// Package testutil provides Docker-based test helpers for ww integration tests.
// Tests use a shared ContainerEnv (started once in TestMain) with per-test
// isolated directories inside the container.
package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/pkg/stdcopy"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

const containerImage = "golang:1.23"

// ContainerEnv is a running Docker container with the ww binary and git pre-configured.
// It is safe to share across tests; each test should call MkdirTemp to get an
// isolated working directory inside the container.
type ContainerEnv struct {
	container testcontainers.Container
	ctx       context.Context
}

// NewContainerEnv builds the ww binary, starts a container, copies the binary
// in, and configures a global git identity.
// Call Terminate when done (typically via defer in TestMain).
func NewContainerEnv(ctx context.Context) (*ContainerEnv, error) {
	binPath, err := buildWWBinary()
	if err != nil {
		return nil, fmt.Errorf("build ww: %w", err)
	}

	req := testcontainers.ContainerRequest{
		Image: containerImage,
		Cmd:   []string{"sleep", "infinity"},
		Env: map[string]string{
			// Prevent host git config from leaking into the container.
			"GIT_CONFIG_GLOBAL": "/dev/null",
		},
		WaitingFor: wait.ForExec([]string{"true"}),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	env := &ContainerEnv{container: c, ctx: ctx}

	if err := c.CopyFileToContainer(ctx, binPath, "/usr/local/bin/ww", 0755); err != nil {
		c.Terminate(ctx) //nolint:errcheck
		return nil, fmt.Errorf("copy ww binary: %w", err)
	}

	// Set a minimal git identity so commits inside the container work.
	for _, cfg := range [][]string{
		{"git", "config", "--global", "user.email", "test@test.com"},
		{"git", "config", "--global", "user.name", "Test User"},
	} {
		if _, err := env.Exec("", cfg[0], cfg[1:]...); err != nil {
			c.Terminate(ctx) //nolint:errcheck
			return nil, fmt.Errorf("git config: %w", err)
		}
	}

	return env, nil
}

// Terminate stops and removes the container.
func (e *ContainerEnv) Terminate() {
	e.container.Terminate(e.ctx) //nolint:errcheck
}

// MkdirTemp creates a uniquely named temporary directory inside the container
// and returns its absolute path.
func (e *ContainerEnv) MkdirTemp(prefix string) (string, error) {
	out, err := e.Exec("", "mktemp", "-d", fmt.Sprintf("/tmp/%s-XXXXXX", prefix))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// MkdirAll creates a directory (and any missing parents) inside the container.
func (e *ContainerEnv) MkdirAll(path string) error {
	_, err := e.Exec("", "mkdir", "-p", path)
	return err
}

// WriteFile writes content to a file inside the container.
// The parent directory must already exist.
func (e *ContainerEnv) WriteFile(path, content string) error {
	tmp, err := os.CreateTemp("", "ww-testutil-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.WriteString(tmp, content); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	return e.container.CopyFileToContainer(e.ctx, tmp.Name(), path, 0644)
}

// ReadFile reads and returns the contents of a file inside the container.
func (e *ContainerEnv) ReadFile(path string) (string, error) {
	out, err := e.Exec("", "cat", path)
	return out, err
}

// PathExists returns true if the path exists inside the container.
func (e *ContainerEnv) PathExists(path string) bool {
	_, err := e.Exec("", "test", "-e", path)
	return err == nil
}

// IsSymlink returns true if path is a symbolic link inside the container.
func (e *ContainerEnv) IsSymlink(path string) bool {
	_, err := e.Exec("", "test", "-L", path)
	return err == nil
}

// Git runs a git command inside the container with repoPath as the working directory.
func (e *ContainerEnv) Git(repoPath string, args ...string) (string, error) {
	return e.Exec(repoPath, "git", args...)
}

// RunWW runs the ww binary inside the container with dir as the working directory.
func (e *ContainerEnv) RunWW(dir string, args ...string) (string, error) {
	return e.Exec(dir, "/usr/local/bin/ww", args...)
}

// Exec runs cmd with args inside the container, optionally in the given working
// directory. Stdout and stderr are combined and returned. A non-zero exit code
// is returned as an error.
func (e *ContainerEnv) Exec(dir string, cmd string, args ...string) (string, error) {
	fullArgs := append([]string{cmd}, args...)

	var shellScript string
	if dir != "" {
		shellScript = fmt.Sprintf("cd %s && %s 2>&1", shellEscape(dir), shellJoin(fullArgs))
	} else {
		shellScript = shellJoin(fullArgs) + " 2>&1"
	}

	exitCode, reader, err := e.container.Exec(e.ctx, []string{"sh", "-c", shellScript}, tcexec.Multiplexed())
	if err != nil {
		return "", fmt.Errorf("exec: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, reader); err != nil {
		return "", fmt.Errorf("read output: %w", err)
	}
	// combined output (stderr already merged into stdout via 2>&1 in the shell
	// command, but we union both buffers for safety)
	out := stdout.String() + stderr.String()

	if exitCode != 0 {
		return out, fmt.Errorf("exit %d", exitCode)
	}
	return out, nil
}

// shellEscape wraps s in single quotes, escaping any embedded single quotes.
func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// shellJoin shell-escapes and joins args into a single string.
func shellJoin(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = shellEscape(a)
	}
	return strings.Join(parts, " ")
}

// buildWWBinary cross-compiles ww for linux/<host-arch> with CGO disabled,
// producing a static binary suitable for any Linux container.
func buildWWBinary() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	modDir := filepath.Dir(strings.TrimSpace(string(out)))

	binPath := filepath.Join(os.TempDir(), "ww-test-linux")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/ww/")
	cmd.Dir = modDir
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH="+runtime.GOARCH,
		"CGO_ENABLED=0",
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build: %w", err)
	}
	return binPath, nil
}
