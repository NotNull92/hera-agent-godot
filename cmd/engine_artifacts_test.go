package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactStagingAndPublication(t *testing.T) {
	destination := t.TempDir()
	old := filepath.Join(destination, "movie00000001.png")
	if err := os.WriteFile(old, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	stage, err := stageArtifacts(destination)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(stage, "movie.png")
	if _, count, err := movieFrameCounts(output, nil); err != nil || count != 0 {
		t.Fatalf("old destination counted: %d %v", count, err)
	}
	generated := filepath.Join(stage, filepath.Base(old))
	if err := os.WriteFile(generated, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldInfo, _ := os.Stat(old)
	if err := os.Chtimes(generated, oldInfo.ModTime(), oldInfo.ModTime()); err != nil {
		t.Fatal(err)
	}
	if written, present, err := movieFrameCounts(output, nil); err != nil || written != 1 || present != 1 {
		t.Fatalf("same metadata run lost: %d %d %v", written, present, err)
	}
	if n, err := publishArtifacts(stage, destination, false); err == nil || n != 0 {
		t.Fatalf("overwrote without force: %d %v", n, err)
	}
	if data, _ := os.ReadFile(old); string(data) != "old" {
		t.Fatal("existing output changed")
	}
	if n, err := publishArtifacts(stage, destination, true); err != nil || n != 1 {
		t.Fatalf("publish: %d %v", n, err)
	}
	if data, _ := os.ReadFile(old); string(data) != "new" {
		t.Fatal("generated output not published")
	}
	if _, err := os.Stat(stage); !os.IsNotExist(err) {
		t.Fatalf("stage not cleaned: %v", err)
	}
}

func TestArtifactPublicationProtectsLinkedDestination(t *testing.T) {
	destination, outside := t.TempDir(), t.TempDir()
	stage, err := stageArtifacts(destination)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(stage, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "docs", "Class.xml"), []byte("xml"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(destination, "docs")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if n, err := publishArtifacts(stage, destination, true); err == nil || n != 0 {
		t.Fatalf("escaped root: %d %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(outside, "Class.xml")); !os.IsNotExist(err) {
		t.Fatal("wrote through destination link")
	}
}

func TestArtifactPublicationIncludesSidecarsAndNestedXML(t *testing.T) {
	for _, replace := range []bool{false, true} {
		destination := t.TempDir()
		stage, err := stageArtifacts(destination)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"movie.avi", "movie.wav", "doc/classes/Node.xml"} {
			path := filepath.Join(stage, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		if n, err := publishArtifacts(stage, destination, replace); err != nil || n != 3 {
			t.Fatalf("publish: %d %v", n, err)
		}
		for _, name := range []string{"movie.avi", "movie.wav", "doc/classes/Node.xml"} {
			if data, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(name))); err != nil || string(data) != name {
				t.Fatalf("%s: %s %v", name, data, err)
			}
		}
	}
}

func TestFailedRunNeverPublishesArtifacts(t *testing.T) {
	for _, run := range []engineRunResult{{ExitCode: 1}, {ExitCode: -1, Abandoned: true}, {ExitCode: 0, CleanupPending: true}} {
		destination := t.TempDir()
		final := filepath.Join(destination, "movie.avi")
		if err := os.WriteFile(final, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		stage, err := stageArtifacts(destination)
		if err != nil {
			t.Fatal(err)
		}
		staged := filepath.Join(stage, "movie.avi")
		if err := os.WriteFile(staged, []byte("new"), 0o600); err != nil {
			t.Fatal(err)
		}
		result := map[string]any{}
		if code := finishArtifactPublication(result, run, stage, destination, true); code != 1 {
			t.Fatalf("failed engine accepted: %v", run)
		}
		if result["published"] != false || result["staging_dir"] != filepath.ToSlash(stage) {
			t.Fatalf("retained output not reported: %v", result)
		}
		if body, _ := os.ReadFile(final); string(body) != "old" {
			t.Fatal("failed run overwrote destination")
		}
		if body, _ := os.ReadFile(staged); string(body) != "new" {
			t.Fatal("failed run lost staged output")
		}
	}
}
