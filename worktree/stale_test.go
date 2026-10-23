package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/yoskeoka/ww/git"
)

func TestElapsedStaleBoundary(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		age  time.Duration
		want bool
	}{{30*24*time.Hour - time.Nanosecond, false}, {30 * 24 * time.Hour, true}, {30*24*time.Hour + time.Nanosecond, true}, {-time.Hour, false}} {
		if got := elapsedStale(now, now.Add(-tc.age), 30); got != tc.want {
			t.Fatalf("age %s: %t", tc.age, got)
		}
	}
}

func TestTimeStaleStatusAndDirtyBranchSafety(t *testing.T) {
	repo := setupGitRepo(t)
	r := &git.Runner{Dir: repo}
	m := &Manager{Git: r, RepoDir: repo, Config: Config{DefaultBase: "main", StaleDays: 30}, Now: func() time.Time { return time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC) }}
	info, _, err := m.Create("topic/old", CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	wr := &git.Runner{Dir: info.Path}
	if err := os.WriteFile(filepath.Join(info.Path, "old.txt"), []byte("committed"), 0600); err != nil {
		t.Fatal(err)
	}
	mustGit(t, wr, "add", "old.txt")
	cmd := exec.Command("git", "commit", "-m", "old")
	cmd.Dir = info.Path
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE=2026-09-01T00:00:00Z", "GIT_AUTHOR_DATE=2026-10-22T00:00:00Z")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
	// An old committer date and recent author date deliberately differ.
	infos, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	var old WorktreeInfo
	for _, v := range infos {
		if v.Branch == info.Branch {
			old = v
		}
	}
	if old.Status != StatusStale || old.StaleReason != "time" || !old.RemoveDirty {
		t.Fatalf("%+v", old)
	}
	zero := int64(0)
	m.Config.StaleDaysOverride = &zero
	infos, err = m.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range infos {
		if v.Branch == info.Branch && v.Status != StatusActive {
			t.Fatalf("disabled: %+v", v)
		}
	}
	for _, f := range []string{"old.txt", "untracked"} {
		if err := os.WriteFile(filepath.Join(info.Path, f), []byte("dirty"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := m.Remove(info.Branch, RemoveOpts{}); err == nil {
		t.Fatal("dirty removal unexpectedly succeeded")
	}
	result, _, err := m.Remove(info.Branch, RemoveOpts{RemoveDirty: true})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Removed || result.BranchDeleted || result.BranchError == "" || !r.BranchExists(info.Branch) {
		t.Fatalf("branch safety: %+v", result)
	}
	if _, err := os.Stat(info.Path); !os.IsNotExist(err) {
		t.Fatalf("worktree remains: %v", err)
	}
}

func TestTimeStaleTimestampFailureAndDisabled(t *testing.T) {
	repo := setupGitRepo(t)
	r := &git.Runner{Dir: repo}
	m := &Manager{Git: r, RepoDir: repo, Config: Config{DefaultBase: "main", StaleDays: 30}}
	info, _, err := m.Create("topic/timestamp", CreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	wr := &git.Runner{Dir: info.Path}
	if err := os.WriteFile(filepath.Join(info.Path, "unique"), []byte("unique"), 0600); err != nil {
		t.Fatal(err)
	}
	mustGit(t, wr, "add", "unique")
	mustGit(t, wr, "commit", "-m", "unique")
	r.GitBin = recoveryGit(t, `if [ "$1" = show ] && [ "$2" = --no-patch ]; then echo bad-timestamp >&2; exit 42; fi`)
	if infos, err := m.List(); err == nil || infos != nil {
		t.Fatalf("timestamp failure: %+v %v", infos, err)
	}
	m.Config.StaleDays = 0
	if _, err := m.List(); err != nil {
		t.Fatal("disabled should skip timestamps", err)
	}
	m.Config.DefaultBase = ""
	m.Config.StaleDays = 30
	infos, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range infos {
		if v.Branch == info.Branch && v.Status != StatusUnknown {
			t.Fatalf("unknown promoted: %+v", v)
		}
	}
}

func TestTimeStalePrecedenceAndRemoteReasons(t *testing.T) {
	repo, _ := setupGitRepoWithRemote(t)
	r := &git.Runner{Dir: repo}
	now := time.Now().Add(31 * 24 * time.Hour)
	m := &Manager{Git: r, RepoDir: repo, Config: Config{StaleDays: 30}, Now: func() time.Time { return now }}
	for _, branch := range []string{"topic/local", "topic/present", "topic/gone", "topic/merged"} {
		info, _, err := m.Create(branch, CreateOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if branch == "topic/merged" {
			continue
		}
		wr := &git.Runner{Dir: info.Path}
		if err := os.WriteFile(filepath.Join(info.Path, "unique"), []byte(branch), 0600); err != nil {
			t.Fatal(err)
		}
		mustGit(t, wr, "add", "unique")
		mustGit(t, wr, "commit", "-m", branch)
		if branch == "topic/present" {
			mustGit(t, wr, "push", "-u", "origin", branch)
		}
		if branch == "topic/gone" {
			mustGit(t, r, "config", "branch."+branch+".remote", "origin")
		}
	}
	mustGit(t, r, "worktree", "add", "--detach", filepath.Join(t.TempDir(), "detached"), "main")
	calls := 0
	m.Now = func() time.Time { calls++; return now }
	infos, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("clock calls=%d", calls)
	}
	for _, v := range infos {
		if v.Main || v.Branch == "" {
			if v.Status != StatusActive || v.StaleReason != "" {
				t.Fatalf("structural: %+v", v)
			}
			continue
		}
		if v.Branch == "topic/merged" {
			if v.Status != StatusMerged || v.StaleReason != "" || v.RemoveDirty {
				t.Fatalf("merged: %+v", v)
			}
			continue
		}
		if v.Status != StatusStale || v.StaleReason != "time" || !v.RemoveDirty {
			t.Fatalf("time: %+v", v)
		}
	}
	no := false
	m.Config.StaleRemoveDirty = &no
	infos, err = m.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range infos {
		if v.RemoveDirty {
			t.Fatalf("policy: %+v", v)
		}
	}
	zero := int64(0)
	m.Config.StaleDaysOverride = &zero
	infos, err = m.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range infos {
		if v.Branch == "topic/gone" {
			if v.Status != StatusStale || v.StaleReason != "remote" || v.RemoveDirty {
				t.Fatalf("remote: %+v", v)
			}
		} else if v.StaleReason != "" {
			t.Fatalf("disabled time: %+v", v)
		}
	}
}
