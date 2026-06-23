// Package protocol defines the JSON request/response contract shared (in spirit)
// between the Go CLI and the Godot addon's HTTP server.
package protocol

// Request is one command sent from the CLI to the editor addon.
type Request struct {
	Tool   string         `json:"tool"`
	Params map[string]any `json:"params,omitempty"`
}

// Response is what the addon returns for a Request.
type Response struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}
