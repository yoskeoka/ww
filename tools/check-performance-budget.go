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

type budgetFile struct {
	FixtureProfile   fixtureProfile             `json:"fixture_profile"`
	BenchmarkCommand []string                   `json:"benchmark_command"`
	Repetitions      int                        `json:"repetitions"`
	Benchmarks       map[string]benchmarkBudget `json:"benchmarks"`
}

type fixtureProfile struct {
	Repos            int `json:"repos"`
	WorktreesPerRepo int `json:"worktrees_per_repo"`
}

type benchmarkBudget struct {
	MaxNSPerOp int64 `json:"max_ns_per_op"`
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
			return fmt.Errorf("benchmark evaluator failed on run %d: %w\n%s", run, err, out)
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
		if observed > limit.MaxNSPerOp {
			exceeded = append(exceeded, fmt.Sprintf("%s observed %d ns/op (median of %v), budget %d ns/op; reproduce: %s", name, observed, samples[name], limit.MaxNSPerOp, commandString(budget.BenchmarkCommand)))
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
		fmt.Printf("%s: median %d ns/op (samples %v; budget %d ns/op)\n", name, median(samples[name]), samples[name], budget.Benchmarks[name].MaxNSPerOp)
	}
	fmt.Printf("performance budgets passed after %d runs (fixture: %d repos, %d worktrees/repo)\n", budget.Repetitions, budget.FixtureProfile.Repos, budget.FixtureProfile.WorktreesPerRepo)
	return nil
}

func validateBudget(budget budgetFile) error {
	if budget.FixtureProfile.Repos < 1 || budget.FixtureProfile.WorktreesPerRepo < 2 {
		return errors.New("fixture_profile requires at least 1 repository and 2 worktrees per repository")
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
		if value.MaxNSPerOp < 1 {
			return fmt.Errorf("%s max_ns_per_op must be positive", name)
		}
	}
	return nil
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
	return strings.Join(command, " ")
}

func runCommand(ctx context.Context, command []string) (string, error) {
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
