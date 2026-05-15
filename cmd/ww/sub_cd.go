package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/worktree"
)

const (
	cdNamedLookupRetryCount    = 5
	cdNamedLookupRetryInterval = 100 * time.Millisecond

	cdTestRetryCountEnv    = "WW_TEST_CD_NAMED_LOOKUP_RETRY_COUNT"
	cdTestRetryIntervalEnv = "WW_TEST_CD_NAMED_LOOKUP_RETRY_INTERVAL_MS"
	cdTestMissMarkerEnv    = "WW_TEST_CD_NAMED_LOOKUP_MISS_MARKER"
	cdTestRetryReleaseEnv  = "WW_TEST_CD_NAMED_LOOKUP_RETRY_RELEASE"
	cdTestWaitPollInterval = 10 * time.Millisecond
)

var cdSleep = time.Sleep

func cdCmd() command {
	fset := pflag.NewFlagSet(mainCmdName+" cd", pflag.ContinueOnError)
	jsonFlag := fset.Bool("json", false, "Output JSON")
	repo := fset.String("repo", "", "Target a detected workspace repository by name")
	fset.Usage = func() {
		out := fset.Output()
		fmt.Fprintln(out, "Print a worktree path for shell navigation")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Usage:")
		fmt.Fprintln(out, "  ww cd [flags] [branch]")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Examples:")
		fmt.Fprintln(out, "  ww cd")
		fmt.Fprintln(out, "  ww cd feat/my-feature")
		fmt.Fprintln(out, "  ww cd --repo backend feat/my-feature")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Notes:")
		fmt.Fprintln(out, "  - ww cd prints the path of an existing worktree")
		fmt.Fprintln(out, "  - to create and enter in one step, use cd \"$(ww create -q feat/my-feature)\"")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Flags:")
		fset.PrintDefaults()
	}

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
				return fmt.Errorf("usage: ww cd [flags] [branch]")
			}

			mgr, err := managerForSelectedRepo(*repo, true, glOpts.sandbox)
			if err != nil {
				return err
			}

			var info *worktree.WorktreeInfo
			if len(remaining) == 0 {
				info, err = mgr.MostRecent(glOpts.json)
			} else {
				info, err = findNamedWorktreeForOutput(mgr, remaining[0], glOpts.json)
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

func findNamedWorktreeForOutput(mgr *worktree.Manager, branch string, withStatus bool) (*worktree.WorktreeInfo, error) {
	info, err := findNamedWorktreeWithRetry(func() (*worktree.WorktreeInfo, error) {
		return mgr.FindByName(branch, false)
	})
	if err != nil || !withStatus {
		return info, err
	}
	return mgr.FindByName(branch, true)
}

func findNamedWorktreeWithRetry(find func() (*worktree.WorktreeInfo, error)) (*worktree.WorktreeInfo, error) {
	cfg := cdNamedLookupRetryConfigFromEnv()
	info, err := find()
	if err == nil || !isNamedWorktreeMiss(err) {
		return info, err
	}
	maybeSignalCDRetryTestHook()

	for i := 0; i < cfg.retryCount; i++ {
		cdSleep(cfg.retryInterval)
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

type cdNamedLookupRetryConfig struct {
	retryCount    int
	retryInterval time.Duration
}

func cdNamedLookupRetryConfigFromEnv() cdNamedLookupRetryConfig {
	cfg := cdNamedLookupRetryConfig{
		retryCount:    cdNamedLookupRetryCount,
		retryInterval: cdNamedLookupRetryInterval,
	}

	if count, ok := parsePositiveIntEnv(cdTestRetryCountEnv); ok {
		cfg.retryCount = count
	}
	if intervalMS, ok := parsePositiveIntEnv(cdTestRetryIntervalEnv); ok {
		cfg.retryInterval = time.Duration(intervalMS) * time.Millisecond
	}
	return cfg
}

func parsePositiveIntEnv(name string) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

func maybeSignalCDRetryTestHook() {
	markerPath := strings.TrimSpace(os.Getenv(cdTestMissMarkerEnv))
	releasePath := strings.TrimSpace(os.Getenv(cdTestRetryReleaseEnv))
	if markerPath == "" && releasePath == "" {
		return
	}

	if markerPath != "" {
		_ = os.WriteFile(markerPath, []byte("miss\n"), 0644)
	}
	if releasePath == "" {
		return
	}

	for {
		if _, err := os.Stat(releasePath); err == nil {
			return
		}
		cdSleep(cdTestWaitPollInterval)
	}
}
