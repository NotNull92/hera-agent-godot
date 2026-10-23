package cmd

import (
	"encoding/binary"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/engineproc"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const captureMovieTimeout = 180 * time.Second

// parseCaptureMovieArgs parses:
//
//	capture movie <output.avi|.png|.ogv> [--frames N] [--fps N] [--resolution WxH]
//	              [--scene <res://...|uid://...>] [--rendering-method NAME]
//	              [--project DIR] [--engine PATH]
//
// Movie Maker forces --fixed-fps and needs a real display driver, so --headless
// is never passed.
func parseCaptureMovieArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: capture movie <output.avi|.png|.ogv> [--frames N] [--fps N] [--resolution WxH]")
	}
	output := args[0]
	if !strings.HasPrefix(output, "res://") {
		if abs, err := filepath.Abs(output); err == nil {
			output = abs
		}
	}
	params := map[string]any{"action": "movie", "output": output, "frames": 60, "fps": 30}
	for i := 1; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--frames"):
			value, next, err := takeFlagValue(args, i, "--frames")
			if err != nil {
				return nil, err
			}
			frames, convErr := strconv.Atoi(value)
			if convErr != nil || frames <= 0 {
				return nil, fmt.Errorf("--frames must be a positive integer")
			}
			params["frames"] = frames
			i = next - 1
		case isFlag(args[i], "--fps"):
			value, next, err := takeFlagValue(args, i, "--fps")
			if err != nil {
				return nil, err
			}
			fps, convErr := strconv.Atoi(value)
			if convErr != nil || fps <= 0 {
				return nil, fmt.Errorf("--fps must be a positive integer")
			}
			params["fps"] = fps
			i = next - 1
		case isFlag(args[i], "--resolution"):
			value, next, err := takeFlagValue(args, i, "--resolution")
			if err != nil {
				return nil, err
			}
			if !regexp.MustCompile(`^\d+x\d+$`).MatchString(value) {
				return nil, fmt.Errorf("--resolution must look like WxH")
			}
			params["resolution"] = value
			i = next - 1
		case isFlag(args[i], "--scene"):
			value, next, err := takeFlagValue(args, i, "--scene")
			if err != nil {
				return nil, err
			}
			params["scene"] = value
			i = next - 1
		case isFlag(args[i], "--rendering-method"):
			value, next, err := takeFlagValue(args, i, "--rendering-method")
			if err != nil {
				return nil, err
			}
			params["rendering_method"] = value
			i = next - 1
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

