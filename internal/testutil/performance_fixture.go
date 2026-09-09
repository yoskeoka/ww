package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// PerformanceFixtureOptions controls the amount of Git state exercised by the
// command-boundary performance benchmarks. Increasing either value makes a
// regression visible without changing the benchmark implementation.
type PerformanceFixtureOptions struct {
	Repos                     int
	WorktreesPerRepo          int
	CleanableWorktreesPerRepo int
	PatchHeavy                bool
	RemoteDelayMS             int
}

// DefaultPerformanceFixtureOptions is deliberately large enough to exercise
// workspace discovery, per-repository status calculation, and remote checks.
var DefaultPerformanceFixtureOptions = PerformanceFixtureOptions{
	Repos:                     6,
	WorktreesPerRepo:          5,
	CleanableWorktreesPerRepo: 1,
}

// PerformanceFixtureOptionsFromEnv applies optional benchmark-only scale
// overrides. This keeps the default profile reviewable while making it easy to
// reproduce a larger workspace locally or in an investigation.
func PerformanceFixtureOptionsFromEnv() (PerformanceFixtureOptions, error) {
	opts := DefaultPerformanceFixtureOptions
	for _, setting := range []struct {
		name string
		set  func(int)
	}{
		{"WW_PERF_REPOS", func(value int) { opts.Repos = value }},
		{"WW_PERF_WORKTREES_PER_REPO", func(value int) { opts.WorktreesPerRepo = value }},
		{"WW_PERF_CLEANABLE_WORKTREES_PER_REPO", func(value int) { opts.CleanableWorktreesPerRepo = value }},
	} {
		value := os.Getenv(setting.name)
		if value == "" {
			continue
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return PerformanceFixtureOptions{}, fmt.Errorf("%s must be a positive integer, got %q", setting.name, value)
		}
		setting.set(n)
	}
	if value := os.Getenv("WW_PERF_PATCH_HEAVY"); value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return PerformanceFixtureOptions{}, fmt.Errorf("WW_PERF_PATCH_HEAVY must be a boolean, got %q", value)
		}
		opts.PatchHeavy = enabled
	}
	if value := os.Getenv("WW_PERF_REMOTE_DELAY_MS"); value != "" {
		delay, err := strconv.Atoi(value)
		if err != nil || delay < 0 {
			return PerformanceFixtureOptions{}, fmt.Errorf("WW_PERF_REMOTE_DELAY_MS must be a non-negative integer, got %q", value)
		}
		opts.RemoteDelayMS = delay
	}
	if opts.WorktreesPerRepo < 2 {
		return PerformanceFixtureOptions{}, fmt.Errorf("WW_PERF_WORKTREES_PER_REPO must be at least 2 to include merged and upstream-tracking branches")
	}
	if opts.CleanableWorktreesPerRepo < 1 || opts.CleanableWorktreesPerRepo >= opts.WorktreesPerRepo {
		return PerformanceFixtureOptions{}, fmt.Errorf("WW_PERF_CLEANABLE_WORKTREES_PER_REPO must be between 1 and %d to retain an upstream-tracking branch", opts.WorktreesPerRepo-1)
	}
	return opts, nil
}

// PerformanceFixture is a non-sandbox workspace with real repositories,
// linked worktrees, and local bare remotes. Setup is intentionally separate
// from benchmark timing.
type PerformanceFixture struct {
	Root              string
	StartDir          string
	ExpectedEntries   int
	ExpectedCleanable int
	cleanupRoot       string
	commandEnv        []string
	remoteProbeLog    string
}

// Cleanup removes the fixture and all linked worktrees.
func (f *PerformanceFixture) Cleanup() error {
	return os.RemoveAll(f.cleanupRoot)
}

// CommandEnv returns environment overrides for the optional delayed-remote
// profile. The default profile returns nil and uses the host environment.
func (f *PerformanceFixture) CommandEnv() []string {
	return append([]string(nil), f.commandEnv...)
}

// ResetRemoteProbeLog clears the opt-in delayed-profile probe log.
func (f *PerformanceFixture) ResetRemoteProbeLog() error {
	if f.remoteProbeLog == "" {
		return nil
	}
	return os.WriteFile(f.remoteProbeLog, nil, 0600)
}

