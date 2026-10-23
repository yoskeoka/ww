package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTimeStaleCleanPolicies(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	t.Parallel()
	for _, tc := range []struct {
		name, policy               string
		force, removed, branchGone bool
	}{
		{"default", "", false, true, false}, {"protected", "stale_remove_dirty=false\n", false, false, false}, {"force", "stale_remove_dirty=false\n", true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := setupRepo(t)
			writeConfig(t, repo, "default_base=\"main\"\n")
			xdg := t.TempDir()
			if err := os.MkdirAll(filepath.Join(xdg, "ww"), 0755); err != nil {
				t.Fatal(err)
			}
			env := []string{"XDG_CONFIG_HOME=" + xdg}
			wt := filepath.Join(t.TempDir(), "old")
			git := func(dir string, args ...string) string {
				t.Helper()
				out, err := globalEnv.Git(dir, args...)
				if err != nil {
					t.Fatalf("%v: %v %s", args, err, out)
				}
				return out
			}
			git(repo, "worktree", "add", "-b", "topic/old", wt)
			if err := globalEnv.WriteFile(filepath.Join(wt, "tracked"), "old"); err != nil {
				t.Fatal(err)
			}
			git(wt, "add", "tracked")
			if out, err := globalEnv.ExecWithEnv(wt, []string{"GIT_COMMITTER_DATE=2000-01-01T00:00:00Z", "GIT_AUTHOR_DATE=2026-10-22T00:00:00Z"}, "git", "commit", "-m", "old"); err != nil {
				t.Fatalf("%s %v", out, err)
			}
			check := func(dir string, want bool, args ...string) string {
				t.Helper()
				out, err := runWWWithEnv(t, dir, env, args...)
				if (err == nil) != want {
					t.Fatalf("%v: %v %s", args, err, out)
				}
				return out
			}
			out := check(wt, true, "list", "--json")
			if strings.Contains(out, `"stale_reason":"time"`) {
				t.Fatal(out)
			}
			if err := os.WriteFile(filepath.Join(xdg, "ww", "config.toml"), []byte("stale_days=30\n"+tc.policy), 0600); err != nil {
				t.Fatal(err)
			}
			out = check(wt, true, "list", "--cleanable", "--json")
			if !strings.Contains(out, `"stale_reason":"time"`) {
				t.Fatal(out)
			}
			if out = check(repo, true, "list", "--cleanable", "--stale-days", "0", "--json"); strings.Contains(out, "topic/old") {
				t.Fatal(out)
			}
			for _, f := range []string{"tracked", "untracked"} {
				if err := globalEnv.WriteFile(filepath.Join(wt, f), "dirty"); err != nil {
					t.Fatal(err)
				}
			}
			out = check(repo, true, "clean", "--dry-run")
			if tc.removed && !tc.force && !strings.Contains(out, "Dirty worktree removal enabled") {
				t.Fatal(out)
			}
			if !globalEnv.PathExists(filepath.Join(wt, "untracked")) {
				t.Fatal("dry-run removed files")
			}
			out = check(repo, true, "clean", "--dry-run", "--json")
			var preview map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &preview); err != nil || preview["stale_reason"] != "time" || preview["remove_dirty"] != (tc.policy == "") {
				t.Fatalf("preview: %s %v", out, err)
			}
			args := []string{"clean", "--json"}
			if tc.force {
				args = append(args, "--force")
			}
			check(repo, tc.removed, args...)
			if globalEnv.PathExists(wt) == tc.removed {
				t.Fatal("worktree removal mismatch")
			}
			_, err := globalEnv.Git(repo, "rev-parse", "--verify", "refs/heads/topic/old")
			if (err != nil) != tc.branchGone {
				t.Fatalf("branch removal mismatch: %v", err)
			}
		})
	}
}

func TestTimeStaleWorkspaceRepoSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("integration")
	}
	t.Parallel()
	root := t.TempDir()
	xdg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(xdg, "ww"), 0755); err != nil {
		t.Fatal(err)
	}
	env := []string{"XDG_CONFIG_HOME=" + xdg}
	git := func(dir string, args ...string) {
		t.Helper()
		out, err := globalEnv.Git(dir, args...)
		if err != nil {
			t.Fatalf("%v: %v %s", args, err, out)
		}
	}
	for _, name := range []string{"alpha", "beta"} {
		repo := filepath.Join(root, name)
		if err := os.MkdirAll(repo, 0755); err != nil {
			t.Fatal(err)
		}
		git(repo, "init", "-b", "main")
		git(repo, "commit", "--allow-empty", "-m", "initial")
		writeConfig(t, repo, "default_base=\"main\"\n")
		wt := filepath.Join(root, ".worktrees", name+"@old")
		git(repo, "worktree", "add", "-b", "topic/old", wt)
		if err := globalEnv.WriteFile(filepath.Join(wt, name), name); err != nil {
			t.Fatal(err)
		}
		git(wt, "add", name)
		if out, err := globalEnv.ExecWithEnv(wt, []string{"GIT_COMMITTER_DATE=2000-01-01T00:00:00Z"}, "git", "commit", "-m", "old"); err != nil {
			t.Fatalf("%s %v", out, err)
		}
	}
	body := fmt.Sprintf("stale_days=30\n[[projects]]\nroot=%q\nstale_days=0\nstale_remove_dirty=false\n", filepath.Join(root, "beta"))
	if err := os.WriteFile(filepath.Join(xdg, "ww", "config.toml"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{root, filepath.Join(root, ".worktrees", "alpha@old")} {
		out, err := runWWWithEnv(t, dir, env, "list", "--cleanable", "--json")
		if err != nil {
			t.Fatalf("%s %v", out, err)
		}
		if !strings.Contains(out, `"repo":"alpha"`) || strings.Contains(out, `"repo":"beta"`) {
			t.Fatal(out)
		}
		out, err = runWWWithEnv(t, dir, env, "clean", "--stale-days", "30", "--dry-run", "--json")
		if err != nil {
			t.Fatalf("%s %v", out, err)
		}
		if !strings.Contains(out, `"repo":"beta"`) || !strings.Contains(out, `"remove_dirty":false`) {
			t.Fatal(out)
		}
	}
}
