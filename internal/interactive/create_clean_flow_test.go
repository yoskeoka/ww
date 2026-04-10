package interactive

import (
	"bytes"
	"fmt"
	"testing"
)

type fakeCreateUI struct {
	repo           string
	branch         string
	confirmed      bool
	repoCalls      int
	branchRepo     string
	previewSeen    CreatePreview
	repoErr        error
	branchErr      error
	confirmErr     error
	availableRepos []RepoOption
}

func (f *fakeCreateUI) SelectCreateRepo(repos []RepoOption) (string, error) {
	f.repoCalls++
	f.availableRepos = append([]RepoOption(nil), repos...)
	return f.repo, f.repoErr
}

func (f *fakeCreateUI) InputCreateBranch(repo string) (string, error) {
	f.branchRepo = repo
	return f.branch, f.branchErr
}

func (f *fakeCreateUI) ConfirmCreate(preview CreatePreview) (bool, error) {
	f.previewSeen = preview
	return f.confirmed, f.confirmErr
}

type fakeCleanUI struct {
	mode        CleanMode
	confirmed   bool
	summarySeen []CleanSummary
	targetsSeen []CleanTarget
	modeErr     error
	confirmErr  error
}

func (f *fakeCleanUI) SelectCleanMode(summary []CleanSummary) (CleanMode, error) {
	f.summarySeen = append([]CleanSummary(nil), summary...)
	return f.mode, f.modeErr
}

func (f *fakeCleanUI) ConfirmClean(mode CleanMode, targets []CleanTarget) (bool, error) {
	if mode != f.mode {
		return false, fmt.Errorf("ConfirmClean mode = %q, want %q", mode, f.mode)
	}
	f.targetsSeen = append([]CleanTarget(nil), targets...)
	return f.confirmed, f.confirmErr
}

func TestCreateFlowWorkspaceSelectsRepoBuildsPreviewAndExecutes(t *testing.T) {
	ui := &fakeCreateUI{
		repo:      "repo2",
		branch:    "feat/interactive",
		confirmed: true,
	}
	execCalls := 0
	flow := CreateFlow{
		UI:            ui,
		WorkspaceMode: true,
		Repos: []RepoOption{
			{Name: "repo1"},
			{Name: "repo2"},
		},
		BuildPreview: func(repo, branch string) (CreatePreview, error) {
			if repo != "repo2" || branch != "feat/interactive" {
				t.Fatalf("BuildPreview(%q, %q), want repo2 / feat/interactive", repo, branch)
			}
			return CreatePreview{Repo: repo, Branch: branch, Path: "/tmp/repo2@feat-interactive"}, nil
		},
		Execute: func(repo, branch string) error {
			execCalls++
			if repo != "repo2" || branch != "feat/interactive" {
				t.Fatalf("Execute(%q, %q), want repo2 / feat/interactive", repo, branch)
			}
			return nil
		},
	}

	if err := flow.Run(); err != nil {
		t.Fatalf("CreateFlow.Run() error = %v", err)
	}
	if ui.repoCalls != 1 {
		t.Fatalf("repoCalls = %d, want 1", ui.repoCalls)
	}
	if ui.branchRepo != "repo2" {
		t.Fatalf("branchRepo = %q, want repo2", ui.branchRepo)
	}
	if execCalls != 1 {
		t.Fatalf("execCalls = %d, want 1", execCalls)
	}
	if ui.previewSeen.Path != "/tmp/repo2@feat-interactive" {
		t.Fatalf("previewSeen.Path = %q, want /tmp/repo2@feat-interactive", ui.previewSeen.Path)
	}
}

func TestCreateFlowSingleRepoSkipsRepoPrompt(t *testing.T) {
	ui := &fakeCreateUI{
		branch:    "feat/local",
		confirmed: false,
	}
	flow := CreateFlow{
		UI:            ui,
		WorkspaceMode: false,
		BuildPreview: func(repo, branch string) (CreatePreview, error) {
			if repo != "" {
				t.Fatalf("single-repo preview repo = %q, want empty", repo)
			}
			return CreatePreview{Branch: branch, Path: "/tmp/repo@feat-local"}, nil
		},
		Execute: func(repo, branch string) error {
			t.Fatal("Execute should not be called when confirmation is declined")
			return nil
		},
	}

	if err := flow.Run(); err != nil {
		t.Fatalf("CreateFlow.Run() error = %v", err)
	}
	if ui.repoCalls != 0 {
		t.Fatalf("repoCalls = %d, want 0", ui.repoCalls)
	}
}

