package validate

import (
	"testing"
)

func TestBranchNameValid(t *testing.T) {
	valid := []string{
		"main",
		"feat/my-feature",
		"fix/bug-123",
		"release/v1.0",
		"user/name/branch",
	}
	for _, name := range valid {
		if err := BranchName(name); err != nil {
			t.Errorf("BranchName(%q) = %v, want nil", name, err)
		}
	}
}

func TestBranchNameInvalid(t *testing.T) {
	invalid := []string{
		"",
		"-starts-with-dash",
		"has..double-dot",
		"has space",
		"has~tilde",
		"has^caret",
		"has:colon",
		"has\\backslash",
		"has*glob",
		".starts-with-dot",
		"ends.lock",
		"has/@{at-brace",
		"@",
		"has\x00control",
	}
	for _, name := range invalid {
		if err := BranchName(name); err == nil {
			t.Errorf("BranchName(%q) = nil, want error", name)
		}
	}
}

func TestWorktreePathValid(t *testing.T) {
	if err := WorktreePath("/home/user/repo@feat", "/home/user/repo"); err != nil {
		t.Errorf("WorktreePath valid sibling: %v", err)
	}
}

func TestWorktreePathInvalid(t *testing.T) {
	if err := WorktreePath("/tmp/evil", "/home/user/repo"); err == nil {
		t.Error("WorktreePath should reject path outside workspace")
	}
}
