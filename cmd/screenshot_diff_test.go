package cmd

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writePNG paints a solid canvas, optionally stamping a differently coloured
// rectangle, and returns the file path.
func writePNG(t *testing.T, name string, w, h int, base color.RGBA, stamp *image.Rectangle, stampColor color.RGBA) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, base)
		}
	}
	if stamp != nil {
		for y := stamp.Min.Y; y < stamp.Max.Y; y++ {
			for x := stamp.Min.X; x < stamp.Max.X; x++ {
				img.Set(x, y, stampColor)
			}
		}
	}
	path := filepath.Join(t.TempDir(), name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDiffImageFiles_reportsIdenticalFrames(t *testing.T) {
	// Given
	grey := color.RGBA{40, 40, 48, 255}
	a := writePNG(t, "a.png", 32, 16, grey, nil, grey)
	b := writePNG(t, "b.png", 32, 16, grey, nil, grey)

	// When
	got, err := diffImageFiles(a, b, defaultDiffThreshold)

	// Then
	if err != nil {
		t.Fatalf("diffImageFiles error: %v", err)
	}
	if got["identical"] != true || got["changed_pixels"] != 0 {
		t.Fatalf("got %#v, want identical", got)
	}
	if _, ok := got["changed_bounds"]; ok {
		t.Fatal("identical frames must not report a bounding box")
	}
}

func TestDiffImageFiles_boundsCoverExactlyTheChangedRegion(t *testing.T) {
	// Given: one 4x3 patch at (10,5) differs.
	grey := color.RGBA{40, 40, 48, 255}
	rect := image.Rect(10, 5, 14, 8)
	a := writePNG(t, "a.png", 40, 20, grey, nil, grey)
	b := writePNG(t, "b.png", 40, 20, grey, &rect, color.RGBA{240, 60, 60, 255})

	// When
	got, err := diffImageFiles(a, b, defaultDiffThreshold)

	// Then
	if err != nil {
		t.Fatalf("diffImageFiles error: %v", err)
	}
	if got["changed_pixels"] != 12 {
		t.Fatalf("changed_pixels = %v, want 12", got["changed_pixels"])
	}
	bounds, ok := got["changed_bounds"].(map[string]any)
	if !ok {
		t.Fatalf("changed_bounds missing: %#v", got)
	}
	if bounds["x"] != 10 || bounds["y"] != 5 || bounds["width"] != 4 || bounds["height"] != 3 {
		t.Fatalf("changed_bounds = %#v, want x=10 y=5 w=4 h=3", bounds)
	}
	if got["identical"] != false {
		t.Fatal("identical must be false when pixels changed")
	}
}

func TestDiffImageFiles_thresholdAbsorbsCaptureNoise(t *testing.T) {
	// Given: every pixel differs by 2 — the wobble a re-capture can produce.
	rect := image.Rect(0, 0, 20, 10)
	a := writePNG(t, "a.png", 20, 10, color.RGBA{40, 40, 48, 255}, nil, color.RGBA{})
	b := writePNG(t, "b.png", 20, 10, color.RGBA{42, 42, 50, 255}, &rect, color.RGBA{42, 42, 50, 255})

	// When: default threshold ignores it, threshold 0 does not.
	lenient, err := diffImageFiles(a, b, defaultDiffThreshold)
	if err != nil {
		t.Fatalf("diffImageFiles error: %v", err)
	}
	strict, err := diffImageFiles(a, b, 0)
	if err != nil {
		t.Fatalf("diffImageFiles error: %v", err)
	}

	// Then
	if lenient["identical"] != true {
		t.Fatalf("a 2/255 wobble should be absorbed by the default threshold: %#v", lenient)
	}
	if strict["identical"] != false {
		t.Fatalf("threshold 0 must report the same wobble: %#v", strict)
	}
	if lenient["max_delta"] != 2 {
		t.Fatalf("max_delta = %v, want 2 — reported even when under threshold", lenient["max_delta"])
	}
}

func TestDiffImageFiles_rejectsMismatchedSizes(t *testing.T) {
	grey := color.RGBA{10, 10, 10, 255}
	a := writePNG(t, "a.png", 20, 10, grey, nil, grey)
	b := writePNG(t, "b.png", 21, 10, grey, nil, grey)

	if _, err := diffImageFiles(a, b, defaultDiffThreshold); err == nil {
		t.Fatal("expected an error when the two frames are different sizes")
	}
}

func TestDiffImageFiles_reportsUnreadableInput(t *testing.T) {
	grey := color.RGBA{10, 10, 10, 255}
	ok := writePNG(t, "a.png", 4, 4, grey, nil, grey)

	if _, err := diffImageFiles(filepath.Join(t.TempDir(), "missing.png"), ok, 4); err == nil {
		t.Fatal("expected an error for a missing before-image")
	}

	notPNG := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(notPNG, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := diffImageFiles(ok, notPNG, 4); err == nil {
		t.Fatal("expected an error for an undecodable after-image")
	}
}

func TestParseScreenshotDiffArgs_validatesThreshold(t *testing.T) {
	if _, _, _, err := parseScreenshotDiffArgs([]string{"a.png"}); err == nil {
		t.Fatal("expected an error when only one image is given")
	}
	if _, _, _, err := parseScreenshotDiffArgs([]string{"a.png", "b.png", "--threshold", "999"}); err == nil {
		t.Fatal("expected an error for an out-of-range threshold")
	}
	if _, _, _, err := parseScreenshotDiffArgs([]string{"a.png", "b.png", "--nope"}); err == nil {
		t.Fatal("expected an error for an unknown flag")
	}
	_, _, threshold, err := parseScreenshotDiffArgs([]string{"a.png", "b.png"})
	if err != nil {
		t.Fatalf("parseScreenshotDiffArgs error: %v", err)
	}
	if threshold != defaultDiffThreshold {
		t.Fatalf("threshold = %d, want the default %d", threshold, defaultDiffThreshold)
	}
}
