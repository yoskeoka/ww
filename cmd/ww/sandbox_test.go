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
			t.Fatal(err)
		}
	}()
	fn()
}
