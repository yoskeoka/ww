package main

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/worktree"
)

func hookCmd() command {
	return command{
		name:        "hook",
		description: "Run explicit lifecycle recovery actions",
		subcommands: []command{hookReplayCmd()},
		fn: func(args []string, _ *globalOpts) error {
			if len(args) == 1 && args[0] == "--help" {
				fmt.Println("Run an explicit lifecycle recovery action")
				fmt.Println()
				fmt.Println("Usage:")
				fmt.Println("  ww hook replay [flags] [branch]")
				return errHelp
			}
			return fmt.Errorf("usage: ww hook replay [flags] [branch]")
		},
	}
}

func hookReplayCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" hook replay", pflag.ContinueOnError)
	repo := fset.String("repo", "", "Target a detected workspace repository by name")
	dryRun := fset.Bool("dry-run", false, "Show replay actions without executing")
	jsonFlag := fset.Bool("json", false, "Output JSON")
	fset.Usage = func() {
		out := fset.Output()
		fmt.Fprintln(out, "Replay post-create materialization for an existing worktree")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintln(out, "  ww hook replay [flags] [branch]")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Run from inside a worktree, or provide an existing branch.")
		fmt.Fprintln(out, "The command only replays copy, symlink, and post-create hook actions.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Flags:")
		fset.PrintDefaults()
	}

	return command{
		name:        "replay",
		description: "Replay post-create materialization",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag
			glOpts.dryRun = glOpts.dryRun || *dryRun
			remaining := fset.Args()
			if len(remaining) > 1 {
				return fmt.Errorf("usage: ww hook replay [flags] [branch]")
			}

			mgr, err := managerForSelectedRepo(*repo, false, glOpts.sandbox)
			if err != nil {
				return err
			}
			var targetPath string
			var branch string
			if len(remaining) == 1 {
				branch = remaining[0]
				info, findErr := mgr.FindByName(branch, false)
				if findErr != nil {
					return findErr
				}
				targetPath = info.Path
			} else if *repo != "" {
				return fmt.Errorf("usage: ww hook replay --repo <name> <branch>")
			} else {
				targetPath, err = mgr.Git.WorktreeRoot()
				if err != nil {
					return fmt.Errorf("replay without a branch must run inside a git worktree: %w", err)
				}
			}

			result, err := mgr.Replay(targetPath, branch, worktree.ReplayOpts{
				DryRun:   glOpts.dryRun,
				Output:   glOpts.output,
				TextMode: !glOpts.json,
			})
			if err != nil {
				return err
			}
			if glOpts.json {
				return outputJSON(glOpts.output, result)
			}
			if glOpts.dryRun {
				for _, action := range result.Actions {
					fmt.Fprintf(glOpts.output, "Would %s\n", action)
				}
				return nil
			}
			_, err = fmt.Fprintf(glOpts.output, "Replayed post-create materialization at %s (branch: %s)\n", result.Path, result.Branch)
			return err
		},
	}
}
