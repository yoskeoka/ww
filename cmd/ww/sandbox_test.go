package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/yoskeoka/ww/workspace"
)

func TestSandboxRepoSelectionRejectedInsideChildRepo(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	gitInitForSandboxTest(t, repoA)
	gitInitForSandboxTest(t, repoB)

	withCwd(t, repoA, func() {
		_, err := managerForSelectedRepo("repo-b", true, true)
		if err == nil {
			t.Fatal("managerForSelectedRepo error = nil, want --repo rejection")
		}
		if got, want := err.Error(), "--repo can only be used inside a detected workspace"; got != want {
			t.Fatalf("error = %q, want %q", got, want)
		}
	})
}

func TestSandboxRepoSelectionAllowedFromCurrentDirectoryWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	gitInitForSandboxTest(t, repoA)
	gitInitForSandboxTest(t, repoB)

	withCwd(t, root, func() {
		mgr, err := managerForSelectedRepo("repo-b", true, true)
		if err != nil {
			t.Fatal(err)
		}
		if mgr.RepoDir != repoB {
			t.Fatalf("RepoDir = %q, want %q", mgr.RepoDir, repoB)
		}
		if mgr.Workspace == nil || mgr.Workspace.Mode != workspace.ModeWorkspace {
			t.Fatalf("Workspace = %+v, want workspace mode", mgr.Workspace)
		}
		if !mgr.Config.Sandbox {
			t.Fatal("Config.Sandbox = false, want true")
		}
	})
}

func TestSandboxConfigFoundViaFallbackRerunsDetectionInSandboxMode(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "repo-a")
	repoB := filepath.Join(root, "repo-b")
	gitInitForSandboxTest(t, repoA)
	gitInitForSandboxTest(t, repoB)

	gitRun(t, repoA, "config", "user.email", "test@example.com")
	gitRun(t, repoA, "config", "user.name", "Test User")
	gitRun(t, repoA, "commit", "--allow-empty", "-m", "initial")

	wtDir := filepath.Join(root, "repo-a-worktrees", "feat-sandbox")
	if err := os.MkdirAll(filepath.Dir(wtDir), 0755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repoA, "worktree", "add", "-b", "feat/sandbox", wtDir, "main")

	if err := os.WriteFile(filepath.Join(repoA, ".ww.toml"), []byte("sandbox = true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	withCwd(t, wtDir, func() {
		mgr, err := newManagerWithOptions(false, false)
		if err != nil {
			t.Fatal(err)
		}
		if !mgr.Config.Sandbox {
			t.Fatal("Config.Sandbox = false, want true from fallback-loaded config")
		}
		if mgr.Workspace == nil {
			t.Fatal("Workspace = nil, want single-repo workspace")
		}
		if mgr.Workspace.Mode != workspace.ModeSingleRepo {
			t.Fatalf("Workspace.Mode = %q, want %q", mgr.Workspace.Mode, workspace.ModeSingleRepo)
		}
		if mgr.Workspace.Root != repoA {
			t.Fatalf("Workspace.Root = %q, want %q", mgr.Workspace.Root, repoA)
		}
	})
}

func gitInitForSandboxTest(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init in %s: %v\n%s", dir, err, string(out))
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, string(out))
	}
}

func withCwd(t *testing.T, dir string, fn func()) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(old); err != nil {
			t.Errorf("restore working directory to %q: %v", old, err)
		}
	}()
	fn()
}