// runCaptureMovie records deterministic frames with Movie Maker. Frame count is
// bounded by --frames (engine --quit-after), so the artifact set is stable.
func runCaptureMovie(params map[string]any) int {
	project, err := resolveProjectDir(stringParam(params, "project"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture movie: %v\n", err)
		return 1
	}
	ref, ctx, cancel, err := runEngineCommand(captureMovieTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture movie: %v\n", err)
		return 1
	}
	defer cancel()

	output := fmt.Sprint(params["output"])
	output = resolveOutputPath(project, output)
	destination := output
	stage, err := stageArtifacts(filepath.Dir(destination))
	if err != nil {
		return failStructured(map[string]any{"error": "artifact_stage_failed", "detail": err.Error()})
	}
	output = filepath.Join(stage, filepath.Base(destination))
	engineArgs := []string{"--path", project}
	if method := stringParam(params, "rendering_method"); method != "" {
		engineArgs = append(engineArgs, "--rendering-method", method)
	}
	if resolution := stringParam(params, "resolution"); resolution != "" {
		engineArgs = append(engineArgs, "--resolution", resolution)
	}
	if scene := stringParam(params, "scene"); scene != "" {
		engineArgs = append(engineArgs, "--scene", scene)
	}
	engineArgs = append(engineArgs,
		"--write-movie", output,
		"--fixed-fps", strconv.Itoa(intValue(params, "fps", 30)),
		"--quit-after", strconv.Itoa(intValue(params, "frames", 60)),
	)
	// Version identity is collected before the workload consumes the deadline.
	version, _ := readEngineVersion(ctx, ref.Path)
	run := engineproc.Run(ctx, ref.Path, engineArgs, project, nil, engineOutputLimit)

	notes := []string{
		"Movie Maker requires a working display driver; --headless uses the dummy renderer and crashed in testing",
		"the recording size follows the project viewport setting; --resolution only requests the window size",
		"the recording is fixed-FPS, so frame N always maps to the same simulated time",
		"artifacts are generated in an isolated empty directory; counts describe this run, not older files left in the destination",
	}
	result := mergeResult(map[string]any{
		"project_path":     filepath.ToSlash(project),
		"output_path":      filepath.ToSlash(destination),
		"frames_requested": intValue(params, "frames", 60), "fps": intValue(params, "fps", 30),
		"engine": engineIdentity(ref, version),
		"notes":  notes,
		"source": "engine_process",
	}, run)
	if stat, statErr := os.Stat(output); statErr == nil && !stat.IsDir() {
		result["output_bytes"] = stat.Size()
		result["output_exists"] = true
	} else {
		result["output_exists"] = false
	}
	if width, height, ok := firstFrameSize(output); ok {
		result["frame_size"] = fmt.Sprintf("%dx%d", width, height)
	}
	if movieFramePatternMeasurable(output) {
		present, scanErr := movieFrameCounts(output)
		if scanErr != nil {
			result["error"] = "sequence_scan_failed"
			result["detail"] = "the frame sequence could not be listed: " + scanErr.Error()
		} else {
			result["frames_present"] = present
			result["frames_written"] = present
			if present == 0 {
				result["error"] = "no_frame_produced"
				result["detail"] = "the engine exited without writing a frame"
			}
		}
	} else {
		info, statErr := os.Stat(output)
		switch {
		case os.IsNotExist(statErr):
			result["error"] = "no_artifact_produced"
			result["detail"] = "the engine exited without writing the recording: " + filepath.ToSlash(output)
		case statErr != nil || !info.Mode().IsRegular():
			if statErr == nil {
				statErr = fmt.Errorf("recording target is not a regular file: %s", output)
			}
			delete(result, "output_exists")
			delete(result, "output_bytes")
			result["error"] = "artifact_stat_failed"
			result["detail"] = "the recording could not be observed: " + statErr.Error()
		}
	}
	if failure := runFailureCode(run); failure != "" && resultError(result) == "" {
		result["error"] = failure
	}
	return finishArtifactPublication(result, run, stage, filepath.Dir(destination), true)
}

// resolveOutputPath maps a res:// output into the project directory so local
// inspection reads the file the engine actually wrote.
func resolveOutputPath(projectDir, output string) string {
	if rest, ok := strings.CutPrefix(output, "res://"); ok {
		return filepath.Join(projectDir, filepath.FromSlash(rest))
	}
	return output
}

// movieFramePattern returns the directory and name pattern of a PNG sequence.
func movieFramePattern(output string) (string, *regexp.Regexp, bool) {
	if !strings.EqualFold(filepath.Ext(output), ".png") {
		return "", nil, false
	}
	dir := filepath.Dir(output)
	base := strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))
	return dir, regexp.MustCompile(`^` + regexp.QuoteMeta(base) + `\d+\.png$`), true
}

// movieFramePatternMeasurable reports whether an output path is a PNG sequence,
// which is the only case in which frames are measured.
func movieFramePatternMeasurable(output string) bool {
	_, _, measurable := movieFramePattern(output)
	return measurable
}

// movieFrameCounts counts this run's frames in its isolated staging directory.
func movieFrameCounts(output string) (int, error) {
	dir, pattern, measurable := movieFramePattern(output)
	if !measurable {
		return 0, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	present := 0
	for _, entry := range entries {
		if !entry.IsDir() && pattern.MatchString(entry.Name()) {
			present++
		}
	}
	return present, nil
}

// firstFrameSize reads the first PNG frame this run produced. Movie Maker sizes
// the recording from the project viewport, so callers need the observed size
// rather than the requested --resolution.
func firstFrameSize(output string) (int, int, bool) {
	dir, pattern, measurable := movieFramePattern(output)
	if !measurable {
		return 0, 0, false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, false
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return 0, 0, false
	}
	sort.Strings(names)
	file, err := os.Open(filepath.Join(dir, names[0]))
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()
	header := make([]byte, 24)
	if _, err := io.ReadFull(file, header); err != nil {
		return 0, 0, false
	}
	if string(header[12:16]) != "IHDR" {
		return 0, 0, false
	}
	width := int(binary.BigEndian.Uint32(header[16:20]))
	height := int(binary.BigEndian.Uint32(header[20:24]))
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

func intValue(params map[string]any, key string, fallback int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return fallback
}
