package interactive

import (
	"fmt"
	"io"
	"slices"
)

type CleanMode string

const (
	CleanModeSafe  CleanMode = "safe"
	CleanModeForce CleanMode = "force"
)

type CleanSummary struct {
	Repo  string
	Count int
}

type CleanTarget struct {
	Repo   string
	Branch string
	Status string
	Path   string
}

type CleanSnapshot struct {
	Targets []CleanTarget
	State   any
}

type CleanPrompter interface {
	SelectCleanMode(summary []CleanSummary) (CleanMode, error)
	ConfirmClean(mode CleanMode, targets []CleanTarget) (bool, error)
}

type CleanFlow struct {
	UI        CleanPrompter
	Output    io.Writer
	RepoNames []string
	Load      func() (CleanSnapshot, error)
	Execute   func(mode CleanMode, snapshot CleanSnapshot) error
}

func (f CleanFlow) Run() error {
	snapshot, err := f.Load()
	if err != nil {
		return err
	}
	targets := snapshot.Targets
	if len(targets) == 0 {
		_, err := fmt.Fprintln(f.Output, "No cleanable worktrees found.")
		return err
	}

	mode, err := f.UI.SelectCleanMode(BuildCleanSummary(f.RepoNames, targets))
	if err != nil {
		return err
	}

	confirmed, err := f.UI.ConfirmClean(mode, targets)
	if err != nil {
		return err
	}
	if !confirmed {
		return nil
	}

	return f.Execute(mode, snapshot)
}

func BuildCleanSummary(repoNames []string, targets []CleanTarget) []CleanSummary {
	counts := make(map[string]int, len(repoNames))
	for _, repo := range repoNames {
		counts[repo] = 0
	}
	for _, target := range targets {
		counts[target.Repo]++
	}

	summary := make([]CleanSummary, 0, len(counts))
	for _, repo := range repoNames {
		summary = append(summary, CleanSummary{Repo: repo, Count: counts[repo]})
		delete(counts, repo)
	}

	var extraRepos []string
	for repo := range counts {
		extraRepos = append(extraRepos, repo)
	}
	slices.Sort(extraRepos)
	for _, repo := range extraRepos {
		summary = append(summary, CleanSummary{Repo: repo, Count: counts[repo]})
	}

	return summary
}
