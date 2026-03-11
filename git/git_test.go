package git

import (
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	input := `worktree /home/user/myrepo
HEAD abc1234def5678901234567890123456789012
branch refs/heads/main

worktree /home/user/myrepo@feat-auth
HEAD def5678abc1234901234567890123456789012
branch refs/heads/feat/auth

`
	entries := parseWorktreeList(input)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Path != "/home/user/myrepo" {
		t.Errorf("entry[0].Path = %q, want /home/user/myrepo", entries[0].Path)
	}
	if entries[0].Branch != "main" {
		t.Errorf("entry[0].Branch = %q, want main", entries[0].Branch)
	}
	if entries[0].Head != "abc1234" {
		t.Errorf("entry[0].Head = %q, want abc1234", entries[0].Head)
	}

	if entries[1].Branch != "feat/auth" {
		t.Errorf("entry[1].Branch = %q, want feat/auth", entries[1].Branch)
	}
}

func TestParseWorktreeListBare(t *testing.T) {
	input := `worktree /home/user/myrepo.git
bare

`
	entries := parseWorktreeList(input)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].Bare {
		t.Error("expected bare = true")
	}
}
