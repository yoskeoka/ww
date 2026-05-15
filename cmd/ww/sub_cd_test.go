package main

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/yoskeoka/ww/worktree"
)

func TestFindNamedWorktreeWithRetryReturnsWithoutRetryOnSuccess(t *testing.T) {
	calls := 0
	restoreSleep := swapCDSleep(func(time.Duration) {
		t.Fatal("sleep should not be called on immediate success")
	})
	defer restoreSleep()

	info, err := findNamedWorktreeWithRetry(func() (*worktree.WorktreeInfo, error) {
		calls++
		return &worktree.WorktreeInfo{Path: "/tmp/repo@feat-alpha"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if info.Path != "/tmp/repo@feat-alpha" {
		t.Fatalf("path = %q, want %q", info.Path, "/tmp/repo@feat-alpha")
	}
}

func TestFindNamedWorktreeWithRetrySucceedsWithinRetryBudget(t *testing.T) {
	var slept []time.Duration
	restoreSleep := swapCDSleep(func(d time.Duration) {
		slept = append(slept, d)
	})
	defer restoreSleep()

	calls := 0
	info, err := findNamedWorktreeWithRetry(func() (*worktree.WorktreeInfo, error) {
		calls++
		if calls <= 3 {
			return nil, errors.New(`no worktree found for branch "feat/alpha"`)
		}
		return &worktree.WorktreeInfo{Path: "/tmp/repo@feat-alpha"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Fatalf("calls = %d, want 4", calls)
	}
	if len(slept) != 3 {
		t.Fatalf("sleep calls = %d, want 3", len(slept))
	}
	for i, d := range slept {
		if d != cdNamedLookupRetryInterval {
			t.Fatalf("sleep[%d] = %s, want %s", i, d, cdNamedLookupRetryInterval)
		}
	}
	if info.Path != "/tmp/repo@feat-alpha" {
		t.Fatalf("path = %q, want %q", info.Path, "/tmp/repo@feat-alpha")
	}
}

func TestFindNamedWorktreeWithRetryStopsAfterBudgetAndPreservesError(t *testing.T) {
	var slept []time.Duration
	restoreSleep := swapCDSleep(func(d time.Duration) {
		slept = append(slept, d)
	})
	defer restoreSleep()

	wantErrText := `no worktree found for branch "feat/missing"`
	calls := 0
	info, err := findNamedWorktreeWithRetry(func() (*worktree.WorktreeInfo, error) {
		calls++
		return nil, errors.New(wantErrText)
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != wantErrText {
		t.Fatalf("error = %q, want %q", err.Error(), wantErrText)
	}
	if info != nil {
		t.Fatalf("info = %#v, want nil", info)
	}
	if calls != cdNamedLookupRetryCount+1 {
		t.Fatalf("calls = %d, want %d", calls, cdNamedLookupRetryCount+1)
	}
	if len(slept) != cdNamedLookupRetryCount {
		t.Fatalf("sleep calls = %d, want %d", len(slept), cdNamedLookupRetryCount)
	}
}

func TestFindNamedWorktreeWithRetryDoesNotRetryOtherErrors(t *testing.T) {
	restoreSleep := swapCDSleep(func(time.Duration) {
		t.Fatal("sleep should not be called for non-lookup errors")
	})
	defer restoreSleep()

	wantErr := errors.New("listing worktrees: git failed")
	calls := 0
	_, err := findNamedWorktreeWithRetry(func() (*worktree.WorktreeInfo, error) {
		calls++
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestCDNamedLookupRetryConfigFromEnvDefaults(t *testing.T) {
	restoreCount := swapEnv(cdTestRetryCountEnv, "")
	defer restoreCount()
	restoreInterval := swapEnv(cdTestRetryIntervalEnv, "")
	defer restoreInterval()

	cfg := cdNamedLookupRetryConfigFromEnv()
	if cfg.retryCount != cdNamedLookupRetryCount {
		t.Fatalf("retryCount = %d, want %d", cfg.retryCount, cdNamedLookupRetryCount)
	}
	if cfg.retryInterval != cdNamedLookupRetryInterval {
		t.Fatalf("retryInterval = %s, want %s", cfg.retryInterval, cdNamedLookupRetryInterval)
	}
}

func TestCDNamedLookupRetryConfigFromEnvOverrides(t *testing.T) {
	restoreCount := swapEnv(cdTestRetryCountEnv, "9")
	defer restoreCount()
	restoreInterval := swapEnv(cdTestRetryIntervalEnv, "250")
	defer restoreInterval()

	cfg := cdNamedLookupRetryConfigFromEnv()
	if cfg.retryCount != 9 {
		t.Fatalf("retryCount = %d, want 9", cfg.retryCount)
	}
	if cfg.retryInterval != 250*time.Millisecond {
		t.Fatalf("retryInterval = %s, want %s", cfg.retryInterval, 250*time.Millisecond)
	}
}

func TestCDNamedLookupRetryConfigFromEnvIgnoresInvalidValues(t *testing.T) {
	restoreCount := swapEnv(cdTestRetryCountEnv, "0")
	defer restoreCount()
	restoreInterval := swapEnv(cdTestRetryIntervalEnv, "bad")
	defer restoreInterval()

	cfg := cdNamedLookupRetryConfigFromEnv()
	if cfg.retryCount != cdNamedLookupRetryCount {
		t.Fatalf("retryCount = %d, want %d", cfg.retryCount, cdNamedLookupRetryCount)
	}
	if cfg.retryInterval != cdNamedLookupRetryInterval {
		t.Fatalf("retryInterval = %s, want %s", cfg.retryInterval, cdNamedLookupRetryInterval)
	}
}

func swapCDSleep(fn func(time.Duration)) func() {
	orig := cdSleep
	cdSleep = fn
	return func() {
		cdSleep = orig
	}
}

func swapEnv(name, value string) func() {
	orig, ok := os.LookupEnv(name)
	if value == "" {
		_ = os.Unsetenv(name)
	} else {
		_ = os.Setenv(name, value)
	}
	return func() {
		if ok {
			_ = os.Setenv(name, orig)
			return
		}
		_ = os.Unsetenv(name)
	}
}
