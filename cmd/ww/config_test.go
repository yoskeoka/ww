package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yoskeoka/ww/internal/config"
)

func TestNewManagerWithoutGlobalConfigKeepsRepoLocalBehavior(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	repoDir := t.TempDir()
	gitInitForSandboxTest(t, repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, ".ww.toml"), []byte(`worktree_dir = "from-local"`), 0644); err != nil {
		t.Fatal(err)
	}

	withCwd(t, repoDir, func() {
		mgr, err := newManagerWithOptions(false, false)
		if err != nil {
			t.Fatal(err)
		}
		if mgr.Config.WorktreeDir != "from-local" {
			t.Fatalf("WorktreeDir = %q, want from-local", mgr.Config.WorktreeDir)
		}
	})
}

func TestNewManagerLoadsGlobalConfigAndLocalOverridesPerKey(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
worktree_dir = ".global-worktrees"
default_base = "origin/main"
copy_files = [".env"]
sandbox = true
`
	if err := os.WriteFile(filepath.Join(globalDir, config.GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	gitInitForSandboxTest(t, repoDir)
	localConfig := `
default_base = "origin/release"
copy_files = ["local.env"]
`
	if err := os.WriteFile(filepath.Join(repoDir, ".ww.toml"), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	withCwd(t, repoDir, func() {
		mgr, err := newManagerWithOptions(false, false)
		if err != nil {
			t.Fatal(err)
		}
		if mgr.Config.WorktreeDir != ".global-worktrees" {
			t.Fatalf("WorktreeDir = %q, want .global-worktrees", mgr.Config.WorktreeDir)
		}
		if mgr.Config.DefaultBase != "origin/release" {
			t.Fatalf("DefaultBase = %q, want origin/release", mgr.Config.DefaultBase)
		}
		if got, want := mgr.Config.CopyFiles, []string{"local.env"}; len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("CopyFiles = %v, want %v", got, want)
		}
		if !mgr.Config.Sandbox {
			t.Fatal("Sandbox = false, want true from global config")
		}
	})
}

func TestNewManagerUsesMainWorktreeRootForGlobalProjectMatching(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	gitInitForSandboxTest(t, repoDir)
	gitRun(t, repoDir, "config", "user.email", "test@example.com")
	gitRun(t, repoDir, "config", "user.name", "Test User")
	gitRun(t, repoDir, "commit", "--allow-empty", "-m", "initial")

	worktreeDir := filepath.Join(t.TempDir(), "repo-worktree")
	gitRun(t, repoDir, "worktree", "add", "-b", "feat/project-target", worktreeDir, "main")

	globalConfig := `
worktree_dir = "from-global"

[[projects]]
root = "` + repoDir + `"
worktree_dir = "from-project"
sandbox = true
`
	if err := os.WriteFile(filepath.Join(globalDir, config.GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	withCwd(t, worktreeDir, func() {
		mgr, err := newManagerWithOptions(false, false)
		if err != nil {
			t.Fatal(err)
		}
		if mgr.Config.WorktreeDir != "from-project" {
			t.Fatalf("WorktreeDir = %q, want from-project", mgr.Config.WorktreeDir)
		}
		if !mgr.Config.Sandbox {
			t.Fatal("Sandbox = false, want true from project match")
		}
	})
}

func TestManagerForSelectedRepoReloadsConfigForSelectedRepo(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	workspaceRoot := t.TempDir()
	repoA := filepath.Join(workspaceRoot, "repo-a")
	repoB := filepath.Join(workspaceRoot, "repo-b")
	gitInitForSandboxTest(t, repoA)
	gitInitForSandboxTest(t, repoB)

	globalConfig := `
[[projects]]
root = "` + repoA + `"
worktree_dir = "from-repo-a"

[[projects]]
root = "` + repoB + `"
materialization_profile = "repo_b_setup"

[materialization_profiles.repo_b_setup]
copy_files = [".env"]
post_create_hook = "make setup"
`
	if err := os.WriteFile(filepath.Join(globalDir, config.GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	withCwd(t, workspaceRoot, func() {
		mgr, err := managerForSelectedRepo("repo-b", true, false)
		if err != nil {
			t.Fatal(err)
		}
		if mgr.RepoDir != repoB {
			t.Fatalf("RepoDir = %q, want %q", mgr.RepoDir, repoB)
		}
		if mgr.Config.WorktreeDir != "" {
			t.Fatalf("WorktreeDir = %q, want empty", mgr.Config.WorktreeDir)
		}
		if got, want := mgr.Config.CopyFiles, []string{".env"}; len(got) != len(want) || got[0] != want[0] {
			t.Fatalf("CopyFiles = %v, want %v", got, want)
		}
		if mgr.Config.PostCreateHook != "make setup" {
			t.Fatalf("PostCreateHook = %q, want make setup", mgr.Config.PostCreateHook)
		}
	})
}
