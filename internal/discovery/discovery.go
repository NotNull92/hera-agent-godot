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

	// rescanDelay covers a gap in how the addon republishes its heartbeat.
	// Heartbeat.write() stages the JSON in a temp file and swaps it in with
	// DirAccess.rename_absolute, which is atomic on POSIX but not on Windows:
	// Godot's DirAccess::rename removes an existing destination *before*
	// MoveFileW, so <pid>.json is briefly absent on every heartbeat (~0.5s).
	// A scan landing in that window sees no instances even though the editor is
	// live, which surfaces as a spurious "no live Godot editor found". One short
	// rescan closes it; the window is sub-millisecond.
	rescanDelay = 40 * time.Millisecond
)

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
// An empty first pass is rescanned once after rescanDelay, so a scan that lands
// in the addon's heartbeat swap window does not report "no editor" while one is
// running. A genuinely empty directory just costs that one short delay.
func Discover() ([]Instance, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, dirName, "instances")
	return discoverRetrying(dir, time.Now, time.Sleep)
}

// discoverRetrying is the testable core of Discover's retry: it rescans once
// when the first pass finds nothing, since an absent file may just be the
// heartbeat mid-swap rather than an absent editor.
func discoverRetrying(dir string, now func() time.Time, sleep func(time.Duration)) ([]Instance, error) {
	live, err := discoverIn(dir, now())
	if err != nil || len(live) > 0 {
		return live, err
	}
	sleep(rescanDelay)
	return discoverIn(dir, now())
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
