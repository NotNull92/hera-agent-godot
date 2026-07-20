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

func TestDiscoverRetrying_rescansWhenHeartbeatIsMidSwap(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1_000_000, 0)
	inst := Instance{PID: 42, Port: 8770, TS: now.Unix()}

	// The addon republishes its heartbeat by swapping <pid>.json, and on Windows
	// the destination is removed before the rename — so the first scan can land
	// in a window where the file does not exist yet. Publish it during the sleep.
	slept := 0
	sleep := func(time.Duration) {
		slept++
		b, err := json.Marshal(inst)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "42.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := discoverRetrying(dir, func() time.Time { return now }, sleep)
	if err != nil {
		t.Fatalf("discoverRetrying error: %v", err)
	}
	if slept != 1 {
		t.Fatalf("rescan delay used %d times, want exactly 1", slept)
	}
	if len(got) != 1 || got[0].PID != 42 {
		t.Fatalf("got %#v, want the instance published during the swap window", got)
	}
}

func TestDiscoverRetrying_doesNotRescanWhenFirstPassFinds(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1_000_000, 0)
	b, err := json.Marshal(Instance{PID: 7, Port: 8770, TS: now.Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "7.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	slept := 0
	got, err := discoverRetrying(dir, func() time.Time { return now }, func(time.Duration) { slept++ })
	if err != nil {
		t.Fatalf("discoverRetrying error: %v", err)
	}
	if slept != 0 {
		t.Fatalf("slept %d times, want 0 — a hit on the first pass must not pay the delay", slept)
	}
	if len(got) != 1 || got[0].PID != 7 {
		t.Fatalf("got %#v, want pid 7", got)
	}
}

func TestDiscoverRetrying_stillEmptyWhenNoEditorIsRunning(t *testing.T) {
	dir := t.TempDir()
	slept := 0
	got, err := discoverRetrying(dir, time.Now, func(time.Duration) { slept++ })
	if err != nil {
		t.Fatalf("discoverRetrying error: %v", err)
	}
	if slept != rescanDelays {
		t.Fatalf("slept %d times, want %d — every retry is spent before believing an empty directory", slept, rescanDelays)
	}
	if len(got) != 0 {
		t.Fatalf("got %d instances, want 0", len(got))
	}
}

func TestDiscoverRetrying_stopsAsSoonAsTheEditorAppears(t *testing.T) {
	// Given: the heartbeat reappears on the second retry, as it would when the
	// swap window is stretched by I/O load.
	dir := t.TempDir()
	now := time.Unix(1_000_000, 0)
	slept := 0
	sleep := func(time.Duration) {
		slept++
		if slept < 2 {
			return
		}
		b, err := json.Marshal(Instance{PID: 5, Port: 8770, TS: now.Unix()})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "5.json"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// When
	got, err := discoverRetrying(dir, func() time.Time { return now }, sleep)

	// Then: it must not keep retrying once it has an answer.
	if err != nil {
		t.Fatalf("discoverRetrying error: %v", err)
	}
	if slept != 2 {
		t.Fatalf("slept %d times, want 2 — retries stop at the first hit", slept)
	}
	if len(got) != 1 || got[0].PID != 5 {
		t.Fatalf("got %#v, want the instance that appeared mid-retry", got)
	}
}

func TestRescanDelayFor_growsAndCoversAUsefulSpread(t *testing.T) {
	// The point of backing off is covering a window that widens under load, so
	// each attempt must wait longer than the last.
	var total time.Duration
	prev := time.Duration(0)
	for attempt := 1; attempt <= rescanDelays; attempt++ {
		d := rescanDelayFor(attempt)
		if d <= prev {
			t.Fatalf("delay %d = %v, want longer than the previous %v", attempt, d, prev)
		}
		prev = d
		total += d
	}
	if total < 300*time.Millisecond {
		t.Fatalf("total retry window = %v, too short to cover a stretched swap", total)
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
