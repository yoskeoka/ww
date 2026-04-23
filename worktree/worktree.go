package worktree

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yoskeoka/ww/git"
	"github.com/yoskeoka/ww/validate"
	"github.com/yoskeoka/ww/workspace"
)

// Exported status constants for WorktreeInfo.Status.
const (
	StatusActive  = "active"
	StatusMerged  = "merged"
	StatusStale   = "stale"
	StatusUnknown = "unknown"
)

// Config holds the configuration values that Manager needs to operate.
// This is decoupled from the TOML config file format (internal/config)
// so that library consumers can construct it directly.
type Config struct {
	WorktreeDir    string
	DefaultBase    string
	CopyFiles      []string
	SymlinkFiles   []string
	PostCreateHook string
	Sandbox        bool
}

// Manager coordinates worktree operations.
type Manager struct {
	Git       *git.Runner
	Config    Config
	RepoDir   string // absolute path to the main repository
	Workspace *workspace.Workspace
}

// CreateOpts configures worktree creation.
type CreateOpts struct {
	DryRun   bool
	Output   io.Writer // destination for text-mode output (nil defaults to os.Stdout)
	TextMode bool      // when true, print human-readable progress (e.g. hook announcement)
}

// RemoveOpts configures worktree removal.
type RemoveOpts struct {
	Force      bool
	KeepBranch bool
	DryRun     bool
}

// WorktreeInfo holds information about a created/listed worktree.
type WorktreeInfo struct {
	Path         string `json:"path"`
	Branch       string `json:"branch"`
	Repo         string `json:"repo,omitempty"`
	Status       string `json:"status,omitempty"`
	StatusDetail string `json:"status_detail,omitempty"`
	Head         string `json:"head,omitempty"`
	Bare         bool   `json:"bare,omitempty"`
	Main         bool   `json:"main,omitempty"`
	Created      bool   `json:"created,omitempty"`
	Base         string `json:"base,omitempty"`
}

type baseRefInfo struct {
	Ref          string
	StatusDetail string
}

// RemoveResult holds information about a removed worktree.
type RemoveResult struct {
	Path          string `json:"path"`
	Branch        string `json:"branch"`
	Removed       bool   `json:"removed"`
	BranchDeleted bool   `json:"branch_deleted"`
	BranchError   string `json:"branch_error,omitempty"`
}

// SanitizeBranch converts a branch name into a safe directory name component.
func SanitizeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

func (m *Manager) isWorkspaceMode() bool {
	return m.Workspace != nil && m.Workspace.Mode == workspace.ModeWorkspace
}

func (m *Manager) worktreeLocation(branch string) (string, string, error) {
	repoName := filepath.Base(m.RepoDir)
	dirName := repoName + "@" + SanitizeBranch(branch)
	repoParent := filepath.Dir(m.RepoDir)

	if m.Config.WorktreeDir != "" {
		base := m.Config.WorktreeDir
		if !filepath.IsAbs(base) {
			var anchor string
			if m.isWorkspaceMode() {
				anchor = m.Workspace.Root
			} else if m.Config.Sandbox {
				anchor = m.RepoDir
			} else {
				anchor = repoParent
			}
			cleanAnchor := filepath.Clean(anchor)
			base = filepath.Join(cleanAnchor, base)
			// Reject relative paths that escape the anchor root via ".." traversal.
			rel, relErr := filepath.Rel(cleanAnchor, base)
			if relErr != nil || strings.HasPrefix(rel, "..") {
				return "", "", fmt.Errorf("worktree_dir %q resolves outside the allowed area %q", m.Config.WorktreeDir, anchor)
			}
			if m.isWorkspaceMode() {
				return filepath.Join(base, dirName), m.Workspace.Root, nil
			}
			return filepath.Join(base, dirName), m.RepoDir, nil
		}
		return filepath.Join(base, dirName), base, nil
	}

	if m.isWorkspaceMode() {
		base := filepath.Join(m.Workspace.Root, ".worktrees")
		return filepath.Join(base, dirName), m.Workspace.Root, nil
	}

	if m.Config.Sandbox {
		base := filepath.Join(m.RepoDir, ".worktrees")
		return filepath.Join(base, dirName), m.RepoDir, nil
	}

	return filepath.Join(repoParent, dirName), m.RepoDir, nil
}

