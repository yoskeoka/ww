// check-performance-budget evaluates the command-boundary ww benchmarks
// against the version-controlled performance budget.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const nanosecondsPerMillisecond = 1_000_000

type budgetFile struct {
	FixtureProfile   fixtureProfile             `json:"fixture_profile"`
	BenchmarkCommand []string                   `json:"benchmark_command"`
	Repetitions      int                        `json:"repetitions"`
	Benchmarks       map[string]benchmarkBudget `json:"benchmarks"`
}

type fixtureProfile struct {
	Repos                     int `json:"repos"`
	WorktreesPerRepo          int `json:"worktrees_per_repo"`
	CleanableWorktreesPerRepo int `json:"cleanable_worktrees_per_repo"`
}

type benchmarkBudget struct {
	MaxMSPerOp float64 `json:"max_ms_per_op"`
}

var benchmarkLine = regexp.MustCompile(`(?m)^(Benchmark[[:alnum:]_/]+)(?:-[0-9]+)?\s+\d+\s+([0-9]+)\s+ns/op`)

type commandRunner func(context.Context, []string) (string, error)

func main() {
	budgetPath := flag.String("budget", "tools/performance-budget.json", "path to the performance budget JSON")
	flag.Parse()
	if err := checkBudget(context.Background(), *budgetPath, runCommand); err != nil {
		fmt.Fprintln(os.Stderr, "performance budget check:", err)
		os.Exit(1)
	}
}

func checkBudget(ctx context.Context, budgetPath string, runner commandRunner) error {
	data, err := os.ReadFile(budgetPath)
	if err != nil {
		return fmt.Errorf("read budget: %w", err)
	}
	var budget budgetFile
	if err := json.Unmarshal(data, &budget); err != nil {
		return fmt.Errorf("parse budget: %w", err)
	}
	if err := validateBudget(budget); err != nil {
		return err
	}

	samples := make(map[string][]int64, len(budget.Benchmarks))
	for run := 1; run <= budget.Repetitions; run++ {
		out, err := runner(ctx, budget.BenchmarkCommand)
		if err != nil {
			if strings.Contains(out, "performance fixture:") {
				return fmt.Errorf("fixture failed on run %d: %w\n%s", run, err, out)
			}
			return fmt.Errorf("benchmark command failed on run %d: %w\n%s", run, err, out)
		}
		observed, err := parseBenchmarkOutput(out, budget.Benchmarks)
		if err != nil {
			return fmt.Errorf("benchmark output invalid on run %d: %w", run, err)
		}
		for name, value := range observed {
			samples[name] = append(samples[name], value)
		}
	}

	var exceeded []string
	for name, limit := range budget.Benchmarks {
		observed := median(samples[name])
		if observed > limit.maxNSPerOp() {
			exceeded = append(exceeded, fmt.Sprintf("%s observed %s/op (median of %s; budget %s/op; reproduce: %s", name, formatMS(observed), formatSamplesMS(samples[name]), formatMS(limit.maxNSPerOp()), commandString(budget.BenchmarkCommand)))
		}
	}
	if len(exceeded) > 0 {
		sort.Strings(exceeded)
		return errors.New("budget exceeded:\n" + strings.Join(exceeded, "\n"))
	}
	benchmarkNames := make([]string, 0, len(budget.Benchmarks))
	for name := range budget.Benchmarks {
		benchmarkNames = append(benchmarkNames, name)
	}
	sort.Strings(benchmarkNames)
	for _, name := range benchmarkNames {
		limit := budget.Benchmarks[name]
		fmt.Printf("%s: median %s/op (samples %s; budget %s/op)\n", name, formatMS(median(samples[name])), formatSamplesMS(samples[name]), formatMS(limit.maxNSPerOp()))
	}
	fmt.Printf("performance budgets passed after %d runs (fixture: %d repos, %d worktrees/repo, %d cleanable worktrees/repo)\n", budget.Repetitions, budget.FixtureProfile.Repos, budget.FixtureProfile.WorktreesPerRepo, budget.FixtureProfile.CleanableWorktreesPerRepo)
	return nil
}

func validateBudget(budget budgetFile) error {
	if budget.FixtureProfile.Repos < 1 || budget.FixtureProfile.WorktreesPerRepo < 2 {
		return errors.New("fixture_profile requires at least 1 repository and 2 worktrees per repository")
	}
	if budget.FixtureProfile.CleanableWorktreesPerRepo < 1 || budget.FixtureProfile.CleanableWorktreesPerRepo >= budget.FixtureProfile.WorktreesPerRepo {
		return fmt.Errorf("fixture_profile cleanable_worktrees_per_repo must be between 1 and %d", budget.FixtureProfile.WorktreesPerRepo-1)
	}
	if len(budget.BenchmarkCommand) == 0 {
		return errors.New("benchmark_command is required")
	}
	if budget.Repetitions < 1 {
		return errors.New("repetitions must be positive")
	}
	if len(budget.Benchmarks) == 0 {
		return errors.New("at least one benchmark budget is required")
	}
	for name, value := range budget.Benchmarks {
		if value.MaxMSPerOp <= 0 {
			return fmt.Errorf("%s max_ms_per_op must be positive", name)
		}
	}
	return nil
}

func (b benchmarkBudget) maxNSPerOp() int64 {
	return int64(b.MaxMSPerOp * nanosecondsPerMillisecond)
}

func formatMS(ns int64) string {
	return fmt.Sprintf("%.1f ms", float64(ns)/nanosecondsPerMillisecond)
}

func formatSamplesMS(samples []int64) string {
	formatted := make([]string, len(samples))
	for i, sample := range samples {
		formatted[i] = formatMS(sample)
	}
	return "[" + strings.Join(formatted, ", ") + "]"
}

func parseBenchmarkOutput(out string, expected map[string]benchmarkBudget) (map[string]int64, error) {
	observed := make(map[string]int64, len(expected))
	for _, match := range benchmarkLine.FindAllStringSubmatch(out, -1) {
		if _, wanted := expected[match[1]]; !wanted {
			continue
		}
		value, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", match[1], err)
		}
		observed[match[1]] = value
	}
	for name := range expected {
		if _, ok := observed[name]; !ok {
			return nil, fmt.Errorf("missing %s result", name)
		}
	}
	return observed, nil
}

func median(values []int64) int64 {
	ordered := append([]int64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	return ordered[len(ordered)/2]
}

func commandString(command []string) string {
	quoted := make([]string, len(command))
	for i, arg := range command {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(arg string) string {
	if arg != "" && strings.IndexFunc(arg, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_@%+=:,./-", r))
	}) == -1 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
}

func runCommand(ctx context.Context, command []string) (string, error) {
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
