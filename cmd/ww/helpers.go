package main

import (
	"errors"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/git"
	"github.com/yoskeoka/ww/internal/config"
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
		cfg, err := loadRepoConfigForSelection(base, repo.Path)
		if err != nil {
			return nil, err
		}
		return &worktree.Manager{
			Git: &git.Runner{Dir: repo.Path},
			Config: worktree.Config{
				WorktreeDir:    cfg.WorktreeDir,
				DefaultBase:    cfg.DefaultBase,
				CopyFiles:      cfg.CopyFiles,
				SymlinkFiles:   cfg.SymlinkFiles,
				PostCreateHook: cfg.PostCreateHook,
				Sandbox:        cfg.Sandbox,
			},
			RepoDir:   repo.Path,
			Workspace: base.Workspace,
		}, nil
	}

	return nil, fmt.Errorf("repo %q not found in workspace", repoName)
}

func loadRepoConfigForSelection(base *worktree.Manager, repoPath string) (*config.Config, error) {
	sandboxMode := base.Config.Sandbox
	if !sandboxMode {
		preCfg, err := config.LoadWithOptions(repoPath, config.LoadOptions{
			ProjectRoot: repoPath,
		})
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		sandboxMode = preCfg.Sandbox
	}

	cfg, err := config.LoadWithOptions(repoPath, config.LoadOptions{
		Sandbox:      sandboxMode,
		Boundary:     sandboxBoundary(base.Workspace, repoPath),
		FallbackDirs: sandboxFallbackDirs(sandboxMode, repoPath, base.Workspace.Root),
		ProjectRoot:  repoPath,
	})
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Preserve explicit CLI sandbox selection from the already-resolved base manager.
	if base.Config.Sandbox {
		cfg.Sandbox = true
	}
	return cfg, nil
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
