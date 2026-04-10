package interactive

import "fmt"

type RepoOption struct {
	Name string
}

type CreatePreview struct {
	Repo         string
	Branch       string
	Path         string
	BranchExists bool
	Base         string
	CopyFiles    []string
	SymlinkFiles []string
	Hook         string
}

type CreatePrompter interface {
	SelectCreateRepo(repos []RepoOption) (string, error)
	InputCreateBranch(repo string) (string, error)
	ConfirmCreate(preview CreatePreview) (bool, error)
}

type CreateFlow struct {
	UI            CreatePrompter
	WorkspaceMode bool
	Repos         []RepoOption
	BuildPreview  func(repo, branch string) (CreatePreview, error)
	Execute       func(repo, branch string) error
}

func (f CreateFlow) Run() error {
	repo := ""
	if f.WorkspaceMode {
		if len(f.Repos) == 0 {
			return fmt.Errorf("no repos available for interactive create")
		}
		selected, err := f.UI.SelectCreateRepo(f.Repos)
		if err != nil {
			return err
		}
		repo = selected
	}

	branch, err := f.UI.InputCreateBranch(repo)
	if err != nil {
		return err
	}

	preview, err := f.BuildPreview(repo, branch)
	if err != nil {
		return err
	}

	confirmed, err := f.UI.ConfirmCreate(preview)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	return f.Execute(repo, branch)
}
