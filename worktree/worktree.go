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
	"github.com/yoskeoka/ww/internal/config"
	"github.com/yoskeoka/ww/validate"
)

// Manager coordinates worktree operations.
type Manager struct {
	Git     *git.Runner
	Config  *config.Config
	RepoDir string // absolute path to the main repository
}

// CreateOpts configures worktree creation.
type CreateOpts struct {
	DryRun bool
}

// RemoveOpts configures worktree removal.
type RemoveOpts struct {
	KeepBranch bool
	DryRun     bool
}

// WorktreeInfo holds information about a created/listed worktree.
type WorktreeInfo struct {
	Path    string `json:"path"`
	Branch  string `json:"branch"`
	Head    string `json:"head,omitempty"`
	Bare    bool   `json:"bare,omitempty"`
	Main    bool   `json:"main,omitempty"`
	Created bool   `json:"created,omitempty"`
	Base    string `json:"base,omitempty"`
}

// RemoveResult holds information about a removed worktree.
type RemoveResult struct {
	Path          string `json:"path"`
	Branch        string `json:"branch"`
	Removed       bool   `json:"removed"`
	BranchDeleted bool   `json:"branch_deleted"`
}

// SanitizeBranch converts a branch name into a safe directory name component.
func SanitizeBranch(branch string) string {
	return strings.ReplaceAll(branch, "/", "-")
}

// WorktreePath computes the worktree directory path for a branch.
func (m *Manager) WorktreePath(branch string) (string, error) {
	repoName := filepath.Base(m.RepoDir)
	dirName := repoName + "@" + SanitizeBranch(branch)

	if m.Config.WorktreeDir != "" {
		base := m.Config.WorktreeDir
		if !filepath.IsAbs(base) {
			base = filepath.Join(filepath.Dir(m.RepoDir), base)
		}
		return filepath.Join(base, dirName), nil
	}

	// Sibling layout
	return filepath.Join(filepath.Dir(m.RepoDir), dirName), nil
}

// Create creates a worktree for the given branch.
func (m *Manager) Create(branch string, opts CreateOpts) (*WorktreeInfo, []string, error) {
	if err := validate.BranchName(branch); err != nil {
		return nil, nil, err
	}

	wtPath, err := m.WorktreePath(branch)
	if err != nil {
		return nil, nil, err
	}

	if err := validate.WorktreePath(wtPath, m.RepoDir); err != nil {
		return nil, nil, err
	}

	if _, err := os.Lstat(wtPath); err == nil {
		return nil, nil, fmt.Errorf("worktree already exists at %s", wtPath)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, fmt.Errorf("cannot access worktree path %s: %w", wtPath, err)
	}

	branchExists := m.Git.BranchExists(branch)

	base := m.Config.DefaultBase
	if base == "" {
		base, err = m.Git.DefaultBranch()
		if err != nil {
			return nil, nil, fmt.Errorf("cannot determine base branch: %w", err)
		}
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
	m.runPostCreateHook(wtPath, branch)

	info := &WorktreeInfo{Path: wtPath, Branch: branch, Created: true, Base: base}
	return info, nil, nil
}

// List returns all worktrees.
func (m *Manager) List() ([]WorktreeInfo, error) {
	entries, err := m.Git.WorktreeList()
	if err != nil {
		return nil, err
	}

	var infos []WorktreeInfo
	for _, e := range entries {
		infos = append(infos, WorktreeInfo{
			Path:   e.Path,
			Branch: e.Branch,
			Head:   e.Head,
			Bare:   e.Bare,
			Main:   e.Main,
		})
	}
	return infos, nil
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

	if err := m.Git.WorktreeRemove(found.Path); err != nil {
		return nil, nil, fmt.Errorf("removing worktree: %w", err)
	}
	result.Removed = true

	if !opts.KeepBranch {
		if err := m.Git.BranchDelete(branch); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not delete branch %s: %v\n", branch, err)
		} else {
			result.BranchDeleted = true
		}
	}

	return result, nil, nil
}

func (m *Manager) copyFiles(wtPath string) {
	for _, pattern := range m.Config.CopyFiles {
		src := filepath.Join(m.RepoDir, pattern)
		dst := filepath.Join(wtPath, pattern)
		if err := copyPath(src, dst); err != nil {
			// Skip silently if source doesn't exist
			continue
		}
	}
}

func (m *Manager) symlinkFiles(wtPath string) {
	for _, pattern := range m.Config.SymlinkFiles {
		src := filepath.Join(m.RepoDir, pattern)
		dst := filepath.Join(wtPath, pattern)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			continue
		}
		os.Symlink(src, dst)
	}
}

func (m *Manager) runPostCreateHook(wtPath, branch string) {
	if m.Config.PostCreateHook == "" {
		return
	}
	cmd := exec.Command("sh", "-c", m.Config.PostCreateHook)
	cmd.Dir = wtPath
	cmd.Env = append(os.Environ(),
		"WW_BRANCH="+branch,
		"WW_WORKTREE_PATH="+wtPath,
	)
	cmd.Stdout = os.Stdout
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
