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