func TestBuildCleanSummaryPreservesZeroCountRepos(t *testing.T) {
	summary := BuildCleanSummary(
		[]string{"repo1", "repo2", "repo3"},
		[]CleanTarget{
			{Repo: "repo1", Branch: "feat/a"},
			{Repo: "repo1", Branch: "feat/b"},
			{Repo: "repo3", Branch: "feat/c"},
		},
	)

	want := []CleanSummary{
		{Repo: "repo1", Count: 2},
		{Repo: "repo2", Count: 0},
		{Repo: "repo3", Count: 1},
	}
	if len(summary) != len(want) {
		t.Fatalf("summary len = %d, want %d", len(summary), len(want))
	}
	for i := range want {
		if summary[i] != want[i] {
			t.Fatalf("summary[%d] = %+v, want %+v", i, summary[i], want[i])
		}
	}
}

func TestCleanFlowNoTargetsPrintsMessage(t *testing.T) {
	var output bytes.Buffer
	ui := &fakeCleanUI{}
	flow := CleanFlow{
		UI:        ui,
		Output:    &output,
		RepoNames: []string{"repo1"},
		Load: func() (CleanSnapshot, error) {
			return CleanSnapshot{}, nil
		},
		Execute: func(mode CleanMode, snapshot CleanSnapshot) error {
			t.Fatal("Execute should not be called with no targets")
			return nil
		},
	}

	if err := flow.Run(); err != nil {
		t.Fatalf("CleanFlow.Run() error = %v", err)
	}
	if got := output.String(); got != "No cleanable worktrees found.\n" {
		t.Fatalf("output = %q, want no-cleanable message", got)
	}
}

func TestCleanFlowForceModeExecutesAfterConfirmation(t *testing.T) {
	ui := &fakeCleanUI{
		mode:      CleanModeForce,
		confirmed: true,
	}
	execCalls := 0
	targets := []CleanTarget{
		{Repo: "repo1", Branch: "feat/merged", Status: "merged", Path: "/tmp/repo1@feat-merged"},
	}
	flow := CleanFlow{
		UI:        ui,
		Output:    &bytes.Buffer{},
		RepoNames: []string{"repo1", "repo2"},
		Load: func() (CleanSnapshot, error) {
			return CleanSnapshot{
				Targets: targets,
				State:   "confirmed-snapshot",
			}, nil
		},
		Execute: func(mode CleanMode, snapshot CleanSnapshot) error {
			execCalls++
			if mode != CleanModeForce {
				t.Fatalf("Execute mode = %q, want force", mode)
			}
			if snapshot.State != "confirmed-snapshot" {
				t.Fatalf("snapshot.State = %#v, want confirmed-snapshot", snapshot.State)
			}
			return nil
		},
	}

	if err := flow.Run(); err != nil {
		t.Fatalf("CleanFlow.Run() error = %v", err)
	}
	if execCalls != 1 {
		t.Fatalf("execCalls = %d, want 1", execCalls)
	}
	if len(ui.summarySeen) != 2 || ui.summarySeen[1] != (CleanSummary{Repo: "repo2", Count: 0}) {
		t.Fatalf("summarySeen = %+v, want repo2 zero-count preserved", ui.summarySeen)
	}
	if len(ui.targetsSeen) != 1 || ui.targetsSeen[0].Branch != "feat/merged" {
		t.Fatalf("targetsSeen = %+v, want feat/merged target", ui.targetsSeen)
	}
}

func TestValidateInteractiveBranch(t *testing.T) {
	if err := validateInteractiveBranch(" feat/valid "); err != nil {
		t.Fatalf("validateInteractiveBranch(valid) error = %v", err)
	}
	if err := validateInteractiveBranch(" "); err == nil {
		t.Fatal("validateInteractiveBranch(empty) should fail")
	}
	if err := validateInteractiveBranch("bad branch"); err == nil {
		t.Fatal("validateInteractiveBranch(space) should fail")
	}
}
