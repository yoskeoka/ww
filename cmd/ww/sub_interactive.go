package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/internal/interactive"
	"github.com/yoskeoka/ww/workspace"
	"github.com/yoskeoka/ww/worktree"
)

const interactiveJSONMessage = "interactive mode does not support --json; use standard ww commands for machine-readable output"

func interactiveCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" i", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Unsupported in interactive mode; use standard ww commands for machine-readable output")
	fset.MarkHidden("json")
	fset.Usage = func() {
		out := fset.Output()
		fmt.Fprintln(out, "Start interactive mode")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintln(out, "  ww i")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Notes:")
		fmt.Fprintln(out, "  - requires a TTY on stdin and stderr")
		fmt.Fprintln(out, "  - --json is not supported; use standard ww commands for machine-readable output")
	}

	return command{
		name:        "i",
		description: "Start interactive mode",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			if *jsonFlag {
				return fmt.Errorf(interactiveJSONMessage)
			}
			if len(fset.Args()) > 0 {
				return fmt.Errorf("usage: ww i")
			}

			if err := interactive.ValidateTTY(os.Stdin, os.Stderr, interactive.StatTTYChecker{}); err != nil {
				return err
			}

			mgr, err := newManager(false)
			if err != nil {
				return err
			}

			prompt := interactive.PromptOutput(glOpts.output, glOpts.errOutput)
			session := interactive.NewHuhSession(os.Stdin, prompt)
			ui := interactive.NewHuhListUI(os.Stdin, prompt)
			flows := interactiveFlows{
				prompt: prompt,
				create: interactive.CreateFlow{
					UI:            ui,
					WorkspaceMode: mgr.Workspace != nil && mgr.Workspace.Mode == workspace.ModeWorkspace,
					Repos:         interactiveRepoOptions(mgr.Workspace),
					BuildPreview: func(repo, branch string) (interactive.CreatePreview, error) {
						return buildInteractiveCreatePreview(mgr, repo, branch)
					},
					Execute: func(repo, branch string) error {
						return executeInteractiveCreate(mgr, prompt, repo, branch)
					},
				},
				list: interactive.ListFlow{
					UI: ui,
					Load: func() ([]interactive.WorktreeItem, error) {
						infos, err := mgr.List()
						if err != nil {
							return nil, err
						}
						return worktreeItems(infos), nil
					},
					Open: func(item interactive.WorktreeItem) error {
						_, err := fmt.Fprintln(glOpts.output, item.Path)
						return err
					},
					Remove: func(item interactive.WorktreeItem) error {
						repoMgr, err := managerForRepo(mgr, item.Repo)
						if err != nil {
							return err
						}
						result, _, err := repoMgr.Remove(item.Branch, worktree.RemoveOpts{})
						if err != nil {
							return err
						}
						return writeInteractiveRemoveResult(prompt, result)
					},
				},
				clean: interactive.CleanFlow{
					UI:        ui,
					Output:    prompt,
					RepoNames: interactiveRepoNames(mgr),
					Load: func() ([]interactive.CleanTarget, error) {
						return interactiveCleanTargets(mgr)
					},
					Execute: func(mode interactive.CleanMode) error {
						infos, err := listCleanableWorktrees(mgr)
						if err != nil {
							return err
						}
						return executeCleanWorktrees(mgr, infos, &globalOpts{
							output:    prompt,
							errOutput: prompt,
						}, mode == interactive.CleanModeForce)
					},
				},
			}

			return interactive.Runner{
				Prompt:  prompt,
				Session: session,
				Flows:   flows,
			}.Run(interactiveOverview(mgr.Workspace))
		},
	}
}

type interactiveFlows struct {
	prompt io.Writer
	create interactive.CreateFlow
	list   interactive.ListFlow
	clean  interactive.CleanFlow
}

func (f interactiveFlows) Create() error {
	return f.create.Run()
}

func (f interactiveFlows) List() error {
	return f.list.Run()
}

func (f interactiveFlows) Clean() error {
	return f.clean.Run()
}

func worktreeItems(infos []worktree.WorktreeInfo) []interactive.WorktreeItem {
	items := make([]interactive.WorktreeItem, 0, len(infos))
	for _, info := range infos {
		items = append(items, interactive.WorktreeItem{
			Repo:        info.Repo,
			Branch:      info.Branch,
			Status:      displayStatus(info),
			Path:        info.Path,
			DisplayPath: shortenInteractivePath(info.Path),
			Main:        info.Main,
		})
	}
	return items
}

