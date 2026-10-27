// Package mcp implements the Model Context Protocol stdio JSON-RPC server.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/soundcheck/soundcheck/internal/app"
)

// Request defines a standard JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response defines a standard JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents a JSON-RPC protocol error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolCallContent holds text content returned by an MCP tool invocation.
type ToolCallContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolCallResult defines the result envelope for tools/call.
type ToolCallResult struct {
	Content []ToolCallContent `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}

// Server provides the MCP stdio JSON-RPC server.
type Server struct {
	app *app.App
	mu  sync.Mutex
	w   io.Writer
}

// NewServer initializes a new MCP Server.
func NewServer(a *app.App, w io.Writer) *Server {
	return &Server{app: a, w: w}
}

// ServeStdio starts reading newline-delimited JSON-RPC from r and writes to w.
func ServeStdio(ctx context.Context, a *app.App, r io.Reader, w io.Writer) error {
	s := NewServer(a, w)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := s.handleLine(ctx, line); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) handleLine(ctx context.Context, line []byte) error {
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return s.sendResponse(&Response{
			JSONRPC: "2.0",
			Error:   &Error{Code: -32700, Message: "Parse error"},
		})
	}
	if req.ID == nil && req.Method == "notifications/initialized" {
		return nil
	}
	res := s.dispatch(ctx, req)
	if res != nil {
		return s.sendResponse(res)
	}
	return nil
}

func (s *Server) sendResponse(res *Response) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = s.w.Write(b)
	return err
}

func (s *Server) dispatch(ctx context.Context, req Request) *Response {
	switch req.Method {
	case "initialize":
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: initResult()}
	case "ping":
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": listTools()}}
	case "tools/call":
		res, err := s.callTool(ctx, req.Params)
		if err != nil {
			return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32603, Message: err.Error()}}
		}
		return &Response{JSONRPC: "2.0", ID: req.ID, Result: res}
	default:
		return &Response{JSONRPC: "2.0", ID: req.ID, Error: &Error{Code: -32601, Message: "Method not found"}}
	}
}

func initResult() map[string]any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]string{
			"name":    "soundcheck",
			"version": "1.4.0",
		},
	}
}

func listTools() []map[string]any {
	return []map[string]any{
		toolDef("soundcheck_doctor", "Run diagnostic health checks across CLI adapters and capabilities", map[string]any{
			"deep":  map[string]any{"type": "boolean", "description": "Execute deep L3 probing"},
			"force": map[string]any{"type": "boolean", "description": "Bypass cache and force live probe"},
		}),
		toolDef("soundcheck_probe", "Probe a specific CLI adapter or all adapters", map[string]any{
			"adapter": map[string]any{"type": "string", "description": "ID of the adapter to probe"},
		}),
		toolDef("soundcheck_resolve", "Resolve CLI tools and capabilities for a specified role or task", map[string]any{
			"role": map[string]any{"type": "string", "description": "Role to resolve (primary, subagent, auxiliary, reviewer)"},
			"task": map[string]any{"type": "string", "description": "Task description or capability requirement"},
		}),
		toolDef("soundcheck_manifest", "Get capability manifest of all available CLI tools and models", map[string]any{}),
	}
}

func toolDef(name, desc string, props map[string]any) map[string]any {
	return map[string]any{
		"name":        name,
		"description": desc,
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": props,
		},
	}
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (s *Server) callTool(ctx context.Context, rawParams json.RawMessage) (*ToolCallResult, error) {
	var p toolCallParams
	if err := json.Unmarshal(rawParams, &p); err != nil {
		return nil, errors.New("invalid tool call params")
	}
	switch p.Name {
	case "soundcheck_doctor":
		return s.toolDoctor(ctx, p.Arguments)
	case "soundcheck_probe":
		return s.toolProbe(ctx, p.Arguments)
	case "soundcheck_resolve":
		return s.toolResolve(ctx, p.Arguments)
	case "soundcheck_manifest":
		return s.toolManifest(ctx)
	default:
		return nil, fmt.Errorf("unknown tool: %s", p.Name)
	}
}

func (s *Server) toolDoctor(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a struct {
		Deep  bool `json:"deep"`
		Force bool `json:"force"`
	}
	_ = json.Unmarshal(args, &a)
	var opts []app.DoctorOption
	if a.Deep {
		opts = append(opts, app.WithDoctorDeep(true))
	}
	if a.Force {
		opts = append(opts, app.WithDoctorForce(true))
	}
	report, err := s.app.Doctor(ctx, opts...)
	if err != nil {
		return &ToolCallResult{IsError: true, Content: []ToolCallContent{{Type: "text", Text: err.Error()}}}, nil
	}
	b, _ := json.MarshalIndent(report, "", "  ")
	return &ToolCallResult{Content: []ToolCallContent{{Type: "text", Text: string(b)}}}, nil
}

func (s *Server) toolProbe(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a struct {
		Adapter string `json:"adapter"`
	}
	_ = json.Unmarshal(args, &a)
	if a.Adapter != "" {
		ad, err := s.app.ProbeOne(ctx, a.Adapter)
		if err != nil {
			return &ToolCallResult{IsError: true, Content: []ToolCallContent{{Type: "text", Text: err.Error()}}}, nil
		}
		b, _ := json.MarshalIndent(ad, "", "  ")
		return &ToolCallResult{Content: []ToolCallContent{{Type: "text", Text: string(b)}}}, nil
	}
	report, err := s.app.Doctor(ctx)
	if err != nil {
		return &ToolCallResult{IsError: true, Content: []ToolCallContent{{Type: "text", Text: err.Error()}}}, nil
	}
	b, _ := json.MarshalIndent(report.Adapters, "", "  ")
	return &ToolCallResult{Content: []ToolCallContent{{Type: "text", Text: string(b)}}}, nil
}

func (s *Server) toolResolve(ctx context.Context, args json.RawMessage) (*ToolCallResult, error) {
	var a struct {
		Role string `json:"role"`
	}
	_ = json.Unmarshal(args, &a)
	res, err := s.app.Resolve(ctx, a.Role)
	if err != nil {
		return &ToolCallResult{IsError: true, Content: []ToolCallContent{{Type: "text", Text: err.Error()}}}, nil
	}
	b, _ := json.MarshalIndent(res, "", "  ")
	return &ToolCallResult{Content: []ToolCallContent{{Type: "text", Text: string(b)}}}, nil
}

func (s *Server) toolManifest(ctx context.Context) (*ToolCallResult, error) {
	m, err := s.app.Manifest(ctx)
	if err != nil {
		return &ToolCallResult{IsError: true, Content: []ToolCallContent{{Type: "text", Text: err.Error()}}}, nil
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return &ToolCallResult{Content: []ToolCallContent{{Type: "text", Text: string(b)}}}, nil
}
