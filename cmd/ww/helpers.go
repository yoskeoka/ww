package main

import (
	"errors"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/worktree"
)

func worktreeCreateOpts(glOpts *globalOpts) worktree.CreateOpts {
	return worktree.CreateOpts{
		DryRun: glOpts.dryRun,
	}
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
