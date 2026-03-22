package main

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/worktree"
)

func removeCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" remove", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output JSON")
	dryRun := fset.Bool("dry-run", false, "Show planned actions without executing")
	force := fset.Bool("force", false, "Force removal even if the worktree is dirty")
	keepBranch := fset.Bool("keep-branch", false, "Do not delete the branch")

	return command{
		name:        "remove",
		description: "Remove a worktree and its branch",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag
			glOpts.dryRun = glOpts.dryRun || *dryRun

			remaining := fset.Args()
			if len(remaining) == 0 {
				return fmt.Errorf("usage: ww remove <branch>")
			}
			branch := remaining[0]

			mgr, err := newManager(true)
			if err != nil {
				return err
			}

			result, dryLog, err := mgr.Remove(branch, worktree.RemoveOpts{
				Force:      *force,
				KeepBranch: *keepBranch,
				DryRun:     glOpts.dryRun,
			})
			if err != nil {
				return err
			}

			if glOpts.dryRun {
				if glOpts.json {
					return outputJSON(glOpts.output, result)
				}
				for _, line := range dryLog {
					fmt.Fprintln(glOpts.output, line)
				}
				return nil
			}

			if glOpts.json {
				return outputJSON(glOpts.output, result)
			}
			fmt.Fprintf(glOpts.output, "Removed worktree at %s\n", result.Path)
			if result.BranchDeleted {
				fmt.Fprintf(glOpts.output, "Deleted branch %s\n", result.Branch)
			}
			return nil
		},
	}
}
