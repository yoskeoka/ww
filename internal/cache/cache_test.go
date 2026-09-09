package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestStoreUsesUserCacheRootHashedFilesAndPrivatePermissions(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("HOME", t.TempDir())

	root, topology := testTopology(t)
	store := New()
	if store == nil {
		t.Fatal("New returned nil")
	}
	store.Save(topology)

	if want := filepath.Join(cacheHome, "ww"); store.Root() != want {
		t.Fatalf("cache root = %q, want %q", store.Root(), want)
	}
	info, err := os.Stat(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("cache directory permissions = %o, broader than 0700", info.Mode().Perm())
	}
	entries, err := os.ReadDir(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || filepath.Ext(entries[0].Name()) != ".json" {
		t.Fatalf("cache files = %v, want one hashed json file", entries)
	}
	if len(entries[0].Name()) < 32 || entries[0].Name() == filepath.Base(root) {
		t.Fatalf("cache filename %q is not path-derived and hashed", entries[0].Name())
	}
	fileInfo, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if fileInfo.Mode().Perm()&0077 != 0 {
		t.Fatalf("cache file permissions = %o, broader than 0600", fileInfo.Mode().Perm())
	}
	if got, ok := store.Load(root, false); !ok || got.Root != topology.Root || len(got.Repos) != 1 {
		t.Fatalf("Load() = %+v, %v; want saved topology", got, ok)
	}
}

func TestStoreTreatsCorruptUnknownAndChangedEntriesAsMisses(t *testing.T) {
	root, topology := testTopology(t)
	store := NewAt(filepath.Join(t.TempDir(), "ww"))
	store.Save(topology)
	path := filepath.Join(store.Root(), fileName(topology.StartPath, false))

	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Load(root, false); ok {
		t.Fatal("corrupt entry unexpectedly loaded")
	}

	store.Save(topology)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var tampered Topology
	if err := json.Unmarshal(data, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered.SchemaVersion++
	data, err = json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Load(root, false); ok {
		t.Fatal("unknown schema unexpectedly loaded")
	}

	store.Save(topology)
	if err := os.MkdirAll(filepath.Join(root, "new-repo", ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Load(root, false); ok {
		t.Fatal("fingerprint change unexpectedly loaded")
	}
}

func TestStoreConcurrentWritersPublishCompleteEntries(t *testing.T) {
	root, topology := testTopology(t)
	store := NewAt(filepath.Join(t.TempDir(), "ww"))

	const writers = 16
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.Save(topology)
		}()
	}
	wg.Wait()
	if _, ok := store.Load(root, false); !ok {
		t.Fatal("concurrent writes did not leave a valid entry")
	}
}

func TestStoreFailsOpenWhenRootIsUnreadableOrSymlinked(t *testing.T) {
	root, topology := testTopology(t)
	parent := t.TempDir()
	cacheRoot := filepath.Join(parent, "cache-file")
	if err := os.WriteFile(cacheRoot, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewAt(cacheRoot)
	if _, ok := store.Load(root, false); ok {
		t.Fatal("file cache root unexpectedly hit")
	}
	store.Save(topology)
	if _, ok := store.Load(root, false); ok {
		t.Fatal("failed cache root unexpectedly hit")
	}
	if err := os.Remove(cacheRoot); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, cacheRoot); err != nil {
		t.Fatal(err)
	}
	store.Save(topology)
}

func testTopology(t *testing.T) (string, Topology) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	topology, ok := NewTopology(root, false, root, "", "workspace", []Repo{{Name: "repo", Path: repo}}, []string{root})
	if !ok {
		t.Fatal("NewTopology returned false")
	}
	return root, topology
}
