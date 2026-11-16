package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRestrictedCheckoutPreservesMain(t *testing.T) {
	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.invalid")
	run("config", "user.name", "Test")
	if err := os.MkdirAll(filepath.Join(repo, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"keep", ".env", "nested/.env", "nested/key.pem"} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run("add", ".")
	run("commit", "-m", "fixture")
	run("branch", "feature")
	r := &Runner{Dir: repo}
	if err := r.CheckSparseIsolation(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "worktree")
	if err := r.WorktreeAddWithoutCheckout(target, "feature", false); err != nil {
		t.Fatal(err)
	}
	if err := r.RestrictedCheckout(target, []string{".env", "*.pem"}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".env", "nested/.env", "nested/key.pem"} {
		if _, err := os.Stat(filepath.Join(target, name)); !os.IsNotExist(err) {
			t.Errorf("excluded %s exists or error: %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(repo, name)); err != nil {
			t.Errorf("main %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "keep")); err != nil {
		t.Fatal(err)
	}

	// A sparse main worktree keeps its own rules and materialized files.
	run("sparse-checkout", "set", "--no-cone", "/keep")
	run("branch", "feature2")
	if err := r.CheckSparseIsolation(); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(t.TempDir(), "second")
	if err := r.WorktreeAddWithoutCheckout(second, "feature2", false); err != nil {
		t.Fatal(err)
	}
	if err := r.RestrictedCheckout(second, []string{".env"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "keep")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, "nested/key.pem")); !os.IsNotExist(err) {
		t.Fatalf("main sparse rules changed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(second, "nested/key.pem")); err != nil {
		t.Fatalf("second worktree inherited main rules: %v", err)
	}
}
