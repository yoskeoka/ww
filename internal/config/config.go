package config

import (
	"fmt"
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
	PreCreateHook  string   `toml:"pre_create_hook"`
	PostCreateHook string   `toml:"post_create_hook"`
	PreRemoveHook  string   `toml:"pre_remove_hook"`
	PostRemoveHook string   `toml:"post_remove_hook"`
	Sandbox        bool     `toml:"sandbox"`
}

// LoadOptions controls config search behavior.
type LoadOptions struct {
	Sandbox      bool
	Boundary     string
	FallbackDirs []string
	ProjectRoot  string
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
	var profiles map[string]*configLayer
	if globalPath != "" {
		globalLayer, projectLayer, decodedProfiles, err := decodeGlobalConfig(globalPath, opts.ProjectRoot)
		if err != nil {
			return nil, err
		}
		profiles = decodedProfiles
		mergeConfig(&cfg, globalLayer)
		mergeConfig(&cfg, projectLayer)
	}

	if localPath != "" {
		localLayer, err := decodeConfigLayer(localPath)
		if err != nil {
			return nil, err
		}
		if err := expandMaterializationProfile(localLayer, profiles); err != nil {
			return nil, err
		}
		mergeConfig(&cfg, localLayer)
	}

	return &cfg, nil
}

type configLayer struct {
	cfg                    Config
	worktreeDir            bool
	defaultBase            bool
	copyFiles              bool
	symlinkFiles           bool
	preCreateHook          bool
	postHook               bool
	preRemoveHook          bool
	postRemoveHook         bool
	sandbox                bool
	materializationProfile string
	hasMaterialization     bool
}

type configFields struct {
	WorktreeDir            *string   `toml:"worktree_dir"`
	DefaultBase            *string   `toml:"default_base"`
	CopyFiles              *[]string `toml:"copy_files"`
	SymlinkFiles           *[]string `toml:"symlink_files"`
	PreCreateHook          *string   `toml:"pre_create_hook"`
	PostCreateHook         *string   `toml:"post_create_hook"`
	PreRemoveHook          *string   `toml:"pre_remove_hook"`
	PostRemoveHook         *string   `toml:"post_remove_hook"`
	MaterializationProfile *string   `toml:"materialization_profile"`
	Sandbox                *bool     `toml:"sandbox"`
}

type globalConfigDocument struct {
	configFields
	Projects                []projectDocument                `toml:"projects"`
	MaterializationProfiles map[string]materializationFields `toml:"materialization_profiles"`
}

type projectDocument struct {
	configFields
	Root       *string `toml:"root"`
	RootPrefix *string `toml:"root_prefix"`
}

type materializationFields struct {
	CopyFiles      *[]string `toml:"copy_files"`
	SymlinkFiles   *[]string `toml:"symlink_files"`
	PostCreateHook *string   `toml:"post_create_hook"`
}

func decodeConfigLayer(path string) (*configLayer, error) {
	var doc configFields
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return nil, err
	}
	return decodeFields(doc), nil
}

func decodeGlobalConfig(path, projectRoot string) (*configLayer, *configLayer, map[string]*configLayer, error) {
	var doc globalConfigDocument
	if _, err := toml.DecodeFile(path, &doc); err != nil {
		return nil, nil, nil, err
	}

	baseLayer := decodeFields(doc.configFields)
	profiles, err := decodeMaterializationProfiles(doc.MaterializationProfiles)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := expandMaterializationProfile(baseLayer, profiles); err != nil {
		return nil, nil, nil, err
	}
	projectLayer, err := selectProjectLayer(doc.Projects, projectRoot)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := expandMaterializationProfile(projectLayer, profiles); err != nil {
		return nil, nil, nil, err
	}
	return baseLayer, projectLayer, profiles, nil
}

