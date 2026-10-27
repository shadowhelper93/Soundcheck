// Package ws provides WebSocket handlers for interactive terminal streaming.
package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/soundcheck/soundcheck/internal/edge/exec"
)

// ClientMessage represents an incoming message from the WebSocket client.
type ClientMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// ChunkEvent represents terminal output streamed to the client.
type ChunkEvent struct {
	Type      string `json:"type"`
	Stream    string `json:"stream"`
	Data      string `json:"data"`
	SessionID string `json:"sessionId"`
	Timestamp int64  `json:"timestamp"`
}

// ExitEvent signals terminal process termination.
type ExitEvent struct {
	Type     string `json:"type"`
	ExitCode int    `json:"exitCode"`
}

// TerminalHandler handles interactive WebSocket PTY sessions.
type TerminalHandler struct {
	runner exec.PTYRunner
}

// NewTerminalHandler creates a new TerminalHandler.
func NewTerminalHandler(runner exec.PTYRunner) *TerminalHandler {
	if runner == nil {
		runner = exec.NewPTYRunner()
	}
	return &TerminalHandler{runner: runner}
}

func (h *TerminalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	defer func() { _ = c.Close(websocket.StatusNormalClosure, "") }()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	cmd, sz := parsePTYRequest(r)
	sess, err := h.runner.Start(ctx, cmd, sz)
	if err != nil {
		_ = wsjson.Write(ctx, c, map[string]string{"type": "error", "error": err.Error()})
		return
	}
	defer func() { _ = sess.Close() }()

	runBridge(ctx, c, sess)
}

func parsePTYRequest(r *http.Request) (exec.Cmd, exec.Winsize) {
	q := r.URL.Query()
	bin := q.Get("cmd")
	if bin == "" {
		bin = defaultShell()
	}
	cols, _ := strconv.ParseUint(q.Get("cols"), 10, 16)
	rows, _ := strconv.ParseUint(q.Get("rows"), 10, 16)
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}
	return exec.Cmd{Binary: bin, Args: parseArgs(q)}, exec.Winsize{Cols: uint16(cols), Rows: uint16(rows)}
}

func parseArgs(q url.Values) []string {
	rawArgs := q["args"]
	if len(rawArgs) > 1 {
		return rawArgs
	}
	if len(rawArgs) == 1 && rawArgs[0] != "" {
		return strings.Fields(rawArgs[0])
	}
	return nil
}

func defaultShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "/bin/sh"
}

func runBridge(ctx context.Context, c *websocket.Conn, sess exec.PTYSession) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer cancel()
		pumpEvents(ctx, c, sess)
	}()
	go func() {
		defer wg.Done()
		defer cancel()
		pumpWS(ctx, c, sess)
	}()
	wg.Wait()
}

func pumpEvents(ctx context.Context, c *websocket.Conn, sess exec.PTYSession) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-sess.Events():
			if !ok {
				return
			}
			if err := writeEvent(ctx, c, sess.ID(), ev); err != nil {
				return
			}
		}
	}
}

func writeEvent(ctx context.Context, c *websocket.Conn, id string, ev exec.Event) error {
	if ev.Type == exec.EventOutput {
		return wsjson.Write(ctx, c, ChunkEvent{
			Type:      "terminal_chunk",
			Stream:    "pty",
			Data:      string(ev.Data),
			SessionID: id,
			Timestamp: time.Now().UnixMilli(),
		})
	}
	if ev.Type == exec.EventExit {
		return wsjson.Write(ctx, c, ExitEvent{
			Type:     "exit",
			ExitCode: ev.ExitCode,
		})
	}
	return nil
}

func pumpWS(ctx context.Context, c *websocket.Conn, sess exec.PTYSession) {
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		handleClientMessage(sess, data)
	}
}

func handleClientMessage(sess exec.PTYSession, data []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err == nil && msg.Type != "" {
		if msg.Type == "resize" && (msg.Cols > 0 || msg.Rows > 0) {
			_ = sess.Resize(msg.Cols, msg.Rows)
			return
		}
		if msg.Type == "input" {
			_, _ = sess.Write([]byte(msg.Data))
			return
		}
	}
	_, _ = sess.Write(data)
}
