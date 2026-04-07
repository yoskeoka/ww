package interactive

import (
	"fmt"
	"strings"
)

type ListAction string

const (
	ListActionOpen   ListAction = "open"
	ListActionRemove ListAction = "remove"
	ListActionBack   ListAction = "back"
)

type WorktreeItem struct {
	Repo        string
	Branch      string
	Status      string
	Path        string
	DisplayPath string
	Main        bool
}

type ListPrompter interface {
	SelectWorktree(items []WorktreeItem, filter string) (*WorktreeItem, string, error)
	SelectListAction(item WorktreeItem, actions []ListAction) (ListAction, error)
	ConfirmRemove(item WorktreeItem) (bool, error)
}

type ListFlow struct {
	UI     ListPrompter
	Load   func() ([]WorktreeItem, error)
	Open   func(WorktreeItem) error
	Remove func(WorktreeItem) error
}

func (f ListFlow) Run() error {
	filter := ""
	for {
		items, err := f.Load()
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return fmt.Errorf("no worktrees found")
		}

		item, nextFilter, err := f.UI.SelectWorktree(items, filter)
		if err != nil {
			return err
		}
		filter = nextFilter
		if item == nil {
			return nil
		}

		actions := AvailableListActions(*item)
		action, err := f.UI.SelectListAction(*item, actions)
		if err != nil {
			return err
		}

		switch action {
		case ListActionOpen:
			if err := f.Open(*item); err != nil {
				return err
			}
			return ErrSessionComplete
		case ListActionRemove:
			confirm, err := f.UI.ConfirmRemove(*item)
			if err != nil {
				return err
			}
			if !confirm {
				continue
			}
			if err := f.Remove(*item); err != nil {
				return err
			}
		case ListActionBack:
			continue
		default:
			return fmt.Errorf("unknown list action %q", action)
		}
	}
}

func FilterWorktreeItems(items []WorktreeItem, filter string) []WorktreeItem {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		out := make([]WorktreeItem, len(items))
		copy(out, items)
		return out
	}

	out := make([]WorktreeItem, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(worktreeSearchText(item)), filter) {
			out = append(out, item)
		}
	}
	return out
}

func FormatWorktreeItem(item WorktreeItem) string {
	branch := item.Branch
	if branch == "" {
		branch = "(no branch)"
	}
	mainMarker := ""
	if item.Main {
		mainMarker = " [main worktree]"
	}
	return fmt.Sprintf("%s | %s | %s | %s%s", item.Repo, branch, item.Status, item.DisplayPath, mainMarker)
}

func AvailableListActions(item WorktreeItem) []ListAction {
	var actions []ListAction
	if item.Branch != "" {
		actions = append(actions, ListActionOpen)
	}
	if !item.Main && item.Branch != "" {
		actions = append(actions, ListActionRemove)
	}
	return append(actions, ListActionBack)
}

func worktreeSearchText(item WorktreeItem) string {
	return strings.Join([]string{
		item.Repo,
		item.Branch,
		item.Status,
		item.Path,
	}, " ")
}
