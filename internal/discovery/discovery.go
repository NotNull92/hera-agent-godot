// Package discovery locates running Godot editors that expose the Hera addon.
//
// The addon's Heartbeat writes one JSON file per editor under
// ~/.hera-agent-godot/instances/<pid>.json. This package scans that directory
// and returns the live ones (most recent first).
package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	dirName   = ".hera-agent-godot"
	freshness = 5 * time.Second // an instance is "live" only if its heartbeat is this recent

	// The addon republishes <pid>.json by staging a temp file and swapping it in
	// with DirAccess.rename_absolute. That is atomic on POSIX but not on
	// Windows: Godot's DirAccess::rename removes an existing destination before
	// MoveFileW, so the file is briefly absent on every heartbeat (~0.5s). A
	// scan landing in that window sees no instances even though the editor is
	// live, and reports a spurious "no live Godot editor found".
	//
	// The window is unobservable on an idle machine — sampling the directory
	// every 1ms for 15s catches it zero times — but widens under I/O load, when
	// the remove and the rename are themselves delayed. So retry over a spread
	// of delays rather than once at a fixed one. The cost is paid only when the
	// directory really is empty, which is already a terminal error path.
	rescanDelays = 4
)

// rescanDelayFor returns the wait before attempt n (1-based), growing so the
// retries cover roughly 25ms, 75ms, 175ms and 375ms of elapsed time.
func rescanDelayFor(attempt int) time.Duration {
	return time.Duration(25*(1<<(attempt-1))) * time.Millisecond
}

// Instance describes a running Godot editor the CLI can talk to.
type Instance struct {
	PID          int    `json:"pid"`
	Port         int    `json:"port"`
	ProjectPath  string `json:"project_path"`
	GodotVersion string `json:"godot_version"`
	Scene        string `json:"scene"`
	TS           int64  `json:"ts"` // unix seconds of last heartbeat
}

// Discover scans the instances directory under the user's home and returns
// editors whose heartbeat is still fresh, most recent first.
//
// An empty pass is retried over growing delays, so a scan that lands in the
// addon's heartbeat swap window does not report "no editor" while one is
// running. A genuinely empty directory pays those delays once and then reports
// nothing, which is fine: that path already ends in an error.
func Discover() ([]Instance, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, dirName, "instances")
	return discoverRetrying(dir, time.Now, time.Sleep)
}

// discoverRetrying is the testable core of Discover's retry: an empty result may
// just be the heartbeat mid-swap rather than an absent editor, so it rescans a
// few times over growing delays before believing it.
func discoverRetrying(dir string, now func() time.Time, sleep func(time.Duration)) ([]Instance, error) {
	live, err := discoverIn(dir, now())
	if err != nil || len(live) > 0 {
		return live, err
	}
	for attempt := 1; attempt <= rescanDelays; attempt++ {
		sleep(rescanDelayFor(attempt))
		live, err = discoverIn(dir, now())
		if err != nil || len(live) > 0 {
			return live, err
		}
	}
	return live, err
}

// discoverIn is the testable core of Discover: it scans dir and drops stale
// entries relative to now.
func discoverIn(dir string, now time.Time) ([]Instance, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no editor has ever advertised — not an error
		}
		return nil, err
	}

	var live []Instance
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var inst Instance
		if err := json.Unmarshal(b, &inst); err != nil {
			continue
		}
		if inst.Port == 0 {
			continue
		}
		if now.Sub(time.Unix(inst.TS, 0)) > freshness {
			continue // stale: crashed or closed editor
		}
		live = append(live, inst)
	}

	sort.Slice(live, func(i, j int) bool { return live[i].TS > live[j].TS })
	return live, nil
}
