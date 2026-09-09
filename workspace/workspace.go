package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yoskeoka/ww/git"
	"github.com/yoskeoka/ww/internal/cache"
)

// Mode identifies how ww should treat the detected directory tree.
type Mode string

const (
	// ModeSingleRepo means ww is operating on one repository only.
	ModeSingleRepo Mode = "single-repo"
	// ModeWorkspace means ww detected a workspace with multiple repositories.
	ModeWorkspace Mode = "workspace"
)

// Repo describes a detected git repository.
type Repo struct {
	Name string
	Path string
}

// Workspace describes the detected workspace layout.
type Workspace struct {
	Root     string
	Repos    []Repo
	Mode     Mode
	MainRoot string
}

// DetectOptions controls workspace discovery behavior.
type DetectOptions struct {
	Sandbox bool
	Cache   cache.DiscoveryCache
}

// ErrNotGitRepository is returned when detection finds no git repository and
// no valid workspace root.
var ErrNotGitRepository = errors.New("not a git repository")

// Detect inspects startDir and returns the detected workspace layout.
func Detect(startDir string) (*Workspace, error) {
	return DetectWithOptions(startDir, DetectOptions{})
}

// DetectWithOptions inspects startDir and returns the detected workspace layout.
func DetectWithOptions(startDir string, opts DetectOptions) (*Workspace, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}
	if opts.Cache != nil {
		if topology, ok := opts.Cache.Load(absStart, opts.Sandbox); ok {
			if ws, err := workspaceFromTopology(topology); err == nil {
				return ws, nil
			}
		}
	}

	discovery := newDiscoveryContext()
	complete := func(ws *Workspace) (*Workspace, error) {
		if opts.Cache != nil {
			repos := make([]cache.Repo, 0, len(ws.Repos))
			for _, repo := range ws.Repos {
				repos = append(repos, cache.Repo{Name: repo.Name, Path: repo.Path})
			}
			candidatePaths := discovery.scannedDirs()
			if opts.Sandbox {
				candidatePaths = append(candidatePaths, ws.Root, ws.MainRoot)
			}
			if topology, ok := cache.NewTopology(absStart, opts.Sandbox, ws.Root, ws.MainRoot, string(ws.Mode), repos, candidatePaths); ok {
				opts.Cache.Save(topology)
			}
		}
		return ws, nil
	}
	childRepos, err := discovery.scanImmediateRepos(absStart)
	if err != nil {
		return nil, err
	}

	if opts.Sandbox && len(childRepos) > 0 {
		repos := childRepos
		if ok, err := isStandaloneRepoRoot(absStart); err != nil {
			return nil, err
		} else if ok {
			repos = append(repos, Repo{Name: filepath.Base(absStart), Path: absStart})
		}
		return complete(&Workspace{Root: absStart, Repos: normalizeRepos(repos), Mode: ModeWorkspace})
	}

	runner := &git.Runner{Dir: absStart}
	mainRoot, err := runner.MainWorktreeDir()
	if err != nil {
		if isGitBinaryMissing(err) {
			return nil, err
		}
		if len(childRepos) > 0 {
			repos := normalizeRepos(childRepos)
			return complete(&Workspace{Root: absStart, Repos: repos, Mode: ModeWorkspace})
		}
		return nil, ErrNotGitRepository
	}

	mainRoot, err = filepath.Abs(mainRoot)
	if err != nil {
		return nil, err
	}

	if opts.Sandbox {
		return complete(&Workspace{
			Root:     mainRoot,
			MainRoot: mainRoot,
			Repos: []Repo{{
				Name: filepath.Base(mainRoot),
				Path: mainRoot,
			}},
			Mode: ModeSingleRepo,
		})
	}

	if wsRoot, ok, err := detectContainingWorkspace(discovery, absStart, mainRoot); err != nil {
		return nil, err
	} else if ok {
		repos, err := reposAtWorkspaceRoot(discovery, wsRoot)
		if err != nil {
			return nil, err
		}
		return complete(&Workspace{Root: wsRoot, Repos: repos, Mode: ModeWorkspace, MainRoot: mainRoot})
	}

	if len(childRepos) > 0 && !isRepoMarker(filepath.Dir(absStart)) {
		repos := childRepos
		if ok, err := isStandaloneRepoRoot(absStart); err != nil {
			return nil, err
		} else if ok {
			repos = append(repos, Repo{Name: filepath.Base(absStart), Path: absStart})
		}
		return complete(&Workspace{Root: absStart, Repos: normalizeRepos(repos), Mode: ModeWorkspace})
	}

	return complete(&Workspace{
		Root:     mainRoot,
		MainRoot: mainRoot,
		Repos: []Repo{{
			Name: filepath.Base(mainRoot),
			Path: mainRoot,
		}},
		Mode: ModeSingleRepo,
	})
}

