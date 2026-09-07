package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

func liveScan(instances []discovery.Instance) discovery.Scan {
	return discovery.Scan{Live: instances}
}

func selectEditor(scan discovery.Scan, requireSingle bool, targetPID int) (discovery.Instance, error) {
	if targetPID != 0 {
		for _, inst := range scan.Live {
			if inst.PID == targetPID {
				return inst, nil
			}
		}
		for _, inst := range scan.Stale {
			if inst.PID == targetPID {
				return discovery.Instance{}, fmt.Errorf("Godot editor pid %d heartbeat expired %s ago; the OS process may still be running. Restart the editor or wait for a fresh heartbeat, then retry (`hera instances` lists stale files)", inst.PID, heartbeatAgeText(inst))
			}
		}
		return discovery.Instance{}, fmt.Errorf("no Godot editor heartbeat for pid %d (%s)", targetPID, describeEditors(scan))
	}
	if len(scan.Live) == 0 {
		if len(scan.Stale) > 0 {
			return discovery.Instance{}, fmt.Errorf("no live Godot editor found; stale heartbeat(s): %s. Open the project in Godot, enable the Hera Agent plugin, or restart a stalled editor, then run `hera instances`", stalePIDs(scan.Stale))
		}
		return discovery.Instance{}, fmt.Errorf("no live Godot editor found; open the project in Godot, enable the Hera Agent plugin, then run `hera instances` to confirm the heartbeat")
	}
	if requireSingle && len(scan.Live) > 1 {
		return discovery.Instance{}, fmt.Errorf("multiple live Godot editors found (%s); pass --instance <pid> or close the extras before running mutation commands", instancePIDs(scan.Live))
	}
	return scan.Live[0], nil
}

func instancePIDs(instances []discovery.Instance) string {
	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		ids = append(ids, fmt.Sprintf("pid %d", inst.PID))
	}
	return strings.Join(ids, ", ")
}

func stalePIDs(instances []discovery.Instance) string {
	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		ids = append(ids, fmt.Sprintf("pid %d expired %s ago", inst.PID, heartbeatAgeText(inst)))
	}
	return strings.Join(ids, ", ")
}

func describeEditors(scan discovery.Scan) string {
	live := "live: none"
	if len(scan.Live) > 0 {
		live = "live: " + instancePIDs(scan.Live)
	}
	if len(scan.Stale) == 0 {
		return live
	}
	return live + "; stale: " + stalePIDs(scan.Stale)
}

func heartbeatAgeText(inst discovery.Instance) string {
	return fmt.Sprintf("%ds", inst.AgeSeconds(time.Now()))
}
