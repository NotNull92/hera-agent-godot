package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type gameInputSequenceEvent struct {
	Frame   *int   `json:"frame"`
	Action  string `json:"action"`
	Pressed *bool  `json:"pressed"`
}

func parseGameInputSequenceArgs(args []string) (map[string]any, error) {
	if len(args) != 2 || args[0] != "--file" || args[1] == "" {
		return nil, fmt.Errorf("usage: game input sequence --file <events.json>")
	}
	file, err := os.Open(args[1])
	if err != nil {
		return nil, fmt.Errorf("read input sequence: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil {
		return nil, fmt.Errorf("read input sequence: %w", err)
	}
	if len(data) > 65536 {
		return nil, fmt.Errorf("input sequence file must not exceed 64 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var events []gameInputSequenceEvent
	if err := decoder.Decode(&events); err != nil {
		return nil, fmt.Errorf("decode input sequence: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("input sequence must contain exactly one JSON array")
	}
	if len(events) == 0 || len(events) > 128 {
		return nil, fmt.Errorf("input sequence requires 1..128 events")
	}
	previous := 0
	for i, event := range events {
		if event.Frame == nil || *event.Frame < previous || *event.Frame > 120 || event.Pressed == nil || strings.TrimSpace(event.Action) == "" {
			return nil, fmt.Errorf("input sequence event %d requires ordered frame 0..120, non-empty action, and boolean pressed", i)
		}
		previous = *event.Frame
	}
	return map[string]any{"action": "input", "kind": "sequence", "events": events}, nil
}
