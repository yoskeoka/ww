package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectStandaloneRepo(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	gitInit(t, repo)

	ws, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeSingleRepo {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeSingleRepo)
	}
	if ws.Root != repo {
		t.Fatalf("Root = %q, want %q", ws.Root, repo)
	}
	if len(ws.Repos) != 1 {
		t.Fatalf("Repos len = %d, want 1", len(ws.Repos))
	}
	if ws.Repos[0].Name != "repo" || ws.Repos[0].Path != repo {
		t.Fatalf("Repos[0] = %+v, want repo at %s", ws.Repos[0], repo)
	}
}

func TestDetectGitParentWithSiblings(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	childA := filepath.Join(root, "child-a")
	childB := filepath.Join(root, "child-b")
	gitInit(t, childA)
	gitInit(t, childB)

	ws, err := Detect(childA)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != root {
		t.Fatalf("Root = %q, want %q", ws.Root, root)
	}
	want := []string{filepath.Base(root), "child-a", "child-b"}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, want) {
		t.Fatalf("Repos = %v, want %v", got, want)
	}
}

func TestDetectNonGitParentWithSiblings(t *testing.T) {
	root := t.TempDir()
	childB := filepath.Join(root, "child-b")
	childA := filepath.Join(root, "child-a")
	gitInit(t, childA)
	gitInit(t, childB)

	ws, err := Detect(childA)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != root {
		t.Fatalf("Root = %q, want %q", ws.Root, root)
	}
	want := []string{"child-a", "child-b"}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, want) {
		t.Fatalf("Repos = %v, want %v", got, want)
	}
}

func TestDetectNonGitWorkspaceRootWithChildren(t *testing.T) {
	root := t.TempDir()
	childB := filepath.Join(root, "repo-b")
	childA := filepath.Join(root, "repo-a")
	gitInit(t, childA)
	if err := os.MkdirAll(childB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childB, ".git"), []byte("gitdir: /tmp/nowhere"), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != root {
		t.Fatalf("Root = %q, want %q", ws.Root, root)
	}
	want := []string{"repo-a", "repo-b"}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, want) {
		t.Fatalf("Repos = %v, want %v", got, want)
	}
}

func TestDetectNestedRepoStopsAtOneLevel(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	sibling := filepath.Join(root, "sibling")
	nested := filepath.Join(child, "nested")
	gitInit(t, child)
	gitInit(t, sibling)
	gitInit(t, nested)

	ws, err := Detect(nested)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeSingleRepo {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeSingleRepo)
	}
	if ws.Root != nested {
		t.Fatalf("Root = %q, want %q", ws.Root, nested)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"nested"}) {
		t.Fatalf("Repos = %v, want [nested]", got)
	}
}

func TestDetectGitFileAndDirectoryMarkers(t *testing.T) {
	root := t.TempDir()
	childA := filepath.Join(root, "child-a")
	childB := filepath.Join(root, "child-b")
	gitInit(t, childA)
	if err := os.MkdirAll(childB, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childB, ".git"), []byte("gitdir: /tmp/example"), 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"child-a", "child-b"}) {
		t.Fatalf("Repos = %v, want [child-a child-b]", got)
	}
}

func TestDetectIgnoresGitWorktreeFileSibling(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	sibling := filepath.Join(root, "sibling")

	gitInit(t, child)
	if err := os.MkdirAll(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	// Simulate a git worktree checkout whose .git file points into another repo's .git/worktrees directory.
	gitFileContents := []byte("gitdir: /tmp/other-repo/.git/worktrees/wt-1")
	if err := os.WriteFile(filepath.Join(sibling, ".git"), gitFileContents, 0644); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(child)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeSingleRepo {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeSingleRepo)
	}
	if ws.Root != child {
		t.Fatalf("Root = %q, want %q", ws.Root, child)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"child"}) {
		t.Fatalf("Repos = %v, want [child]", got)
	}
}
func TestDetectOrdersReposDeterministically(t *testing.T) {
	root := t.TempDir()
	zeta := filepath.Join(root, "zeta")
	alpha := filepath.Join(root, "alpha")
	gitInit(t, zeta)
	gitInit(t, alpha)

	ws, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"alpha", "zeta"}) {
		t.Fatalf("Repos = %v, want [alpha zeta]", got)
	}
}

func repoNames(repos []Repo) []string {
	names := make([]string, len(repos))
	for i, repo := range repos {
		names[i] = repo.Name
	}
	return names
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init in %s: %v\n%s", dir, err, string(out))
	}
}