func workspaceFromTopology(topology cache.Topology) (*Workspace, error) {
	mode := Mode(topology.Mode)
	if mode != ModeSingleRepo && mode != ModeWorkspace {
		return nil, ErrNotGitRepository
	}
	repos := make([]Repo, 0, len(topology.Repos))
	for _, repo := range topology.Repos {
		repos = append(repos, Repo{Name: repo.Name, Path: repo.Path})
	}
	return &Workspace{
		Root:     topology.Root,
		MainRoot: topology.MainRoot,
		Repos:    repos,
		Mode:     mode,
	}, nil
}

func detectContainingWorkspace(discovery *discoveryContext, startDir, mainRoot string) (string, bool, error) {
	for _, candidate := range candidateDirs(startDir, mainRoot) {
		ok, err := isContainingWorkspaceRoot(discovery, candidate, mainRoot)
		if err != nil {
			return "", false, err
		}
		if ok {
			return candidate, true, nil
		}
	}
	return "", false, nil
}

func candidateDirs(startDir, mainRoot string) []string {
	parent := filepath.Dir(mainRoot)
	grandparent := filepath.Dir(parent)

	var dirs []string
	add := func(dir string) {
		if dir == "" {
			return
		}
		for _, existing := range dirs {
			if existing == dir {
				return
			}
		}
		dirs = append(dirs, dir)
	}

	add(startDir)
	add(mainRoot)
	if parent != mainRoot {
		add(parent)
	}
	if grandparent != parent {
		add(grandparent)
	}

	return dirs
}

func isContainingWorkspaceRoot(discovery *discoveryContext, candidate, mainRoot string) (bool, error) {
	if !containsPath(candidate, mainRoot) {
		return false, nil
	}

	repos, err := discovery.scanImmediateRepos(candidate)
	if err != nil {
		return false, err
	}
	if len(repos) < 2 {
		return false, nil
	}
	if filepath.Clean(candidate) == filepath.Clean(mainRoot) {
		return true, nil
	}
	for _, repo := range repos {
		if containsPath(repo.Path, mainRoot) {
			return true, nil
		}
	}
	return false, nil
}

func containsPath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func reposAtWorkspaceRoot(discovery *discoveryContext, root string) ([]Repo, error) {
	repos, err := discovery.scanImmediateRepos(root)
	if err != nil {
		return nil, err
	}
	if ok, err := isStandaloneRepoRoot(root); err != nil {
		return nil, err
	} else if ok {
		repos = append(repos, Repo{Name: filepath.Base(root), Path: root})
	}
	return normalizeRepos(repos), nil
}

func scanImmediateRepos(dir string) ([]Repo, error) {
	entries, err := immediateChildReadDir(dir)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	for _, entry := range entries {
		candidate := filepath.Join(dir, entry.Name())
		if ok, err := isImmediateChildRepo(entry, candidate); err != nil {
			return nil, err
		} else if ok {
			repos = append(repos, Repo{Name: entry.Name(), Path: candidate})
		}
	}
	return repos, nil
}

type discoveryContext struct {
	scans   map[string]scanResult
	scanned map[string]struct{}
}

type scanResult struct {
	repos []Repo
	err   error
}

func newDiscoveryContext() *discoveryContext {
	return &discoveryContext{scans: make(map[string]scanResult), scanned: make(map[string]struct{})}
}

func (d *discoveryContext) scanImmediateRepos(dir string) ([]Repo, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	d.scanned[absDir] = struct{}{}
	if cached, ok := d.scans[absDir]; ok {
		return cloneRepos(cached.repos), cached.err
	}

	repos, err := scanImmediateRepos(absDir)
	if err == nil {
		repos = normalizeRepos(repos)
	}
	d.scans[absDir] = scanResult{repos: cloneRepos(repos), err: err}
	return cloneRepos(repos), err
}

