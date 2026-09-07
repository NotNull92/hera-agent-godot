package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

func TestSelectEditor_reportsActionableMessage_whenNoInstances(t *testing.T) {
	// Given
	instances := []discovery.Instance{}

	// When
	_, err := selectEditor(liveScan(instances), false, 0)

	// Then
	if err == nil {
		t.Fatal("expected an error for an empty live-editor list")
	}
	message := err.Error()
	for _, want := range []string{"no live Godot editor found", "Hera Agent plugin", "hera instances"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not include %q", message, want)
		}
	}
}

func TestSelectEditor_distinguishesExpiredHeartbeatFromMissing(t *testing.T) {
	scan := discovery.Scan{
		Stale: []discovery.Instance{{PID: 33516, Port: 8770, TS: time.Now().Add(-54 * time.Second).Unix()}},
	}

	_, err := selectEditor(scan, false, 33516)
	if err == nil {
		t.Fatal("expected expired-heartbeat error")
	}
	message := err.Error()
	for _, want := range []string{"pid 33516", "expired", "ago", "hera instances"} {
		if !strings.Contains(message, want) {
			t.Fatalf("targeted stale error %q does not include %q", message, want)
		}
	}
	if strings.Contains(message, "no live Godot editor found") {
		t.Fatalf("targeted stale error should not collapse to a missing-editor message: %q", message)
	}

	_, err = selectEditor(scan, false, 0)
	if err == nil {
		t.Fatal("expected stale-list error when no live editor exists")
	}
	message = err.Error()
	for _, want := range []string{"no live Godot editor found", "stale heartbeat", "pid 33516 expired"} {
		if !strings.Contains(message, want) {
			t.Fatalf("untargeted stale error %q does not include %q", message, want)
		}
	}

	_, err = selectEditor(scan, false, 99)
	if err == nil {
		t.Fatal("expected missing-pid error")
	}
	message = err.Error()
	if !strings.Contains(message, "no Godot editor heartbeat for pid 99") || !strings.Contains(message, "stale: pid 33516") {
		t.Fatalf("missing pid error %q, want missing vs stale distinction", message)
	}
}
