package cmd

import (
	"encoding/binary"
	"fmt"
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
	// The sequence state is captured before the run, so this run's frames are
	// identified by change rather than by a timestamp comparison.
	before := snapshotMovieFrames(output)
	run := runEngineBounded(ctx, ref.Path, engineArgs, project, nil, engineOutputLimit)

	notes := []string{
		"Movie Maker requires a working display driver; --headless uses the dummy renderer and crashed in testing",
		"the recording size follows the project viewport setting; --resolution only requests the window size",
		"the recording is fixed-FPS, so frame N always maps to the same simulated time",
		"frames_written counts only files this invocation created or rewrote; unchanged files with the same prefix are reported as frames_present",
	}
	result := mergeResult(map[string]any{
		"project_path":     filepath.ToSlash(project),
		"output_path":      filepath.ToSlash(output),
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
	if width, height, ok := firstFrameSize(output, before); ok {
		result["frame_size"] = fmt.Sprintf("%dx%d", width, height)
	}
	if written, present, ok := movieFrameCounts(output, before); ok {
		result["frames_written"], result["frames_present"] = written, present
		if present > written {
			notes = append(notes, fmt.Sprintf("stale_frames_present: %d frame file(s) with the same prefix are unchanged since before this run", present-written))
		}
		if written == 0 {
			// A clean exit with no frame of its own is not a usable recording.
			result["error"] = "no_frame_produced"
			notes = append(notes, "no_frame_produced: the engine exited without writing a frame for this run")
		}
		result["notes"] = notes
	} else if result["output_exists"] != true {
		// Containers are written once; a clean exit without the file is a failure.
		result["error"] = "no_artifact_produced"
		result["detail"] = "the engine exited without writing the recording: " + filepath.ToSlash(output)
	}
	if failure := runFailureCode(run); failure != "" && resultError(result) == "" {
		result["error"] = failure
	}
	return finishEngineResult(result, run)
}

// resolveOutputPath maps a res:// output into the project directory so local
// inspection reads the file the engine actually wrote.
func resolveOutputPath(projectDir, output string) string {
	if rest, ok := strings.CutPrefix(output, "res://"); ok {
		return filepath.Join(projectDir, filepath.FromSlash(rest))
	}
	return output
}

// frameSnapshot is the PNG sequence state before a run, so this run's output can
// be told from an older file with the same name without trusting timestamp
// granularity (coarse filesystems can report a fresh write in the past).
type frameSnapshot map[string]frameStamp

// frameStamp is one frame's size and modification time.
type frameStamp struct {
	bytes int64
	mod   time.Time
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

// snapshotMovieFrames records the PNG sequence state before a run.
func snapshotMovieFrames(output string) frameSnapshot {
	before := frameSnapshot{}
	dir, pattern, measurable := movieFramePattern(output)
	if !measurable {
		return before
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return before
	}
	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			before[entry.Name()] = frameStamp{bytes: info.Size(), mod: info.ModTime()}
		}
	}
	return before
}

// changed reports whether a frame is this run's output: it is new, or its size or
// modification time differs from the snapshot.
func (s frameSnapshot) changed(name string, info os.FileInfo) bool {
	stamp, seen := s[name]
	if !seen {
		return true
	}
	return stamp.bytes != info.Size() || !stamp.mod.Equal(info.ModTime())
}

// movieFrameCounts separates frames this run produced from older files with the
// same prefix. ok is false for non-PNG containers, where no frame is measured.
func movieFrameCounts(output string, before frameSnapshot) (written int, present int, ok bool) {
	dir, pattern, measurable := movieFramePattern(output)
	if !measurable {
		return 0, 0, false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, false
	}
	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		present++
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		if before.changed(entry.Name(), info) {
			written++
		}
	}
	return written, present, true
}

// firstFrameSize reads the first PNG frame this run produced. Movie Maker sizes
// the recording from the project viewport, so callers need the observed size
// rather than the requested --resolution.
func firstFrameSize(output string, before frameSnapshot) (int, int, bool) {
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
		info, infoErr := entry.Info()
		if infoErr != nil || !before.changed(entry.Name(), info) {
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
