package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoverIn_returnsFreshInstancesMostRecentFirst(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1_000_000, 0)

	instances := map[string]Instance{
		"1.json": {PID: 1, Port: 8770, TS: now.Unix()},                        // freshest
		"2.json": {PID: 2, Port: 8771, TS: now.Add(-2 * time.Second).Unix()},  // fresh, older
		"3.json": {PID: 3, Port: 8772, TS: now.Add(-30 * time.Second).Unix()}, // stale
	}
	for name, inst := range instances {
		b, err := json.Marshal(inst)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := discoverIn(dir, now)
	if err != nil {
		t.Fatalf("discoverIn error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d live instances, want 2 (stale dropped): %#v", len(got), got)
	}
	if got[0].PID != 1 || got[1].PID != 2 {
		t.Fatalf("order = [pid %d, pid %d], want [1, 2] (most recent first)", got[0].PID, got[1].PID)
	}
}

func TestDiscoverIn_missingDirReturnsEmpty(t *testing.T) {
	got, err := discoverIn(filepath.Join(t.TempDir(), "nope"), time.Now())
	if err != nil {
		t.Fatalf("error = %v, want nil for a missing directory", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d instances, want 0", len(got))
	}
}
