package testutil

import (
	"fmt"
	"path"
	"sync"
	"testing"
)

var (
	repoSeedOnce sync.Once
	repoSeedPath string
	repoSeedErr  error
)

// RepoOpts configures a single test git repository.
type RepoOpts struct {
	// Name is the repository directory name. Defaults to "myrepo".
	Name string
	// DefaultBranch is the initial branch name. Defaults to "main".
	DefaultBranch string
}

// Workspace represents a multi-repo workspace created on the host filesystem.
type Workspace struct {
	// RootDir is the parent directory containing all child repos.
	RootDir string
	// RepoDirs contains the absolute path of each child repo.
	RepoDirs []string
}

// WorkspaceOpts configures a multi-repo workspace.
type WorkspaceOpts struct {
	// NumRepos is the number of child repositories to create.
	NumRepos int
	// RepoOpts provides per-repo configuration. If shorter than NumRepos,
	// remaining repos use defaults with auto-generated names (repo1, repo2, …).
	RepoOpts []RepoOpts
}

// SetupRepo creates a single git repository with realistic seed data.
// Returns the absolute path to the repository.
func SetupRepo(t *testing.T, env *HostEnv, opts RepoOpts) string {
	t.Helper()
	if opts.Name == "" {
		opts.Name = "myrepo"
	}
	if opts.DefaultBranch == "" {
		opts.DefaultBranch = "main"
	}

	if opts.Name == "myrepo" && opts.DefaultBranch == "main" {
		seedRepo, err := ensureRepoSeed(env)
		if err != nil {
			t.Fatal(err)
		}
		return cloneRepoSeed(t, env, seedRepo, opts)
	}

	return setupRepoFromScratch(t, env, opts)
}

func ensureRepoSeed(env *HostEnv) (string, error) {
	repoSeedOnce.Do(func() {
		repoSeedPath, repoSeedErr = createRepoSeed(env)
	})
	return repoSeedPath, repoSeedErr
}

func createRepoSeed(env *HostEnv) (string, error) {
	baseDir, err := env.MkdirTemp("ww-seed")
	if err != nil {
		return "", err
	}

	repo := path.Join(baseDir, "myrepo")
	if err := env.MkdirAll(repo); err != nil {
		return "", err
	}

	git := func(args ...string) error {
		if _, err := env.Git(repo, args...); err != nil {
			return fmt.Errorf("git %v: %w", args, err)
		}
		return nil
	}

	writeFile := func(name, content string) error {
		filePath := path.Join(repo, name)
		if err := env.MkdirAll(path.Dir(filePath)); err != nil {
			return err
		}
		if err := env.WriteFile(filePath, content); err != nil {
			return fmt.Errorf("writeFile %s: %w", name, err)
		}
		return nil
	}

	if err := git("init", "-b", "main"); err != nil {
		return "", err
	}

	// Commit 1: initial project structure
	if err := writeFile("README.md", "# My Repo\n\nA test repository for ww integration tests.\n"); err != nil {
		return "", err
	}
	if err := writeFile("go.mod", "module example.com/myrepo\n\ngo 1.23.0\n"); err != nil {
		return "", err
	}
	if err := writeFile("main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"); err != nil {
		return "", err
	}
	if err := git("add", "."); err != nil {
		return "", err
	}
	if err := git("commit", "-m", "initial: project scaffold"); err != nil {
		return "", err
	}

	// Commit 2: add utility package
	if err := writeFile("internal/util.go", "package internal\n\nfunc Add(a, b int) int { return a + b }\n"); err != nil {
		return "", err
	}
	if err := writeFile("internal/util_test.go", "package internal\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n"); err != nil {
		return "", err
	}
	if err := git("add", "."); err != nil {
		return "", err
	}
	if err := git("commit", "-m", "feat: add util package"); err != nil {
		return "", err
	}

	// Commit 3: docs update
	if err := writeFile("README.md", "# My Repo\n\nA test repository for ww integration tests.\n\n## Usage\n\nRun `go run main.go`\n"); err != nil {
		return "", err
	}
	if err := git("add", "."); err != nil {
		return "", err
	}
	if err := git("commit", "-m", "docs: update readme with usage"); err != nil {
		return "", err
	}

	// Pre-existing branch for TestCreateExistingBranch.
	if err := git("branch", "feat/existing"); err != nil {
		return "", err
	}

	return repo, nil
}

func cloneRepoSeed(t *testing.T, env *HostEnv, seedRepo string, opts RepoOpts) string {
	t.Helper()

	baseDir, err := env.MkdirTemp("ww-test")
	if err != nil {
		t.Fatal(err)
	}

	repo := path.Join(baseDir, opts.Name)
	if _, err := env.Git(baseDir, "clone", seedRepo, repo); err != nil {
		t.Fatalf("git clone: %v", err)
	}
	return repo
}

