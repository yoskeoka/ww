package interactive

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"

	"github.com/yoskeoka/ww/validate"
)

const (
	backValue                     = "__back__"
	noMatchesKey                  = "__no_matches__"
	huhSelectTitleDescriptionRows = 2
	worktreeBrowserVisibleRows    = 5
)

var topLevelActions = []Action{
	ActionCreate,
	ActionList,
	ActionClean,
	ActionQuit,
}

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
			Options(topLevelActionOptions()...).
			Value(&action).
			Height(selectHeightForOptions(len(topLevelActions))),
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
			Height(selectHeightForVisibleOptions(worktreeBrowserVisibleRows)),
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
			Height(selectHeightForOptions(len(options))),
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

func (ui *HuhListUI) SelectCreateRepo(repos []RepoOption) (string, error) {
	var repo string
	options := make([]huh.Option[string], 0, len(repos))
	for _, candidate := range repos {
		options = append(options, huh.NewOption(candidate.Name, candidate.Name))
	}
	err := runHuhForm(ui.Input, ui.Output,
		huh.NewSelect[string]().
			Title("Select repo").
			Description("Choose the repo for `ww create`. Use arrows or j/k to move, enter to confirm, q to quit.").
			Options(options...).
			Value(&repo).
			Height(cappedSelectHeightForOptions(len(options), 10)),
	)
	if err != nil {
		return "", err
	}
	return repo, nil
}

func (ui *HuhListUI) InputCreateBranch(repo string) (string, error) {
	var branch string
	title := "Branch name"
	description := "Enter the branch to create or open with `ww create` parity."
	if repo != "" {
		description = fmt.Sprintf("Repo: %s. Enter the branch to create or open with `ww create --repo %s` parity.", repo, repo)
	}
	err := runHuhForm(ui.Input, ui.Output,
		huh.NewInput().
			Title(title).
			Description(description).
			Value(&branch).
			Validate(validateInteractiveBranch),
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(branch), nil
}

func (ui *HuhListUI) ConfirmCreate(preview CreatePreview) (bool, error) {
	confirmed := false
	err := runHuhForm(ui.Input, ui.Output,
		huh.NewConfirm().
			Title("Create worktree?").
			Description(formatCreatePreview(preview)).
			Affirmative("Create").
			Negative("Cancel").
			Value(&confirmed),
	)
	if err != nil {
		return false, err
	}
	return confirmed, nil
}

func (ui *HuhListUI) SelectCleanMode(summary []CleanSummary) (CleanMode, error) {
	var mode CleanMode
	err := runHuhForm(ui.Input, ui.Output,
		huh.NewSelect[CleanMode]().
			Title("Clean mode").
			Description(formatCleanSummary(summary)).
			Options(
				huh.NewOption("safe (`ww clean`)", CleanModeSafe),
				huh.NewOption("force (`ww clean --force`)", CleanModeForce),
			).
			Value(&mode).
			Height(selectHeightForOptions(2)),
	)
	if err != nil {
		return "", err
	}
	return mode, nil
}

func (ui *HuhListUI) ConfirmClean(mode CleanMode, targets []CleanTarget) (bool, error) {
	confirmed := false
	err := runHuhForm(ui.Input, ui.Output,
		huh.NewConfirm().
			Title(fmt.Sprintf("Run %s clean?", mode)).
			Description(formatCleanConfirmation(mode, targets)).
			Affirmative("Run").
			Negative("Cancel").
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

func topLevelActionOptions() []huh.Option[Action] {
	options := make([]huh.Option[Action], 0, len(topLevelActions))
	for _, action := range topLevelActions {
		options = append(options, huh.NewOption(string(action), action))
	}
	return options
}

func selectHeightForOptions(optionCount int) int {
	return selectHeightForVisibleOptions(optionCount)
}

func cappedSelectHeightForOptions(optionCount, maxVisible int) int {
	return selectHeightForVisibleOptions(min(max(1, optionCount), maxVisible))
}

func selectHeightForVisibleOptions(visibleOptions int) int {
	return max(1, visibleOptions) + huhSelectTitleDescriptionRows
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

func formatCreatePreview(preview CreatePreview) string {
	lines := []string{
		fmt.Sprintf("Branch: %s", preview.Branch),
		fmt.Sprintf("Path: %s", preview.Path),
	}
	if preview.Repo != "" {
		lines = append([]string{fmt.Sprintf("Repo: %s", preview.Repo)}, lines...)
	}
	if preview.BranchExists {
		lines = append(lines, "Branch source: existing branch")
	} else {
		lines = append(lines, fmt.Sprintf("Branch source: new branch from %s", preview.Base))
	}
	lines = append(lines, formatActionList("Copy", preview.CopyFiles))
	lines = append(lines, formatActionList("Symlink", preview.SymlinkFiles))
	if preview.Hook != "" {
		lines = append(lines, fmt.Sprintf("Hook: %s", preview.Hook))
	} else {
		lines = append(lines, "Hook: none")
	}
	return strings.Join(lines, "\n")
}

func formatCleanSummary(summary []CleanSummary) string {
	lines := make([]string, 0, len(summary)+1)
	lines = append(lines, "Cleanable worktrees by repo:")
	for _, entry := range summary {
		lines = append(lines, fmt.Sprintf("- %s: %d", entry.Repo, entry.Count))
	}
	return strings.Join(lines, "\n")
}

func formatCleanConfirmation(mode CleanMode, targets []CleanTarget) string {
	lines := []string{fmt.Sprintf("Mode: %s", mode), "Targets:"}
	for _, target := range targets {
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | %s", target.Repo, target.Branch, target.Status, target.Path))
	}
	return strings.Join(lines, "\n")
}

func formatActionList(label string, values []string) string {
	if len(values) == 0 {
		return label + ": none"
	}
	return fmt.Sprintf("%s: %s", label, strings.Join(values, ", "))
}

func validateInteractiveBranch(branch string) error {
	return validate.BranchName(strings.TrimSpace(branch))
}
