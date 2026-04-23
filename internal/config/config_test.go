package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
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
