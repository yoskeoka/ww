package validate

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BranchName validates a branch name against git check-ref-format rules (subset).
func BranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("branch name contains control character: %q", name)
		}
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("branch name cannot contain '..': %q", name)
	}
	if strings.Contains(name, "~") || strings.Contains(name, "^") || strings.Contains(name, ":") {
		return fmt.Errorf("branch name contains invalid character (~, ^, or :): %q", name)
	}
	if strings.Contains(name, "*") {
		return fmt.Errorf("branch name contains invalid character '*': %q", name)
	}
	if strings.Contains(name, " ") || strings.Contains(name, "\\") {
		return fmt.Errorf("branch name contains space or backslash: %q", name)
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("branch name cannot start with '-': %q", name)
	}
	if strings.HasSuffix(name, ".lock") {
		return fmt.Errorf("branch name cannot end with '.lock': %q", name)
	}
	if strings.HasPrefix(name, ".") || strings.Contains(name, "/.") {
		return fmt.Errorf("branch name component cannot start with '.': %q", name)
	}
	if strings.HasSuffix(name, ".") || strings.Contains(name, "./") {
		return fmt.Errorf("branch name component cannot end with '.': %q", name)
	}
	if strings.Contains(name, "@{") {
		return fmt.Errorf("branch name cannot contain '@{': %q", name)
	}
	if name == "@" {
		return fmt.Errorf("branch name cannot be '@'")
	}
	return nil
}

// WorktreePath validates that a worktree path is safe.
func WorktreePath(path, workspaceRoot string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return fmt.Errorf("invalid workspace root: %w", err)
	}
	// Path must be under or adjacent to the workspace root's parent
	rootParent := filepath.Dir(absRoot)
	if !strings.HasPrefix(absPath, rootParent+string(filepath.Separator)) && absPath != rootParent {
		return fmt.Errorf("worktree path %q is outside workspace area %q", absPath, rootParent)
	}
	return nil
}
