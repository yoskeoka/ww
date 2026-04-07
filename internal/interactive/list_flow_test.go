package interactive

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type scriptedPrompter struct {
	inputs []string
	index  int
	output *bytes.Buffer
}

func (s *scriptedPrompter) ReadLine(prompt string) (string, error) {
	if s.output != nil {
		if _, err := s.output.WriteString(prompt); err != nil {
			return "", err
		}
	}
	if s.index >= len(s.inputs) {
		return "", fmt.Errorf("unexpected prompt read")
	}
	value := s.inputs[s.index]
	s.index++
	return value, nil
}

func TestFilterWorktreeItemsMatchesExpectedFields(t *testing.T) {
	items := []WorktreeItem{
		{
			Repo:        "repo1",
			Branch:      "feat/alpha",
			Status:      "merged(heuristic-base)",
			Path:        "/tmp/workspace/.worktrees/repo1@feat-alpha",
			DisplayPath: "/tmp/workspace/.worktrees/repo1@feat-alpha",
		},
		{
			Repo:        "repo2",
			Branch:      "bugfix/beta",
			Status:      "active",
			Path:        "/tmp/workspace/.worktrees/repo2@bugfix-beta",
			DisplayPath: "/tmp/workspace/.worktrees/repo2@bugfix-beta",
		},
	}

	cases := map[string]string{
		"repo":   "repo2",
		"branch": "alpha",
		"status": "heuristic-base",
		"path":   "repo2@bugfix-beta",
	}

	for name, filter := range cases {
		t.Run(name, func(t *testing.T) {
			got := FilterWorktreeItems(items, filter)
			if len(got) != 1 {
				t.Fatalf("FilterWorktreeItems(%q) returned %d items, want 1", filter, len(got))
			}
		})
	}
}

func TestListFlowOpenReturnsSessionComplete(t *testing.T) {
	var prompt bytes.Buffer
	var stdout bytes.Buffer
	flow := ListFlow{
		Prompt:  &prompt,
		Session: &scriptedPrompter{inputs: []string{"1", "open"}, output: &prompt},
		Load: func() ([]WorktreeItem, error) {
			return []WorktreeItem{{
				Repo:        "repo1",
				Branch:      "feat/alpha",
				Status:      "active",
				Path:        "/tmp/repo1@feat-alpha",
				DisplayPath: "/tmp/repo1@feat-alpha",
			}}, nil
		},
		Open: func(item WorktreeItem) error {
			_, err := fmt.Fprintln(&stdout, item.Path)
			return err
		},
		Remove: func(WorktreeItem) error {
			t.Fatal("remove should not be called")
			return nil
		},
	}

	err := flow.Run()
	if err != ErrSessionComplete {
		t.Fatalf("ListFlow.Run() error = %v, want ErrSessionComplete", err)
	}
	if got := stdout.String(); got != "/tmp/repo1@feat-alpha\n" {
		t.Fatalf("stdout = %q, want selected path only", got)
	}
}

func TestListFlowDoesNotOfferRemoveForMainWorktree(t *testing.T) {
	var prompt bytes.Buffer
	removeCalls := 0
	flow := ListFlow{
		Prompt:  &prompt,
		Session: &scriptedPrompter{inputs: []string{"1", "remove", "back", "back"}, output: &prompt},
		Load: func() ([]WorktreeItem, error) {
			return []WorktreeItem{{
				Repo:        "repo1",
				Branch:      "main",
				Status:      "active",
				Path:        "/tmp/repo1",
				DisplayPath: "/tmp/repo1",
				Main:        true,
			}}, nil
		},
		Open: func(WorktreeItem) error { return nil },
		Remove: func(WorktreeItem) error {
			removeCalls++
			return nil
		},
	}

	if err := flow.Run(); err != nil {
		t.Fatalf("ListFlow.Run() error = %v", err)
	}
	if removeCalls != 0 {
		t.Fatalf("removeCalls = %d, want 0", removeCalls)
	}
	if !strings.Contains(prompt.String(), "Select action [open/back]: ") {
		t.Fatalf("prompt should omit remove for main worktree:\n%s", prompt.String())
	}
}

func TestListFlowRemoveConfirmsAndReloads(t *testing.T) {
	var prompt bytes.Buffer
	removeCalls := 0
	flow := ListFlow{
		Prompt:  &prompt,
		Session: &scriptedPrompter{inputs: []string{"1", "remove", "yes"}, output: &prompt},
		Load: func() ([]WorktreeItem, error) {
			if removeCalls > 0 {
				return nil, nil
			}
			return []WorktreeItem{{
				Repo:        "repo1",
				Branch:      "feat/remove-me",
				Status:      "merged",
				Path:        "/tmp/repo1@feat-remove-me",
				DisplayPath: "/tmp/repo1@feat-remove-me",
			}}, nil
		},
		Open: func(WorktreeItem) error { return nil },
		Remove: func(item WorktreeItem) error {
			removeCalls++
			if item.Branch != "feat/remove-me" {
				t.Fatalf("Remove() branch = %q, want feat/remove-me", item.Branch)
			}
			return nil
		},
	}

	if err := flow.Run(); err != nil {
		t.Fatalf("ListFlow.Run() error = %v", err)
	}
	if removeCalls != 1 {
		t.Fatalf("removeCalls = %d, want 1", removeCalls)
	}
	if !strings.Contains(prompt.String(), "Remove worktree /tmp/repo1@feat-remove-me (branch: feat/remove-me)? [y/N]: ") {
		t.Fatalf("prompt should include removal preview:\n%s", prompt.String())
	}
}

func TestAvailableListActionsDetachedWorktreeAllowsBackOnly(t *testing.T) {
	actions := AvailableListActions(WorktreeItem{Repo: "repo1", Path: "/tmp/detached"})
	if len(actions) != 1 || actions[0] != ListActionBack {
		t.Fatalf("AvailableListActions(detached) = %v, want [back]", actions)
	}
}
