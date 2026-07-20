package cmd

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"strconv"
)

// defaultDiffThreshold ignores per-channel differences this small. Captures of
// the same frame are not always bit-identical — anti-aliasing and dithering
// wobble by a step or two — so an exact compare reports noise as change.
const defaultDiffThreshold = 4

// runScreenshotDiff implements `screenshot diff <before.png> <after.png>`.
//
// This runs entirely locally: both frames are already on disk, so there is no
// reason to round-trip the editor, and it works with no editor running at all.
func runScreenshotDiff(args []string) int {
	before, after, threshold, err := parseScreenshotDiffArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "screenshot: %v\n", err)
		return 2
	}
	result, err := diffImageFiles(before, after, threshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "screenshot: %v\n", err)
		return 1
	}
	var out []byte
	if outputMode == "json" {
		out, err = json.MarshalIndent(result, "", "  ")
	} else {
		out, err = json.Marshal(result)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "screenshot: %v\n", err)
		return 1
	}
	fmt.Println(string(out))
	return 0
}

func parseScreenshotDiffArgs(args []string) (string, string, int, error) {
	usage := "usage: screenshot diff <before.png> <after.png> [--threshold N]"
	if len(args) < 2 {
		return "", "", 0, fmt.Errorf("%s", usage)
	}
	before, after := args[0], args[1]
	threshold := defaultDiffThreshold
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--threshold":
			if i+1 >= len(args) {
				return "", "", 0, fmt.Errorf("--threshold requires a value")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 0 || n > 255 {
				return "", "", 0, fmt.Errorf("--threshold: want 0..255, got %q", args[i+1])
			}
			threshold = n
			i++
		default:
			return "", "", 0, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return before, after, threshold, nil
}

func diffImageFiles(beforePath, afterPath string, threshold int) (map[string]any, error) {
	before, err := decodeImageFile(beforePath)
	if err != nil {
		return nil, err
	}
	after, err := decodeImageFile(afterPath)
	if err != nil {
		return nil, err
	}

	bb, ab := before.Bounds(), after.Bounds()
	if bb.Dx() != ab.Dx() || bb.Dy() != ab.Dy() {
		return nil, fmt.Errorf("size mismatch: before is %dx%d, after is %dx%d — a diff needs the same viewport",
			bb.Dx(), bb.Dy(), ab.Dx(), ab.Dy())
	}

	width, height := bb.Dx(), bb.Dy()
	changed := 0
	maxDelta := 0
	minX, minY, maxX, maxY := width, height, -1, -1

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			delta := channelDelta(before.At(bb.Min.X+x, bb.Min.Y+y), after.At(ab.Min.X+x, ab.Min.Y+y))
			if delta > maxDelta {
				maxDelta = delta
			}
			if delta <= threshold {
				continue
			}
			changed++
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}

	total := width * height
	result := map[string]any{
		"before":         beforePath,
		"after":          afterPath,
		"width":          width,
		"height":         height,
		"threshold":      threshold,
		"total_pixels":   total,
		"changed_pixels": changed,
		"changed_ratio":  ratio(changed, total),
		"max_delta":      maxDelta,
		"identical":      changed == 0,
	}
	// The bounding box is the useful half of the answer: it separates "the side
	// panel restyled" from "the whole frame shifted".
	if changed > 0 {
		result["changed_bounds"] = map[string]any{
			"x": minX, "y": minY,
			"width":  maxX - minX + 1,
			"height": maxY - minY + 1,
		}
	}
	return result, nil
}

func decodeImageFile(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("cannot decode %s: %w", path, err)
	}
	return img, nil
}

// channelDelta is the largest per-channel difference in 0..255. Comparing
// channels separately keeps a hue shift visible where averaging would hide it.
func channelDelta(a, b color.Color) int {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return maxInt(
		maxInt(absInt(int(ar>>8)-int(br>>8)), absInt(int(ag>>8)-int(bg>>8))),
		maxInt(absInt(int(ab>>8)-int(bb>>8)), absInt(int(aa>>8)-int(ba>>8))),
	)
}

func ratio(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
