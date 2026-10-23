package cmd

import (
	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
	"time"
)

const maxAssertionWaitMS = 30000

// Extend only this request, preserving the caller's transport and later timeouts.
func postWithAssertionWait(c *client.Client, tool string, params map[string]any) (*protocol.Response, error) {
	wait := assertionWait(tool, params)
	if tool == "batch" {
		forEachBatchChild(params, func(child string, values map[string]any) {
			wait += assertionWait(child, values)
		})
	}
	if wait > 0 {
		requestClient := *c
		if c.HTTP == nil {
			requestClient.HTTP = client.New(c.BaseURL).HTTP
		} else {
			httpClient := *c.HTTP
			requestClient.HTTP = &httpClient
		}
		minimum := wait + 5*time.Second
		if requestClient.HTTP.Timeout > 0 {
			if tool == "batch" {
				requestClient.HTTP.Timeout += wait
			}
			if requestClient.HTTP.Timeout < minimum {
				requestClient.HTTP.Timeout = minimum
			}
		}
		return requestClient.Post(tool, params)
	}
	return c.Post(tool, params)
}

// Batch JSON numbers decode as float64; direct CLI and QA use int.
func assertionWait(tool string, params map[string]any) time.Duration {
	if tool != "game" || params["action"] != "assert" {
		return 0
	}
	var n float64
	switch value := params["within_ms"].(type) {
	case int:
		n = float64(value)
	case float64:
		n = value
	default:
		return 0
	}
	if n < 1 || n > maxAssertionWaitMS || n != float64(int(n)) {
		return 0
	}
	return time.Duration(n) * time.Millisecond
}
