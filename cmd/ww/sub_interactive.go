package main

import (
	"fmt"
	"io"
	"os"

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
			session := interactive.NewLineSession(os.Stdin, prompt)
			flows := interactiveFlows{
				prompt: prompt,
				list: interactive.ListFlow{
					Prompt:  prompt,
					Session: session,
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
	list   interactive.ListFlow
}

func (f interactiveFlows) Create() error {
	_, err := fmt.Fprintln(f.prompt, "Interactive create flow is not implemented yet. Use `ww create` for now.")
	return err
}

func (f interactiveFlows) List() error {
	return f.list.Run()
}

func (f interactiveFlows) Clean() error {
	_, err := fmt.Fprintln(f.prompt, "Interactive clean flow is not implemented yet. Use `ww clean` for now.")
	return err
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
	if len(path) <= maxLen {
		return path
	}
	return path[:22] + "..." + path[len(path)-23:]
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
