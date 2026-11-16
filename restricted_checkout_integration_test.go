package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWithCheckoutExclusions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	repo := setupRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nested/.env", "nested/key.pem", "nested/keep"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"add", "-f", "nested"}, {"commit", "-m", "fixture"}} {
		if out, err := globalEnv.Git(repo, args...); err != nil {
			t.Fatalf("git: %v: %s", err, out)
		}
	}
	writeConfig(t, repo, "default_base = 'main'\ncheckout_exclude = ['.env', '*.pem']\n")
	preview, err := runWW(t, repo, "create", "--dry-run", "feat/test")
	if err != nil || !strings.Contains(preview, "*.pem") {
		t.Fatalf("preview: %s: %v", preview, err)
	}
	out, err := runWW(t, repo, "create", "--quiet", "feat/test")
	if err != nil {
		t.Fatalf("create: %s: %v", out, err)
	}
	wt := strings.TrimSpace(out)
	for _, name := range []string{"nested/.env", "nested/key.pem"} {
		if _, err := os.Stat(filepath.Join(wt, name)); !os.IsNotExist(err) {
			t.Errorf("excluded %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(wt, "nested/keep")); err != nil {
		t.Fatal(err)
	}
}