// RemoteProbeCount reports delayed-profile ls-remote calls.
func (f *PerformanceFixture) RemoteProbeCount() (int, error) {
	if f.remoteProbeLog == "" {
		return 0, nil
	}
	data, err := os.ReadFile(f.remoteProbeLog)
	if err != nil {
		return 0, err
	}
	return strings.Count(string(data), "ls-remote\n"), nil
}

// NewPerformanceFixture creates the named benchmark profile using HostEnv's
// isolated Git configuration. Every repository has one merged worktree and at
// least one pushed, upstream-tracking active worktree. The optional PatchHeavy
// profile makes those secondary branches squash-equivalent to main.
func NewPerformanceFixture(env *HostEnv, opts PerformanceFixtureOptions) (*PerformanceFixture, error) {
	if opts.Repos < 1 || opts.WorktreesPerRepo < 2 || opts.CleanableWorktreesPerRepo < 1 || opts.CleanableWorktreesPerRepo >= opts.WorktreesPerRepo {
		return nil, fmt.Errorf("invalid performance fixture scale: repos=%d worktrees_per_repo=%d cleanable_worktrees_per_repo=%d", opts.Repos, opts.WorktreesPerRepo, opts.CleanableWorktreesPerRepo)
	}

	base, err := env.MkdirTemp("ww-performance")
	if err != nil {
		return nil, fmt.Errorf("create fixture root: %w", err)
	}
	failure := func(err error) (*PerformanceFixture, error) {
		_ = os.RemoveAll(base)
		return nil, err
	}
	workspaceRoot := filepath.Join(base, "workspace")
	if err := env.MkdirAll(workspaceRoot); err != nil {
		return failure(fmt.Errorf("create workspace: %w", err))
	}
	commandEnv, remoteProbeLog, err := configureRemoteDelay(base, opts.RemoteDelayMS)
	if err != nil {
		return failure(err)
	}

	for repoIndex := 1; repoIndex <= opts.Repos; repoIndex++ {
		repoName := fmt.Sprintf("repo%d", repoIndex)
		repoPath := filepath.Join(workspaceRoot, repoName)
		remotePath := filepath.Join(base, "remotes", repoName+".git")
		if err := env.MkdirAll(filepath.Dir(remotePath)); err != nil {
			return failure(fmt.Errorf("create remote parent for %s: %w", repoName, err))
		}
		if _, err := env.Git("", "init", "--bare", remotePath); err != nil {
			return failure(fmt.Errorf("init bare remote for %s: %w", repoName, err))
		}
		if _, err := env.Git("", "--git-dir", remotePath, "symbolic-ref", "HEAD", "refs/heads/main"); err != nil {
			return failure(fmt.Errorf("set remote HEAD for %s: %w", repoName, err))
		}
		if err := env.MkdirAll(repoPath); err != nil {
			return failure(fmt.Errorf("create %s: %w", repoName, err))
		}
		if _, err := env.Git(repoPath, "init", "-b", "main"); err != nil {
			return failure(fmt.Errorf("init %s: %w", repoName, err))
		}
		if _, err := env.Git(repoPath, "remote", "add", "origin", remotePath); err != nil {
			return failure(fmt.Errorf("add remote for %s: %w", repoName, err))
		}
		if _, err := env.Git(repoPath, "commit", "--allow-empty", "-m", "fixture initial"); err != nil {
			return failure(fmt.Errorf("initial commit for %s: %w", repoName, err))
		}
		if _, err := env.Git(repoPath, "push", "-u", "origin", "main"); err != nil {
			return failure(fmt.Errorf("push main for %s: %w", repoName, err))
		}
		if _, err := env.Git(repoPath, "remote", "set-head", "origin", "-a"); err != nil {
			return failure(fmt.Errorf("set origin HEAD for %s: %w", repoName, err))
		}

		secondaryBranches := make([]string, 0, opts.WorktreesPerRepo-1)
		for worktreeIndex := 0; worktreeIndex < opts.WorktreesPerRepo; worktreeIndex++ {
			branch := fmt.Sprintf("feat/active-%d", worktreeIndex)
			if worktreeIndex < opts.CleanableWorktreesPerRepo {
				branch = fmt.Sprintf("feat/merged-%d", worktreeIndex)
			}
			worktreePath := filepath.Join(base, "worktrees", fmt.Sprintf("%s-%d", repoName, worktreeIndex))
			if err := env.MkdirAll(filepath.Dir(worktreePath)); err != nil {
				return failure(fmt.Errorf("create worktree parent for %s: %w", repoName, err))
			}
			if _, err := env.Git(repoPath, "worktree", "add", "-b", branch, worktreePath, "main"); err != nil {
				return failure(fmt.Errorf("add worktree %s/%s: %w", repoName, branch, err))
			}
			if opts.PatchHeavy && worktreeIndex > 0 {
				for commitIndex := 1; commitIndex <= 2; commitIndex++ {
					fileName := fmt.Sprintf("%s-%d.txt", branch[5:], commitIndex)
					if err := env.WriteFile(filepath.Join(worktreePath, fileName), fmt.Sprintf("%s change %d\\n", branch, commitIndex)); err != nil {
						return failure(fmt.Errorf("write %s/%s change %d: %w", repoName, branch, commitIndex, err))
					}
					if _, err := env.Git(worktreePath, "add", fileName); err != nil {
						return failure(fmt.Errorf("add %s/%s change %d: %w", repoName, branch, commitIndex, err))
					}
					if _, err := env.Git(worktreePath, "commit", "-m", fmt.Sprintf("fixture %s change %d", branch, commitIndex)); err != nil {
						return failure(fmt.Errorf("commit %s/%s change %d: %w", repoName, branch, commitIndex, err))
					}
				}
			} else if _, err := env.Git(worktreePath, "commit", "--allow-empty", "-m", "fixture "+branch); err != nil {
				return failure(fmt.Errorf("commit %s/%s: %w", repoName, branch, err))
			}
			if worktreeIndex < opts.CleanableWorktreesPerRepo {
				if _, err := env.Git(repoPath, "merge", "--no-ff", branch, "-m", "merge fixture branch"); err != nil {
					return failure(fmt.Errorf("merge %s/%s: %w", repoName, branch, err))
				}
				if _, err := env.Git(repoPath, "push", "origin", "main"); err != nil {
					return failure(fmt.Errorf("push merged main for %s: %w", repoName, err))
				}
				continue
			}
			if _, err := env.Git(worktreePath, "push", "-u", "origin", branch); err != nil {
				return failure(fmt.Errorf("push active branch %s/%s: %w", repoName, branch, err))
			}
			secondaryBranches = append(secondaryBranches, branch)
		}
		if opts.PatchHeavy {
			for _, branch := range secondaryBranches {
				if _, err := env.Git(repoPath, "merge", "--squash", branch); err != nil {
					return failure(fmt.Errorf("squash merge %s/%s: %w", repoName, branch, err))
				}
				if _, err := env.Git(repoPath, "commit", "-m", "squash fixture "+branch); err != nil {
					return failure(fmt.Errorf("commit squash merge %s/%s: %w", repoName, branch, err))
				}
			}
			if _, err := env.Git(repoPath, "push", "origin", "main"); err != nil {
				return failure(fmt.Errorf("push patch-heavy main for %s: %w", repoName, err))
			}
		}
	}

	return &PerformanceFixture{
		Root:              workspaceRoot,
		StartDir:          filepath.Join(workspaceRoot, "repo1"),
		ExpectedEntries:   opts.Repos * (opts.WorktreesPerRepo + 1),
		ExpectedCleanable: expectedCleanable(opts),
		cleanupRoot:       base,
		commandEnv:        commandEnv,
		remoteProbeLog:    remoteProbeLog,
	}, nil
}

func configureRemoteDelay(base string, delayMS int) ([]string, string, error) {
	if delayMS == 0 {
		return nil, "", nil
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		return nil, "", fmt.Errorf("locate git for delayed remote profile: %w", err)
	}
	logPath := filepath.Join(base, "remote-probes.log")
	delaySeconds := float64(delayMS) / 1000
	script := fmt.Sprintf("#!/bin/sh\nif [ \"$1\" = \"ls-remote\" ]; then\n  printf 'ls-remote\\n' >> %s\n  sleep %.3f\nfi\nexec %s \"$@\"\n", shellQuote(logPath), delaySeconds, shellQuote(realGit))
	wrapperPath := filepath.Join(base, "git")
	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		return nil, "", fmt.Errorf("write delayed git wrapper: %w", err)
	}
	return []string{"PATH=" + base + string(os.PathListSeparator) + os.Getenv("PATH")}, logPath, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func expectedCleanable(opts PerformanceFixtureOptions) int {
	if opts.PatchHeavy {
		return opts.Repos * opts.WorktreesPerRepo
	}
	return opts.Repos * opts.CleanableWorktreesPerRepo
}