// WorktreePath computes the worktree directory path for a branch.
func (m *Manager) WorktreePath(branch string) (string, error) {
	path, _, err := m.worktreeLocation(branch)
	return path, err
}

// Create creates a worktree for the given branch.
func (m *Manager) Create(branch string, opts CreateOpts) (*WorktreeInfo, []string, error) {
	if err := validate.BranchName(branch); err != nil {
		return nil, nil, err
	}

	wtPath, validationRoot, err := m.worktreeLocation(branch)
	if err != nil {
		return nil, nil, err
	}

	if err := validate.WorktreePath(wtPath, validationRoot); err != nil {
		return nil, nil, err
	}

	if _, err := os.Lstat(wtPath); err == nil {
		return nil, nil, fmt.Errorf("worktree already exists at %s", wtPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("cannot access worktree path %s: %w", wtPath, err)
	}

	branchExists := m.Git.BranchExists(branch)

	var base string
	if !branchExists {
		baseInfo, err := m.baseRef(m.Git)
		if err != nil {
			return nil, nil, unresolvedCreateBaseError(err)
		}
		base = baseInfo.Ref
	}

	var dryRunLog []string

	if opts.DryRun {
		if branchExists {
			dryRunLog = append(dryRunLog, fmt.Sprintf("Would create worktree at %s (existing branch: %s)", wtPath, branch))
		} else {
			dryRunLog = append(dryRunLog, fmt.Sprintf("Would create worktree at %s (branch: %s, base: %s)", wtPath, branch, base))
		}
		for _, f := range m.Config.CopyFiles {
			dryRunLog = append(dryRunLog, fmt.Sprintf("Would copy: %s", f))
		}
		for _, f := range m.Config.SymlinkFiles {
			dryRunLog = append(dryRunLog, fmt.Sprintf("Would symlink: %s", f))
		}
		if m.Config.PostCreateHook != "" {
			dryRunLog = append(dryRunLog, fmt.Sprintf("Would run hook: %s", m.Config.PostCreateHook))
		}
		info := &WorktreeInfo{Path: wtPath, Branch: branch, Created: true, Base: base}
		return info, dryRunLog, nil
	}

	if branchExists {
		if err := m.Git.WorktreeAddExisting(wtPath, branch); err != nil {
			return nil, nil, fmt.Errorf("adding worktree for existing branch: %w", err)
		}
	} else {
		if err := m.Git.WorktreeAdd(wtPath, branch, base); err != nil {
			return nil, nil, fmt.Errorf("creating worktree with new branch: %w", err)
		}
	}

	m.copyFiles(wtPath)
	m.symlinkFiles(wtPath)
	m.runPostCreateHook(wtPath, branch, opts)

	info := &WorktreeInfo{Path: wtPath, Branch: branch, Created: true, Base: base}
	return info, nil, nil
}

// List returns all worktrees.
func (m *Manager) List() ([]WorktreeInfo, error) {
	if m.isWorkspaceMode() {
		return m.listWorkspace()
	}
	return m.listRepo(filepath.Base(m.RepoDir), m.RepoDir)
}

// FindByName returns the worktree whose branch matches name.
// refs/heads/<branch> and <branch> are treated as equivalent.
// When withStatus is true the returned WorktreeInfo includes the computed
// Status field (requires merged-branch and remote-branch discovery).
// When withStatus is false only path, branch, head, and structural fields
// are populated, which is significantly faster.
func (m *Manager) FindByName(name string, withStatus bool) (*WorktreeInfo, error) {
	target := normalizeBranchName(name)
	listFn := m.listRepoFast
	if withStatus {
		listFn = m.listRepo
	}
	infos, err := listFn(filepath.Base(m.RepoDir), m.RepoDir)
	if err != nil {
		return nil, err
	}
	for i := range infos {
		if normalizeBranchName(infos[i].Branch) != target {
			continue
		}
		info := infos[i]
		return &info, nil
	}
	return nil, fmt.Errorf("no worktree found for branch %q", target)
}

// MostRecent returns the most recently created secondary worktree.
// Recency is determined by the mtime of .git/worktrees/<name> admin dirs.
// When withStatus is true the returned WorktreeInfo includes the computed
// Status field (requires merged-branch and remote-branch discovery).
// When withStatus is false only the Path field is guaranteed to be populated,
// which avoids all additional git calls after the admin-dir scan.
func (m *Manager) MostRecent(withStatus bool) (*WorktreeInfo, error) {
	adminRoot := filepath.Join(m.RepoDir, ".git", "worktrees")
	entries, err := os.ReadDir(adminRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("no secondary worktrees found")
		}
		return nil, fmt.Errorf("reading worktree admin directory: %w", err)
	}

	var newestPath string
	var newestTime int64
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading worktree admin info for %s: %w", entry.Name(), err)
		}
		wtPath, err := worktreePathFromAdminDir(adminRoot, entry.Name())
		if err != nil {
			return nil, err
		}
		if newestPath == "" || info.ModTime().UnixNano() > newestTime {
			newestPath = wtPath
			newestTime = info.ModTime().UnixNano()
		}
	}
	if newestPath == "" {
		return nil, fmt.Errorf("no secondary worktrees found")
	}

	if !withStatus {
		return &WorktreeInfo{Path: newestPath}, nil
	}

	infos, err := m.listRepo(filepath.Base(m.RepoDir), m.RepoDir)
	if err != nil {
		return nil, err
	}
	for i := range infos {
		if infos[i].Path != newestPath {
			continue
		}
		info := infos[i]
		return &info, nil
	}
	return nil, fmt.Errorf("no worktree found for path %q", newestPath)
}