func decodeFields(fields configFields) *configLayer {
	layer := &configLayer{}
	if fields.WorktreeDir != nil {
		layer.cfg.WorktreeDir = *fields.WorktreeDir
		layer.worktreeDir = true
	}
	if fields.DefaultBase != nil {
		layer.cfg.DefaultBase = *fields.DefaultBase
		layer.defaultBase = true
	}
	if fields.CopyFiles != nil {
		layer.cfg.CopyFiles = cloneStrings(*fields.CopyFiles)
		layer.copyFiles = true
	}
	if fields.SymlinkFiles != nil {
		layer.cfg.SymlinkFiles = cloneStrings(*fields.SymlinkFiles)
		layer.symlinkFiles = true
	}
	if fields.PreCreateHook != nil {
		layer.cfg.PreCreateHook = *fields.PreCreateHook
		layer.preCreateHook = true
	}
	if fields.PostCreateHook != nil {
		layer.cfg.PostCreateHook = *fields.PostCreateHook
		layer.postHook = true
	}
	if fields.PreRemoveHook != nil {
		layer.cfg.PreRemoveHook = *fields.PreRemoveHook
		layer.preRemoveHook = true
	}
	if fields.PostRemoveHook != nil {
		layer.cfg.PostRemoveHook = *fields.PostRemoveHook
		layer.postRemoveHook = true
	}
	if fields.MaterializationProfile != nil {
		layer.materializationProfile = strings.TrimSpace(*fields.MaterializationProfile)
		layer.hasMaterialization = layer.materializationProfile != ""
	}
	if fields.Sandbox != nil {
		layer.cfg.Sandbox = *fields.Sandbox
		layer.sandbox = true
	}
	return layer
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
	if layer.preCreateHook {
		dst.PreCreateHook = layer.cfg.PreCreateHook
	}
	if layer.postHook {
		dst.PostCreateHook = layer.cfg.PostCreateHook
	}
	if layer.preRemoveHook {
		dst.PreRemoveHook = layer.cfg.PreRemoveHook
	}
	if layer.postRemoveHook {
		dst.PostRemoveHook = layer.cfg.PostRemoveHook
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

func decodeMaterializationProfiles(profiles map[string]materializationFields) (map[string]*configLayer, error) {
	if len(profiles) == 0 {
		return nil, nil
	}

	decoded := make(map[string]*configLayer, len(profiles))
	for name, profile := range profiles {
		layer := &configLayer{}
		if profile.CopyFiles != nil {
			layer.cfg.CopyFiles = cloneStrings(*profile.CopyFiles)
			layer.copyFiles = true
		}
		if profile.SymlinkFiles != nil {
			layer.cfg.SymlinkFiles = cloneStrings(*profile.SymlinkFiles)
			layer.symlinkFiles = true
		}
		if profile.PostCreateHook != nil {
			layer.cfg.PostCreateHook = *profile.PostCreateHook
			layer.postHook = true
		}
		if !layer.copyFiles && !layer.symlinkFiles && !layer.postHook {
			return nil, fmt.Errorf("invalid materialization_profiles.%s: define at least one of copy_files, symlink_files, or post_create_hook", name)
		}
		decoded[name] = layer
	}
	return decoded, nil
}

func expandMaterializationProfile(layer *configLayer, profiles map[string]*configLayer) error {
	if layer == nil || !layer.hasMaterialization {
		return nil
	}
	if layer.copyFiles || layer.symlinkFiles || layer.postHook {
		return fmt.Errorf("invalid config: materialization_profile cannot be combined with copy_files, symlink_files, or post_create_hook in the same config layer")
	}
	profile, ok := profiles[layer.materializationProfile]
	if !ok {
		return fmt.Errorf("unknown materialization_profile %q", layer.materializationProfile)
	}
	if profile.copyFiles {
		layer.cfg.CopyFiles = cloneStrings(profile.cfg.CopyFiles)
		layer.copyFiles = true
	}
	if profile.symlinkFiles {
		layer.cfg.SymlinkFiles = cloneStrings(profile.cfg.SymlinkFiles)
		layer.symlinkFiles = true
	}
	if profile.postHook {
		layer.cfg.PostCreateHook = profile.cfg.PostCreateHook
		layer.postHook = true
	}
	return nil
}

func selectProjectLayer(projects []projectDocument, projectRoot string) (*configLayer, error) {
	if len(projects) == 0 {
		return nil, nil
	}
	for i, project := range projects {
		if err := validateProjectDocument(project, i); err != nil {
			return nil, err
		}
	}
	if projectRoot == "" {
		return nil, nil
	}

	cleanRoot := filepath.Clean(projectRoot)
	for _, project := range projects {
		if projectMatches(cleanRoot, project) {
			return decodeFields(project.configFields), nil
		}
	}
	return nil, nil
}

func validateProjectDocument(project projectDocument, index int) error {
	hasRoot := project.Root != nil
	hasPrefix := project.RootPrefix != nil
	if hasRoot == hasPrefix {
		return fmt.Errorf("invalid projects[%d]: define exactly one of root or root_prefix", index)
	}

	targetValue := ""
	targetName := ""
	if hasRoot {
		targetName = "root"
		targetValue = strings.TrimSpace(*project.Root)
	} else {
		targetName = "root_prefix"
		targetValue = strings.TrimSpace(*project.RootPrefix)
	}
	if targetValue == "" {
		return fmt.Errorf("invalid projects[%d]: %s must not be empty", index, targetName)
	}
	if !filepath.IsAbs(targetValue) {
		return fmt.Errorf("invalid projects[%d]: %s must be an absolute path", index, targetName)
	}
	return nil
}

func projectMatches(projectRoot string, project projectDocument) bool {
	if project.Root != nil {
		return filepath.Clean(projectRoot) == filepath.Clean(strings.TrimSpace(*project.Root))
	}
	if project.RootPrefix != nil {
		return hasPathPrefix(filepath.Clean(projectRoot), filepath.Clean(strings.TrimSpace(*project.RootPrefix)))
	}
	return false
}

func hasPathPrefix(path, prefix string) bool {
	rel, err := filepath.Rel(prefix, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
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
