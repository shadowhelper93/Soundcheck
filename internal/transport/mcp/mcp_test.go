package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/soundcheck/soundcheck/internal/adapters"
	"github.com/soundcheck/soundcheck/internal/app"
	"github.com/soundcheck/soundcheck/internal/core/domain"
	edgeExec "github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/store"
	"github.com/soundcheck/soundcheck/internal/transport/mcp"
)

var (
	fakecliOnce sync.Once
	fakecliPath string
	fakecliErr  error
)

func getFakeCLI(t testing.TB) string {
	t.Helper()
	fakecliOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "fakecli-mcp-*")
		if err != nil {
			fakecliErr = err
			return
		}
		bin := filepath.Join(tmpDir, "fakecli")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		root := findModuleRoot(t)
		// #nosec G204
		cmd := exec.Command("go", "build", "-o", bin, filepath.Join(root, "testdata", "fakecli"))
		if out, err := cmd.CombinedOutput(); err != nil {
			fakecliErr = fmt.Errorf("build fakecli: %w: %s", err, string(out))
			return
		}
		fakecliPath = bin
	})
	if fakecliErr != nil {
		t.Fatalf("failed to build fakecli: %v", fakecliErr)
	}
	return fakecliPath
}

func findModuleRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate go.mod")
		}
		dir = parent
	}
}

func testDefinitions(bin string) []adapters.Definition {
	agentIDs := []string{"claude", "codex", "opencode", "agy", "gemini", "copilot"}
	defs := make([]adapters.Definition, len(agentIDs))

	for i, id := range agentIDs {
		orig, _ := adapters.Get(id)
		defs[i] = adapters.Definition{
			Adapter: domain.Adapter{
				ID:              id,
				Name:            orig.Name,
				Vendor:          orig.Vendor,
				Binary:          bin,
				Category:        orig.Category,
				Models:          orig.Models,
				HeadlessCommand: bin,
			},
			ProbeSpecs: map[domain.Tier]domain.Probe{
				domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
				domain.TierL1: {Level: domain.TierL1, Command: bin},
				domain.TierL2: {Level: domain.TierL2, Command: bin},
				domain.TierL3: {Level: domain.TierL3, Command: bin},
			},
		}
	}
	return defs
}

func runMCPExchange(t *testing.T, input string) []mcp.Response {
	t.Helper()
	bin := getFakeCLI(t)
	defs := testDefinitions(bin)
	runner := edgeExec.NewPipeRunner()
	cache := store.NewTTLCache(nil)
	appInstance, err := app.New(
		app.WithRunner(runner),
		app.WithCache(cache),
		app.WithDefinitions(defs),
	)
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
