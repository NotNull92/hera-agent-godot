// Package dap is a small Debug Adapter Protocol client: Content-Length framing,
// request/response correlation and bounded event buffering. It speaks to
// Godot's GDScript debug adapter, which accepts an "attach" request for a
// session created by the editor debug server.
package dap

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Message is one decoded DAP message.
type Message map[string]any

// String returns a field as a string.
func (m Message) String(key string) string {
	value, _ := m[key].(string)
	return value
}

// Int returns a field as an int.
func (m Message) Int(key string) int {
	if value, ok := m[key].(float64); ok {
		return int(value)
	}
	return 0
}

// Bool returns a field as a bool.
func (m Message) Bool(key string) bool {
	value, _ := m[key].(bool)
	return value
}

// Map returns a nested object.
func (m Message) Map(key string) Message {
	if value, ok := m[key].(map[string]any); ok {
		return Message(value)
	}
	return nil
}

// List returns a nested array of objects.
func (m Message) List(key string) []Message {
	raw, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]Message, 0, len(raw))
	for _, item := range raw {
		if entry, ok := item.(map[string]any); ok {
			out = append(out, Message(entry))
		}
	}
	return out
}

const (
	// MaxFrameBytes bounds one decoded message body.
	MaxFrameBytes = 16 << 20
	// maxHeaderBytes bounds the header block before Content-Length is parsed.
	maxHeaderBytes = 8 << 10
	// maxPendingMessages and maxPendingBytes bound buffered events.
	maxPendingMessages = 512
	maxPendingBytes    = 4 << 20
)

// ErrBroken marks a connection that must not be reused because a partial frame
// was consumed when a deadline expired.
var ErrBroken = errors.New("dap connection state is no longer trustworthy after a partial frame read")

// ProtocolError is a response the adapter rejected or mis-addressed.
type ProtocolError struct {
	Command string
	Message string
}

func (e *ProtocolError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("the adapter rejected %q", e.Command)
	}
	return fmt.Sprintf("the adapter rejected %q: %s", e.Command, e.Message)
}

// Client is a DAP connection with a bounded read deadline per call.
type Client struct {
	conn         net.Conn
	reader       *bufio.Reader
	seq          int
	pending      []Message
	pendingBytes int
	broken       error
}

// Dial connects to a DAP endpoint.
func Dial(address string, timeout time.Duration) (*Client, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}
	return &Client{conn: conn, reader: bufio.NewReader(conn)}, nil
}

// Close closes the connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// send writes one framed message with both deadlines set, so a peer that stops
// reading cannot block the caller.
func (c *Client) send(message Message, timeout time.Duration) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if timeout > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			return err
		}
		defer c.conn.SetWriteDeadline(time.Time{})
	}
	_, err = fmt.Fprintf(c.conn, "Content-Length: %d\r\n\r\n%s", len(body), body)
	return err
}

