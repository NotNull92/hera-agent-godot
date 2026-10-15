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
	// identified by change rather than by a timestamp comparison. A failed
	// observation is recorded and reported as inconclusive, never as freshness.
	before, beforeErr := snapshotMovieFrames(output)
	containerBefore, containerSeen, containerBeforeErr := statFileStamp(output)
	run := runEngineBounded(ctx, ref.Path, engineArgs, project, nil, engineOutputLimit)

	notes := []string{
		"Movie Maker requires a working display driver; --headless uses the dummy renderer and crashed in testing",
		"the recording size follows the project viewport setting; --resolution only requests the window size",
		"the recording is fixed-FPS, so frame N always maps to the same simulated time",
		"frames_written counts frames whose size or modification time changed since before this run; frames whose metadata did not change are counted in frames_unchanged and may be older files or same-metadata rewrites",
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
	if movieFramePatternMeasurable(output) {
		written, present, scanErr := movieFrameCounts(output, before)
		verdict := classifySequence(written, present, scanErr, beforeErr)
		if verdict.ErrorCode != "" {
			result["error"] = verdict.ErrorCode
			if verdict.Detail != "" {
				result["detail"] = verdict.Detail
			}
		}
		if scanErr == nil {
			result["frames_present"] = present
		}
		if verdict.ClaimCounts {
			result["frames_written"] = written
			if written < present {
				// Frames whose metadata did not change may be older files or rewrites
				// that this observation cannot tell apart.
				result["frames_unchanged"] = present - written
			}
		}
		if verdict.Inconclusive {
			result["frames_verification"] = "inconclusive"
			if verdict.Detail != "" {
				result["frames_verification_detail"] = verdict.Detail
			}
		}
		if verdict.Note != "" {
			notes = append(notes, verdict.Note)
			result["notes"] = notes
		}
	} else {
		after, afterSeen, afterErr := statFileStamp(output)
		code, inconclusive := containerVerification(containerBefore, containerSeen, after, afterSeen, containerBeforeErr, afterErr)
		switch {
		case afterErr != nil:
			delete(result, "output_exists")
			delete(result, "output_bytes")
			result["error"] = code
			result["detail"] = "the recording could not be observed: " + afterErr.Error()
		case code != "":
			result["error"] = code
			result["detail"] = "the engine exited without writing the recording: " + filepath.ToSlash(output)
		case inconclusive:
			// The file is unchanged since before the run, which is either an
			// identical rewrite or no work at all; the difference is not observable.
			result["output_verification"] = "inconclusive"
			if containerBeforeErr != nil {
				result["output_verification_detail"] = "the recording could not be observed before the run: " + containerBeforeErr.Error()
			} else {
				notes = append(notes, "output_verification inconclusive: the recording file has unchanged size and modification time")
			}
			result["notes"] = notes
		}
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
type frameSnapshot map[string]fileStamp

// fileStamp is one file's size and modification time.
type fileStamp struct {
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

// snapshotMovieFrames records the PNG sequence state before a run. A destination
// that cannot be observed is an error, so verification reports inconclusive
// instead of treating unobserved files as this run's output.
func snapshotMovieFrames(output string) (frameSnapshot, error) {
	before := frameSnapshot{}
	dir, pattern, measurable := movieFramePattern(output)
	if !measurable {
		return before, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return before, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return before, infoErr
		}
		before[entry.Name()] = fileStamp{bytes: info.Size(), mod: info.ModTime()}
	}
	return before, nil
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

// movieFramePatternMeasurable reports whether an output path is a PNG sequence,
// which is the only case in which frames are measured.
func movieFramePatternMeasurable(output string) bool {
	_, _, measurable := movieFramePattern(output)
	return measurable
}

// containerVerification classifies a single-file recording target from stamps
// taken before and after the run: a missing target fails, an unchanged one is
// inconclusive (an identical rewrite and no work are indistinguishable), and a
// changed or newly created one is verified.
func containerVerification(before fileStamp, beforeSeen bool, after fileStamp, afterSeen bool, beforeErr, afterErr error) (errorCode string, inconclusive bool) {
	if afterErr != nil {
		return "artifact_stat_failed", false
	}
	if !afterSeen {
		return "no_artifact_produced", false
	}
	if beforeErr != nil || (beforeSeen && before == after) {
		return "", true
	}
	return "", false
}

// sequenceVerdict is the PNG sequence verification verdict for one run.
type sequenceVerdict struct {
	ErrorCode    string
	Detail       string
	Inconclusive bool
	// ClaimCounts reports whether frames_written/frames_unchanged may be claimed:
	// that needs a pre-run observation to compare against.
	ClaimCounts bool
	Note        string
}

// classifySequence decides the sequence verdict. A sequence that cannot be listed
// or holds no frame fails whatever the pre-run state was, so an unobservable
// destination never hides a missing recording; only the comparison-derived fields
// are withheld when the pre-run state is unknown.
func classifySequence(written, present int, scanErr, beforeErr error) sequenceVerdict {
	switch {
	case scanErr != nil:
		return sequenceVerdict{ErrorCode: "sequence_scan_failed", Detail: "the frame sequence could not be listed: " + scanErr.Error()}
	case present == 0:
		return sequenceVerdict{ErrorCode: "no_frame_produced", Detail: "the engine exited without writing a frame", ClaimCounts: true}
	case beforeErr != nil:
		return sequenceVerdict{Inconclusive: true,
			Detail: "the destination could not be observed before the run: " + beforeErr.Error()}
	}
	code, inconclusive, note := movieVerification(written, present)
	return sequenceVerdict{ErrorCode: code, Inconclusive: inconclusive, ClaimCounts: true, Note: note}
}

// movieVerification classifies a frame-count comparison. Missing frames are a
// failure; frames that are all unchanged may be a rewrite inside one filesystem
// timestamp interval or no work at all, which is reported as inconclusive rather
// than claimed as this run's output or failed.
func movieVerification(written, present int) (errorCode string, inconclusive bool, note string) {
	switch {
	case present == 0:
		return "no_frame_produced", false, "no_frame_produced: the engine exited without writing a frame"
	case written == 0:
		return "", true, "frames_verification inconclusive: every frame has unchanged size and modification time"
	case present > written:
		return "", false, fmt.Sprintf("%d frame file(s) have unchanged size and modification time; they may predate this run or be rewrites this observation cannot tell apart", present-written)
	}
	return "", false, ""
}

// statFileStamp records a regular file's identity, so a container that this run
// did not replace can be reported instead of assumed fresh.
func statFileStamp(path string) (fileStamp, bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fileStamp{}, false, nil
	}
	if err != nil {
		return fileStamp{}, false, err
	}
	if !info.Mode().IsRegular() {
		return fileStamp{}, false, fmt.Errorf("recording target is not a regular file: %s", path)
	}
	return fileStamp{bytes: info.Size(), mod: info.ModTime()}, true, nil
}

// movieFrameCounts separates frames whose metadata changed from older files with
// the same prefix. A sequence that cannot be listed is an error.
func movieFrameCounts(output string, before frameSnapshot) (written int, present int, scanErr error) {
	dir, pattern, measurable := movieFramePattern(output)
	if !measurable {
		return 0, 0, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		present++
		info, infoErr := entry.Info()
		if infoErr != nil {
			return 0, present, infoErr
		}
		if before.changed(entry.Name(), info) {
			written++
		}
	}
	return written, present, nil
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
