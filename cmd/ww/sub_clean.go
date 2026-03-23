package main

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/worktree"
)

type cleanResult struct {
	Repo          string `json:"repo"`
	Path          string `json:"path"`
	Branch        string `json:"branch"`
	Status        string `json:"status"`
	Removed       bool   `json:"removed"`
	BranchDeleted bool   `json:"branch_deleted"`
	BranchError   string `json:"branch_error,omitempty"`
	Error         string `json:"error,omitempty"`
}

func cleanCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" clean", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output NDJSON")
	dryRun := fset.Bool("dry-run", false, "Show planned actions without executing")
	force := fset.Bool("force", false, "Force removal even if the worktree is dirty")

	return command{
		name:        "clean",
		description: "Remove all merged or stale worktrees",
		fset:        fset,
		fn: func(args []string, glOpts *globalOpts) error {
			if err := parseFlags(fset, args); err != nil {
				return err
			}
			glOpts.json = glOpts.json || *jsonFlag
			glOpts.dryRun = glOpts.dryRun || *dryRun

			mgr, err := newManager(false)
			if err != nil {
				return err
			}

			infos, err := mgr.List()
			if err != nil {
				return err
			}
			infos = filterCleanableWorktrees(infos)
			if len(infos) == 0 {
				return nil
			}

			var failures []string
			for _, info := range infos {
				repoMgr, err := managerForRepo(mgr, info.Repo)
				if err != nil {
					failures = append(failures, fmt.Sprintf("%s (%s): %v", info.Branch, info.Path, err))
					if glOpts.json {
						if outErr := outputJSON(glOpts.output, cleanResult{
							Repo:   info.Repo,
							Path:   info.Path,
							Branch: info.Branch,
							Status: info.Status,
							Error:  err.Error(),
						}); outErr != nil {
							return outErr
						}
					} else {
						fmt.Fprintf(glOpts.output, "Failed to clean %s at %s: %v\n", info.Branch, info.Path, err)
					}
					continue
				}

				result, dryLog, err := repoMgr.Remove(info.Branch, worktree.RemoveOpts{
					Force:  *force,
					DryRun: glOpts.dryRun,
				})
				if err != nil {
					failures = append(failures, fmt.Sprintf("%s (%s): %v", info.Branch, info.Path, err))
					if glOpts.json {
						if outErr := outputJSON(glOpts.output, cleanResult{
							Repo:   info.Repo,
							Path:   info.Path,
							Branch: info.Branch,
							Status: info.Status,
							Error:  err.Error(),
						}); outErr != nil {
							return outErr
						}
					} else {
						fmt.Fprintf(glOpts.output, "Failed to clean %s at %s: %v\n", info.Branch, info.Path, err)
					}
					continue
				}

				if glOpts.json {
					out := cleanResult{
						Repo:          info.Repo,
						Path:          result.Path,
						Branch:        result.Branch,
						Status:        info.Status,
						Removed:       result.Removed,
						BranchDeleted: result.BranchDeleted,
						BranchError:   result.BranchError,
					}
					if glOpts.dryRun {
						out.Removed = false
						out.BranchDeleted = false
						out.BranchError = ""
					}
					if err := outputJSON(glOpts.output, out); err != nil {
						return err
					}
					continue
				}

				if glOpts.dryRun {
					for _, line := range dryLog {
						fmt.Fprintln(glOpts.output, line)
					}
					continue
				}

				fmt.Fprintf(glOpts.output, "Removed worktree at %s\n", result.Path)
				if result.BranchDeleted {
					fmt.Fprintf(glOpts.output, "Deleted branch %s\n", result.Branch)
				} else if result.BranchError != "" {
					fmt.Fprintf(glOpts.errOutput, "warning: could not delete branch %s: %s\n", result.Branch, result.BranchError)
				}
			}

			if len(failures) > 0 {
				return fmt.Errorf("one or more worktrees failed to clean: %s", strings.Join(failures, "; "))
			}
			return nil
		},
	}
}