func shortenInteractivePath(path string) string {
	const maxLen = 48
	runes := []rune(path)
	if len(runes) <= maxLen {
		return path
	}
	return string(runes[:22]) + "..." + string(runes[len(runes)-23:])
}

func writeInteractiveRemoveResult(w io.Writer, result *worktree.RemoveResult) error {
	if _, err := fmt.Fprintf(w, "Removed worktree at %s\n", result.Path); err != nil {
		return err
	}
	if result.BranchDeleted {
		_, err := fmt.Fprintf(w, "Deleted branch %s\n", result.Branch)
		return err
	}
	if result.BranchError != "" {
		_, err := fmt.Fprintf(w, "warning: could not delete branch %s: %s\n", result.Branch, result.BranchError)
		return err
	}
	return nil
}

func interactiveOverview(ws *workspace.Workspace) interactive.Overview {
	overview := interactive.Overview{
		Mode: string(ws.Mode),
		Root: ws.Root,
	}
	if ws.Mode == workspace.ModeWorkspace {
		overview.Repos = make([]string, 0, len(ws.Repos))
		for _, repo := range ws.Repos {
			overview.Repos = append(overview.Repos, repo.Name)
		}
	}
	return overview
}

func interactiveRepoOptions(ws *workspace.Workspace) []interactive.RepoOption {
	if ws == nil || ws.Mode != workspace.ModeWorkspace {
		return nil
	}
	options := make([]interactive.RepoOption, 0, len(ws.Repos))
	for _, repo := range ws.Repos {
		options = append(options, interactive.RepoOption{Name: repo.Name})
	}
	return options
}

func interactiveRepoNames(mgr *worktree.Manager) []string {
	if mgr.Workspace != nil && mgr.Workspace.Mode == workspace.ModeWorkspace {
		names := make([]string, 0, len(mgr.Workspace.Repos))
		for _, repo := range mgr.Workspace.Repos {
			names = append(names, repo.Name)
		}
		return names
	}
	return []string{filepath.Base(mgr.RepoDir)}
}

func buildInteractiveCreatePreview(baseMgr *worktree.Manager, repoName, branch string) (interactive.CreatePreview, error) {
	repoMgr, err := interactiveRepoManager(baseMgr, repoName)
	if err != nil {
		return interactive.CreatePreview{}, err
	}

	info, _, err := repoMgr.Create(branch, worktree.CreateOpts{DryRun: true})
	if err != nil {
		return interactive.CreatePreview{}, err
	}

	preview := interactive.CreatePreview{
		Repo:         repoName,
		Branch:       branch,
		Path:         info.Path,
		BranchExists: repoMgr.Git.BranchExists(branch),
		CopyFiles:    append([]string(nil), repoMgr.Config.CopyFiles...),
		SymlinkFiles: append([]string(nil), repoMgr.Config.SymlinkFiles...),
		Hook:         repoMgr.Config.PostCreateHook,
	}
	if !preview.BranchExists {
		preview.Base = info.Base
	}
	return preview, nil
}

func executeInteractiveCreate(baseMgr *worktree.Manager, prompt io.Writer, repoName, branch string) error {
	repoMgr, err := interactiveRepoManager(baseMgr, repoName)
	if err != nil {
		return err
	}

	info, _, err := repoMgr.Create(branch, worktree.CreateOpts{
		Output:   prompt,
		TextMode: true,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(prompt, "Created worktree at %s (branch: %s)\n", info.Path, info.Branch)
	return err
}

func interactiveRepoManager(baseMgr *worktree.Manager, repoName string) (*worktree.Manager, error) {
	if repoName == "" {
		return baseMgr, nil
	}
	return managerForRepo(baseMgr, repoName)
}

func interactiveCleanTargets(mgr *worktree.Manager) ([]interactive.CleanTarget, error) {
	infos, err := listCleanableWorktrees(mgr)
	if err != nil {
		return nil, err
	}

	targets := make([]interactive.CleanTarget, 0, len(infos))
	for _, info := range infos {
		targets = append(targets, interactive.CleanTarget{
			Repo:   info.Repo,
			Branch: info.Branch,
			Status: displayStatus(info),
			Path:   info.Path,
		})
	}
	return targets, nil
}
