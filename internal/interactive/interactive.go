package interactive

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const ttyRequirementMessage = "interactive mode requires a TTY on stdin and stderr; use standard ww commands and see ww --help"

type Action string

const (
	ActionCreate Action = "create"
	ActionList   Action = "list"
	ActionClean  Action = "clean"
	ActionQuit   Action = "quit"
)

type Overview struct {
	Mode  string
	Root  string
	Repos []string
}

type TTYChecker interface {
	IsTerminal(file *os.File) bool
}

type StatTTYChecker struct{}

func (StatTTYChecker) IsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func ValidateTTY(stdin, stderr *os.File, checker TTYChecker) error {
	if checker == nil {
		checker = StatTTYChecker{}
	}
	if checker.IsTerminal(stdin) && checker.IsTerminal(stderr) {
		return nil
	}
	return fmt.Errorf(ttyRequirementMessage)
}

func PromptOutput(_ io.Writer, stderr io.Writer) io.Writer {
	return stderr
}

func WriteOverview(w io.Writer, overview Overview) error {
	if _, err := fmt.Fprintln(w, "Interactive mode"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Mode: %s\n", overview.Mode); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Root: %s\n", overview.Root); err != nil {
		return err
	}
	if len(overview.Repos) > 0 {
		if _, err := fmt.Fprintln(w, "Repos:"); err != nil {
			return err
		}
		for _, repo := range overview.Repos {
			if _, err := fmt.Fprintf(w, "  - %s\n", repo); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprintln(w)
	return err
}

type Session interface {
	SelectAction() (Action, error)
}

type Prompter interface {
	ReadLine(prompt string) (string, error)
}

type LineSession struct {
	in     *bufio.Reader
	output io.Writer
}

func NewLineSession(in io.Reader, output io.Writer) *LineSession {
	return &LineSession{
		in:     bufio.NewReader(in),
		output: output,
	}
}

func (s *LineSession) ReadLine(prompt string) (string, error) {
	if _, err := fmt.Fprint(s.output, prompt); err != nil {
		return "", err
	}

	line, err := s.in.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (s *LineSession) SelectAction() (Action, error) {
	for {
		line, err := s.ReadLine("Select action [create/list/clean/quit]: ")
		if err != nil {
			return "", err
		}

		switch strings.ToLower(strings.TrimSpace(line)) {
		case "create", "c", "1":
			return ActionCreate, nil
		case "list", "l", "2":
			return ActionList, nil
		case "clean", "3":
			return ActionClean, nil
		case "quit", "q", "4":
			return ActionQuit, nil
		default:
			if _, writeErr := fmt.Fprintf(s.output, "Unknown action %q. Choose create, list, clean, or quit.\n", strings.TrimSpace(line)); writeErr != nil {
				return "", writeErr
			}
		}
	}
}

var ErrSessionComplete = errors.New("interactive session complete")

type Flows interface {
	Create() error
	List() error
	Clean() error
}

type PlaceholderFlows struct {
	Output io.Writer
}

func (f PlaceholderFlows) Create() error {
	_, err := fmt.Fprintln(f.Output, "Interactive create flow is not implemented yet. Use `ww create` for now.")
	return err
}

func (f PlaceholderFlows) List() error {
	_, err := fmt.Fprintln(f.Output, "Interactive list flow is not implemented yet. Use `ww list` for now.")
	return err
}

func (f PlaceholderFlows) Clean() error {
	_, err := fmt.Fprintln(f.Output, "Interactive clean flow is not implemented yet. Use `ww clean` for now.")
	return err
}

type Runner struct {
	Prompt  io.Writer
	Session Session
	Flows   Flows
}

func (r Runner) Run(overview Overview) error {
	if err := WriteOverview(r.Prompt, overview); err != nil {
		return err
	}

	for {
		action, err := r.Session.SelectAction()
		if err != nil {
			return err
		}

		switch action {
		case ActionCreate:
			if err := r.Flows.Create(); err != nil {
				if errors.Is(err, ErrSessionComplete) {
					return nil
				}
				return err
			}
		case ActionList:
			if err := r.Flows.List(); err != nil {
				if errors.Is(err, ErrSessionComplete) {
					return nil
				}
				return err
			}
		case ActionClean:
			if err := r.Flows.Clean(); err != nil {
				if errors.Is(err, ErrSessionComplete) {
					return nil
				}
				return err
			}
		case ActionQuit:
			return nil
		default:
			return fmt.Errorf("unknown interactive action %q", action)
		}
	}
}
