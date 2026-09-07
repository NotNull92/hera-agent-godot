package cmd

import "testing"

func TestParseGameArgs_returnsClockSnapshot(t *testing.T) {
	got, err := parseGameArgs([]string{"clock"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "clock" {
		t.Fatalf("action = %v, want clock", got["action"])
	}
	if gameClockMutates(got) {
		t.Fatal("bare game clock should not mutate")
	}
}

func TestParseGameArgs_returnsClockPauseAndScale(t *testing.T) {
	got, err := parseGameArgs([]string{"clock", "--pause", "--time-scale", "0.5"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["paused"] != true {
		t.Fatalf("paused = %v, want true", got["paused"])
	}
	if got["time_scale"] != 0.5 {
		t.Fatalf("time_scale = %v, want 0.5", got["time_scale"])
	}
	if !gameClockMutates(got) {
		t.Fatal("pause/time-scale should mutate")
	}
}

func TestParseGameArgs_returnsClockStepPhysics(t *testing.T) {
	got, err := parseGameArgs([]string{"clock", "--step", "--physics"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["step"] != true || got["physics"] != true {
		t.Fatalf("step params = %v", got)
	}
}

func TestParseGameArgs_rejectsInvalidClockArgs(t *testing.T) {
	tests := [][]string{
		{"clock", "--pause", "--resume"},
		{"clock", "--resume", "--step"},
		{"clock", "--physics"},
		{"clock", "--time-scale", "0"},
		{"clock", "--time-scale", "-1"},
		{"clock", "--time-scale"},
		{"clock", "--bad"},
	}
	for _, args := range tests {
		if _, err := parseGameArgs(args); err == nil {
			t.Fatalf("expected parse error for %v", args)
		}
	}
}

func TestGameClockParamsFromQAStep_passesParamsThrough(t *testing.T) {
	step := gameQAStep{Params: map[string]any{"paused": true, "step": true}}
	params := gameClockParamsFromQAStep(step)
	if params["action"] != "clock" {
		t.Fatalf("action = %v, want clock", params["action"])
	}
	if params["paused"] != true || params["step"] != true {
		t.Fatalf("clock qa params = %v", params)
	}
}