func (d *discoveryContext) scannedDirs() []string {
	dirs := make([]string, 0, len(d.scanned))
	for dir := range d.scanned {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}

func cloneRepos(repos []Repo) []Repo {
	cloned := make([]Repo, len(repos))
	copy(cloned, repos)
	return cloned
}

func isImmediateChildRepo(entry os.DirEntry, dir string) (bool, error) {
	kind, err := classifyImmediateChild(entry, dir)
	if err != nil {
		if isIgnorableImmediateChildError(err) {
			return false, nil
		}
		return false, err
	}
	if kind.isSymlink || !kind.isDir {
		return false, nil
	}
	if !isEligibleRepoMarker(dir) {
		return false, nil
	}
	return standaloneRepoRoot(dir)
}

type immediateChildKind struct {
	isDir     bool
	isSymlink bool
}

var immediateChildReadDir = os.ReadDir
var immediateChildLstat = os.Lstat
var standaloneRepoRoot = isStandaloneRepoRoot

func classifyImmediateChild(entry os.DirEntry, path string) (immediateChildKind, error) {
	if entry.IsDir() {
		return immediateChildKind{isDir: true}, nil
	}

	if entry.Type()&os.ModeSymlink != 0 {
		return immediateChildKind{isSymlink: true}, nil
	}

	if entry.Type() != 0 {
		return immediateChildKind{}, nil
	}

	info, err := immediateChildLstat(path)
	if err != nil {
		return immediateChildKind{}, err
	}

	return immediateChildKind{
		isDir:     info.IsDir(),
		isSymlink: info.Mode()&os.ModeSymlink != 0,
	}, nil
}

func isIgnorableImmediateChildError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission)
}

func isStandaloneRepoRoot(dir string) (bool, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}
	canonicalDir, err := canonicalPath(absDir)
	if err != nil {
		return false, err
	}

	runner := &git.Runner{Dir: canonicalDir}

	topLevel, err := runner.TopLevelDir()
	if err != nil {
		if isGitBinaryMissing(err) {
			return false, err
		}
		return false, nil
	}
	topLevel, err = canonicalPath(topLevel)
	if err != nil {
		return false, err
	}
	if filepath.Clean(topLevel) != filepath.Clean(canonicalDir) {
		return false, nil
	}

	gitDir, err := runner.GitDir()
	if err != nil {
		if isGitBinaryMissing(err) {
			return false, err
		}
		return false, nil
	}
	gitCommonDir, err := runner.GitCommonDir()
	if err != nil {
		if isGitBinaryMissing(err) {
			return false, err
		}
		return false, nil
	}
	gitDir, err = canonicalPath(gitDir)
	if err != nil {
		return false, err
	}
	gitCommonDir, err = canonicalPath(gitCommonDir)
	if err != nil {
		return false, err
	}
	if filepath.Clean(gitDir) != filepath.Clean(gitCommonDir) {
		return false, nil
	}

	return true, nil
}

func isRepoMarker(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return true
	}
	return info.Mode().IsRegular()
}

func isEligibleRepoMarker(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := immediateChildLstat(gitPath)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

func isGitBinaryMissing(err error) bool {
	return strings.Contains(err.Error(), "git not found in PATH")
}

func canonicalPath(path string) (string, error) {
	evaluated, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return filepath.Clean(path), nil
		}
		return "", err
	}
	return filepath.Clean(evaluated), nil
}

func normalizeRepos(repos []Repo) []Repo {
	seen := make(map[string]Repo, len(repos))
	for _, repo := range repos {
		if repo.Path == "" {
			continue
		}
		absPath, err := filepath.Abs(repo.Path)
		if err != nil {
			continue
		}
		repo.Path = absPath
		if repo.Name == "" {
			repo.Name = filepath.Base(absPath)
		}
		seen[absPath] = repo
	}

	normalized := make([]Repo, 0, len(seen))
	for _, repo := range seen {
		normalized = append(normalized, repo)
	}

	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Name == normalized[j].Name {
			return normalized[i].Path < normalized[j].Path
		}
		return normalized[i].Name < normalized[j].Name
	})
	return normalized
}
