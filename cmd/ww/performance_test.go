package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/yoskeoka/ww/internal/cache"
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
	if err := env.ResetCache(); err != nil {
		b.Fatalf("reset performance cache: %v", err)
	}
	if err := fixture.ResetRemoteProbeLog(); err != nil {
		b.Fatalf("reset remote probe log: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := runPerformanceWW(env, fixture, "list")
		if err != nil {
			b.Fatalf("ww list: %v\n%s", err, out)
		}
		if got := strings.Count(out, "\n") - 1; got != fixture.ExpectedEntries {
			b.Fatalf("ww list entries = %d, want %d\n%s", got, fixture.ExpectedEntries, out)
		}
	}
	b.StopTimer()
	reportRemoteProbeCount(b, fixture)
}

func BenchmarkWorkspaceCleanDryRun(b *testing.B) {
	env, fixture := benchmarkPerformanceFixture(b)
	if err := env.ResetCache(); err != nil {
		b.Fatalf("reset performance cache: %v", err)
	}
	if err := fixture.ResetRemoteProbeLog(); err != nil {
		b.Fatalf("reset remote probe log: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := runPerformanceWW(env, fixture, "clean", "--dry-run")
		if err != nil {
			b.Fatalf("ww clean --dry-run: %v\n%s", err, out)
		}
		if got := strings.Count(out, "Would remove worktree at "); got != fixture.ExpectedCleanable {
			b.Fatalf("ww clean --dry-run removals = %d, want %d\n%s", got, fixture.ExpectedCleanable, out)
		}
	}
	b.StopTimer()
	reportRemoteProbeCount(b, fixture)
}

func BenchmarkWorkspaceListWarm(b *testing.B) {
	env, fixture := benchmarkPerformanceFixture(b)
	primePerformanceCache(b, env, fixture.StartDir, "list")
	if err := fixture.ResetRemoteProbeLog(); err != nil {
		b.Fatalf("reset remote probe log: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := runPerformanceWW(env, fixture, "list")
		if err != nil {
			b.Fatalf("warm ww list: %v\n%s", err, out)
		}
		if got := strings.Count(out, "\n") - 1; got != fixture.ExpectedEntries {
			b.Fatalf("warm ww list entries = %d, want %d\n%s", got, fixture.ExpectedEntries, out)
		}
	}
	b.StopTimer()
	reportRemoteProbeCount(b, fixture)
}

func BenchmarkWorkspaceCleanDryRunWarm(b *testing.B) {
	env, fixture := benchmarkPerformanceFixture(b)
	primePerformanceCache(b, env, fixture.StartDir, "clean", "--dry-run")
	if err := fixture.ResetRemoteProbeLog(); err != nil {
		b.Fatalf("reset remote probe log: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := runPerformanceWW(env, fixture, "clean", "--dry-run")
		if err != nil {
			b.Fatalf("warm ww clean --dry-run: %v\n%s", err, out)
		}
		if got := strings.Count(out, "Would remove worktree at "); got != fixture.ExpectedCleanable {
			b.Fatalf("warm ww clean --dry-run removals = %d, want %d\n%s", got, fixture.ExpectedCleanable, out)
		}
	}
	b.StopTimer()
	reportRemoteProbeCount(b, fixture)
}

func primePerformanceCache(b *testing.B, env *testutil.HostEnv, dir string, args ...string) {
	b.Helper()
	if err := env.ResetCache(); err != nil {
		b.Fatalf("reset performance cache: %v", err)
	}
	if out, err := runPerformanceWW(env, performanceFixtureState.fixture, args...); err != nil {
		b.Fatalf("prime performance cache: %v\n%s", err, out)
	}
	if _, ok := cache.NewAt(filepath.Join(env.CacheDir(), "ww")).Load(dir, false); !ok {
		b.Fatal("prime performance cache did not produce a validated hit")
	}
}

func runPerformanceWW(env *testutil.HostEnv, fixture *testutil.PerformanceFixture, args ...string) (string, error) {
	if commandEnv := fixture.CommandEnv(); len(commandEnv) > 0 {
		return env.RunWWWithEnv(fixture.StartDir, commandEnv, args...)
	}
	return env.RunWW(fixture.StartDir, args...)
}

func reportRemoteProbeCount(b *testing.B, fixture *testutil.PerformanceFixture) {
	b.Helper()
	if len(fixture.CommandEnv()) == 0 {
		return
	}
	count, err := fixture.RemoteProbeCount()
	if err != nil {
		b.Fatalf("read remote probe log: %v", err)
	}
	if b.N > 0 {
		b.ReportMetric(float64(count)/float64(b.N), "ls-remote/op")
	}
}
