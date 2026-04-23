package main

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/worktree"
)

func cdCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" cd", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output JSON")
	repo := fset.String("repo", "", "Target a detected workspace repository by name")

	return command{
		name:        "cd",
		description: "Print a worktree path for shell navigation",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag

			remaining := fset.Args()
			if len(remaining) > 1 {
				return fmt.Errorf("usage: ww cd [branch]")
			}

			mgr, err := managerForSelectedRepo(*repo, true, glOpts.sandbox)
			if err != nil {
				return err
			}

			var info *worktree.WorktreeInfo
			if len(remaining) == 0 {
				info, err = mgr.MostRecent(glOpts.json)
			} else {
				info, err = mgr.FindByName(remaining[0], glOpts.json)
			}
			if err != nil {
				return err
			}

			if glOpts.json {
				return outputJSON(glOpts.output, info)
			}
			fmt.Fprintln(glOpts.output, info.Path)
			return nil
		},
	}
}
