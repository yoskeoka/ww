package main

import (
	"fmt"

	"github.com/spf13/pflag"
)

func createCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" create", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output JSON")
	dryRun := fset.Bool("dry-run", false, "Show planned actions without executing")
	quiet := fset.BoolP("quiet", "q", false, "Print only the worktree path")
	repo := fset.String("repo", "", "Target a detected workspace repository by name")

	return command{
		name:        "create",
		description: "Create a new worktree for a branch",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag
			glOpts.dryRun = glOpts.dryRun || *dryRun

			remaining := fset.Args()
			if len(remaining) == 0 {
				return fmt.Errorf("usage: ww create <branch>")
			}
			branch := remaining[0]

			mgr, err := managerForSelectedRepo(*repo, true)
			if err != nil {
				return err
			}

			info, dryLog, err := mgr.Create(branch, worktreeCreateOpts(glOpts, *quiet))
			if err != nil {
				return err
			}

			if glOpts.dryRun {
				if glOpts.json {
					return outputJSON(glOpts.output, info)
				}
				if *quiet {
					fmt.Fprintln(glOpts.output, info.Path)
					return nil
				}
				for _, line := range dryLog {
					fmt.Fprintln(glOpts.output, line)
				}
				return nil
			}

			if glOpts.json {
				return outputJSON(glOpts.output, info)
			}
			if *quiet {
				fmt.Fprintln(glOpts.output, info.Path)
				return nil
			}
			fmt.Fprintf(glOpts.output, "Created worktree at %s (branch: %s)\n", info.Path, info.Branch)
			return nil
		},
	}
}
