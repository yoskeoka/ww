package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"

	"github.com/yoskeoka/ww/git"
	"github.com/yoskeoka/ww/internal/config"
	"github.com/yoskeoka/ww/workspace"
	"github.com/yoskeoka/ww/worktree"
)

// errHelp is returned when --help is requested. Not an error — just signals clean exit.
var errHelp = errors.New("")

func main() {
	os.Exit(cliMain())
}

var mainCmdName = "ww"

type globalOpts struct {
	output io.Writer
	json   bool
	dryRun bool
}

type command struct {
	name        string
	description string
	subcommands []command
	fset        *pflag.FlagSet
	fn          func(args []string, gOpts *globalOpts) error
}

func cliMain() int {
	commands := []command{
		createCmd(),
		cleanCmd(),
		listCmd(),
		removeCmd(),
		versionCmd(),
	}

	fset := pflag.NewFlagSet(mainCmdName, pflag.ContinueOnError)
	fset.SetInterspersed(false)
	showVersion := fset.Bool("version", false, "Print version")

	fset.Usage = func() {
		fmt.Fprintf(fset.Output(), "Usage: %s <command> [flags]\n\n", mainCmdName)
		fmt.Fprintln(fset.Output(), "Commands:")
		printCommands(fset.Output(), commands)
		fmt.Fprintln(fset.Output())
		fmt.Fprintln(fset.Output(), "Flags:")
		fset.PrintDefaults()
	}

	if err := fset.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return 0
		}
		return 1
	}

	if *showVersion {
		printVersion(os.Stdout)
		return 0
	}

	args := fset.Args()
	if len(args) == 0 {
		fset.Usage()
		return 1
	}

	glOpts := &globalOpts{output: os.Stdout}
	if err := runSubcmd(mainCmdName, commands, args, glOpts); err != nil {
		if errors.Is(err, errHelp) {
			return 0
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func printCommands(w io.Writer, commands []command) {
	for _, cmd := range commands {
		fmt.Fprintf(w, "  %-14s%s\n", cmd.name, cmd.description)
	}
}

func runSubcmd(parentCmd string, subCommands []command, args []string, glOpts *globalOpts) error {
	name := args[0]
	for _, cmd := range subCommands {
		if cmd.name != name {
			continue
		}
		if len(cmd.subcommands) > 0 && len(args) > 1 {
			return runSubcmd(cmd.name, cmd.subcommands, args[1:], glOpts)
		}
		return cmd.fn(args[1:], glOpts)
	}
	return fmt.Errorf("unknown command: '%s' for '%s'", name, parentCmd)
}

func newManager(requireRepo bool) (*worktree.Manager, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	ws, err := workspace.Detect(dir)
	if err != nil {
		return nil, err
	}

	runner := &git.Runner{Dir: dir}
	mainDir, err := runner.MainWorktreeDir()
	if err != nil {
		if ws.Mode == workspace.ModeWorkspace && ws.Root == dir && !requireRepo {
			mainDir = ws.Root
		} else if ws.Mode == workspace.ModeWorkspace && ws.Root == dir {
			return nil, fmt.Errorf("repo selection is not supported from a non-git workspace root")
		} else {
			return nil, fmt.Errorf("not a git repository: %w", err)
		}
	}

	cfg, err := config.Load(dir, mainDir, ws.Root)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	return &worktree.Manager{
		Git: runner,
		Config: worktree.Config{
			WorktreeDir:    cfg.WorktreeDir,
			DefaultBase:    cfg.DefaultBase,
			CopyFiles:      cfg.CopyFiles,
			SymlinkFiles:   cfg.SymlinkFiles,
			PostCreateHook: cfg.PostCreateHook,
		},
		RepoDir:   mainDir,
		Workspace: ws,
	}, nil
}

func managerForRepo(base *worktree.Manager, repoName string) (*worktree.Manager, error) {
	if base.Workspace == nil || base.Workspace.Mode != workspace.ModeWorkspace {
		return base, nil
	}

	for _, repo := range base.Workspace.Repos {
		if repo.Name != repoName {
			continue
		}
		return &worktree.Manager{
			Git: &git.Runner{Dir: repo.Path},
			Config: worktree.Config{
				WorktreeDir:    base.Config.WorktreeDir,
				DefaultBase:    base.Config.DefaultBase,
				CopyFiles:      base.Config.CopyFiles,
				SymlinkFiles:   base.Config.SymlinkFiles,
				PostCreateHook: base.Config.PostCreateHook,
			},
			RepoDir:   repo.Path,
			Workspace: base.Workspace,
		}, nil
	}

	return nil, fmt.Errorf("repo %q not found in workspace", repoName)
}

func outputJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func printVersion(w io.Writer) {
	hash := CommitHash
	if hash == "" {
		hash = "dev"
	}
	fmt.Fprintf(w, "ww version %s\n", hash)
}
