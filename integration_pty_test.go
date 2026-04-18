//go:build !windows

package integration_test

import (
	"context"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/yoskeoka/ww/internal/testutil"
)

func TestInteractivePTYQuitSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	session := startInteractivePTY(t, repo)

	if err := session.WaitForOutput("Select action", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := session.WriteKeys("4\r"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := session.Wait(5 * time.Second)
	if err != nil {
		t.Fatalf("ww i quit via PTY: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("ww i quit should not write stdout, got: %q", stdout)
	}
	if !strings.Contains(stderr, "Interactive mode") {
		t.Fatalf("PTY output should include interactive overview, got:\n%s", stderr)
	}
}

func TestInteractivePTYListOpenSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: integration test")
	}
	t.Parallel()

	repo := setupRepo(t)
	writeConfig(t, repo, `default_base = "main"`)
	if _, err := runWW(t, repo, "create", "feat/pty-open"); err != nil {
		t.Fatal(err)
	}

	session := startInteractivePTY(t, repo)
	if err := session.WaitForOutput("Select action", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := session.WriteKeys("2\r"); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitForOutput("Filter worktrees", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := session.WriteKeys("\r"); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitForOutput("Select worktree", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := session.WriteKeys("2\r"); err != nil {
		t.Fatal(err)
	}
	if err := session.WaitForOutput("Selected worktree", 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := session.WriteKeys("1\r"); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := session.Wait(5 * time.Second)
	if err != nil {
		t.Fatalf("ww i list -> open via PTY: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
	wantPath := path.Join(path.Dir(repo), "myrepo@feat-pty-open")
	if stdout != wantPath+"\n" {
		t.Fatalf("ww i open stdout = %q, want %q\nstderr:\n%s", stdout, wantPath+"\n", stderr)
	}
	if strings.Contains(stdout, "Interactive mode") || strings.Contains(stdout, "Select action") {
		t.Fatalf("stdout should contain only path output, got: %q", stdout)
	}
}

func startInteractivePTY(t *testing.T, repo string) *testutil.PTYSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	session, err := globalEnv.RunWWPTY(ctx, repo, "i")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		session.Close(time.Second)
	})
	return session
}
