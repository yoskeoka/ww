package testutil

import "testing"

func TestPerformanceFixtureOptionsFromEnvCleanableDensity(t *testing.T) {
	t.Setenv("WW_PERF_WORKTREES_PER_REPO", "4")
	t.Setenv("WW_PERF_CLEANABLE_WORKTREES_PER_REPO", "3")
	opts, err := PerformanceFixtureOptionsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if opts.CleanableWorktreesPerRepo != 3 {
		t.Fatalf("cleanable density = %d, want 3", opts.CleanableWorktreesPerRepo)
	}
}

func TestPerformanceFixtureOptionsFromEnvRejectsAllCleanable(t *testing.T) {
	t.Setenv("WW_PERF_WORKTREES_PER_REPO", "3")
	t.Setenv("WW_PERF_CLEANABLE_WORKTREES_PER_REPO", "3")
	if _, err := PerformanceFixtureOptionsFromEnv(); err == nil {
		t.Fatal("PerformanceFixtureOptionsFromEnv() error = nil, want validation error")
	}
}

func TestPerformanceFixtureOptionsFromEnvRemoteDelay(t *testing.T) {
	t.Setenv("WW_PERF_REMOTE_DELAY_MS", "25")
	opts, err := PerformanceFixtureOptionsFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if opts.RemoteDelayMS != 25 {
		t.Fatalf("remote delay = %d, want 25", opts.RemoteDelayMS)
	}
}

func TestPerformanceFixtureOptionsFromEnvRejectsNegativeRemoteDelay(t *testing.T) {
	t.Setenv("WW_PERF_REMOTE_DELAY_MS", "-1")
	if _, err := PerformanceFixtureOptionsFromEnv(); err == nil {
		t.Fatal("PerformanceFixtureOptionsFromEnv() error = nil, want validation error")
	}
}