// readHeaderLine reads one header line with an explicit byte budget, so an
// unterminated line cannot grow memory before the limit is checked.
func (c *Client) readHeaderLine(budget *int) (string, error) {
	var builder []byte
	for {
		fragment, err := c.reader.ReadSlice('\n')
		*budget -= len(fragment)
		builder = append(builder, fragment...)
		if *budget < 0 {
			return "", c.poison(fmt.Errorf("DAP header exceeded %d bytes", maxHeaderBytes))
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil {
			return string(builder), err
		}
		return string(builder), nil
	}
}

// read decodes the next message. A deadline that expires after part of a frame
// was consumed poisons the connection, because the remainder of that frame would
// be mistaken for a new one.
func (c *Client) read(timeout time.Duration) (Message, error) {
	if c.broken != nil {
		return nil, c.broken
	}
	if err := c.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	consumed := false
	budget := maxHeaderBytes
	length := -1
	for {
		line, err := c.readHeaderLine(&budget)
		if len(line) > 0 {
			consumed = true
		}
		if err != nil {
			return nil, c.frameError(err, consumed)
		}
		header := strings.TrimRight(line, "\r\n")
		if header == "" {
			break
		}
		if value, ok := strings.CutPrefix(strings.ToLower(header), "content-length:"); ok {
			parsed, convErr := strconv.Atoi(strings.TrimSpace(value))
			if convErr != nil {
				return nil, c.poison(fmt.Errorf("invalid Content-Length: %q", header))
			}
			length = parsed
		}
	}
	if length < 0 {
		return nil, c.poison(fmt.Errorf("missing Content-Length header"))
	}
	if length > MaxFrameBytes {
		return nil, c.poison(fmt.Errorf("frame of %d bytes exceeds the %d-byte limit", length, MaxFrameBytes))
	}
	buffer := make([]byte, length)
	if _, err := readFull(c.reader, buffer); err != nil {
		return nil, c.frameError(err, true)
	}
	var message Message
	if err := json.Unmarshal(buffer, &message); err != nil {
		return nil, c.poison(fmt.Errorf("invalid DAP body: %w", err))
	}
	return message, nil
}

// frameError records a read failure, poisoning the connection when part of a
// frame was already consumed.
func (c *Client) frameError(err error, consumed bool) error {
	if isTimeout(err) && !consumed {
		// Nothing was consumed, so the next read can safely start fresh.
		return err
	}
	return c.poison(err)
}

func (c *Client) poison(err error) error {
	if c.broken == nil {
		c.broken = fmt.Errorf("%w (%v)", ErrBroken, err)
		if c.conn != nil {
			// Close so a peer cannot act on anything Hera would not read.
			_ = c.conn.Close()
		}
	}
	return c.broken
}

// usable reports the connection state before any outbound or buffered operation.
func (c *Client) usable() error {
	if c.broken != nil {
		return c.broken
	}
	return nil
}

func isTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func readFull(reader *bufio.Reader, buffer []byte) (int, error) {
	total := 0
	for total < len(buffer) {
		n, err := reader.Read(buffer[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// buffer keeps an event for later reads, with an explicit bound: a noisy
// debuggee must not consume unbounded memory while a command waits.
func (c *Client) buffer(message Message) error {
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(c.pending) >= maxPendingMessages || c.pendingBytes+len(encoded) > maxPendingBytes {
		return c.poison(fmt.Errorf("buffered DAP events exceeded the bound of %d messages / %d bytes", maxPendingMessages, maxPendingBytes))
	}
	c.pending = append(c.pending, message)
	c.pendingBytes += len(encoded)
	return nil
}

// Request sends a request and returns its successful response. A response that
// is rejected, mis-addressed or malformed is an error. Events that arrive while
// waiting are buffered for later reads.
func (c *Client) Request(command string, arguments map[string]any, timeout time.Duration) (Message, error) {
	if err := c.usable(); err != nil {
		return nil, err
	}
	c.seq++
	seq := c.seq
	request := Message{"type": "request", "seq": seq, "command": command}
	if len(arguments) > 0 {
		request["arguments"] = arguments
	}
	if err := c.send(request, timeout); err != nil {
		return nil, c.poison(err)
	}
	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("no response to %q within %s", command, timeout)
		}
		message, err := c.read(remaining)
		if err != nil {
			return nil, err
		}
		switch message.String("type") {
		case "response":
			if message.Int("request_seq") != seq {
				if err := c.buffer(message); err != nil {
					return nil, err
				}
				continue
			}
			if !message.Bool("success") {
				reason := message.String("message")
				if body := message.Map("body"); body != nil {
					if text := body.String("error"); text != "" {
						reason = text
					}
				}
				return nil, &ProtocolError{Command: command, Message: reason}
			}
			if reported := message.String("command"); reported != "" && reported != command {
				return nil, c.poison(fmt.Errorf("response %d reports command %q for %q", seq, reported, command))
			}
			return message, nil
		case "event":
			if err := c.buffer(message); err != nil {
				return nil, err
			}
		default:
			if err := c.buffer(message); err != nil {
				return nil, err
			}
		}
	}
}

// NextEvent returns the next event of any name, buffered ones first, so a caller
// can react to several possible events in arrival order.
func (c *Client) NextEvent(timeout time.Duration) (Message, error) {
	if err := c.usable(); err != nil {
		return nil, err
	}
	for i, message := range c.pending {
		if message.String("type") == "event" {
			encoded, _ := json.Marshal(message)
			c.pendingBytes -= len(encoded)
			c.pending = append(c.pending[:i], c.pending[i+1:]...)
			return message, nil
		}
	}
	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, ErrNotObserved
		}
		message, err := c.read(remaining)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			return nil, err
		}
		if message.String("type") == "event" {
			return message, nil
		}
	}
}

// ErrNotObserved means the requested event did not arrive inside the window;
// it says nothing about the debuggee's state.
var ErrNotObserved = errors.New("the event was not observed within the wait window")

// ReconcileToIdle consumes unread socket data and buffered events until the
// connection has been quiet for the idle window. It is bounded by an absolute
// budget, so steady output traffic (a game printing every frame) fails instead
// of hanging, and terminal events are returned rather than discarded.
//
// A quiet window reduces the chance that earlier events are attributed to a
// later request; it does not prove such attribution, which is why callers still
// require the adapter's own resume confirmation before accepting a stop.
func (c *Client) ReconcileToIdle(idle time.Duration, budget time.Duration) (int, Message, error) {
	if err := c.usable(); err != nil {
		return 0, nil, err
	}
	// Buffered messages are inspected before they are cleared: a terminal event
	// that arrived earlier is a reason to refuse the action.
	dropped := 0
	for _, message := range c.pending {
		if message.String("type") == "event" {
			if name := message.String("event"); name == "terminated" || name == "exited" {
				return dropped, message, nil
			}
			dropped++
		}
	}
	c.pending = nil
	c.pendingBytes = 0
	deadline := time.Now().Add(budget)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return dropped, nil, fmt.Errorf("the connection never went idle within %s, so the debuggee's state is unknown", budget)
		}
		window := idle
		if remaining < window {
			window = remaining
		}
		event, err := c.NextEvent(window)
		if err != nil {
			if errors.Is(err, ErrNotObserved) {
				if window < idle {
					// The budget ended before a full quiet window: no boundary.
					return dropped, nil, fmt.Errorf("the connection never went quiet for %s within the %s budget, so the debuggee's state is unknown", idle, budget)
				}
				return dropped, nil, nil
			}
			return dropped, nil, err
		}
		dropped++
		switch event.String("event") {
		case "terminated", "exited":
			return dropped, event, nil
		}
	}
}

// DrainEvents drops every buffered event of the given names and returns them, so
// an action can establish a boundary before it resumes the debuggee.
func (c *Client) DrainEvents(names ...string) []Message {
	wanted := map[string]bool{}
	for _, name := range names {
		wanted[name] = true
	}
	drained := []Message{}
	remaining := make([]Message, 0, len(c.pending))
	for _, message := range c.pending {
		if message.String("type") == "event" && wanted[message.String("event")] {
			encoded, _ := json.Marshal(message)
			c.pendingBytes -= len(encoded)
			drained = append(drained, message)
			continue
		}
		remaining = append(remaining, message)
	}
	c.pending = remaining
	return drained
}
