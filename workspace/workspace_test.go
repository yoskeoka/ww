package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yoskeoka/ww/git"
)

// evalTempDir resolves symlinks in t.TempDir() so that path comparisons
// work on macOS where /tmp is a symlink to /private/tmp.
func evalTempDir(t *testing.T) string {
	t.Helper()
	d, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDetectStandaloneRepo(t *testing.T) {
	root := evalTempDir(t)
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
	root := evalTempDir(t)
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

func TestDetectNearestContainingGitWorkspaceWinsOverGrandparent(t *testing.T) {
	root := evalTempDir(t)
	grandparent := filepath.Join(root, "grandparent")
	workspaceRoot := filepath.Join(grandparent, "workspace")
	otherRepo := filepath.Join(grandparent, "other")
	childA := filepath.Join(workspaceRoot, "child-a")
	childB := filepath.Join(workspaceRoot, "child-b")

	gitInit(t, grandparent)
	gitInit(t, workspaceRoot)
	gitInit(t, otherRepo)
	gitInit(t, childA)
	gitInit(t, childB)

	nestedStart := filepath.Join(childA, "internal", "deep")
	if err := os.MkdirAll(nestedStart, 0755); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(nestedStart)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != workspaceRoot {
		t.Fatalf("Root = %q, want %q", ws.Root, workspaceRoot)
	}
	want := []string{"child-a", "child-b", "workspace"}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, want) {
		t.Fatalf("Repos = %v, want %v", got, want)
	}
}

func TestDetectNonGitParentWithSiblings(t *testing.T) {
	root := evalTempDir(t)
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
	root := evalTempDir(t)
	childB := filepath.Join(root, "repo-b")
	childA := filepath.Join(root, "repo-a")
	gitInit(t, childA)
	gitInitSeparateGitDir(t, childB, filepath.Join(root, "repo-b.git"))

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

func TestDetectNestedRepoCanResolveContainingGrandparentWorkspace(t *testing.T) {
	root := evalTempDir(t)
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
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != root {
		t.Fatalf("Root = %q, want %q", ws.Root, root)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"child", "sibling"}) {
		t.Fatalf("Repos = %v, want [child sibling]", got)
	}
}

func TestDetectIgnoresInvalidGitMarkerDirectory(t *testing.T) {
	root := evalTempDir(t)
	childA := filepath.Join(root, "child-a")
	childB := filepath.Join(root, "child-b")
	gitInit(t, childA)
	if err := os.MkdirAll(filepath.Join(childB, ".git", "gk"), 0755); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"child-a"}) {
		t.Fatalf("Repos = %v, want [child-a]", got)
	}
}

