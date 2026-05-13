package main

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/git"
	"github.com/yoskeoka/ww/workspace"
	"github.com/yoskeoka/ww/worktree"
)

func worktreeCreateOpts(glOpts *globalOpts, quiet bool, guessRemote bool) worktree.CreateOpts {
	return worktree.CreateOpts{
		DryRun:      glOpts.dryRun,
		Output:      glOpts.output,
		TextMode:    !glOpts.json && !quiet,
		GuessRemote: guessRemote,
	}
}

func managerForSelectedRepo(repoName string, requireRepo bool, sandbox bool) (*worktree.Manager, error) {
	if repoName == "" {
		return newManagerWithOptions(requireRepo, sandbox)
	}

	base, err := newManagerWithOptions(false, sandbox)
	if err != nil {
		return nil, err
	}
	if base.Workspace == nil || base.Workspace.Mode != workspace.ModeWorkspace {
		return nil, fmt.Errorf("--repo can only be used inside a detected workspace")
	}
	return managerForRepo(base, repoName)
}

func managerForRepo(base *worktree.Manager, repoName string) (*worktree.Manager, error) {
	if base.Workspace == nil || base.Workspace.Mode != workspace.ModeWorkspace {
		return base, nil
	}

	for _, repo := range base.Workspace.Repos {
		if repo.Name != repoName {
			continue
		}
		return &worktree.Manager{
			Git: &git.Runner{Dir: repo.Path},
			Config: worktree.Config{
				WorktreeDir:    base.Config.WorktreeDir,
				DefaultBase:    base.Config.DefaultBase,
				CopyFiles:      base.Config.CopyFiles,
				SymlinkFiles:   base.Config.SymlinkFiles,
				PostCreateHook: base.Config.PostCreateHook,
				Sandbox:        base.Config.Sandbox,
			},
			RepoDir:   repo.Path,
			Workspace: base.Workspace,
		}, nil
	}

	return nil, fmt.Errorf("repo %q not found in workspace", repoName)
}

// parseFlags parses a subcommand flagset, returning errHelp for --help
// so the caller can exit cleanly without leaking pflag internals.
func parseFlags(fset *pflag.FlagSet, args []string) error {
	if err := fset.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return errHelp
		}
		return err
	}
	return nil
}
