package ws_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/transport/ws"
)

type mockSession struct {
	mu       sync.Mutex
	id       string
	written  []byte
	resized  []exec.Winsize
	eventsCh chan exec.Event
	closed   bool
}

func newMockSession() *mockSession {
	return &mockSession{
		id:       "mock-sess-1",
		eventsCh: make(chan exec.Event, 10),
	}
}

func (s *mockSession) ID() string { return s.id }

func (s *mockSession) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.written = append(s.written, p...)
	return len(p), nil
}

func (s *mockSession) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resized = append(s.resized, exec.Winsize{Cols: cols, Rows: rows})
	return nil
}

func (s *mockSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.eventsCh)
	}
	return nil
}

func (s *mockSession) Wait() (int, error) { return 0, nil }

func (s *mockSession) Events() <-chan exec.Event { return s.eventsCh }

func (s *mockSession) History() []byte { return nil }

type mockRunner struct {
	sess    *mockSession
	err     error
	lastCmd exec.Cmd
}

func (r *mockRunner) Start(_ context.Context, cmd exec.Cmd, _ exec.Winsize) (exec.PTYSession, error) {
	r.lastCmd = cmd
	if r.err != nil {
		return nil, r.err
	}
	return r.sess, nil
}

func testSendClientMessages(ctx context.Context, conn *websocket.Conn) error {
	if err := wsjson.Write(ctx, conn, ws.ClientMessage{Type: "input", Data: "ls -la\n"}); err != nil {
		return err
	}
	return wsjson.Write(ctx, conn, ws.ClientMessage{Type: "resize", Cols: 120, Rows: 40})
}

func testReceiveSessionEvents(ctx context.Context, conn *websocket.Conn, sess *mockSession) error {
	sess.eventsCh <- exec.Event{Type: exec.EventOutput, Data: []byte("total 42\n")}
	var chunk ws.ChunkEvent
	if err := wsjson.Read(ctx, conn, &chunk); err != nil {
		return err
	}
	sess.eventsCh <- exec.Event{Type: exec.EventExit, ExitCode: 0}
	var exitEv ws.ExitEvent
	return wsjson.Read(ctx, conn, &exitEv)
}

func TestTerminalHandlerBidirectional(t *testing.T) {
	sess := newMockSession()
	runner := &mockRunner{sess: sess}
	srv := httptest.NewServer(ws.NewTerminalHandler(runner))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	if err := testSendClientMessages(ctx, conn); err != nil {
		t.Fatalf("failed to send: %v", err)
	}
	if err := testReceiveSessionEvents(ctx, conn, sess); err != nil {
		t.Fatalf("failed to receive: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if string(sess.written) != "ls -la\n" {
		t.Errorf("expected written 'ls -la\\n', got %q", string(sess.written))
	}
	if len(sess.resized) != 1 || sess.resized[0].Cols != 120 || sess.resized[0].Rows != 40 {
		t.Errorf("expected resize to 120x40, got %+v", sess.resized)
	}
}

func TestTerminalHandlerArgsParsing(t *testing.T) {
	runner := &mockRunner{sess: newMockSession()}
	srv := httptest.NewServer(ws.NewTerminalHandler(runner))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "?args=hello%20world&args=again"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")

	if len(runner.lastCmd.Args) != 2 || runner.lastCmd.Args[0] != "hello world" || runner.lastCmd.Args[1] != "again" {
		t.Errorf("expected multi-word repeated args, got %+v", runner.lastCmd.Args)
	}
}

func TestTerminalHandlerArgsFallback(t *testing.T) {
	runner := &mockRunner{sess: newMockSession()}
	srv := httptest.NewServer(ws.NewTerminalHandler(runner))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "?args=-l%20-a"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")

	if len(runner.lastCmd.Args) != 2 || runner.lastCmd.Args[0] != "-l" || runner.lastCmd.Args[1] != "-a" {
		t.Errorf("expected split single arg fallback, got %+v", runner.lastCmd.Args)
	}
}

func TestTerminalHandlerQuotedArgs(t *testing.T) {
	runner := &mockRunner{sess: newMockSession()}
	srv := httptest.NewServer(ws.NewTerminalHandler(runner))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + `?args=%22hello%20world%22`
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")

	if len(runner.lastCmd.Args) != 1 || runner.lastCmd.Args[0] != "hello world" {
		t.Errorf("expected quoted multi-word arg ['hello world'], got %+v", runner.lastCmd.Args)
	}
}
