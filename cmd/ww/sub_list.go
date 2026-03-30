package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/workspace"
	"github.com/yoskeoka/ww/worktree"
)

func listCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" list", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output NDJSON")
	cleanable := fset.Bool("cleanable", false, "Show only merged or stale worktrees")

	return command{
		name:        "list",
		description: "List all worktrees",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag

			mgr, err := newManager(false)
			if err != nil {
				return err
			}

			infos, err := mgr.List()
			if err != nil {
				return err
			}
			if *cleanable {
				infos = filterCleanableWorktrees(infos)
			}

			if glOpts.json {
				for _, info := range infos {
					if err := outputJSON(glOpts.output, info); err != nil {
						return err
					}
				}
				return nil
			}

			tw := tabwriter.NewWriter(glOpts.output, 0, 0, 2, ' ', 0)
			if mgr.Workspace != nil && mgr.Workspace.Mode == workspace.ModeWorkspace {
				fmt.Fprintln(tw, "REPO\tPATH\tBRANCH\tHEAD\tSTATUS")
				for _, info := range infos {
					path := info.Path
					if info.Main {
						path += " (main worktree)"
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", info.Repo, path, info.Branch, info.Head, displayStatus(info))
				}
			} else {
				fmt.Fprintln(tw, "PATH\tBRANCH\tHEAD\tSTATUS")
				for _, info := range infos {
					path := info.Path
					if info.Main {
						path += " (main worktree)"
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", path, info.Branch, info.Head, displayStatus(info))
				}
			}
			return tw.Flush()
		},
	}
}

func displayStatus(info worktree.WorktreeInfo) string {
	if info.StatusDetail != "" {
		return fmt.Sprintf("%s(%s)", info.Status, info.StatusDetail)
	}
	return info.Status
}

func filterCleanableWorktrees(infos []worktree.WorktreeInfo) []worktree.WorktreeInfo {
	out := make([]worktree.WorktreeInfo, 0, len(infos))
	for _, info := range infos {
		if info.Status == worktree.StatusMerged || info.Status == worktree.StatusStale {
			out = append(out, info)
		}
	}
	return out
}
