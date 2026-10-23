package cmd

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/dap"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDAPPort    = 6006
	defaultStopWaitMS = 10000
	maxWaitMS         = 3600000
	maxDAPVariables   = 50
)

// debugSessionIDPattern keeps a session identifier to one path component.
var debugSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// debugSession is the endpoint recipe one attach records so later calls can
// reconnect. It is not evidence that the original debuggee is still running:
// reconnects address whatever now answers on that host and port.
type debugSession struct {
	ID          string              `json:"id"`
	Host        string              `json:"host"`
	Port        int                 `json:"port"`
	Project     string              `json:"project"`
	Breakpoints []sessionBreakpoint `json:"breakpoints"`
	AttachedAt  string              `json:"attached_at"`
}

type sessionBreakpoint struct {
	Path string `json:"path"`
	Line int    `json:"line"`
}

// dapSessionRoot returns the private session directory. A missing or unusable
// home fails instead of silently sharing a predictable temporary namespace.
func dapSessionRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("no user home directory for debug sessions: %v", err)
	}
	return filepath.Join(home, ".hera-agent-godot", "debug"), nil
}

// validSessionID rejects separators, traversal and anything that is not a single
// safe path component.
func validSessionID(id string) bool {
	if !debugSessionIDPattern.MatchString(id) || id == "." || id == ".." {
		return false
	}
	return !strings.ContainsAny(id, `/\`)
}

func writeDebugSession(session debugSession) error {
	root, err := dapSessionRoot()
	if err != nil {
		return err
	}
	if !validSessionID(session.ID) {
		return fmt.Errorf("refusing to persist an invalid session id %q", session.ID)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	recordPath := filepath.Join(root, session.ID+".json")
	if err := writeFileAtomic(recordPath, data); err != nil {
		return err
	}
	// The pointer is a convenience, not part of the record's durability.
	if err := writeFileAtomic(filepath.Join(root, "latest"), []byte(session.ID)); err != nil {
		return fmt.Errorf("session record written but the latest pointer failed: %w", err)
	}
	return nil
}

// loadDebugSession resolves an id (explicit, or the latest pointer) and validates
// both the id and the record before anything else happens.
func loadDebugSession(id string) (debugSession, string, error) {
	root, err := dapSessionRoot()
	if err != nil {
		return debugSession{}, "", err
	}
	if id == "" || id == "latest" {
		data, readErr := os.ReadFile(filepath.Join(root, "latest"))
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				return debugSession{}, "", fmt.Errorf("no debug session recorded; run debug attach first")
			}
			return debugSession{}, "", fmt.Errorf("cannot read the latest session pointer: %w", readErr)
		}
		id = strings.TrimSpace(string(data))
	}
	if !validSessionID(id) {
		return debugSession{}, "", fmt.Errorf("invalid session id %q", id)
	}
	recordPath := filepath.Join(root, id+".json")
	raw, readErr := os.ReadFile(recordPath)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return debugSession{}, "", fmt.Errorf("debug session %q not found", id)
		}
		return debugSession{}, "", fmt.Errorf("cannot read debug session %q: %w", id, readErr)
	}
	var session debugSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return debugSession{}, "", fmt.Errorf("debug session %q is unreadable: %w", id, err)
	}
	if session.ID != id || !validSessionID(session.ID) {
		return debugSession{}, "", fmt.Errorf("debug session %q does not describe itself correctly", id)
	}
	if session.Port <= 0 || session.Port > 65535 || session.Host == "" {
		return debugSession{}, "", fmt.Errorf("debug session %q has no usable endpoint", id)
	}
	return session, recordPath, nil
}

// dapAddress joins host and port, including IPv6 hosts.
func dapAddress(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// connectDebug attaches to the recorded endpoint, configures breakpoints once and
// completes configuration once. Adapter rejections surface as errors.
func connectDebug(session debugSession, wait time.Duration) (*dap.Client, []map[string]any, error) {
	client, err := dap.Dial(dapAddress(session.Host, session.Port), wait)
	if err != nil {
		return nil, nil, err
	}
	if _, err := client.Request("initialize", map[string]any{
		"clientID": "hera-godot", "adapterID": "godot", "pathFormat": "path",
		"linesStartAt1": true, "columnsStartAt1": true,
	}, wait); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("initialize: %w", err)
	}
	if _, err := client.Request("attach", map[string]any{"type": "godot", "request": "attach", "project": session.Project}, wait); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("attach: %w", err)
	}
	results, err := applyBreakpoints(client, session.Breakpoints, wait)
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	return client, results, nil
}

// applyBreakpoints sends every source's breakpoints and then completes
// configuration exactly once for the connection.
func applyBreakpoints(client *dap.Client, breakpoints []sessionBreakpoint, wait time.Duration) ([]map[string]any, error) {
	byPath := map[string][]int{}
	order := []string{}
	for _, breakpoint := range breakpoints {
		if _, seen := byPath[breakpoint.Path]; !seen {
			order = append(order, breakpoint.Path)
		}
		byPath[breakpoint.Path] = append(byPath[breakpoint.Path], breakpoint.Line)
	}
	results := []map[string]any{}
	for _, path := range order {
		lines := byPath[path]
		entries := make([]map[string]any, 0, len(lines))
		for _, line := range lines {
			entries = append(entries, map[string]any{"line": line})
		}
		source := map[string]any{"name": filepath.Base(path), "path": path}
		checksums, hashErr := fileChecksums(path)
		if hashErr == nil {
			source["checksums"] = checksums
		}
		response, err := client.Request("setBreakpoints", map[string]any{
			"source": source, "breakpoints": entries, "lines": lines,
		}, wait)
		if err != nil {
			return nil, fmt.Errorf("setBreakpoints %s: %w", path, err)
		}
		// Reconcile: every requested breakpoint must be accounted for.
		reported := response.Map("body").List("breakpoints")
		for index, line := range lines {
			entry := map[string]any{
				"path": filepath.ToSlash(path), "line": line, "verified": false,
				"checksums_source": "local_disk",
			}
			if hashErr != nil {
				entry["checksums_available"] = false
			}
			if index < len(reported) {
				entry["line"] = reported[index].Int("line")
				entry["verified"] = reported[index].Bool("verified")
			} else {
				entry["unavailable"] = true
			}
			results = append(results, entry)
		}
	}
	if _, err := client.Request("configurationDone", nil, wait); err != nil {
		return nil, fmt.Errorf("configurationDone: %w", err)
	}
	return results, nil
}

// fileChecksums returns the DAP checksum entries for a local script. These are
// local disk hashes sent to the adapter; they do not prove which revision the
// running game loaded.
func fileChecksums(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	md5sum := md5.Sum(data)
	sha := sha256.Sum256(data)
	return []map[string]any{
		{"algorithm": "MD5", "checksum": hex.EncodeToString(md5sum[:])},
		{"algorithm": "SHA256", "checksum": hex.EncodeToString(sha[:])},
	}, nil
}
