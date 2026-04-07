package interactive

import (
	"fmt"
	"io"
	"strconv"
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

type ListFlow struct {
	Prompt  io.Writer
	Session Prompter
	Load    func() ([]WorktreeItem, error)
	Open    func(WorktreeItem) error
	Remove  func(WorktreeItem) error
}

func (f ListFlow) Run() error {
	filter := ""
	for {
		items, err := f.Load()
		if err != nil {
			return err
		}
		if len(items) == 0 {
			_, err := fmt.Fprintln(f.Prompt, "No worktrees found.")
			return err
		}

		matches := FilterWorktreeItems(items, filter)
		if err := writeListBrowser(f.Prompt, filter, matches); err != nil {
			return err
		}

		input, err := f.Session.ReadLine("Enter filter text, worktree number, or 'back': ")
		if err != nil {
			return err
		}
		switch {
		case strings.EqualFold(input, "back"):
			return nil
		case input == "":
			filter = ""
			continue
		}

		index, ok := parseSelectionIndex(input, len(matches))
		if !ok {
			filter = input
			continue
		}

		if err := f.runSelected(matches[index]); err != nil {
			return err
		}
	}
}

func (f ListFlow) runSelected(item WorktreeItem) error {
	for {
		actions := AvailableListActions(item)
		if _, err := fmt.Fprintf(f.Prompt, "Selected: %s\n", FormatWorktreeItem(item)); err != nil {
			return err
		}

		input, err := f.Session.ReadLine(fmt.Sprintf("Select action [%s]: ", strings.Join(listActionNames(actions), "/")))
		if err != nil {
			return err
		}

		action, ok := parseListAction(input, actions)
		if !ok {
			if _, err := fmt.Fprintf(f.Prompt, "Unknown action %q.\n", input); err != nil {
				return err
			}
			continue
		}

		switch action {
		case ListActionOpen:
			if err := f.Open(item); err != nil {
				return err
			}
			return ErrSessionComplete
		case ListActionRemove:
			if _, err := fmt.Fprintf(f.Prompt, "Remove worktree %s (branch: %s)? [y/N]: ", item.Path, item.Branch); err != nil {
				return err
			}
			confirm, err := f.Session.ReadLine("")
			if err != nil {
				return err
			}
			if !isConfirmed(confirm) {
				if _, err := fmt.Fprintln(f.Prompt, "Removal cancelled."); err != nil {
					return err
				}
				return nil
			}
			if err := f.Remove(item); err != nil {
				return err
			}
			return nil
		case ListActionBack:
			return nil
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

func writeListBrowser(w io.Writer, filter string, matches []WorktreeItem) error {
	if _, err := fmt.Fprintln(w, "Worktrees"); err != nil {
		return err
	}
	if filter == "" {
		if _, err := fmt.Fprintln(w, "Filter: (all)"); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(w, "Filter: %s\n", filter); err != nil {
		return err
	}

	if len(matches) == 0 {
		if _, err := fmt.Fprintln(w, "  No worktrees match that filter."); err != nil {
			return err
		}
		_, err := fmt.Fprintln(w)
		return err
	}

	for i, item := range matches {
		if _, err := fmt.Fprintf(w, "  %d. %s\n", i+1, FormatWorktreeItem(item)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

func parseSelectionIndex(input string, count int) (int, bool) {
	index, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || index < 1 || index > count {
		return 0, false
	}
	return index - 1, true
}

func parseListAction(input string, actions []ListAction) (ListAction, bool) {
	input = strings.ToLower(strings.TrimSpace(input))
	for _, action := range actions {
		switch action {
		case ListActionOpen:
			if input == "open" || input == "o" || input == "1" {
				return action, true
			}
		case ListActionRemove:
			if input == "remove" || input == "r" || input == "2" {
				return action, true
			}
		case ListActionBack:
			if input == "back" || input == "b" || input == "3" {
				return action, true
			}
		}
	}
	return "", false
}

func listActionNames(actions []ListAction) []string {
	names := make([]string, 0, len(actions))
	for _, action := range actions {
		names = append(names, string(action))
	}
	return names
}

func isConfirmed(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