func (m *Manager) listWorkspace() ([]WorktreeInfo, error) {
	var infos []WorktreeInfo
	for _, repo := range m.Workspace.Repos {
		repoInfos, err := m.listRepo(repo.Name, repo.Path)
		if err != nil {
			return nil, err
		}
		infos = append(infos, repoInfos...)
	}
	return infos, nil
}

func (m *Manager) listRepo(repoName, repoPath string) ([]WorktreeInfo, error) {
	runner := &git.Runner{Dir: repoPath}
	entries, err := runner.WorktreeList()
	if err != nil {
		return nil, fmt.Errorf("listing worktrees for %s: %w", repoName, err)
	}

	baseInfo, baseErr := m.baseRef(runner)
	if baseErr != nil {
		// Base branch could not be determined; degrade to unknown status.
		return listRepoUnknown(entries, repoName, "base-detect-failed"), nil
	}

	merged, err := runner.MergedBranches(baseInfo.Ref)
	if err != nil {
		return nil, fmt.Errorf("listing merged branches for %s: %w", repoName, err)
	}
	mergedSet := make(map[string]struct{}, len(merged))
	for _, branch := range merged {
		mergedSet[branch] = struct{}{}
	}
	// The base branch itself is always active even though git reports it as merged.
	delete(mergedSet, baseInfo.Ref)

	// Precompute branch→remote and batch ls-remote calls (one per unique remote).
	branchRemote := make(map[string]string)
	remoteBranches := make(map[string]map[string]struct{})
	for _, e := range entries {
		if e.Main || e.Branch == "" {
			continue
		}
		if _, ok := mergedSet[e.Branch]; ok {
			continue
		}
		remote, err := runner.BranchRemote(e.Branch)
		if err != nil {
			return nil, fmt.Errorf("getting remote for %s: %w", e.Branch, err)
		}
		branchRemote[e.Branch] = remote
		if remote != "" {
			if _, cached := remoteBranches[remote]; !cached {
				branches, err := runner.ListRemoteBranches(remote)
				if err != nil {
					return nil, fmt.Errorf("listing remote branches for %s: %w", remote, err)
				}
				remoteBranches[remote] = branches
			}
		}
	}

	infos := make([]WorktreeInfo, 0, len(entries))
	for _, e := range entries {
		status := resolveStatus(e, mergedSet, branchRemote, remoteBranches)
		infos = append(infos, WorktreeInfo{
			Path:         e.Path,
			Branch:       e.Branch,
			Repo:         repoName,
			Status:       status,
			StatusDetail: baseInfo.StatusDetail,
			Head:         e.Head,
			Bare:         e.Bare,
			Main:         e.Main,
		})
	}
	return infos, nil
}

// listRepoFast returns worktree entries without computing Status.
// It calls git worktree list --porcelain once and skips merged-branch and
// remote-branch discovery, making it safe for use when only path/branch/head
// fields are needed.
func (m *Manager) listRepoFast(repoName, repoPath string) ([]WorktreeInfo, error) {
	runner := &git.Runner{Dir: repoPath}
	entries, err := runner.WorktreeList()
	if err != nil {
		return nil, fmt.Errorf("listing worktrees for %s: %w", repoName, err)
	}
	infos := make([]WorktreeInfo, 0, len(entries))
	for _, e := range entries {
		infos = append(infos, WorktreeInfo{
			Path:   e.Path,
			Branch: e.Branch,
			Repo:   repoName,
			Head:   e.Head,
			Bare:   e.Bare,
			Main:   e.Main,
		})
	}
	return infos, nil
}

