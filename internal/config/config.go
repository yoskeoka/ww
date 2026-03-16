package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const FileName = ".ww.toml"

// Config represents the ww configuration.
type Config struct {
	WorktreeDir    string   `toml:"worktree_dir"`
	DefaultBase    string   `toml:"default_base"`
	CopyFiles      []string `toml:"copy_files"`
	SymlinkFiles   []string `toml:"symlink_files"`
	PostCreateHook string   `toml:"post_create_hook"`
}

// Load searches upward from startDir for .ww.toml and parses it.
// If the upward search fails and startDir is inside a git worktree,
// it also checks the main worktree's root directory as a fallback.
// Returns default config if no file is found.
func Load(startDir string) (*Config, error) {
	path := findConfig(startDir)
	if path == "" {
		path = findConfigFromMainWorktree(startDir)
	}
	if path == "" {
		return &Config{}, nil
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// findConfig searches upward from dir for .ww.toml.
func findConfig(dir string) string {
	dir, _ = filepath.Abs(dir)
	for {
		candidate := filepath.Join(dir, FileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// findConfigFromMainWorktree resolves the main worktree directory via git
// and checks for .ww.toml there. Returns empty string if not in a git repo
// or if the config file does not exist in the main worktree.
func findConfigFromMainWorktree(startDir string) string {
	cmd := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir")
	cmd.Dir = startDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	gitCommonDir := strings.TrimRight(string(out), "\n")
	mainWorktreeDir := filepath.Dir(gitCommonDir)

	candidate := filepath.Join(mainWorktreeDir, FileName)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}
