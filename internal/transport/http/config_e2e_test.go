package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	"github.com/soundcheck/soundcheck/internal/core/resolve"
	edgeExec "github.com/soundcheck/soundcheck/internal/edge/exec"
	"github.com/soundcheck/soundcheck/internal/edge/store"
	transphttp "github.com/soundcheck/soundcheck/internal/transport/http"
)

var (
	e2eFakecliOnce sync.Once
	e2eFakecliPath string
	e2eFakecliErr  error
)

func getE2EFakeCLI(t testing.TB) string {
	t.Helper()
	e2eFakecliOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "fakecli-e2e-*")
		if err != nil {
			e2eFakecliErr = err
			return
		}
		bin := filepath.Join(tmpDir, "fakecli")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		root := findE2EModuleRoot(t)
		// #nosec G204
		cmd := exec.Command("go", "build", "-o", bin, filepath.Join(root, "testdata", "fakecli"))
		if out, err := cmd.CombinedOutput(); err != nil {
			e2eFakecliErr = fmt.Errorf("build fakecli: %w: %s", err, string(out))
			return
		}
		e2eFakecliPath = bin
	})
	if e2eFakecliErr != nil {
		t.Fatalf("failed to build fakecli: %v", e2eFakecliErr)
	}
	return e2eFakecliPath
}

func findE2EModuleRoot(t testing.TB) string {
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

func setupE2EApp(t *testing.T, bin, cfgPath string) (*app.App, *transphttp.Server) {
	t.Helper()
	defs := []adapters.Definition{
		{
			Adapter: domain.Adapter{
				ID:       "codex",
				Name:     "OpenAI Codex",
				Vendor:   "OpenAI",
				Binary:   bin,
				Category: "platform",
				Models: []domain.ModelInfo{
					{ID: "gpt-4o", DisplayName: "GPT-4o"},
				},
			},
			ModelsCommand: bin + " debug models",
			ProbeSpecs: map[domain.Tier]domain.Probe{
				domain.TierL0: {Level: domain.TierL0, Command: bin + " --version"},
				domain.TierL1: {Level: domain.TierL1, Command: bin},
				domain.TierL2: {Level: domain.TierL2, Command: bin},
				domain.TierL3: {Level: domain.TierL3, Command: bin},
			},
		},
	}

	application, err := app.New(
		app.WithRunner(edgeExec.NewPipeRunner()),
		app.WithCache(store.NewTTLCache(nil)),
		app.WithDefinitions(defs),
		app.WithConfigPath(cfgPath),
	)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}
	server := transphttp.NewServer(application)
	return application, server
}

func testConfigGetE2E(t *testing.T, server *transphttp.Server, expectedPath string) domain.Config {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/config returned status %d: %s", rec.Code, rec.Body.String())
	}
	if pathHdr := rec.Header().Get("X-Config-Path"); pathHdr != expectedPath {
		t.Errorf("expected X-Config-Path %q, got %q", expectedPath, pathHdr)
	}
	var cfg domain.Config
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatalf("failed to decode GET /api/v1/config json: %v", err)
	}
	return cfg
}

func testConfigValidateE2E(t *testing.T, server *transphttp.Server, baseCfg domain.Config) {
	t.Helper()
	valCfg := baseCfg
	valCfg.Roles = map[domain.Role][]string{
		domain.RolePrimary: {"codex:non-existent-model-xyz"},
	}
	body, err := json.Marshal(valCfg)
	if err != nil {
		t.Fatalf("marshal validation config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/config/validate status %d: %s", rec.Code, rec.Body.String())
	}

	var checks []resolve.CandidateCheck
	if err := json.Unmarshal(rec.Body.Bytes(), &checks); err != nil {
		t.Fatalf("decode checks: %v", err)
	}
	if len(checks) == 0 {
		t.Fatal("expected candidate checks, got none")
	}
	ch := checks[0]
	if ch.Verified {
		t.Errorf("expected verified false for unlisted model, got true")
	}
	if !strings.Contains(ch.Reason, "Available:") {
		t.Errorf("expected reason containing 'Available:', got %q", ch.Reason)
	}
}

func testConfigPutValidE2E(t *testing.T, server *transphttp.Server, cfgPath string, baseCfg domain.Config) {
	t.Helper()
	validCfg := baseCfg
	validCfg.Roles[domain.RolePrimary] = []string{"codex:gpt-6-astra"}
	validBody, err := json.Marshal(validCfg)
	if err != nil {
		t.Fatalf("marshal valid config: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(validBody))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/v1/config valid returned status %d: %s", rec.Code, rec.Body.String())
	}

	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("failed to stat persisted config file: %v", err)
	}
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm != 0o600 {
		t.Errorf("expected file mode 0600, got %#o", perm)
	}

	data, err := os.ReadFile(filepath.Clean(cfgPath)) // #nosec G304
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	if !strings.Contains(string(data), "gpt-6-astra") {
		t.Errorf("persisted config content missing 'gpt-6-astra':\n%s", string(data))
	}
}

func testConfigPutInvalidE2E(t *testing.T, server *transphttp.Server, cfgPath string) {
	t.Helper()
	beforeData, err := os.ReadFile(filepath.Clean(cfgPath)) // #nosec G304
	if err != nil {
		t.Fatalf("read config before invalid put: %v", err)
	}

	invalidBody := []byte(`{"version": 999, "roles": {}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/config", bytes.NewReader(invalidBody))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid PUT config, got %d: %s", rec.Code, rec.Body.String())
	}

	afterData, err := os.ReadFile(filepath.Clean(cfgPath)) // #nosec G304
	if err != nil {
		t.Fatalf("read config after invalid put: %v", err)
	}
	if !bytes.Equal(beforeData, afterData) {
		t.Errorf("config file was modified despite invalid PUT 400")
	}
}

func TestConfigE2E(t *testing.T) {
	bin := getE2EFakeCLI(t)
	cfgPath := filepath.Join(t.TempDir(), "agent-doctor.yaml")
	_, server := setupE2EApp(t, bin, cfgPath)

	cfg := testConfigGetE2E(t, server, cfgPath)
	testConfigValidateE2E(t, server, cfg)
	testConfigPutValidE2E(t, server, cfgPath, cfg)
	testConfigPutInvalidE2E(t, server, cfgPath)
}