type baseDetectionError struct {
	originHeadErr error
	heuristicErr  error
}

func (e baseDetectionError) Error() string {
	if e.heuristicErr != nil {
		return fmt.Sprintf("origin/HEAD could not be used and heuristic fallback failed: %v (origin/HEAD error: %v)", e.heuristicErr, e.originHeadErr)
	}
	return fmt.Sprintf("origin/HEAD could not be used and heuristic fallback found no usable origin/main or origin/master: %v", e.originHeadErr)
}

func (e baseDetectionError) Unwrap() []error {
	var errs []error
	if e.originHeadErr != nil {
		errs = append(errs, e.originHeadErr)
	}
	if e.heuristicErr != nil {
		errs = append(errs, e.heuristicErr)
	}
	return errs
}

func (m *Manager) baseRef(runner *git.Runner) (baseRefInfo, error) {
	if m.Config.DefaultBase != "" {
		return baseRefInfo{Ref: m.Config.DefaultBase}, nil
	}
	ref, err := runner.DefaultBranch()
	if err == nil {
		return baseRefInfo{Ref: ref}, nil
	}
	ref, ok, heuristicErr := runner.HeuristicDefaultBranch()
	if heuristicErr != nil {
		return baseRefInfo{}, baseDetectionError{originHeadErr: err, heuristicErr: heuristicErr}
	}
	if ok {
		return baseRefInfo{Ref: ref, StatusDetail: "heuristic-base"}, nil
	}
	return baseRefInfo{}, baseDetectionError{originHeadErr: err}
}

func unresolvedCreateBaseError(err error) error {
	var baseErr baseDetectionError
	heuristicDetail := "heuristic fallback could not find a usable origin/main or origin/master"
	if errors.As(err, &baseErr) && baseErr.heuristicErr != nil {
		heuristicDetail = "heuristic fallback failed before it could choose origin/main or origin/master"
	}
	return fmt.Errorf("cannot determine base branch: no default_base is configured, origin/HEAD could not be used, and %s.\nSet default_base in .ww.toml or run: git remote set-head origin --auto when origin exposes a default branch.\nOriginal error: %w", heuristicDetail, err)
}

// listRepoUnknown builds WorktreeInfo entries when the base branch cannot be
// determined. Main worktrees get "active" status; all others get "unknown"
// with the supplied detail string.
func listRepoUnknown(entries []git.WorktreeEntry, repoName, detail string) []WorktreeInfo {
	infos := make([]WorktreeInfo, 0, len(entries))
	for _, e := range entries {
		status := StatusUnknown
		statusDetail := detail
		if e.Main || e.Branch == "" {
			status = StatusActive
			statusDetail = ""
		}
		infos = append(infos, WorktreeInfo{
			Path:         e.Path,
			Branch:       e.Branch,
			Repo:         repoName,
			Status:       status,
			StatusDetail: statusDetail,
			Head:         e.Head,
			Bare:         e.Bare,
			Main:         e.Main,
		})
	}
	return infos
}

func resolveStatus(entry git.WorktreeEntry, merged map[string]struct{}, branchRemote map[string]string, remoteBranches map[string]map[string]struct{}) string {
	if entry.Main || entry.Branch == "" {
		return StatusActive
	}
	if _, ok := merged[entry.Branch]; ok {
		return StatusMerged
	}
	remote := branchRemote[entry.Branch]
	if remote == "" {
		return StatusActive
	}
	if _, exists := remoteBranches[remote][entry.Branch]; !exists {
		return StatusStale
	}
	return StatusActive
}

func normalizeBranchName(name string) string {
	return strings.TrimPrefix(name, "refs/heads/")
}

func worktreePathFromAdminDir(adminRoot, name string) (string, error) {
	gitdirPath := filepath.Join(adminRoot, name, "gitdir")
	data, err := os.ReadFile(gitdirPath)
	if err != nil {
		return "", fmt.Errorf("reading worktree admin gitdir for %s: %w", name, err)
	}
	gitdir := strings.TrimSpace(string(data))
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Clean(filepath.Join(adminRoot, name, gitdir))
	}
	return filepath.Dir(gitdir), nil
}

