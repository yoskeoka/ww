package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStaleConfigLayers(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	global := filepath.Join(root, "xdg", "ww")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(global, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(global))
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := Load(repo)
	if err != nil || cfg.StaleDays != 0 || cfg.StaleRemoveDirty != nil {
		t.Fatalf("defaults: %+v %v", cfg, err)
	}
	write(filepath.Join(global, GlobalFileName), "stale_days=30\nstale_remove_dirty=false\n[[projects]]\nroot="+`"`+repo+`"`+"\nstale_days=10\nstale_remove_dirty=true\n")
	cfg, err = LoadWithOptions(repo, LoadOptions{ProjectRoot: repo})
	if err != nil || cfg.StaleDays != 10 || cfg.StaleRemoveDirty == nil || !*cfg.StaleRemoveDirty {
		t.Fatalf("project: %+v %v", cfg, err)
	}
	write(filepath.Join(repo, FileName), "stale_days=0\nstale_remove_dirty=false\n")
	cfg, err = LoadWithOptions(repo, LoadOptions{ProjectRoot: repo})
	if err != nil || cfg.StaleDays != 0 || cfg.StaleRemoveDirty == nil || *cfg.StaleRemoveDirty {
		t.Fatalf("local: %+v %v", cfg, err)
	}
	for _, body := range []string{"stale_days=-1", "stale_days=106752", "stale_days=9223372036854775808", `stale_days="30"`, `stale_remove_dirty="false"`} {
		write(filepath.Join(repo, FileName), body)
		if _, err := Load(repo); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
