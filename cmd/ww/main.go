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
	output    io.Writer
	errOutput io.Writer
	json      bool
	dryRun    bool
	sandbox   bool
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
		cdCmd(),
		createCmd(),
		cleanCmd(),
		interactiveCmd(),
		listCmd(),
		removeCmd(),
		versionCmd(),
	}

	fset := pflag.NewFlagSet(mainCmdName, pflag.ContinueOnError)
	fset.SetInterspersed(false)
	showVersion := fset.Bool("version", false, "Print version")
	sandbox := fset.Bool("sandbox", false, "Constrain discovery and worktree defaults to the current sandbox boundary")

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

	glOpts := &globalOpts{output: os.Stdout, errOutput: os.Stderr, sandbox: *sandbox}
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
	return newManagerWithOptions(requireRepo, false)
}

type managerContext struct {
	ws      *workspace.Workspace
	mainDir string
	cfg     *config.Config
}

func newManagerWithOptions(requireRepo bool, sandboxFlag bool) (*worktree.Manager, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	sandboxMode := sandboxFlag
	if !sandboxMode {
		preCfg, err := config.Load(dir)
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
		}
		sandboxMode = preCfg.Sandbox
	}

	ctx, err := loadManagerContext(dir, requireRepo, sandboxMode)
	if err != nil {
		return nil, err
	}

	// sandbox=true discovered only via fallback config paths must still
	// affect workspace/config behavior, so rerun once in sandbox mode.
	if !sandboxMode && ctx.cfg.Sandbox {
		sandboxMode = true
		ctx, err = loadManagerContext(dir, requireRepo, true)
		if err != nil {
			return nil, err
		}
	}
	sandboxMode = sandboxMode || ctx.cfg.Sandbox

	runner := &git.Runner{Dir: dir}

	return &worktree.Manager{
		Git: runner,
		Config: worktree.Config{
			WorktreeDir:    ctx.cfg.WorktreeDir,
			DefaultBase:    ctx.cfg.DefaultBase,
			CopyFiles:      ctx.cfg.CopyFiles,
			SymlinkFiles:   ctx.cfg.SymlinkFiles,
			PostCreateHook: ctx.cfg.PostCreateHook,
			Sandbox:        sandboxMode,
		},
		RepoDir:   ctx.mainDir,
		Workspace: ctx.ws,
	}, nil
}

func loadManagerContext(dir string, requireRepo bool, sandboxMode bool) (*managerContext, error) {
	ws, err := workspace.DetectWithOptions(dir, workspace.DetectOptions{Sandbox: sandboxMode})
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

	cfg, err := config.LoadWithOptions(dir, config.LoadOptions{
		Sandbox:      sandboxMode,
		Boundary:     sandboxBoundary(ws, mainDir),
		FallbackDirs: sandboxFallbackDirs(sandboxMode, mainDir, ws.Root),
	})
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &managerContext{ws: ws, mainDir: mainDir, cfg: cfg}, nil
}

func sandboxBoundary(ws *workspace.Workspace, mainDir string) string {
	if ws != nil && ws.Mode == workspace.ModeWorkspace && ws.Root != "" {
		return ws.Root
	}
	return mainDir
}

func sandboxFallbackDirs(sandbox bool, mainDir, workspaceRoot string) []string {
	if sandbox {
		return []string{mainDir}
	}
	return []string{mainDir, workspaceRoot}
}

func outputJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
