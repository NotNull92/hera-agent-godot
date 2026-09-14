package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanPreservesOptionalEditorSession(t *testing.T) {
	for _, session := range []string{"", "editor-session-a"} {
		t.Run(session, func(t *testing.T) {
			dir := t.TempDir()
			data := struct {
				PID     int    `json:"pid"`
				Port    int    `json:"port"`
				TS      int64  `json:"ts"`
				Session string `json:"editor_session_id,omitempty"`
			}{42, 8770, 1000, session}
			encoded, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "42.json"), encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			scan, err := scanIn(dir, time.Unix(1000, 0))
			if err != nil || len(scan.Live) != 1 {
				t.Fatalf("scan = %#v, error = %v", scan, err)
			}
			encoded, err = json.Marshal(scan.Live[0])
			if err != nil {
				t.Fatal(err)
			}
			var roundtrip map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &roundtrip); err != nil {
				t.Fatal(err)
			}
			got, present := roundtrip["editor_session_id"]
			if session == "" && present || session != "" && string(got) != `"`+session+`"` {
				t.Fatalf("session = %v (present %t), want %q with legacy omission", got, present, session)
			}
		})
	}
}
