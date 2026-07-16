package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckBudgetWithinLimit(t *testing.T) {
	path := writeBudget(t, 100)
	err := checkBudget(context.Background(), path, func(context.Context, []string) (string, error) {
		return benchmarkOutput(90, 80), nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckBudgetReportsExceededBenchmark(t *testing.T) {
	path := writeBudget(t, 100)
	err := checkBudget(context.Background(), path, func(context.Context, []string) (string, error) {
		return benchmarkOutput(101, 80), nil
	})
	if err == nil || !strings.Contains(err.Error(), "BenchmarkWorkspaceList observed 101 ns/op") {
		t.Fatalf("error = %v, want named budget exceedance", err)
	}
}

func TestCheckBudgetRejectsMissingBenchmarkOutput(t *testing.T) {
	path := writeBudget(t, 100)
	err := checkBudget(context.Background(), path, func(context.Context, []string) (string, error) {
		return "BenchmarkWorkspaceList\t1\t90 ns/op\n", nil
	})
	if err == nil || !strings.Contains(err.Error(), "missing BenchmarkWorkspaceCleanDryRun result") {
		t.Fatalf("error = %v, want missing-output failure", err)
	}
}

func TestCheckBudgetClassifiesFixtureFailure(t *testing.T) {
	path := writeBudget(t, 100)
	err := checkBudget(context.Background(), path, func(context.Context, []string) (string, error) {
		return "--- FAIL: BenchmarkWorkspaceList\nperformance fixture: git init failed\n", errors.New("exit status 1")
	})
	if err == nil || !strings.Contains(err.Error(), "fixture failed") {
		t.Fatalf("error = %v, want fixture failure", err)
	}
}

func TestCommandStringQuotesShellMetacharacters(t *testing.T) {
	got := commandString([]string{"go", "test", "^BenchmarkWorkspace(List|CleanDryRun)$", "contains space", "plain"})
	want := "go test '^BenchmarkWorkspace(List|CleanDryRun)$' 'contains space' plain"
	if got != want {
		t.Fatalf("commandString() = %q, want %q", got, want)
	}
}

func writeBudget(t *testing.T, limit int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "budget.json")
	content := fmt.Sprintf(`{
  "fixture_profile": {"repos": 2, "worktrees_per_repo": 2},
  "benchmark_command": ["go", "test"],
  "repetitions": 1,
  "benchmarks": {
    "BenchmarkWorkspaceList": {"max_ns_per_op": %d},
    "BenchmarkWorkspaceCleanDryRun": {"max_ns_per_op": %d}
  }

}`, limit, limit)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func benchmarkOutput(list, clean int) string {
	return fmt.Sprintf("BenchmarkWorkspaceList-8\t1\t%d ns/op\nBenchmarkWorkspaceCleanDryRun-8\t1\t%d ns/op\n", list, clean)
}
