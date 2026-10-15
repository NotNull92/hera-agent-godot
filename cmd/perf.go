package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

const perfSampleTimeout = 30 * time.Second

// parsePerfSampleArgs parses:
//
//	perf sample [--top N] [--scene <res://...|uid://...>] [--headless]
//	            [--project DIR] [--engine PATH]
func parsePerfSampleArgs(args []string) (map[string]any, error) {
	params := map[string]any{"top": 10, "headless": false}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--top"):
			value, next, err := takeFlagValue(args, i, "--top")
			if err != nil {
				return nil, err
			}
			top, convErr := strconv.Atoi(value)
			if convErr != nil || top <= 0 {
				return nil, fmt.Errorf("--top must be a positive integer")
			}
			params["top"] = top
			i = next - 1
		case isFlag(args[i], "--scene"):
			value, next, err := takeFlagValue(args, i, "--scene")
			if err != nil {
				return nil, err
			}
			params["scene"] = value
			i = next - 1
		case args[i] == "--headless":
			params["headless"] = true
		case isFlag(args[i], "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return nil, err
			}
			params["project"] = value
			i = next - 1
		case isFlag(args[i], "--engine"):
			value, next, err := takeFlagValue(args, i, "--engine")
			if err != nil {
				return nil, err
			}
			engineFlagPath = value
			i = next - 1
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

// runPerfSample runs the engine with --benchmark-file and reports the phases
// that cost the most. Only the JSON artifact is parsed; stdout text is not.
//
// Godot 4.7.2 ignores --quit-after while either benchmark flag is set, so the
// run is stopped by the timeout and the written JSON is read afterwards.
func runPerfSample(params map[string]any) int {
	project, err := resolveProjectDir(stringParam(params, "project"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "perf sample: %v\n", err)
		return 1
	}
	ref, ctx, cancel, err := runEngineCommand(perfSampleTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "perf sample: %v\n", err)
		return 1
	}
	defer cancel()

	benchFile, err := os.CreateTemp("", "hera-bench-*.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "perf sample: %v\n", err)
		return 1
	}
	benchPath := benchFile.Name()
	benchFile.Close()
	defer os.Remove(benchPath)

	headless, _ := params["headless"].(bool)
	engineArgs := []string{}
	if headless {
		engineArgs = append(engineArgs, "--headless")
	}
	engineArgs = append(engineArgs, "--path", project)
	if scene := stringParam(params, "scene"); scene != "" {
		engineArgs = append(engineArgs, "--scene", scene)
	}
	engineArgs = append(engineArgs, "--benchmark", "--benchmark-file", benchPath)
	// Version identity is collected before the workload consumes the deadline.
	version, _ := readEngineVersion(ctx, ref.Path)
	run := runEngineBounded(ctx, ref.Path, engineArgs, project, nil, engineOutputLimit)

	boundedBy := "engine_exit"
	if run.TimedOut {
		boundedBy = "timeout"
	}
	result := mergeResult(map[string]any{
		"project_path": filepath.ToSlash(project), "headless": headless,
		"bounded_by": boundedBy,
		"engine":     engineIdentity(ref, version),
		"notes": []string{
			"Godot 4.7.2 ignores --quit-after while benchmark flags are set; the run stops at the timeout and the JSON is read afterwards",
			"phases cover engine startup and system setup, which is what a regression baseline compares",
			"exit code 0 means the benchmark JSON was read, not that the examined game run succeeded",
		},
		"source": "engine_process",
	}, run)
	benchmark, benchErr := readBenchmarkFile(benchPath)
	if benchErr != nil {
		result["benchmark_error"] = benchErr.Error()
	} else {
		top := intValue(params, "top", 10)
		entries := sortedBenchmarkEntries(benchmark)
		total := 0.0
		for _, value := range benchmark {
			total += value
		}
		shown := entries
		if len(shown) > top {
			shown = shown[:top]
		}
		result["benchmark"] = map[string]any{
			"entries": len(entries), "total_seconds": total, "top": shown,
		}
	}
	switch {
	case benchErr != nil:
		result["error"] = "benchmark_file_unavailable"
		result["benchmark_error"] = benchErr.Error()
	default:
		if failure := runFailureCode(run); failure != "" && !run.TimedOut {
			result["engine_error"] = failure
		}
	}
	if benchErr != nil {
		return failEngineResult(result)
	}
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	return 0
}

// readBenchmarkFile reads the engine's benchmark JSON, which maps phase names to
// seconds. Non-numeric entries are ignored instead of failing the whole read.
func readBenchmarkFile(path string) (map[string]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	values := map[string]float64{}
	for name, value := range raw {
		if seconds, ok := value.(float64); ok {
			values[name] = seconds
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("benchmark file had no numeric phases")
	}
	return values, nil
}

func runPerf(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: perf sample [--headless] [--top N]")
		return 2
	}
	switch args[0] {
	case "sample":
		params, err := parsePerfSampleArgs(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "perf sample: %v\n", err)
			return 2
		}
		return runPerfSample(params)
	default:
		fmt.Fprintf(os.Stderr, "unknown perf subcommand %q (want sample)\n", args[0])
		return 2
	}
}

func runCapture(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: capture movie <output.avi|.png|.ogv> [--frames N] [--fps N]")
		return 2
	}
	switch args[0] {
	case "movie":
		params, err := parseCaptureMovieArgs(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "capture movie: %v\n", err)
			return 2
		}
		return runCaptureMovie(params)
	default:
		fmt.Fprintf(os.Stderr, "unknown capture subcommand %q (want movie)\n", args[0])
		return 2
	}
}
