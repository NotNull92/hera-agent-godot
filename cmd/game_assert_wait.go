package cmd

import (
	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
	"time"
)

// Extend only this request, preserving the caller's transport and later timeouts.
func postWithAssertionWait(c *client.Client, tool string, params map[string]any) (*protocol.Response, error) {
	if tool == "game" && params["action"] == "assert" {
		n, ok := params["within_ms"].(int)
		if ok && n >= 1 && n <= maxWaitMS {
			requestClient := *c
			if c.HTTP == nil {
				requestClient.HTTP = client.New(c.BaseURL).HTTP
			} else {
				httpClient := *c.HTTP
				requestClient.HTTP = &httpClient
			}
			minimum := time.Duration(n)*time.Millisecond + 5*time.Second
			if requestClient.HTTP.Timeout > 0 && requestClient.HTTP.Timeout < minimum {
				requestClient.HTTP.Timeout = minimum
			}
			return requestClient.Post(tool, params)
		}
	}
	return c.Post(tool, params)
}
