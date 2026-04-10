package interactive

import (
	"fmt"
	"testing"
)

type fakeListUI struct {
	selectedItems  []*WorktreeItem
	selectIndex    int
	nextFilter     string
	actions        []ListAction
	actionIndex    int
	confirm        bool
	selectErr      error
	actionErr      error
	confirmErr     error
	selectedFilter string
	actionsSeen    []ListAction
}

func (f *fakeListUI) SelectWorktree(_ []WorktreeItem, filter string) (*WorktreeItem, string, error) {
	f.selectedFilter = filter
	if f.selectErr != nil {
		return nil, f.nextFilter, f.selectErr
	}
	if f.selectIndex >= len(f.selectedItems) {
		return nil, f.nextFilter, nil
	}
	item := f.selectedItems[f.selectIndex]
	f.selectIndex++
	return item, f.nextFilter, nil
}

func (f *fakeListUI) SelectListAction(_ WorktreeItem, actions []ListAction) (ListAction, error) {
	f.actionsSeen = append([]ListAction(nil), actions...)
	if f.actionErr != nil {
		return "", f.actionErr
	}
	if f.actionIndex >= len(f.actions) {
		return "", fmt.Errorf("unexpected action prompt")
	}
	action := f.actions[f.actionIndex]
	f.actionIndex++
	return action, nil
}

func (f *fakeListUI) ConfirmRemove(_ WorktreeItem) (bool, error) {
	return f.confirm, f.confirmErr
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
	item := WorktreeItem{
		Repo:        "repo1",
		Branch:      "feat/alpha",
		Status:      "active",
		Path:        "/tmp/repo1@feat-alpha",
		DisplayPath: "/tmp/repo1@feat-alpha",
	}
	ui := &fakeListUI{
		selectedItems: []*WorktreeItem{&item},
		actions:       []ListAction{ListActionOpen},
	}
	openCalls := 0
	flow := ListFlow{
		UI: ui,
		Load: func() ([]WorktreeItem, error) {
			return []WorktreeItem{item}, nil
		},
		Open: func(got WorktreeItem) error {
			openCalls++
			if got.Path != item.Path {
				t.Fatalf("Open() path = %q, want %q", got.Path, item.Path)
			}
			return nil
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
	if openCalls != 1 {
		t.Fatalf("openCalls = %d, want 1", openCalls)
	}
}

func TestListFlowDoesNotOfferRemoveForMainWorktree(t *testing.T) {
	item := WorktreeItem{
		Repo:        "repo1",
		Branch:      "main",
		Status:      "active",
		Path:        "/tmp/repo1",
		DisplayPath: "/tmp/repo1",
		Main:        true,
	}
	ui := &fakeListUI{
		selectedItems: []*WorktreeItem{&item, nil},
		actions:       []ListAction{ListActionBack},
	}
	removeCalls := 0
	flow := ListFlow{
		UI: ui,
		Load: func() ([]WorktreeItem, error) {
			return []WorktreeItem{item}, nil
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
	if len(ui.actionsSeen) != 2 || ui.actionsSeen[0] != ListActionOpen || ui.actionsSeen[1] != ListActionBack {
		t.Fatalf("actionsSeen = %v, want [open back]", ui.actionsSeen)
	}
}

func TestListFlowRemoveConfirmsAndReloads(t *testing.T) {
	item := WorktreeItem{
		Repo:        "repo1",
		Branch:      "feat/remove-me",
		Status:      "merged",
		Path:        "/tmp/repo1@feat-remove-me",
		DisplayPath: "/tmp/repo1@feat-remove-me",
	}
	ui := &fakeListUI{
		selectedItems: []*WorktreeItem{&item, nil},
		nextFilter:    "repo1",
		actions:       []ListAction{ListActionRemove},
		confirm:       true,
	}
	removeCalls := 0
	flow := ListFlow{
		UI: ui,
		Load: func() ([]WorktreeItem, error) {
			if removeCalls > 0 {
				return []WorktreeItem{item}, nil
			}
			return []WorktreeItem{item}, nil
		},
		Open: func(WorktreeItem) error { return nil },
		Remove: func(got WorktreeItem) error {
			removeCalls++
			if got.Branch != item.Branch {
				t.Fatalf("Remove() branch = %q, want %q", got.Branch, item.Branch)
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
	if ui.selectedFilter != "repo1" {
		t.Fatalf("selectedFilter = %q, want repo1 after reload", ui.selectedFilter)
	}
}

func TestAvailableListActionsDetachedWorktreeAllowsBackOnly(t *testing.T) {
	actions := AvailableListActions(WorktreeItem{Repo: "repo1", Path: "/tmp/detached"})
	if len(actions) != 1 || actions[0] != ListActionBack {
		t.Fatalf("AvailableListActions(detached) = %v, want [back]", actions)
	}
}

func TestListFlowReturnsSelectError(t *testing.T) {
	wantErr := fmt.Errorf("boom")
	flow := ListFlow{
		UI: &fakeListUI{selectErr: wantErr},
		Load: func() ([]WorktreeItem, error) {
			return []WorktreeItem{{Repo: "repo1", Branch: "main", Path: "/tmp/repo1"}}, nil
		},
	}
	if err := flow.Run(); err != wantErr {
		t.Fatalf("ListFlow.Run() error = %v, want %v", err, wantErr)
	}
}
