package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/transport/mcp"
)

func runMCPExchange(t *testing.T, input string) []mcp.Response {
	t.Helper()
	appInstance, err := app.New()
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	in := strings.NewReader(input)
	out := &bytes.Buffer{}
	if err := mcp.ServeStdio(context.Background(), appInstance, in, out); err != nil {
		t.Fatalf("ServeStdio failed: %v", err)
	}

	var responses []mcp.Response
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var res mcp.Response
		if err := json.Unmarshal([]byte(line), &res); err != nil {
			t.Fatalf("failed to parse response %q: %v", line, err)
		}
		responses = append(responses, res)
	}
	return responses
}

func TestMCPInitializeAndPing(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n"

	responses := runMCPExchange(t, input)
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses (notification skipped), got %d", len(responses))
	}

	if responses[0].ID.(float64) != 1 {
		t.Errorf("expected id 1, got %v", responses[0].ID)
	}
	initRes := responses[0].Result.(map[string]any)
	if initRes["protocolVersion"] != "2024-11-05" {
		t.Errorf("unexpected protocolVersion: %v", initRes["protocolVersion"])
	}

	if responses[1].ID.(float64) != 2 {
		t.Errorf("expected id 2, got %v", responses[1].ID)
	}
}

func TestMCPToolsList(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":"tools-1","method":"tools/list"}` + "\n"
	responses := runMCPExchange(t, input)

	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	resMap := responses[0].Result.(map[string]any)
	tools := resMap["tools"].([]any)
	if len(tools) < 4 {
		t.Errorf("expected at least 4 tools, got %d", len(tools))
	}
}

func TestMCPToolsCall(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"soundcheck_doctor","arguments":{"deep":false}}}` + "\n" +
		`{"jsonrpc":"2.0","id":"call-2","method":"tools/call","params":{"name":"soundcheck_manifest","arguments":{}}}` + "\n" +
		`{"jsonrpc":"2.0","id":"call-3","method":"tools/call","params":{"name":"soundcheck_resolve","arguments":{"role":"primary"}}}` + "\n" +
		`{"jsonrpc":"2.0","id":"call-4","method":"tools/call","params":{"name":"soundcheck_probe","arguments":{"adapter":"claude"}}}` + "\n"

	responses := runMCPExchange(t, input)
	if len(responses) != 4 {
		t.Fatalf("expected 4 tool call responses, got %d", len(responses))
	}

	for i, res := range responses {
		if res.Error != nil {
			t.Errorf("tool call %d returned unexpected error: %+v", i+1, res.Error)
		}
		resMap, ok := res.Result.(map[string]any)
		if !ok {
			t.Fatalf("tool call %d result is not map: %+v", i+1, res.Result)
		}
		content, ok := resMap["content"].([]any)
		if !ok || len(content) == 0 {
			t.Errorf("tool call %d returned empty content", i+1)
		}
	}
}

func TestMCPMethodNotFound(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":99,"method":"nonexistent_method"}` + "\n"
	responses := runMCPExchange(t, input)

	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}
	if responses[0].Error == nil || responses[0].Error.Code != -32601 {
		t.Errorf("expected -32601 Method not found, got %+v", responses[0].Error)
	}
}
