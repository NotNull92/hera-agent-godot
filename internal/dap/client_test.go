package dap

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeAdapter serves scripted frames over one connection.
func fakeAdapter(t *testing.T, script func(reader *bufio.Reader, writer *bufio.Writer)) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		script(bufio.NewReader(conn), bufio.NewWriter(conn))
	}()
	return listener.Addr().String(), func() {
		listener.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
}

// fakeAdapterConn is fakeAdapter with the raw connection, for tests that must
// probe the socket (a buffered-only check misses the kernel receive queue).
func fakeAdapterConn(t *testing.T, script func(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer)) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		script(conn, bufio.NewReader(conn), bufio.NewWriter(conn))
	}()
	return listener.Addr().String(), func() {
		listener.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
}

func writeFrame(writer *bufio.Writer, body string) {
	fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n%s", len(body), body)
	writer.Flush()
}

func readRequest(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	length := 0
	if strings.HasPrefix(line, "Content-Length:") {
		fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")), "%d", &length)
		reader.ReadString('\n')
	}
	buffer := make([]byte, length)
	for read := 0; read < length; {
		n, err := reader.Read(buffer[read:])
		read += n
		if err != nil {
			break
		}
	}
	return string(buffer)
}

func TestRequestCorrelatesAndBuffersEvents(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		writeFrame(writer, `{"type":"event","event":"initialized","seq":1}`)
		writeFrame(writer, `{"type":"response","request_seq":1,"command":"initialize","success":true,"seq":2}`)
		readRequest(reader)
		writeFrame(writer, `{"type":"response","request_seq":2,"command":"attach","success":true,"seq":3}`)
		time.Sleep(300 * time.Millisecond)
	})
	defer stop()

	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	response, err := client.Request("initialize", map[string]any{"clientID": "test"}, 2*time.Second)
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if response.String("command") != "initialize" || response.Int("seq") != 2 {
		t.Fatalf("unexpected response: %v", response)
	}
	event, err := client.Event("initialized", 2*time.Second)
	if err != nil {
		t.Fatalf("buffered event must be returned: %v", err)
	}
	if event.Int("seq") != 1 {
		t.Fatalf("unexpected event: %v", event)
	}
	if _, err := client.Request("attach", nil, 2*time.Second); err != nil {
		t.Fatalf("attach: %v", err)
	}
}

func TestEventTimesOutWithoutMatch(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		writeFrame(writer, `{"type":"response","request_seq":1,"command":"initialize","success":true,"seq":1}`)
		time.Sleep(400 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := client.Event("stopped", 300*time.Millisecond); err == nil {
		t.Fatal("a missing event must time out")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("event wait was not bounded: %s", elapsed)
	}
}

func TestReadRejectsMalformedFrames(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		fmt.Fprint(writer, "Content-Length: abc\r\n\r\n")
		writer.Flush()
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, time.Second); err == nil {
		t.Fatal("a malformed Content-Length must fail")
	}
}