// Remove removes a worktree and optionally its branch.
// It uses git worktree list as the source of truth (not os.Stat) and
// rejects attempts to remove the main worktree with a clear error.
func (m *Manager) Remove(branch string, opts RemoveOpts) (*RemoveResult, []string, error) {
	if err := validate.BranchName(branch); err != nil {
		return nil, nil, err
	}

	// Look up the branch in git worktree list output
	entries, err := m.Git.WorktreeList()
	if err != nil {
		return nil, nil, fmt.Errorf("listing worktrees: %w", err)
	}

	var found *git.WorktreeEntry
	for i := range entries {
		if entries[i].Branch == branch {
			found = &entries[i]
			break
		}
	}
	if found == nil {
		return nil, nil, fmt.Errorf("no worktree found for branch %q", branch)
	}

	if found.Main {
		return nil, nil, fmt.Errorf("cannot remove the main worktree")
	}

	result := &RemoveResult{Path: found.Path, Branch: branch}

	if opts.DryRun {
		var dryRunLog []string
		dryRunLog = append(dryRunLog, fmt.Sprintf("Would remove worktree at %s", found.Path))
		if !opts.KeepBranch {
			dryRunLog = append(dryRunLog, fmt.Sprintf("Would delete branch %s", branch))
		}
		result.Removed = true
		result.BranchDeleted = !opts.KeepBranch
		return result, dryRunLog, nil
	}

	if err := m.Git.WorktreeRemove(found.Path, opts.Force); err != nil {
		if git.IsWorktreeRemoveSubmoduleError(err) {
			return nil, nil, submoduleWorktreeRemoveError(found.Path, m.RepoDir, err)
		}
		return nil, nil, fmt.Errorf("removing worktree: %w", err)
	}
	result.Removed = true

	if !opts.KeepBranch {
		if err := m.Git.BranchDelete(branch, opts.Force); err != nil {
			result.BranchError = err.Error()
		} else {
			result.BranchDeleted = true
		}
	}

	return result, nil, nil
}

func submoduleWorktreeRemoveError(worktreePath, repoDir string, err error) error {
	return fmt.Errorf(`removing worktree: Git cannot remove worktrees containing submodules.
Target worktree: %s
Manual cleanup permanently deletes uncommitted work in that directory.
To clean it up manually, run:
  rm -rf -- %s
  git -C %s worktree prune
Original error: %w`, worktreePath, shellQuotePOSIX(worktreePath), shellQuotePOSIX(repoDir), err)
}

func shellQuotePOSIX(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func (m *Manager) copyFiles(wtPath string) {
	for _, pattern := range m.Config.CopyFiles {
		src := filepath.Join(m.RepoDir, pattern)
		dst := filepath.Join(wtPath, pattern)
		if err := copyPath(src, dst); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "warning: could not copy %s: %v\n", pattern, err)
			}
		}
	}
}

func (m *Manager) symlinkFiles(wtPath string) {
	for _, pattern := range m.Config.SymlinkFiles {
		src := filepath.Join(m.RepoDir, pattern)
		dst := filepath.Join(wtPath, pattern)
		if _, err := os.Stat(src); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "warning: could not access %s: %v\n", pattern, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create directory for %s: %v\n", pattern, err)
			continue
		}
		if err := os.Symlink(src, dst); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not symlink %s: %v\n", pattern, err)
		}
	}
}

func (m *Manager) runPostCreateHook(wtPath, branch string, opts CreateOpts) {
	if m.Config.PostCreateHook == "" {
		return
	}
	var hookOut io.Writer = os.Stderr
	if opts.TextMode {
		hookOut = opts.Output
		if hookOut == nil {
			hookOut = os.Stdout
		}
		fmt.Fprintf(hookOut, "Running post_create_hook: %s\n", m.Config.PostCreateHook)
	}
	cmd := exec.Command("sh", "-c", m.Config.PostCreateHook)
	cmd.Dir = wtPath
	cmd.Env = append(os.Environ(),
		"WW_BRANCH="+branch,
		"WW_WORKTREE_PATH="+wtPath,
	)
	cmd.Stdout = hookOut
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: post-create hook failed: %v\n", err)
	}
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
