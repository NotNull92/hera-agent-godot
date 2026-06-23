// Package client is a thin HTTP client targeting a single Godot editor instance
// over localhost.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

const defaultTimeout = 5 * time.Second

// Client talks to one editor's HTTP server at http://127.0.0.1:<port>/rpc.
type Client struct {
	BaseURL string // e.g. "http://127.0.0.1:8770"
	HTTP    *http.Client
}

// New builds a Client with a sane default timeout.
func New(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTP: newDefaultHTTPClient()}
}

// Post sends a single tool request and returns the decoded response.
func (c *Client) Post(tool string, params map[string]any) (*protocol.Response, error) {
	body, err := json.Marshal(protocol.Request{Tool: tool, Params: params})
	if err != nil {
		return nil, fmt.Errorf("encode %q request: %w", tool, err)
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = newDefaultHTTPClient()
		c.HTTP = httpClient
	}

	resp, err := httpClient.Post(c.BaseURL+"/rpc", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("post %q: %w", tool, err)
	}
	defer resp.Body.Close()

	var out protocol.Response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode %q response: %w", tool, err)
	}
	return &out, nil
}

func newDefaultHTTPClient() *http.Client {
	return &http.Client{Timeout: defaultTimeout}
}
