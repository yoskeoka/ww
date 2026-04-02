package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yoskeoka/ww/git"
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
	Root  string
	Repos []Repo
	Mode  Mode
}

// ErrNotGitRepository is returned when detection finds no git repository and
// no valid workspace root.
var ErrNotGitRepository = errors.New("not a git repository")

// Detect inspects startDir and returns the detected workspace layout.
func Detect(startDir string) (*Workspace, error) {
	absStart, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	childRepos, err := scanImmediateRepos(absStart)
	if err != nil {
		return nil, err
	}

	runner := &git.Runner{Dir: absStart}
	mainRoot, err := runner.MainWorktreeDir()
	if err != nil {
		if isGitBinaryMissing(err) {
			return nil, err
		}
		if len(childRepos) > 0 {
			repos := normalizeRepos(childRepos)
			return &Workspace{Root: absStart, Repos: repos, Mode: ModeWorkspace}, nil
		}
		return nil, ErrNotGitRepository
	}

	mainRoot, err = filepath.Abs(mainRoot)
	if err != nil {
		return nil, err
	}

	if wsRoot, ok, err := detectContainingWorkspace(absStart, mainRoot); err != nil {
		return nil, err
	} else if ok {
		repos, err := reposAtWorkspaceRoot(wsRoot)
		if err != nil {
			return nil, err
		}
		return &Workspace{Root: wsRoot, Repos: repos, Mode: ModeWorkspace}, nil
	}

	if len(childRepos) > 0 && !hasGitEntry(filepath.Dir(absStart)) {
		repos := childRepos
		if hasGitEntry(absStart) {
			repos = append(repos, Repo{Name: filepath.Base(absStart), Path: absStart})
		}
		return &Workspace{Root: absStart, Repos: normalizeRepos(repos), Mode: ModeWorkspace}, nil
	}

	return &Workspace{
		Root: mainRoot,
		Repos: []Repo{{
			Name: filepath.Base(mainRoot),
			Path: mainRoot,
		}},
		Mode: ModeSingleRepo,
	}, nil
}

func detectContainingWorkspace(startDir, mainRoot string) (string, bool, error) {
	for _, candidate := range candidateDirs(startDir, mainRoot) {
		ok, err := isContainingWorkspaceRoot(candidate, mainRoot)
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

func isContainingWorkspaceRoot(candidate, mainRoot string) (bool, error) {
	if !containsPath(candidate, mainRoot) {
		return false, nil
	}

	repos, err := scanImmediateRepos(candidate)
	if err != nil {
		return false, err
	}
	return len(repos) >= 2, nil
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

func reposAtWorkspaceRoot(root string) ([]Repo, error) {
	repos, err := scanImmediateRepos(root)
	if err != nil {
		return nil, err
	}
	if hasGitEntry(root) {
		repos = append(repos, Repo{Name: filepath.Base(root), Path: root})
	}
	return normalizeRepos(repos), nil
}

func scanImmediateRepos(dir string) ([]Repo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	for _, entry := range entries {
		candidate := filepath.Join(dir, entry.Name())
		if hasGitEntry(candidate) {
			repos = append(repos, Repo{Name: entry.Name(), Path: candidate})
		}
	}
	return repos, nil
}

func hasGitEntry(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return true
	}
	if info.Mode().IsRegular() {
		data, err := os.ReadFile(gitPath)
		if err != nil {
			return false
		}
		return !strings.Contains(string(data), ".git/worktrees/")
	}
	return true
}

func isGitBinaryMissing(err error) bool {
	return strings.Contains(err.Error(), "git not found in PATH")
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
