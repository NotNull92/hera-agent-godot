package cmd

import (
	"fmt"
	"strings"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

func selectEditor(instances []discovery.Instance, requireSingle bool, targetPID int) (discovery.Instance, error) {
	if len(instances) == 0 {
		return discovery.Instance{}, fmt.Errorf("no live Godot editor found; open the project in Godot, enable the Hera Agent plugin, then run `hera instances` to confirm the heartbeat")
	}
	if targetPID != 0 {
		for _, inst := range instances {
			if inst.PID == targetPID {
				return inst, nil
			}
		}
		return discovery.Instance{}, fmt.Errorf("no live Godot editor with pid %d (live: %s)", targetPID, instancePIDs(instances))
	}
	if requireSingle && len(instances) > 1 {
		return discovery.Instance{}, fmt.Errorf("multiple live Godot editors found (%s); pass --instance <pid> or close the extras before running mutation commands", instancePIDs(instances))
	}
	return instances[0], nil
}

func instancePIDs(instances []discovery.Instance) string {
	ids := make([]string, 0, len(instances))
	for _, inst := range instances {
		ids = append(ids, fmt.Sprintf("pid %d", inst.PID))
	}
	return strings.Join(ids, ", ")
}
