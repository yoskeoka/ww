package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/internal/interactive"
	"github.com/yoskeoka/ww/workspace"
)

const interactiveJSONMessage = "interactive mode does not support --json; use standard ww commands for machine-readable output"

func interactiveCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" i", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Unsupported in interactive mode; use standard ww commands for machine-readable output")

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
			flows := interactive.PlaceholderFlows{Output: prompt}

			return interactive.Runner{
				Prompt:  prompt,
				Session: session,
				Flows:   flows,
			}.Run(interactiveOverview(mgr.Workspace))
		},
	}
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
