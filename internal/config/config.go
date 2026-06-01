package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const FileName = ".ww.toml"
const GlobalFileName = "config.toml"

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

// Load resolves the global config layer plus any repo-local .ww.toml layer.
// Repo-local values replace same-key global values completely. If no config is
// found, it returns the default zero-value config.
func Load(startDir string, fallbackDirs ...string) (*Config, error) {
	return LoadWithOptions(startDir, LoadOptions{FallbackDirs: fallbackDirs})
}

// LoadWithOptions resolves the global config layer plus any repo-local
// .ww.toml using the provided search options.
func LoadWithOptions(startDir string, opts LoadOptions) (*Config, error) {
	globalPath := globalConfigPath()
	localPath := ""
	if opts.Sandbox {
		localPath = findConfigBounded(startDir, opts.Boundary)
	} else {
		localPath = findConfig(startDir)
	}
	if localPath == "" {
		localPath = findConfigInDirs(opts.FallbackDirs)
	}

	var cfg Config
	if globalPath != "" {
		globalLayer, err := decodeConfigLayer(globalPath)
		if err != nil {
			return nil, err
		}
		mergeConfig(&cfg, globalLayer)
	}

	if localPath != "" {
		localLayer, err := decodeConfigLayer(localPath)
		if err != nil {
			return nil, err
		}
		mergeConfig(&cfg, localLayer)
	}

	return &cfg, nil
}

type configLayer struct {
	cfg          Config
	worktreeDir  bool
	defaultBase  bool
	copyFiles    bool
	symlinkFiles bool
	postHook     bool
	sandbox      bool
}

func decodeConfigLayer(path string) (*configLayer, error) {
	var cfg Config
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}

	return &configLayer{
		cfg:          cfg,
		worktreeDir:  md.IsDefined("worktree_dir"),
		defaultBase:  md.IsDefined("default_base"),
		copyFiles:    md.IsDefined("copy_files"),
		symlinkFiles: md.IsDefined("symlink_files"),
		postHook:     md.IsDefined("post_create_hook"),
		sandbox:      md.IsDefined("sandbox"),
	}, nil
}

func mergeConfig(dst *Config, layer *configLayer) {
	if layer == nil {
		return
	}
	if layer.worktreeDir {
		dst.WorktreeDir = layer.cfg.WorktreeDir
	}
	if layer.defaultBase {
		dst.DefaultBase = layer.cfg.DefaultBase
	}
	if layer.copyFiles {
		dst.CopyFiles = cloneStrings(layer.cfg.CopyFiles)
	}
	if layer.symlinkFiles {
		dst.SymlinkFiles = cloneStrings(layer.cfg.SymlinkFiles)
	}
	if layer.postHook {
		dst.PostCreateHook = layer.cfg.PostCreateHook
	}
	if layer.sandbox {
		dst.Sandbox = layer.cfg.Sandbox
	}
}

func cloneStrings(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func globalConfigPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	candidate := filepath.Join(base, "ww", GlobalFileName)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
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
