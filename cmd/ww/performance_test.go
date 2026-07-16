package main

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/yoskeoka/ww/internal/testutil"
)

var performanceFixtureState struct {
	once    sync.Once
	env     *testutil.HostEnv
	fixture *testutil.PerformanceFixture
	err     error
}

func TestMain(m *testing.M) {
	code := m.Run()
	if performanceFixtureState.fixture != nil {
		_ = performanceFixtureState.fixture.Cleanup()
	}
	if performanceFixtureState.env != nil {
		performanceFixtureState.env.Terminate()
	}
	os.Exit(code)
}

func benchmarkPerformanceFixture(b *testing.B) (*testutil.HostEnv, *testutil.PerformanceFixture) {
	b.Helper()
	performanceFixtureState.once.Do(func() {
		performanceFixtureState.env, performanceFixtureState.err = testutil.NewHostEnv(context.Background())
		if performanceFixtureState.err != nil {
			return
		}
		opts, err := testutil.PerformanceFixtureOptionsFromEnv()
		if err != nil {
			performanceFixtureState.err = err
			return
		}
		performanceFixtureState.fixture, performanceFixtureState.err = testutil.NewPerformanceFixture(performanceFixtureState.env, opts)
	})
	if performanceFixtureState.err != nil {
		b.Fatalf("performance fixture: %v", performanceFixtureState.err)
	}
	return performanceFixtureState.env, performanceFixtureState.fixture
}

func BenchmarkWorkspaceList(b *testing.B) {
	env, fixture := benchmarkPerformanceFixture(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := env.RunWW(fixture.StartDir, "list")
		if err != nil {
			b.Fatalf("ww list: %v\n%s", err, out)
		}
		if got := strings.Count(out, "\n") - 1; got != fixture.ExpectedEntries {
			b.Fatalf("ww list entries = %d, want %d\n%s", got, fixture.ExpectedEntries, out)
		}
	}
}

func BenchmarkWorkspaceCleanDryRun(b *testing.B) {
	env, fixture := benchmarkPerformanceFixture(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := env.RunWW(fixture.StartDir, "clean", "--dry-run")
		if err != nil {
			b.Fatalf("ww clean --dry-run: %v\n%s", err, out)
		}
		if got := strings.Count(out, "Would remove worktree at "); got != fixture.ExpectedCleanable {
			b.Fatalf("ww clean --dry-run removals = %d, want %d\n%s", got, fixture.ExpectedCleanable, out)
		}
	}
}
