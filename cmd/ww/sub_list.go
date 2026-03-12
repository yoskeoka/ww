package main

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/pflag"
)

func listCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" list", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output NDJSON")

	return command{
		name:        "list",
		description: "List all worktrees",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag

			mgr, err := newManager()
			if err != nil {
				return err
			}

			infos, err := mgr.List()
			if err != nil {
				return err
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
			fmt.Fprintln(tw, "PATH\tBRANCH\tHEAD")
			for _, info := range infos {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", info.Path, info.Branch, info.Head)
			}
			return tw.Flush()
		},
	}
}