func TestRejectedResponseIsAnError(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		writeFrame(writer, `{"type":"response","request_seq":1,"command":"attach","success":false,"message":"no active debug session"}`)
		time.Sleep(200 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_, err = client.Request("attach", nil, 2*time.Second)
	if err == nil {
		t.Fatal("success:false must fail the request")
	}
	if !strings.Contains(err.Error(), "no active debug session") {
		t.Fatalf("the adapter message must be preserved: %v", err)
	}
}

func TestMisaddressedResponseIsRejected(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		// The right sequence number but a different command name.
		writeFrame(writer, `{"type":"response","request_seq":1,"command":"initialize","success":true}`)
		time.Sleep(200 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("attach", nil, time.Second); err == nil {
		t.Fatal("a response naming another command must fail")
	}
}

func TestPartialFramePoisonsTheConnection(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		// Half of a body arrives, then silence: the rest would later look like a
		// new header on the same connection.
		fmt.Fprint(writer, "Content-Length: 40\r\n\r\n{\"type\":\"resp")
		writer.Flush()
		time.Sleep(1500 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, 300*time.Millisecond); err == nil {
		t.Fatal("a partial frame with an expired deadline must fail")
	}
	// Every later call on the same connection must fail instead of misparsing.
	if _, err := client.Request("stackTrace", nil, time.Second); err == nil || !errors.Is(err, ErrBroken) {
		t.Fatalf("the connection must stay broken: %v", err)
	}
}

func TestOversizedHeaderIsRejected(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		for i := 0; i < 40; i++ {
			fmt.Fprintf(writer, "X-Pad-%02d: %s\r\n", i, strings.Repeat("p", 400))
		}
		writer.Flush()
		time.Sleep(300 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, 2*time.Second); err == nil {
		t.Fatal("an oversized header must be rejected")
	}
}

func TestEventFloodIsBounded(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		for i := 0; i < maxPendingMessages+50; i++ {
			writeFrame(writer, fmt.Sprintf(`{"type":"event","event":"output","seq":%d,"body":{"output":"noise"}}`, i))
		}
		time.Sleep(300 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, 2*time.Second); err == nil {
		t.Fatal("an unbounded event flood must fail instead of growing memory")
	}
}

func TestNextEventReturnsArrivalOrder(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		writeFrame(writer, `{"type":"response","request_seq":1,"command":"initialize","success":true}`)
		writeFrame(writer, `{"type":"event","event":"continued","seq":2,"body":{"threadId":1}}`)
		writeFrame(writer, `{"type":"event","event":"stopped","seq":3,"body":{"reason":"breakpoint","threadId":1}}`)
		time.Sleep(300 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	first, err := client.NextEvent(2 * time.Second)
	if err != nil || first.String("event") != "continued" {
		t.Fatalf("first event=%v err=%v", first, err)
	}
	second, err := client.NextEvent(2 * time.Second)
	if err != nil || second.String("event") != "stopped" {
		t.Fatalf("second event=%v err=%v", second, err)
	}
	if _, err := client.NextEvent(200 * time.Millisecond); !errors.Is(err, ErrNotObserved) {
		t.Fatalf("an exhausted queue must report not-observed: %v", err)
	}
}

func TestSingleUnterminatedHeaderIsBounded(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		// One header line that never ends: the reader must refuse it before the
		// sender finishes, so the line cannot grow without bound.
		fmt.Fprint(writer, "X-Huge: "+strings.Repeat("p", 64<<10))
		writer.Flush()
		time.Sleep(500 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	start := time.Now()
	if _, err := client.Request("initialize", nil, 5*time.Second); err == nil {
		t.Fatal("an unterminated oversized header must be refused")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("the limit must trigger before the sender stops: %s", elapsed)
	}
}

func TestPoisonedClientRefusesFurtherRequests(t *testing.T) {
	peerSaw := make(chan string, 1)
	address, stop := fakeAdapterConn(t, func(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		fmt.Fprint(writer, "Content-Length: 40\r\n\r\n{\"type\":\"resp")
		writer.Flush()
		// Give the client time to poison, then probe the socket (a buffered-only
		// check cannot see data still in the kernel receive queue).
		time.Sleep(1500 * time.Millisecond)
		conn.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
		extra := ""
		if peeked, err := reader.Peek(1); err == nil && len(peeked) > 0 {
			extra = string(peeked)
		}
		peerSaw <- extra
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, 250*time.Millisecond); err == nil {
		t.Fatal("the partial frame must fail")
	}
	if _, err := client.Request("continue", nil, time.Second); !errors.Is(err, ErrBroken) {
		t.Fatalf("a poisoned client must refuse further requests: %v", err)
	}
	if _, err := client.NextEvent(100 * time.Millisecond); err == nil {
		t.Fatal("a poisoned client must not report buffered success")
	}
	select {
	case extra := <-peerSaw:
		if extra != "" {
			t.Fatalf("a request reached the peer after poisoning: %q", extra)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the peer never reported its observation")
	}
}

// continuousOutputConn keeps input available without scheduler-created idle gaps.
type continuousOutputConn struct {
	net.Conn
	frame  string
	offset int
}

func (c *continuousOutputConn) Read(p []byte) (int, error) {
	n := copy(p, c.frame[c.offset:])
	c.offset = (c.offset + n) % len(c.frame)
	return n, nil
}

func (c *continuousOutputConn) SetReadDeadline(time.Time) error { return nil }

func TestReconcileToIdleIsBoundedUnderContinuousOutput(t *testing.T) {
	body := `{"type":"event","event":"output","body":{"output":"noise"}}`
	conn := &continuousOutputConn{frame: fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)}
	client := &Client{conn: conn, reader: bufio.NewReader(conn)}
	start := time.Now()
	dropped, terminal, err := client.ReconcileToIdle(50*time.Millisecond, 300*time.Millisecond)
	if err == nil {
		t.Fatal("steady output must fail the boundary instead of hanging")
	}
	if dropped == 0 || terminal != nil || !strings.Contains(err.Error(), "never went") {
		t.Fatalf("continuous valid events must exhaust the budget: dropped=%d terminal=%v err=%v", dropped, terminal, err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("reconciliation was not bounded: %s", elapsed)
	}
}

func TestReconcileToIdleReportsTerminalEvents(t *testing.T) {
	address, stop := fakeAdapter(t, func(reader *bufio.Reader, writer *bufio.Writer) {
		readRequest(reader)
		writeFrame(writer, `{"type":"response","request_seq":1,"command":"initialize","success":true}`)
		writeFrame(writer, `{"type":"event","event":"terminated"}`)
		time.Sleep(300 * time.Millisecond)
	})
	defer stop()
	client, err := Dial(address, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Request("initialize", nil, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	_, terminal, err := client.ReconcileToIdle(50*time.Millisecond, time.Second)
	if err != nil {
		t.Fatalf("reconciliation should succeed: %v", err)
	}
	if terminal == nil || terminal.String("event") != "terminated" {
		t.Fatalf("a terminal event must be preserved: %v", terminal)
	}
}

func TestDrainEventsEstablishesAnActionBoundary(t *testing.T) {
	client := &Client{}
	for _, name := range []string{"stopped", "output", "stopped"} {
		if err := client.buffer(Message{"type": "event", "event": name}); err != nil {
			t.Fatal(err)
		}
	}
	drained := client.DrainEvents("stopped")
	if len(drained) != 2 || client.Buffered() != 1 {
		t.Fatalf("drained=%d buffered=%d", len(drained), client.Buffered())
	}
	if _, err := client.NextEvent(50 * time.Millisecond); err != nil {
		t.Fatalf("the remaining output event should still be readable: %v", err)
	}
}

func TestNestedAccessors(t *testing.T) {
	message := Message{
		"body": map[string]any{
			"scene": "res://Main.tscn",
			"count": float64(3),
			"scopes": []any{
				map[string]any{"name": "Locals", "variablesReference": float64(1)},
				"ignored",
			},
		},
	}
	body := message.Map("body")
	if body.String("scene") != "res://Main.tscn" || body.Int("count") != 3 {
		t.Fatalf("unexpected body: %v", body)
	}
	scopes := body.List("scopes")
	if len(scopes) != 1 || scopes[0].String("name") != "Locals" || scopes[0].Int("variablesReference") != 1 {
		t.Fatalf("unexpected scopes: %v", scopes)
	}
	if message.Map("missing") != nil || message.List("missing") != nil || message.String("missing") != "" {
		t.Fatal("missing fields must be empty")
	}
}
