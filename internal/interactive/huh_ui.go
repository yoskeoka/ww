package interactive

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

const (
	backValue    = "__back__"
	noMatchesKey = "__no_matches__"
)

type HuhSession struct {
	Input  io.Reader
	Output io.Writer
}

func NewHuhSession(input io.Reader, output io.Writer) *HuhSession {
	return &HuhSession{Input: input, Output: output}
}

func (s *HuhSession) SelectAction() (Action, error) {
	var action Action
	err := runHuhForm(s.Input, s.Output,
		huh.NewSelect[Action]().
			Title("Select action").
			Description("Use arrows or j/k to move, enter to confirm, q to quit.").
			Options(
				huh.NewOption("create (planned)", ActionCreate),
				huh.NewOption("list", ActionList),
				huh.NewOption("clean (planned)", ActionClean),
				huh.NewOption("quit", ActionQuit),
			).
			Value(&action).
			Height(4),
	)
	if err != nil {
		return "", err
	}
	return action, nil
}

type HuhListUI struct {
	Input  io.Reader
	Output io.Writer
}

func NewHuhListUI(input io.Reader, output io.Writer) *HuhListUI {
	return &HuhListUI{Input: input, Output: output}
}

func (ui *HuhListUI) SelectWorktree(items []WorktreeItem, filter string) (*WorktreeItem, string, error) {
	selected := backValue
	currentFilter := filter
	index := make(map[string]WorktreeItem, len(items))

	err := runHuhForm(ui.Input, ui.Output,
		huh.NewInput().
			Title("Filter worktrees").
			Description("Match repo, branch, status, or full path. Leave empty for all.").
			Value(&currentFilter),
		huh.NewSelect[string]().
			Title("Select worktree").
			Description("Use arrows or j/k to move, enter to select, / to filter the visible list, q to quit.").
			OptionsFunc(func() []huh.Option[string] {
				return worktreeOptions(items, currentFilter, index)
			}, &currentFilter).
			Value(&selected).
			Height(min(10, max(2, len(items)+1))),
	)
	if err != nil {
		return nil, filter, err
	}
	if selected == backValue {
		return nil, currentFilter, nil
	}
	if selected == noMatchesKey {
		return nil, currentFilter, nil
	}
	item, ok := index[selected]
	if !ok {
		return nil, currentFilter, fmt.Errorf("unknown worktree selection %q", selected)
	}
	return &item, currentFilter, nil
}

func (ui *HuhListUI) SelectListAction(item WorktreeItem, actions []ListAction) (ListAction, error) {
	var action ListAction
	options := make([]huh.Option[ListAction], 0, len(actions))
	for _, candidate := range actions {
		options = append(options, huh.NewOption(listActionLabel(candidate), candidate))
	}
	err := runHuhForm(ui.Input, ui.Output,
		huh.NewSelect[ListAction]().
			Title("Selected worktree").
			Description(FormatWorktreeItem(item)).
			Options(options...).
			Value(&action).
			Height(len(options)),
	)
	if err != nil {
		return "", err
	}
	return action, nil
}

func (ui *HuhListUI) ConfirmRemove(item WorktreeItem) (bool, error) {
	confirmed := false
	err := runHuhForm(ui.Input, ui.Output,
		huh.NewConfirm().
			Title(fmt.Sprintf("Remove %s?", item.Branch)).
			Description(fmt.Sprintf("This will remove the worktree at %s using `ww remove` parity.", item.Path)).
			Affirmative("Yes").
			Negative("No").
			Value(&confirmed),
	)
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

func runHuhForm(input io.Reader, output io.Writer, fields ...huh.Field) error {
	groupFields := make([]huh.Field, 0, len(fields))
	groupFields = append(groupFields, fields...)

	keymap := huh.NewDefaultKeyMap()
	keymap.Quit = key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	)

	form := huh.NewForm(huh.NewGroup(groupFields...)).
		WithInput(input).
		WithOutput(output).
		WithKeyMap(keymap)

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return ErrSessionComplete
		}
		return err
	}
	return nil
}

func worktreeOptions(items []WorktreeItem, filter string, index map[string]WorktreeItem) []huh.Option[string] {
	clear(index)
	filtered := FilterWorktreeItems(items, filter)
	options := make([]huh.Option[string], 0, len(filtered)+2)
	if len(filtered) == 0 {
		options = append(options, huh.NewOption("No worktrees match the current filter. Back to actions.", noMatchesKey))
	} else {
		for _, item := range filtered {
			value := worktreeValue(item)
			index[value] = item
			options = append(options, huh.NewOption(FormatWorktreeItem(item), value))
		}
	}
	options = append(options, huh.NewOption("Back to actions", backValue))
	return options
}

func worktreeValue(item WorktreeItem) string {
	return item.Repo + "\x00" + item.Path + "\x00" + item.Branch
}

func listActionLabel(action ListAction) string {
	switch action {
	case ListActionOpen:
		return "open"
	case ListActionRemove:
		return "remove"
	case ListActionBack:
		return "back"
	default:
		return string(action)
	}
}
