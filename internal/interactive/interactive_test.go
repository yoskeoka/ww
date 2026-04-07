package interactive

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

type stubTTYChecker struct {
	files map[*os.File]bool
}

func (s stubTTYChecker) IsTerminal(file *os.File) bool {
	return s.files[file]
}

func TestValidateTTY(t *testing.T) {
	stdin, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()

	stderr, err := os.CreateTemp(t.TempDir(), "stderr-*")
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()

	t.Run("accepts tty stdin and stderr", func(t *testing.T) {
		err := ValidateTTY(stdin, stderr, stubTTYChecker{
			files: map[*os.File]bool{
				stdin:  true,
				stderr: true,
			},
		})
		if err != nil {
			t.Fatalf("ValidateTTY() error = %v, want nil", err)
		}
	})

	t.Run("rejects non tty stdin", func(t *testing.T) {
		err := ValidateTTY(stdin, stderr, stubTTYChecker{
			files: map[*os.File]bool{
				stdin:  false,
				stderr: true,
			},
		})
		if err == nil || err.Error() != ttyRequirementMessage {
			t.Fatalf("ValidateTTY() error = %v, want %q", err, ttyRequirementMessage)
		}
	})

	t.Run("rejects non tty stderr", func(t *testing.T) {
		err := ValidateTTY(stdin, stderr, stubTTYChecker{
			files: map[*os.File]bool{
				stdin:  true,
				stderr: false,
			},
		})
		if err == nil || err.Error() != ttyRequirementMessage {
			t.Fatalf("ValidateTTY() error = %v, want %q", err, ttyRequirementMessage)
		}
	})
}

func TestStatTTYCheckerRejectsCharacterDeviceThatIsNotTTY(t *testing.T) {
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	checker := StatTTYChecker{}
	if checker.IsTerminal(file) {
		t.Fatalf("StatTTYChecker should reject %s as non-TTY", os.DevNull)
	}
}

func TestPromptOutputUsesStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	got := PromptOutput(&stdout, &stderr)
	if got != &stderr {
		t.Fatalf("PromptOutput() did not return stderr writer")
	}
}

type fakeSession struct {
	actions []Action
	index   int
}

func (f *fakeSession) SelectAction() (Action, error) {
	action := f.actions[f.index]
	f.index++
	return action, nil
}

type fakeFlows struct {
	createCalls int
	listCalls   int
	cleanCalls  int
}

func (f *fakeFlows) Create() error {
	f.createCalls++
	return nil
}

func (f *fakeFlows) List() error {
	f.listCalls++
	return nil
}

func (f *fakeFlows) Clean() error {
	f.cleanCalls++
	return nil
}

func TestRunnerDispatchesActionsAndPrintsOverviewOnce(t *testing.T) {
	var prompt bytes.Buffer
	session := &fakeSession{
		actions: []Action{ActionCreate, ActionList, ActionClean, ActionQuit},
	}
	flows := &fakeFlows{}

	err := Runner{
		Prompt:  &prompt,
		Session: session,
		Flows:   flows,
	}.Run(Overview{
		Mode: "workspace",
		Root: "/tmp/ws",
		Repos: []string{
			"repo1",
			"repo2",
		},
	})
	if err != nil {
		t.Fatalf("Runner.Run() error = %v", err)
	}

	if flows.createCalls != 1 || flows.listCalls != 1 || flows.cleanCalls != 1 {
		t.Fatalf("flow calls = create:%d list:%d clean:%d, want 1 each", flows.createCalls, flows.listCalls, flows.cleanCalls)
	}

	output := prompt.String()
	if strings.Count(output, "Interactive mode") != 1 {
		t.Fatalf("overview should be printed once, got output:\n%s", output)
	}
	if !strings.Contains(output, "Mode: workspace") || !strings.Contains(output, "Root: /tmp/ws") {
		t.Fatalf("overview missing expected fields:\n%s", output)
	}
	if !strings.Contains(output, "  - repo1") || !strings.Contains(output, "  - repo2") {
		t.Fatalf("overview missing repo list:\n%s", output)
	}
}
