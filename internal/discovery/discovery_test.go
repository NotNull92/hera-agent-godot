package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanIn_returnsFreshInstancesMostRecentFirst(t *testing.T) {
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

	scan, err := scanIn(dir, now)
	if err != nil {
		t.Fatalf("scanIn error: %v", err)
	}
	if len(scan.Live) != 2 {
		t.Fatalf("got %d live instances, want 2 (stale dropped): %#v", len(scan.Live), scan)
	}
	if scan.Live[0].PID != 1 || scan.Live[1].PID != 2 {
		t.Fatalf("order = [pid %d, pid %d], want [1, 2] (most recent first)", scan.Live[0].PID, scan.Live[1].PID)
	}
	if len(scan.Stale) != 1 || scan.Stale[0].PID != 3 {
		t.Fatalf("stale = %#v, want pid 3", scan.Stale)
	}
}

func TestScanIn_keepsExpiredHeartbeatsSeparateFromLive(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1_000_000, 0)
	instances := map[string]Instance{
		"1.json": {PID: 1, Port: 8770, TS: now.Unix()},
		"3.json": {PID: 3, Port: 8772, TS: now.Add(-54 * time.Second).Unix()},
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

	scan, err := scanIn(dir, now)
	if err != nil {
		t.Fatalf("scanIn error: %v", err)
	}
	if len(scan.Live) != 1 || scan.Live[0].PID != 1 {
		t.Fatalf("live = %#v, want pid 1", scan.Live)
	}
	if len(scan.Stale) != 1 || scan.Stale[0].PID != 3 {
		t.Fatalf("stale = %#v, want pid 3", scan.Stale)
	}
	if got := scan.Stale[0].AgeSeconds(now); got != 54 {
		t.Fatalf("stale age = %d, want 54", got)
	}
}

func TestScanRetrying_rescansWhenHeartbeatIsMidSwap(t *testing.T) {
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

	scan, err := scanRetrying(dir, func() time.Time { return now }, sleep)
	if err != nil {
		t.Fatalf("scanRetrying error: %v", err)
	}
	if slept != 1 {
		t.Fatalf("rescan delay used %d times, want exactly 1", slept)
	}
	if len(scan.Live) != 1 || scan.Live[0].PID != 42 {
		t.Fatalf("got %#v, want the instance published during the swap window", scan)
	}
}

func TestScanRetrying_doesNotRescanWhenFirstPassFinds(t *testing.T) {
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
	scan, err := scanRetrying(dir, func() time.Time { return now }, func(time.Duration) { slept++ })
	if err != nil {
		t.Fatalf("scanRetrying error: %v", err)
	}
	if slept != 0 {
		t.Fatalf("slept %d times, want 0 — a hit on the first pass must not pay the delay", slept)
	}
	if len(scan.Live) != 1 || scan.Live[0].PID != 7 {
		t.Fatalf("got %#v, want pid 7", scan)
	}
}

func TestScanRetrying_stillEmptyWhenNoEditorIsRunning(t *testing.T) {
	dir := t.TempDir()
	slept := 0
	scan, err := scanRetrying(dir, time.Now, func(time.Duration) { slept++ })
	if err != nil {
		t.Fatalf("scanRetrying error: %v", err)
	}
	if slept != rescanDelays {
		t.Fatalf("slept %d times, want %d — every retry is spent before believing an empty directory", slept, rescanDelays)
	}
	if len(scan.Live) != 0 {
		t.Fatalf("got %d instances, want 0", len(scan.Live))
	}
}

func TestScanRetrying_stopsAsSoonAsTheEditorAppears(t *testing.T) {
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
	scan, err := scanRetrying(dir, func() time.Time { return now }, sleep)

	// Then: it must not keep retrying once it has an answer.
	if err != nil {
		t.Fatalf("scanRetrying error: %v", err)
	}
	if slept != 2 {
		t.Fatalf("slept %d times, want 2 — retries stop at the first hit", slept)
	}
	if len(scan.Live) != 1 || scan.Live[0].PID != 5 {
		t.Fatalf("got %#v, want the instance that appeared mid-retry", scan)
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

func TestScanIn_missingDirReturnsEmpty(t *testing.T) {
	scan, err := scanIn(filepath.Join(t.TempDir(), "nope"), time.Now())
	if err != nil {
		t.Fatalf("error = %v, want nil for a missing directory", err)
	}
	if len(scan.Live) != 0 || len(scan.Stale) != 0 {
		t.Fatalf("got %#v, want empty live and stale", scan)
	}
}
