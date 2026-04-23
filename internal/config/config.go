package config

import (
	"os"
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
	Sandbox        bool     `toml:"sandbox"`
}

// LoadOptions controls config search behavior.
type LoadOptions struct {
	Sandbox      bool
	Boundary     string
	FallbackDirs []string
}

// Load searches upward from startDir for .ww.toml and parses it.
// If the upward search fails, it checks each directory in fallbackDirs
// for .ww.toml. Returns default config if no file is found.
func Load(startDir string, fallbackDirs ...string) (*Config, error) {
	return LoadWithOptions(startDir, LoadOptions{FallbackDirs: fallbackDirs})
}

// LoadWithOptions searches for .ww.toml using the provided search options and parses it.
func LoadWithOptions(startDir string, opts LoadOptions) (*Config, error) {
	var path string
	if opts.Sandbox {
		path = findConfigBounded(startDir, opts.Boundary)
	} else {
		path = findConfig(startDir)
	}
	if path == "" {
		path = findConfigInDirs(opts.FallbackDirs)
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

// findConfigBounded searches upward from dir for .ww.toml, stopping after
// checking boundary. If boundary is empty, it behaves like an unbounded search.
func findConfigBounded(dir, boundary string) string {
	if boundary == "" {
		return findConfig(dir)
	}
	dir, _ = filepath.Abs(dir)
	boundary, _ = filepath.Abs(boundary)
	for {
		candidate := filepath.Join(dir, FileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		if filepath.Clean(dir) == filepath.Clean(boundary) {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		rel, err := filepath.Rel(boundary, parent)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return ""
		}
		dir = parent
	}
}

// findConfigInDirs checks each directory for .ww.toml, returning the
// first match. Returns empty string if none found.
func findConfigInDirs(dirs []string) string {
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, FileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
