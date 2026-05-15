package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/worktree"
)

const (
	cdNamedLookupRetryCount    = 5
	cdNamedLookupRetryInterval = 100 * time.Millisecond
)

var cdSleep = time.Sleep

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
				info, err = findNamedWorktreeWithRetry(func() (*worktree.WorktreeInfo, error) {
					return mgr.FindByName(remaining[0], glOpts.json)
				})
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

func findNamedWorktreeWithRetry(find func() (*worktree.WorktreeInfo, error)) (*worktree.WorktreeInfo, error) {
	info, err := find()
	if err == nil || !isNamedWorktreeMiss(err) {
		return info, err
	}

	for i := 0; i < cdNamedLookupRetryCount; i++ {
		cdSleep(cdNamedLookupRetryInterval)
		info, err = find()
		if err == nil || !isNamedWorktreeMiss(err) {
			return info, err
		}
	}
	return nil, err
}

func isNamedWorktreeMiss(err error) bool {
	return strings.Contains(err.Error(), "no worktree found for branch ")
}
