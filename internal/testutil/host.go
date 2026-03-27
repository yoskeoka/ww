// Package testutil provides host-based test helpers for ww integration tests.
// Tests use a shared HostEnv (started once in TestMain) with per-test
// isolated directories on the host filesystem.
package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// HostEnv runs ww and git commands directly on the host machine.
// It is safe to share across tests; each test should call MkdirTemp to get an
// isolated working directory.
type HostEnv struct {
	wwBinaryPath  string
	gitConfigPath string
	ctx           context.Context
}

// NewHostEnv builds the ww binary for the host OS/arch, creates a temporary
// git config file with a test identity, and returns a ready-to-use HostEnv.
// Call Terminate when done (typically via defer in TestMain).
func NewHostEnv(ctx context.Context) (*HostEnv, error) {
	binPath, err := buildWWBinaryHost()
	if err != nil {
		return nil, fmt.Errorf("build ww: %w", err)
	}

	gitCfg, err := os.CreateTemp("", "ww-test-gitconfig-*")
	if err != nil {
		return nil, fmt.Errorf("create temp gitconfig: %w", err)
	}
	gitConfigPath := gitCfg.Name()
	gitCfg.Close()

	env := &HostEnv{
		wwBinaryPath:  binPath,
		gitConfigPath: gitConfigPath,
		ctx:           ctx,
	}

	for _, cfg := range [][]string{
		{"git", "config", "--global", "user.email", "test@test.com"},
		{"git", "config", "--global", "user.name", "Test User"},
	} {
		if _, err := env.Exec("", cfg[0], cfg[1:]...); err != nil {
			env.Terminate()
			return nil, fmt.Errorf("git config: %w", err)
		}
	}

	return env, nil
}

// Terminate cleans up temporary files.
func (e *HostEnv) Terminate() {
	os.Remove(e.wwBinaryPath)
	os.Remove(e.gitConfigPath)
}

// MkdirTemp creates a uniquely named temporary directory and returns its
// real absolute path (symlinks resolved). This is important on macOS where
// /tmp is a symlink to /private/tmp, and git resolves to the real path.
func (e *HostEnv) MkdirTemp(prefix string) (string, error) {
	dir, err := os.MkdirTemp("", prefix+"-")
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(dir)
}

// MkdirAll creates a directory and any missing parents.
func (e *HostEnv) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

// WriteFile writes content to a file on the host.
func (e *HostEnv) WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// ReadFile reads and returns the contents of a file on the host.
func (e *HostEnv) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}

// PathExists returns true if the path exists on the host.
func (e *HostEnv) PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsSymlink returns true if path is a symbolic link on the host.
func (e *HostEnv) IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// Git runs a git command with repoPath as the working directory.
func (e *HostEnv) Git(repoPath string, args ...string) (string, error) {
	return e.Exec(repoPath, "git", args...)
}

// RunWW runs the ww binary with dir as the working directory.
func (e *HostEnv) RunWW(dir string, args ...string) (string, error) {
	return e.Exec(dir, e.wwBinaryPath, args...)
}

// Exec runs cmd with args on the host, optionally in the given working
// directory. Stdout and stderr are combined and returned. A non-zero exit code
// is returned as an error.
func (e *HostEnv) Exec(dir string, cmd string, args ...string) (string, error) {
	c := exec.CommandContext(e.ctx, cmd, args...)
	if dir != "" {
		c.Dir = dir
	}
	c.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+e.gitConfigPath)

	out, err := c.CombinedOutput()
	outStr := string(out)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return outStr, fmt.Errorf("exit %d", exitErr.ExitCode())
		}
		return outStr, err
	}
	return outStr, nil
}

// buildWWBinaryHost compiles ww for the host OS/arch.
func buildWWBinaryHost() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOMOD: %w", err)
	}
	modDir := filepath.Dir(strings.TrimSpace(string(out)))

	tmpFile, err := os.CreateTemp("", "ww-test-host-*")
	if err != nil {
		return "", fmt.Errorf("create temp ww binary: %w", err)
	}
	binPath := tmpFile.Name()
	tmpFile.Close()

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/ww/")
	cmd.Dir = modDir
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build: %w", err)
	}
	return binPath, nil
}