func setupRepoFromScratch(t *testing.T, env *HostEnv, opts RepoOpts) string {
	t.Helper()

	baseDir, err := env.MkdirTemp("ww-test")
	if err != nil {
		t.Fatal(err)
	}

	repo := path.Join(baseDir, opts.Name)
	if err := env.MkdirAll(repo); err != nil {
		t.Fatal(err)
	}

	git := func(args ...string) {
		t.Helper()
		if _, err := env.Git(repo, args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	writeFile := func(name, content string) {
		t.Helper()
		filePath := path.Join(repo, name)
		if err := env.MkdirAll(path.Dir(filePath)); err != nil {
			t.Fatal(err)
		}
		if err := env.WriteFile(filePath, content); err != nil {
			t.Fatalf("writeFile %s: %v", name, err)
		}
	}

	git("init", "-b", opts.DefaultBranch)

	// Commit 1: initial project structure
	writeFile("README.md", "# My Repo\n\nA test repository for ww integration tests.\n")
	writeFile("go.mod", "module example.com/myrepo\n\ngo 1.23.0\n")
	writeFile("main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")
	git("add", ".")
	git("commit", "-m", "initial: project scaffold")

	// Commit 2: add utility package
	writeFile("internal/util.go", "package internal\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile("internal/util_test.go", "package internal\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")
	git("add", ".")
	git("commit", "-m", "feat: add util package")

	// Commit 3: docs update
	writeFile("README.md", "# My Repo\n\nA test repository for ww integration tests.\n\n## Usage\n\nRun `go run main.go`\n")
	git("add", ".")
	git("commit", "-m", "docs: update readme with usage")

	// Pre-existing branch for TestCreateExistingBranch
	git("branch", "feat/existing")

	return repo
}

// SetupWorkspace creates a workspace root containing NumRepos child git
// repositories, each with a single empty initial commit. This is intended for
// Phase 2 workspace-mode tests (plans 009–011).
func SetupWorkspace(t *testing.T, env *HostEnv, opts WorkspaceOpts) *Workspace {
	t.Helper()

	root, err := env.MkdirTemp("ww-workspace")
	if err != nil {
		t.Fatal(err)
	}

	repoDirs := make([]string, opts.NumRepos)
	for i := 0; i < opts.NumRepos; i++ {
		ro := repoOptAt(opts, i)
		repoDir := path.Join(root, ro.Name)
		if err := env.MkdirAll(repoDir); err != nil {
			t.Fatal(err)
		}
		if _, err := env.Git(repoDir, "init", "-b", ro.DefaultBranch); err != nil {
			t.Fatal(err)
		}
		if _, err := env.Git(repoDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
			t.Fatal(err)
		}
		repoDirs[i] = repoDir
	}

	return &Workspace{RootDir: root, RepoDirs: repoDirs}
}

// SetupNonGitWorkspace creates a workspace where the root directory is NOT a
// git repository but contains NumRepos git child repositories. Intended for
// Phase 2 workspace-mode tests.
func SetupNonGitWorkspace(t *testing.T, env *HostEnv, opts WorkspaceOpts) *Workspace {
	t.Helper()

	root, err := env.MkdirTemp("ww-nongit-workspace")
	if err != nil {
		t.Fatal(err)
	}

	repoDirs := make([]string, opts.NumRepos)
	for i := 0; i < opts.NumRepos; i++ {
		ro := repoOptAt(opts, i)
		repoDir := path.Join(root, ro.Name)
		if err := env.MkdirAll(repoDir); err != nil {
			t.Fatal(err)
		}
		if _, err := env.Git(repoDir, "init", "-b", ro.DefaultBranch); err != nil {
			t.Fatal(err)
		}
		if _, err := env.Git(repoDir, "commit", "--allow-empty", "-m", "initial"); err != nil {
			t.Fatal(err)
		}
		repoDirs[i] = repoDir
	}

	return &Workspace{RootDir: root, RepoDirs: repoDirs}
}

// repoOptAt returns the RepoOpts for index i, using defaults when the slice is
// shorter than i.
func repoOptAt(opts WorkspaceOpts, i int) RepoOpts {
	ro := RepoOpts{}
	if i < len(opts.RepoOpts) {
		ro = opts.RepoOpts[i]
	}
	if ro.Name == "" {
		ro.Name = fmt.Sprintf("repo%d", i+1)
	}
	if ro.DefaultBranch == "" {
		ro.DefaultBranch = "main"
	}
	return ro
}
