package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRemoteStoreHitsOnlyForFreshPositiveCoverage(t *testing.T) {
	commonDir := t.TempDir()
	storeRoot := filepath.Join(t.TempDir(), "ww")
	now := time.Unix(100, 0).UTC()
	store := NewAtWithClock(storeRoot, func() time.Time { return now })
	urls := []string{"https://user:secret@example.test/repo.git"}
	positive := map[string]struct{}{"feat/one": {}, "release/two": {}}
	store.SaveRemote(commonDir, "origin", urls, positive)

	got, ok := store.LoadRemote(commonDir, "origin", urls, []string{"feat/one", "release/two"})
	if !ok || len(got) != len(positive) {
		t.Fatalf("fresh positive load = %#v, %v; want full hit", got, ok)
	}

	now = now.Add(RemotePositiveTTL - time.Nanosecond)
	if _, ok := store.LoadRemote(commonDir, "origin", urls, []string{"feat/one"}); !ok {
		t.Fatal("entry before TTL boundary unexpectedly missed")
	}
	now = time.Unix(100, 0).Add(RemotePositiveTTL).UTC()
	if _, ok := store.LoadRemote(commonDir, "origin", urls, []string{"feat/one"}); ok {
		t.Fatal("entry at exact TTL boundary unexpectedly hit")
	}

	now = time.Unix(100, 0).UTC()
	if _, ok := store.LoadRemote(commonDir, "origin", urls, []string{"feat/missing"}); ok {
		t.Fatal("candidate absent from positive set unexpectedly hit")
	}

	entries, err := os.ReadDir(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("remote cache entries = %d, want one", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(store.Root(), entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret") || strings.Contains(string(data), "example.test") {
		t.Fatalf("remote URL leaked into cache entry: %s", data)
	}
}

func TestRemoteStoreIdentityChangesMiss(t *testing.T) {
	commonDir := t.TempDir()
	otherCommonDir := t.TempDir()
	store := NewAtWithClock(filepath.Join(t.TempDir(), "ww"), func() time.Time {
		return time.Unix(200, 0).UTC()
	})
	urls := []string{"/tmp/origin.git", "/tmp/origin-mirror.git"}
	store.SaveRemote(commonDir, "origin", urls, map[string]struct{}{"main": {}})

	for name, args := range map[string]struct {
		commonDir string
		remote    string
		urls      []string
	}{
		"common directory": {commonDir: otherCommonDir, remote: "origin", urls: urls},
		"remote name":      {commonDir: commonDir, remote: "backup", urls: urls},
		"remote URL":       {commonDir: commonDir, remote: "origin", urls: []string{"/tmp/changed.git"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := store.LoadRemote(args.commonDir, args.remote, args.urls, []string{"main"}); ok {
				t.Fatal("changed remote identity unexpectedly hit")
			}
		})
	}
}

func TestRemoteStoreRejectsCorruptAndTamperedEntries(t *testing.T) {
	commonDir := t.TempDir()
	store := NewAtWithClock(filepath.Join(t.TempDir(), "ww"), func() time.Time {
		return time.Unix(300, 0).UTC()
	})
	urls := []string{"/tmp/origin.git"}
	store.SaveRemote(commonDir, "origin", urls, map[string]struct{}{"main": {}, "feat/one": {}})
	path := remoteEntryPath(t, store)

	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.LoadRemote(commonDir, "origin", urls, []string{"main"}); ok {
		t.Fatal("corrupt entry unexpectedly hit")
	}

	store.SaveRemote(commonDir, "origin", urls, map[string]struct{}{"main": {}, "feat/one": {}})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entry RemoteEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatal(err)
	}
	entry.Branches = []string{"main", "main"}
	data, err = json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.LoadRemote(commonDir, "origin", urls, []string{"main"}); ok {
		t.Fatal("tampered duplicate branch entry unexpectedly hit")
	}

	store.SaveRemote(commonDir, "origin", urls, map[string]struct{}{"main": {}, "feat/one": {}})
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("junk")...)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.LoadRemote(commonDir, "origin", urls, []string{"main"}); ok {
		t.Fatal("entry with trailing data unexpectedly hit")
	}
}

func TestRemoteStoreConcurrentWritersPublishCompleteEntries(t *testing.T) {
	commonDir := t.TempDir()
	store := NewAtWithClock(filepath.Join(t.TempDir(), "ww"), func() time.Time {
		return time.Unix(400, 0).UTC()
	})
	urls := []string{"/tmp/origin.git"}

	const writers = 16
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store.SaveRemote(commonDir, "origin", urls, map[string]struct{}{
				"feat/one":                           {},
				"feat/writer-" + string(rune('a'+i)): {},
			})
		}(i)
	}
	wg.Wait()
	if _, ok := store.LoadRemote(commonDir, "origin", urls, []string{"feat/one"}); !ok {
		t.Fatal("concurrent writes did not leave a valid positive entry")
	}
}

func remoteEntryPath(t *testing.T, store *Store) string {
	t.Helper()
	entries, err := os.ReadDir(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("cache entries = %d, want one", len(entries))
	}
	return filepath.Join(store.Root(), entries[0].Name())
}
