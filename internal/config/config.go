package config

import (
	"os"
	"path/filepath"

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
// If the upward search fails, it checks each directory in fallbackDirs
// for .ww.toml. Returns default config if no file is found.
func Load(startDir string, fallbackDirs ...string) (*Config, error) {
	path := findConfig(startDir)
	if path == "" {
		path = findConfigInDirs(fallbackDirs)
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

// findConfigInDirs checks each directory for .ww.toml, returning the
// first match. Returns empty string if none found.
func findConfigInDirs(dirs []string) string {
	for _, dir := range dirs {
		candidate := filepath.Join(dir, FileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