func TestDetectStandaloneRepoWithSeparateGitDirFileCounts(t *testing.T) {
	root := evalTempDir(t)
	childA := filepath.Join(root, "child-a")
	childB := filepath.Join(root, "child-b")
	gitInit(t, childA)
	gitInitSeparateGitDir(t, childB, filepath.Join(root, "child-b.git"))

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

func TestDetectIgnoresChildSymlinkToRepo(t *testing.T) {
	root := evalTempDir(t)
	child := filepath.Join(root, "child")
	sibling := filepath.Join(root, "sibling")

	gitInit(t, child)
	targetRoot := evalTempDir(t)
	target := filepath.Join(targetRoot, "target")
	gitInit(t, target)
	if err := os.Symlink(target, sibling); err != nil {
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

func TestScanImmediateReposIgnoresRegularFilesAndChildSymlinks(t *testing.T) {
	root := evalTempDir(t)
	repo := filepath.Join(root, "repo")
	file := filepath.Join(root, "notes.txt")
	symlink := filepath.Join(root, "repo-link")

	gitInit(t, repo)
	if err := os.WriteFile(file, []byte("not a repo"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repo, symlink); err != nil {
		t.Fatal(err)
	}

	repos, err := scanImmediateRepos(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := repoNames(repos); !reflect.DeepEqual(got, []string{"repo"}) {
		t.Fatalf("Repos = %v, want [repo]", got)
	}
}

func TestDetectIgnoresLinkedWorktreeSibling(t *testing.T) {
	root := evalTempDir(t)
	child := filepath.Join(root, "child")
	gitInit(t, child)

	worktreePath := filepath.Join(root, "child-worktree")
	runner := git.Runner{Dir: child}
	if _, err := runner.Run("config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("branch", "feat/worktree"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run("worktree", "add", worktreePath, "feat/worktree"); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(child)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeSingleRepo {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeSingleRepo)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"child"}) {
		t.Fatalf("Repos = %v, want [child]", got)
	}
}

func TestDetectCurrentDirectoryWorkspaceWinsImmediately(t *testing.T) {
	root := evalTempDir(t)
	outer := filepath.Join(root, "outer")
	workspaceRoot := filepath.Join(outer, "workspace")
	otherRepo := filepath.Join(outer, "other")
	childA := filepath.Join(workspaceRoot, "child-a")
	childB := filepath.Join(workspaceRoot, "child-b")

	gitInit(t, outer)
	gitInit(t, workspaceRoot)
	gitInit(t, otherRepo)
	gitInit(t, childA)
	gitInit(t, childB)

	ws, err := Detect(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != workspaceRoot {
		t.Fatalf("Root = %q, want %q", ws.Root, workspaceRoot)
	}
	want := []string{"child-a", "child-b", "workspace"}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, want) {
		t.Fatalf("Repos = %v, want %v", got, want)
	}
}

func TestDetectGitWorkspaceRootSelectedFromNestedPath(t *testing.T) {
	root := evalTempDir(t)
	outer := filepath.Join(root, "outer")
	workspaceRoot := filepath.Join(outer, "workspace")
	otherRepo := filepath.Join(outer, "other")
	childA := filepath.Join(workspaceRoot, "child-a")
	childB := filepath.Join(workspaceRoot, "child-b")
	internal := filepath.Join(workspaceRoot, "internal")

	gitInit(t, outer)
	gitInit(t, workspaceRoot)
	gitInit(t, otherRepo)
	gitInit(t, childA)
	gitInit(t, childB)
	if err := os.MkdirAll(internal, 0755); err != nil {
		t.Fatal(err)
	}

	ws, err := Detect(internal)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != workspaceRoot {
		t.Fatalf("Root = %q, want %q", ws.Root, workspaceRoot)
	}
	want := []string{"child-a", "child-b", "workspace"}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, want) {
		t.Fatalf("Repos = %v, want %v", got, want)
	}
}
func TestDetectOrdersReposDeterministically(t *testing.T) {
	root := evalTempDir(t)
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

func TestDetectWorktreeSiblingNotCountedAsWorkspaceMember(t *testing.T) {
	root := evalTempDir(t)
	repoA := filepath.Join(root, "repo-a")
	gitInit(t, repoA)

	// Simulate a ww-created worktree sibling: .git file points into .git/worktrees/
	wtSibling := filepath.Join(root, "repo-a@feat-branch")
	if err := os.MkdirAll(wtSibling, 0755); err != nil {
		t.Fatal(err)
	}
	gitfileContent := "gitdir: " + repoA + "/.git/worktrees/feat-branch"
	if err := os.WriteFile(filepath.Join(wtSibling, ".git"), []byte(gitfileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Only one real repo → should be single-repo, not workspace
	ws, err := Detect(repoA)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeSingleRepo {
		t.Fatalf("Mode = %q, want %q — worktree sibling should not trigger workspace mode", ws.Mode, ModeSingleRepo)
	}
}

func TestDetectSandboxFromChildRepoDoesNotDiscoverParentWorkspace(t *testing.T) {
	root := evalTempDir(t)
	childA := filepath.Join(root, "child-a")
	childB := filepath.Join(root, "child-b")
	gitInit(t, childA)
	gitInit(t, childB)

	ws, err := DetectWithOptions(childA, DetectOptions{Sandbox: true})
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeSingleRepo {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeSingleRepo)
	}
	if ws.Root != childA {
		t.Fatalf("Root = %q, want %q", ws.Root, childA)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"child-a"}) {
		t.Fatalf("Repos = %v, want [child-a]", got)
	}
}

func TestDetectSandboxCurrentDirectoryWorkspaceRoot(t *testing.T) {
	root := evalTempDir(t)
	childA := filepath.Join(root, "child-a")
	childB := filepath.Join(root, "child-b")
	gitInit(t, childA)
	gitInit(t, childB)

	ws, err := DetectWithOptions(root, DetectOptions{Sandbox: true})
	if err != nil {
		t.Fatal(err)
	}
	if ws.Mode != ModeWorkspace {
		t.Fatalf("Mode = %q, want %q", ws.Mode, ModeWorkspace)
	}
	if ws.Root != root {
		t.Fatalf("Root = %q, want %q", ws.Root, root)
	}
	if got := repoNames(ws.Repos); !reflect.DeepEqual(got, []string{"child-a", "child-b"}) {
		t.Fatalf("Repos = %v, want [child-a child-b]", got)
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

func gitInitSeparateGitDir(t *testing.T, dir, gitDir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(gitDir), 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "--separate-git-dir", gitDir, "-b", "main", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --separate-git-dir in %s: %v\n%s", dir, err, string(out))
	}
}
