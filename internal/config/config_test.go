package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func disableGlobalConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
}

func TestLoadDefaults(t *testing.T) {
	disableGlobalConfig(t)
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "" {
		t.Errorf("WorktreeDir = %q, want empty", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "" {
		t.Errorf("DefaultBase = %q, want empty", cfg.DefaultBase)
	}
	if len(cfg.CopyFiles) != 0 {
		t.Errorf("CopyFiles = %v, want empty", cfg.CopyFiles)
	}
}

func TestLoadFromFile(t *testing.T) {
	disableGlobalConfig(t)
	dir := t.TempDir()
	content := `
worktree_dir = ".worktrees"
default_base = "origin/main"
copy_files = [".env"]
symlink_files = ["node_modules"]
post_create_hook = "npm install"
sandbox = true
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != ".worktrees" {
		t.Errorf("WorktreeDir = %q, want .worktrees", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "origin/main" {
		t.Errorf("DefaultBase = %q, want origin/main", cfg.DefaultBase)
	}
	if len(cfg.CopyFiles) != 1 || cfg.CopyFiles[0] != ".env" {
		t.Errorf("CopyFiles = %v, want [.env]", cfg.CopyFiles)
	}
	if cfg.PostCreateHook != "npm install" {
		t.Errorf("PostCreateHook = %q, want 'npm install'", cfg.PostCreateHook)
	}
	if !cfg.Sandbox {
		t.Errorf("Sandbox = false, want true")
	}
}

func TestLoadSearchUpward(t *testing.T) {
	disableGlobalConfig(t)
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	content := `worktree_dir = "found"`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(sub)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "found" {
		t.Errorf("WorktreeDir = %q, want found", cfg.WorktreeDir)
	}
}

func TestLoadFallbackDir(t *testing.T) {
	disableGlobalConfig(t)
	// startDir has no config, but fallback dir does
	startDir := t.TempDir()
	fallbackDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(fallbackDir, FileName), []byte(`worktree_dir = "from-fallback"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(startDir, fallbackDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-fallback" {
		t.Errorf("WorktreeDir = %q, want from-fallback", cfg.WorktreeDir)
	}
}

func TestUpwardSearchTakesPriorityOverFallback(t *testing.T) {
	disableGlobalConfig(t)
	parentDir := t.TempDir()
	startDir := filepath.Join(parentDir, "sub")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatal(err)
	}
	fallbackDir := t.TempDir()

	// Config in parent (found via upward search)
	if err := os.WriteFile(filepath.Join(parentDir, FileName), []byte(`worktree_dir = "from-parent"`), 0644); err != nil {
		t.Fatal(err)
	}
	// Config in fallback dir
	if err := os.WriteFile(filepath.Join(fallbackDir, FileName), []byte(`worktree_dir = "from-fallback"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(startDir, fallbackDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-parent" {
		t.Errorf("WorktreeDir = %q, want from-parent (upward search should win)", cfg.WorktreeDir)
	}
}

func TestLoadFallbackDirWithoutConfig(t *testing.T) {
	disableGlobalConfig(t)
	startDir := t.TempDir()
	fallbackDir := t.TempDir() // no .ww.toml here

	cfg, err := Load(startDir, fallbackDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "" {
		t.Errorf("WorktreeDir = %q, want empty (default)", cfg.WorktreeDir)
	}
}

func TestLoadFallbackSkipsEmptyString(t *testing.T) {
	disableGlobalConfig(t)
	startDir := t.TempDir()
	fallbackDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(fallbackDir, FileName), []byte(`worktree_dir = "from-fallback"`), 0644); err != nil {
		t.Fatal(err)
	}

	// Empty string fallback should be skipped, second fallback should be used
	cfg, err := Load(startDir, "", fallbackDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-fallback" {
		t.Errorf("WorktreeDir = %q, want from-fallback", cfg.WorktreeDir)
	}
}

func TestLoadSandboxStopsAtBoundary(t *testing.T) {
	disableGlobalConfig(t)
	parentDir := t.TempDir()
	boundary := filepath.Join(parentDir, "repo")
	startDir := filepath.Join(boundary, "sub")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parentDir, FileName), []byte(`worktree_dir = "from-parent"`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(boundary, FileName), []byte(`worktree_dir = "from-boundary"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(startDir, LoadOptions{Sandbox: true, Boundary: boundary})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-boundary" {
		t.Errorf("WorktreeDir = %q, want from-boundary", cfg.WorktreeDir)
	}
}

func TestLoadSandboxIgnoresConfigAboveBoundary(t *testing.T) {
	disableGlobalConfig(t)
	parentDir := t.TempDir()
	boundary := filepath.Join(parentDir, "repo")
	startDir := filepath.Join(boundary, "sub")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parentDir, FileName), []byte(`worktree_dir = "from-parent"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(startDir, LoadOptions{Sandbox: true, Boundary: boundary})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "" {
		t.Errorf("WorktreeDir = %q, want default", cfg.WorktreeDir)
	}
}

func TestLoadSandboxAllowsMainWorktreeFallback(t *testing.T) {
	disableGlobalConfig(t)
	currentCheckout := t.TempDir()
	mainWorktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(mainWorktree, FileName), []byte(`worktree_dir = "from-main"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(currentCheckout, LoadOptions{
		Sandbox:      true,
		Boundary:     mainWorktree,
		FallbackDirs: []string{mainWorktree},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-main" {
		t.Errorf("WorktreeDir = %q, want from-main", cfg.WorktreeDir)
	}
}

func TestLoadUsesXDGGlobalConfig(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(`worktree_dir = "from-global"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-global" {
		t.Fatalf("WorktreeDir = %q, want from-global", cfg.WorktreeDir)
	}
}

func TestLoadUsesHomeConfigWhenXDGUnset(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", "")

	globalDir := filepath.Join(homeDir, ".config", "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(`worktree_dir = "from-home-global"`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-home-global" {
		t.Fatalf("WorktreeDir = %q, want from-home-global", cfg.WorktreeDir)
	}
}

func TestLoadRepoLocalOverlaysGlobalPerKey(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
worktree_dir = "from-global"
default_base = "origin/main"
copy_files = [".env", ".tool-versions"]
symlink_files = ["node_modules"]
post_create_hook = "global-hook"
sandbox = true
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	localConfig := `
default_base = "origin/release"
copy_files = ["local.env"]
post_create_hook = "local-hook"
`
	if err := os.WriteFile(filepath.Join(repoDir, FileName), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-global" {
		t.Fatalf("WorktreeDir = %q, want from-global", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "origin/release" {
		t.Fatalf("DefaultBase = %q, want origin/release", cfg.DefaultBase)
	}
	if got, want := cfg.CopyFiles, []string{"local.env"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("CopyFiles = %v, want %v", got, want)
	}
	if got, want := cfg.SymlinkFiles, []string{"node_modules"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("SymlinkFiles = %v, want %v", got, want)
	}
	if cfg.PostCreateHook != "local-hook" {
		t.Fatalf("PostCreateHook = %q, want local-hook", cfg.PostCreateHook)
	}
	if !cfg.Sandbox {
		t.Fatal("Sandbox = false, want true from global config")
	}
}

func TestLoadSelectedRepoLocalConfigUsesProjectRootAndNearestRepoLocalConfig(t *testing.T) {
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
	if err := os.MkdirAll(repoA, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(repoB, 0755); err != nil {
		t.Fatal(err)
	}

	globalConfig := `
[[projects]]
root = "` + repoA + `"
worktree_dir = "from-repo-a"

[[projects]]
root = "` + repoB + `"
default_base = "origin/release"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspaceRoot, FileName), []byte(`copy_files = ["workspace.env"]`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoB, FileName), []byte(`copy_files = ["repo-b.env"]`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(repoB, LoadOptions{
		FallbackDirs: []string{repoB, workspaceRoot},
		ProjectRoot:  repoB,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "" {
		t.Fatalf("WorktreeDir = %q, want empty", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "origin/release" {
		t.Fatalf("DefaultBase = %q, want origin/release", cfg.DefaultBase)
	}
	if got, want := cfg.CopyFiles, []string{"repo-b.env"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("CopyFiles = %v, want %v", got, want)
	}
}

func TestLoadSelectedRepoLocalConfigUsesProjectRootAndNearestSandboxRepoLocalConfig(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	workspaceRoot := t.TempDir()
	repoB := filepath.Join(workspaceRoot, "repo-b")
	if err := os.MkdirAll(repoB, 0755); err != nil {
		t.Fatal(err)
	}

	globalConfig := `
[[projects]]
root = "` + repoB + `"
default_base = "origin/release"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspaceRoot, FileName), []byte(`copy_files = ["workspace.env"]`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoB, FileName), []byte(`copy_files = ["repo-b.env"]`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(repoB, LoadOptions{
		Sandbox:      true,
		Boundary:     workspaceRoot,
		FallbackDirs: []string{repoB},
		ProjectRoot:  repoB,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultBase != "origin/release" {
		t.Fatalf("DefaultBase = %q, want origin/release", cfg.DefaultBase)
	}
	if got, want := cfg.CopyFiles, []string{"repo-b.env"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("CopyFiles = %v, want %v", got, want)
	}
}

func TestLoadRepoLocalReplacesArraysHooksAndFalseValues(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
copy_files = [".env", ".tool-versions"]
post_create_hook = "global-hook"
sandbox = true
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	localConfig := `
copy_files = []
post_create_hook = ""
sandbox = false
`
	if err := os.WriteFile(filepath.Join(repoDir, FileName), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CopyFiles) != 0 {
		t.Fatalf("CopyFiles = %v, want empty replacement", cfg.CopyFiles)
	}
	if cfg.PostCreateHook != "" {
		t.Fatalf("PostCreateHook = %q, want empty replacement", cfg.PostCreateHook)
	}
	if cfg.Sandbox {
		t.Fatal("Sandbox = true, want false replacement")
	}
}

func TestLoadGlobalProjectExactRootMatchOverridesBaseConfig(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
worktree_dir = "from-global"

[[projects]]
root = "/target/repo"
worktree_dir = "from-project"
default_base = "origin/main"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(t.TempDir(), LoadOptions{ProjectRoot: "/target/repo"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-project" {
		t.Fatalf("WorktreeDir = %q, want from-project", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "origin/main" {
		t.Fatalf("DefaultBase = %q, want origin/main", cfg.DefaultBase)
	}
}

func TestLoadMaterializationProfileExpandsInBaseGlobalConfig(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
materialization_profile = "dev_setup"

[materialization_profiles.dev_setup]
copy_files = [".env"]
symlink_files = ["node_modules"]
post_create_hook = "make setup"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.CopyFiles, []string{".env"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("CopyFiles = %v, want %v", got, want)
	}
	if got, want := cfg.SymlinkFiles, []string{"node_modules"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("SymlinkFiles = %v, want %v", got, want)
	}
	if cfg.PostCreateHook != "make setup" {
		t.Fatalf("PostCreateHook = %q, want make setup", cfg.PostCreateHook)
	}
}

func TestLoadGlobalProjectFirstMatchWins(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
[[projects]]
root_prefix = "/workspace"
worktree_dir = "first"

[[projects]]
root = "/workspace/repo-a"
worktree_dir = "second"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(t.TempDir(), LoadOptions{ProjectRoot: "/workspace/repo-a"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "first" {
		t.Fatalf("WorktreeDir = %q, want first", cfg.WorktreeDir)
	}
}

func TestLoadGlobalProjectRootPrefixMatchesFilesystemRoot(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
[[projects]]
root_prefix = "/"
worktree_dir = "from-root-prefix"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(t.TempDir(), LoadOptions{ProjectRoot: "/workspace/repo-a"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-root-prefix" {
		t.Fatalf("WorktreeDir = %q, want from-root-prefix", cfg.WorktreeDir)
	}
}

func TestHasPathPrefixIsSegmentAware(t *testing.T) {
	if hasPathPrefix("/workspace/repo2", "/workspace/repo") {
		t.Fatal("hasPathPrefix matched sibling path, want false")
	}
	if !hasPathPrefix("/workspace/repo/sub", "/workspace/repo") {
		t.Fatal("hasPathPrefix = false, want true for child path")
	}
}

func TestLoadGlobalProjectUnmatchedLeavesBaseGlobalConfig(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
worktree_dir = "from-global"

[[projects]]
root = "/workspace/repo-a"
default_base = "origin/main"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(t.TempDir(), LoadOptions{ProjectRoot: "/workspace/repo-b"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorktreeDir != "from-global" {
		t.Fatalf("WorktreeDir = %q, want from-global", cfg.WorktreeDir)
	}
	if cfg.DefaultBase != "" {
		t.Fatalf("DefaultBase = %q, want empty", cfg.DefaultBase)
	}
}

func TestLoadRepoLocalOverridesSelectedProjectPerKey(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
worktree_dir = "from-global"
copy_files = [".env"]

[[projects]]
root = "/workspace/repo-a"
copy_files = ["project.env"]
post_create_hook = "project-hook"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	localConfig := `
copy_files = ["local.env"]
`
	if err := os.WriteFile(filepath.Join(repoDir, FileName), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadWithOptions(repoDir, LoadOptions{ProjectRoot: "/workspace/repo-a"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.CopyFiles, []string{"local.env"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("CopyFiles = %v, want %v", got, want)
	}
	if cfg.PostCreateHook != "project-hook" {
		t.Fatalf("PostCreateHook = %q, want project-hook", cfg.PostCreateHook)
	}
}

func TestLoadRepoLocalMaterializationProfileOverridesGlobalMaterializationFields(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
materialization_profile = "global_setup"

[materialization_profiles.global_setup]
copy_files = [".env"]
post_create_hook = "global-hook"

[materialization_profiles.repo_setup]
copy_files = ["repo.env"]
symlink_files = ["node_modules"]
post_create_hook = "repo-hook"
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	localConfig := `
materialization_profile = "repo_setup"
`
	if err := os.WriteFile(filepath.Join(repoDir, FileName), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.CopyFiles, []string{"repo.env"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("CopyFiles = %v, want %v", got, want)
	}
	if got, want := cfg.SymlinkFiles, []string{"node_modules"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("SymlinkFiles = %v, want %v", got, want)
	}
	if cfg.PostCreateHook != "repo-hook" {
		t.Fatalf("PostCreateHook = %q, want repo-hook", cfg.PostCreateHook)
	}
}

func TestLoadEmptyMaterializationProfileActsAsUnset(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	t.Setenv("HOME", t.TempDir())

	globalDir := filepath.Join(xdgDir, "ww")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	globalConfig := `
[materialization_profiles.dev_setup]
copy_files = [".env"]
`
	if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(globalConfig), 0644); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	localConfig := `
materialization_profile = "   "
copy_files = ["local.env"]
`
	if err := os.WriteFile(filepath.Join(repoDir, FileName), []byte(localConfig), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.CopyFiles, []string{"local.env"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("CopyFiles = %v, want %v", got, want)
	}
}

func TestLoadGlobalProjectRejectsInvalidTargetDefinitions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "both root and prefix",
			content: `
[[projects]]
root = "/workspace/repo-a"
root_prefix = "/workspace"
`,
			want: "define exactly one of root or root_prefix",
		},
		{
			name: "missing target",
			content: `
[[projects]]
worktree_dir = "from-project"
`,
			want: "define exactly one of root or root_prefix",
		},
		{
			name: "relative root",
			content: `
[[projects]]
root = "workspace/repo-a"
`,
			want: "root must be an absolute path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdgDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdgDir)
			t.Setenv("HOME", t.TempDir())

			globalDir := filepath.Join(xdgDir, "ww")
			if err := os.MkdirAll(globalDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := LoadWithOptions(t.TempDir(), LoadOptions{ProjectRoot: "/workspace/repo-a"})
			if err == nil {
				t.Fatal("LoadWithOptions error = nil, want invalid project error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestLoadRejectsInvalidMaterializationProfileConfig(t *testing.T) {
	tests := []struct {
		name   string
		global string
		local  string
		want   string
	}{
		{
			name: "unknown profile reference",
			global: `
[materialization_profiles.dev_setup]
copy_files = [".env"]
`,
			local: `materialization_profile = "missing"`,
			want:  `unknown materialization_profile "missing"`,
		},
		{
			name: "mixed profile and direct fields",
			global: `
[materialization_profiles.dev_setup]
copy_files = [".env"]
`,
			local: `
materialization_profile = "dev_setup"
copy_files = ["local.env"]
`,
			want: "materialization_profile cannot be combined with copy_files, symlink_files, or post_create_hook",
		},
		{
			name: "empty profile definition",
			global: `
[materialization_profiles.empty]
`,
			want: "define at least one of copy_files, symlink_files, or post_create_hook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xdgDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdgDir)
			t.Setenv("HOME", t.TempDir())

			globalDir := filepath.Join(xdgDir, "ww")
			if err := os.MkdirAll(globalDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(globalDir, GlobalFileName), []byte(tt.global), 0644); err != nil {
				t.Fatal(err)
			}

			repoDir := t.TempDir()
			if tt.local != "" {
				if err := os.WriteFile(filepath.Join(repoDir, FileName), []byte(tt.local), 0644); err != nil {
					t.Fatal(err)
				}
			}

			_, err := Load(repoDir)
			if err == nil {
				t.Fatal("Load error = nil, want invalid materialization profile error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}
