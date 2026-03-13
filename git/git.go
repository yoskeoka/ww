package git

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

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

func (r *Runner) gitBin() string {
	if r.GitBin != "" {
		return r.GitBin
	}
	return "git"
}

// Run executes a git command and returns stdout.
func (r *Runner) Run(args ...string) (string, error) {
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

func parseWorktreeList(output string) []WorktreeEntry {
	var entries []WorktreeEntry
	var current WorktreeEntry
	isFirst := true

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = WorktreeEntry{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")[:7]
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
func (r *Runner) WorktreeRemove(path string) error {
	_, err := r.Run("worktree", "remove", path)
	return err
}

// BranchDelete deletes a local branch (safe delete).
func (r *Runner) BranchDelete(branch string) error {
	_, err := r.Run("branch", "-d", branch)
	return err
}

// BranchExists checks if a local branch exists.
func (r *Runner) BranchExists(branch string) bool {
	_, err := r.Run("rev-parse", "--verify", "refs/heads/"+branch)
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

// Fetch fetches from origin.
func (r *Runner) Fetch() error {
	_, err := r.Run("fetch", "origin")
	return err
}

// MainWorktreeDir returns the absolute path of the main working tree.
// This works correctly even when called from inside a secondary worktree.
func (r *Runner) MainWorktreeDir() (string, error) {
	out, err := r.Run("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	// --git-common-dir returns the .git directory; parent is the repo root
	return filepath.Dir(out), nil
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
