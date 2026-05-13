package git

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var heuristicDefaultBranchCandidates = []string{"main", "master"}

const worktreeRemoveSubmoduleFailure = "working trees containing submodules cannot be moved or removed"

// Runner executes git commands by shelling out to the git binary.
type Runner struct {
	GitBin string // path to git binary, defaults to "git"
	Dir    string // working directory for git commands
}

// WorktreeEntry represents a single git worktree.
type WorktreeEntry struct {
	Path   string
	Head   string
	Branch string
	Bare   bool
	Main   bool // true for the main working tree (first entry from git)
}

type BranchTracking struct {
	Remote   string
	MergeRef string
}

func (r *Runner) gitBin() string {
	if r.GitBin != "" {
		return r.GitBin
	}
	return "git"
}

// Run executes a git command and returns stdout.
func (r *Runner) Run(args ...string) (string, error) {
	return r.run(args, false)
}

func (r *Runner) run(args []string, allowExitCode1 bool) (string, error) {
	cmd := exec.Command(r.gitBin(), args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	out, err := cmd.Output()
	if err != nil {
		var pathErr *exec.Error
		if errors.As(err, &pathErr) {
			return "", fmt.Errorf("git not found in PATH: install git and try again")
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			if allowExitCode1 && exitErr.ExitCode() == 1 && len(exitErr.Stderr) == 0 {
				return "", nil
			}
			return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// WorktreeAdd creates a new worktree with a new branch from base.
func (r *Runner) WorktreeAdd(path, branch, base string) error {
	_, err := r.Run("worktree", "add", "-b", branch, path, base)
	return err
}

// WorktreeAddGuessRemote creates a worktree by asking Git to resolve a
// same-named remote branch.
func (r *Runner) WorktreeAddGuessRemote(path, branch string) error {
	_, err := r.Run("worktree", "add", "--guess-remote", path, branch)
	return err
}

// WorktreeAddExisting creates a worktree for an existing branch.
func (r *Runner) WorktreeAddExisting(path, branch string) error {
	_, err := r.Run("worktree", "add", path, branch)
	return err
}

// WorktreeList returns all worktrees using porcelain format.
func (r *Runner) WorktreeList() ([]WorktreeEntry, error) {
	out, err := r.Run("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out), nil
}

// MergedBranches returns local branches merged into base.
func (r *Runner) MergedBranches(base string) ([]string, error) {
	out, err := r.Run("branch", "--merged", base)
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "*+ ")
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// BranchRemote returns the remote configured for branch, or empty string if
// the branch has no tracking remote.
func (r *Runner) BranchRemote(branch string) (string, error) {
	out, err := r.run([]string{"config", "--get", "branch." + branch + ".remote"}, true)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// BranchMergeRef returns the merge ref configured for branch, or empty string if
// the branch has no configured upstream merge target.
func (r *Runner) BranchMergeRef(branch string) (string, error) {
	out, err := r.run([]string{"config", "--get", "branch." + branch + ".merge"}, true)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HasRemote reports whether a remote is configured locally.
func (r *Runner) HasRemote(remote string) (bool, error) {
	out, err := r.run([]string{"config", "--get", "remote." + remote + ".url"}, true)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// RemoteBranchExists reports whether remote/branch exists on the remote.
func (r *Runner) RemoteBranchExists(remote, branch string) (bool, error) {
	out, err := r.Run("ls-remote", "--heads", remote, branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// ListRemoteBranches returns the set of branch names present on a remote.
// It makes a single ls-remote --heads call so callers can batch lookups.
func (r *Runner) ListRemoteBranches(remote string) (map[string]struct{}, error) {
	out, err := r.Run("ls-remote", "--heads", remote)
	if err != nil {
		return nil, err
	}
	branches := make(map[string]struct{})
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		// format: "<sha>\trefs/heads/<branch>"
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		ref := strings.TrimSpace(parts[1])
		branch := strings.TrimPrefix(ref, "refs/heads/")
		if branch != ref {
			branches[branch] = struct{}{}
		}
	}
	return branches, nil
}

func parseWorktreeList(output string) []WorktreeEntry {
	var entries []WorktreeEntry
	var current WorktreeEntry
	isFirst := true

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = WorktreeEntry{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			hash := strings.TrimPrefix(line, "HEAD ")
			if len(hash) > 7 {
				hash = hash[:7]
			}
			current.Head = hash
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			current.Bare = true
		case line == "":
			if current.Path != "" {
				if isFirst {
					current.Main = true
					isFirst = false
				}
				entries = append(entries, current)
				current = WorktreeEntry{}
			}
		}
	}
	if current.Path != "" {
		if isFirst {
			current.Main = true
		}
		entries = append(entries, current)
	}
	return entries
}

// WorktreeRemove removes a worktree at the given path.
func (r *Runner) WorktreeRemove(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := r.Run(args...)
	return err
}

// IsWorktreeRemoveSubmoduleError reports whether err is Git's known refusal to
// remove a worktree that contains submodules.
func IsWorktreeRemoveSubmoduleError(err error) bool {
	return err != nil && strings.Contains(err.Error(), worktreeRemoveSubmoduleFailure)
}

// BranchDelete deletes a local branch.
func (r *Runner) BranchDelete(branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.Run("branch", flag, branch)
	return err
}

// BranchExists checks if a local branch exists.
func (r *Runner) BranchExists(branch string) bool {
	_, err := r.Run("rev-parse", "--verify", "refs/heads/"+branch)
	return err == nil
}

// RefExists checks if a ref exists locally.
func (r *Runner) RefExists(ref string) bool {
	_, err := r.Run("rev-parse", "--verify", ref)
	return err == nil
}

// DefaultBranch returns the default branch name (e.g., "origin/main").
func (r *Runner) DefaultBranch() (string, error) {
	out, err := r.Run("symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", fmt.Errorf("cannot detect default branch: %w", err)
	}
	// refs/remotes/origin/main -> origin/main
	ref := strings.TrimPrefix(out, "refs/remotes/")
	return ref, nil
}

// HeuristicDefaultBranch attempts to infer a usable base ref when origin/HEAD
// is unavailable. It returns the resolved remote ref and true on success.
func (r *Runner) HeuristicDefaultBranch() (string, bool, error) {
	hasOrigin, err := r.HasRemote("origin")
	if err != nil {
		return "", false, err
	}
	if !hasOrigin {
		return "", false, nil
	}

	tracking, err := r.BranchTrackingConfig()
	if err != nil {
		return "", false, err
	}

	for _, candidate := range heuristicDefaultBranchCandidates {
		mergeRef := "refs/heads/" + candidate
		resolvedRef := "origin/" + candidate
		localTrackingRef := "refs/remotes/" + resolvedRef
		if cfg, ok := tracking[candidate]; ok && cfg.Remote == "origin" && cfg.MergeRef == mergeRef {
			if r.RefExists(localTrackingRef) {
				return resolvedRef, true, nil
			}
		}

		if tracksRemoteBranch(tracking, "origin", mergeRef) && r.RefExists(localTrackingRef) {
			return resolvedRef, true, nil
		}

		exists, err := r.RemoteBranchExists("origin", candidate)
		if err != nil {
			return "", false, err
		}
		if exists && r.RefExists(localTrackingRef) {
			return resolvedRef, true, nil
		}
	}

	return "", false, nil
}

func (r *Runner) BranchTrackingConfig() (map[string]BranchTracking, error) {
	remoteConfig, err := r.run([]string{"config", "--get-regexp", `^branch\..*\.remote$`}, true)
	if err != nil {
		return nil, err
	}
	mergeConfig, err := r.run([]string{"config", "--get-regexp", `^branch\..*\.merge$`}, true)
	if err != nil {
		return nil, err
	}

	tracking := make(map[string]BranchTracking)
	for _, line := range strings.Split(remoteConfig, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		branch, ok := branchNameFromConfigKey(fields[0], ".remote")
		if !ok {
			continue
		}
		cfg := tracking[branch]
		cfg.Remote = fields[1]
		tracking[branch] = cfg
	}
	for _, line := range strings.Split(mergeConfig, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		branch, ok := branchNameFromConfigKey(fields[0], ".merge")
		if !ok {
			continue
		}
		cfg := tracking[branch]
		cfg.MergeRef = fields[1]
		tracking[branch] = cfg
	}
	return tracking, nil
}

func tracksRemoteBranch(tracking map[string]BranchTracking, wantRemote, wantMergeRef string) bool {
	for _, cfg := range tracking {
		if cfg.Remote == wantRemote && cfg.MergeRef == wantMergeRef {
			return true
		}
	}
	return false
}

func branchNameFromConfigKey(key, suffix string) (string, bool) {
	branch := strings.TrimPrefix(key, "branch.")
	branch = strings.TrimSuffix(branch, suffix)
	if branch == "" || branch == key {
		return "", false
	}
	return branch, true
}

// Fetch fetches from origin.
func (r *Runner) Fetch() error {
	_, err := r.Run("fetch", "origin")
	return err
}

// MainWorktreeDir returns the absolute path of the main working tree.
// This works correctly even when called from inside a secondary worktree.
func (r *Runner) MainWorktreeDir() (string, error) {
	out, err := r.GitCommonDir()
	if err != nil {
		return "", err
	}
	// --git-common-dir returns the .git directory; parent is the repo root
	return filepath.Dir(out), nil
}

// TopLevelDir returns the absolute path of the current repository worktree root.
func (r *Runner) TopLevelDir() (string, error) {
	return r.Run("rev-parse", "--path-format=absolute", "--show-toplevel")
}

// GitDir returns the absolute path to the current repository git dir.
func (r *Runner) GitDir() (string, error) {
	return r.Run("rev-parse", "--path-format=absolute", "--git-dir")
}

// GitCommonDir returns the absolute path to the current repository shared git dir.
func (r *Runner) GitCommonDir() (string, error) {
	return r.Run("rev-parse", "--path-format=absolute", "--git-common-dir")
}

// RepoName returns the repository directory name.
// Uses the main working tree to determine the name, so it returns the
// correct name even when called from a secondary worktree.
func (r *Runner) RepoName() (string, error) {
	dir, err := r.MainWorktreeDir()
	if err != nil {
		return "", err
	}
	return filepath.Base(dir), nil
}
