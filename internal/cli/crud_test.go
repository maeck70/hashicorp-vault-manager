package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

func TestRunPutSecret_Table(t *testing.T) {
	var mu sync.Mutex
	var lastWrittenData map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodPost && r.URL.Path == "/v1/secret/data/infra/redis" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			lastWrittenData = body["data"].(map[string]any)
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/v1/secret/data/infra/redis" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"data": {"host": "127.0.0.1", "port": "6379"},
					"metadata": {"version": 3, "created_time": "2026-09-07T08:00:00Z"}
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var buf bytes.Buffer
	OutputWriter = &buf

	client := vault.NewClient(server.URL, "token", "secret")
	cfg := &config.Config{
		Command:    "put",
		TargetPath: "infra/redis",
		DataArgs: map[string]string{
			"host": "127.0.0.1",
			"port": "6379",
		},
		Format: "table",
	}

	err := RunPutSecret(client, cfg)
	if err != nil {
		t.Fatalf("RunPutSecret failed: %v", err)
	}

	mu.Lock()
	if lastWrittenData["host"] != "127.0.0.1" || lastWrittenData["port"] != "6379" {
		t.Errorf("unexpected written data: %+v", lastWrittenData)
	}
	mu.Unlock()

	out := buf.String()
	if !strings.Contains(out, "Secret successfully written") {
		t.Errorf("expected success message, got: %s", out)
	}
	if !strings.Contains(out, "Version: 3") {
		t.Errorf("expected Version: 3, got: %s", out)
	}
}

func TestRunPutSecret_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"data": {"api_key": "sk_test_123"},
					"metadata": {"version": 1, "created_time": "2026-09-07T08:00:00Z"}
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var buf bytes.Buffer
	OutputWriter = &buf

	client := vault.NewClient(server.URL, "token", "secret")
	cfg := &config.Config{
		Command:    "put",
		TargetPath: "api/stripe",
		DataArgs: map[string]string{
			"api_key": "sk_test_123",
		},
		Format: "json",
	}

	err := RunPutSecret(client, cfg)
	if err != nil {
		t.Fatalf("RunPutSecret failed: %v", err)
	}

	out := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("failed to parse output json: %v, out: %s", err, out)
	}
	if parsed["status"] != "success" || parsed["path"] != "api/stripe" {
		t.Errorf("unexpected json output: %+v", parsed)
	}
}

func TestRunPutSecret_RawData_FileAndMerge(t *testing.T) {
	// Create temporary JSON file
	tempDir := t.TempDir()
	jsonFile := filepath.Join(tempDir, "payload.json")
	if err := os.WriteFile(jsonFile, []byte(`{"env":"production","timeout":"30s"}`), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	var mu sync.Mutex
	var lastWritten map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodGet && r.URL.Path == "/v1/secret/data/services/auth" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"data": {"existing_key": "keep_this"},
					"metadata": {"version": 1}
				}
			}`))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/v1/secret/data/services/auth" {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			lastWritten = body["data"].(map[string]any)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	var buf bytes.Buffer
	OutputWriter = &buf

	client := vault.NewClient(server.URL, "token", "secret")
	cfg := &config.Config{
		Command:    "put",
		TargetPath: "services/auth",
		RawData:    "@" + jsonFile,
		DataArgs: map[string]string{
			"timeout": "60s", // CLI arg overrides file
		},
		Merge:  true,
		Format: "table",
	}

	err := RunPutSecret(client, cfg)
	if err != nil {
		t.Fatalf("RunPutSecret failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if lastWritten["existing_key"] != "keep_this" {
		t.Errorf("expected existing_key to be preserved under merge, got: %+v", lastWritten)
	}
	if lastWritten["env"] != "production" {
		t.Errorf("expected env from file, got: %+v", lastWritten)
	}
	if lastWritten["timeout"] != "60s" {
		t.Errorf("expected timeout overridden by CLI arg, got: %+v", lastWritten)
	}
}

func TestRunDeleteSecret(t *testing.T) {
	var mu sync.Mutex
	var permanentDeleted bool
	var softDeleted bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodDelete && r.URL.Path == "/v1/secret/metadata/services/auth" {
			permanentDeleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodDelete && r.URL.Path == "/v1/secret/data/services/auth" {
			softDeleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := vault.NewClient(server.URL, "token", "secret")

	// 1. Permanent delete with -f
	var buf bytes.Buffer
	OutputWriter = &buf
	cfgPerm := &config.Config{
		Command:    "delete",
		TargetPath: "services/auth",
		Force:      true,
		Format:     "table",
	}
	if err := RunDeleteSecret(client, cfgPerm); err != nil {
		t.Fatalf("RunDeleteSecret failed: %v", err)
	}
	if !permanentDeleted {
		t.Errorf("expected metadata permanent deletion endpoint to be called")
	}
	if !strings.Contains(buf.String(), "permanently deleted") {
		t.Errorf("unexpected output: %s", buf.String())
	}

	// 2. Soft delete with -soft -f
	buf.Reset()
	cfgSoft := &config.Config{
		Command:    "delete",
		TargetPath: "services/auth",
		Force:      true,
		SoftDelete: true,
		Format:     "json",
	}
	if err := RunDeleteSecret(client, cfgSoft); err != nil {
		t.Fatalf("RunDeleteSecret soft failed: %v", err)
	}
	if !softDeleted {
		t.Errorf("expected data soft deletion endpoint to be called")
	}
	if !strings.Contains(buf.String(), `"action": "soft_delete"`) {
		t.Errorf("unexpected json output: %s", buf.String())
	}

	// 3. Confirmation abort
	buf.Reset()
	InputReader = strings.NewReader("n\n")
	cfgAbort := &config.Config{
		Command:    "delete",
		TargetPath: "services/auth",
		Force:      false,
	}
	if err := RunDeleteSecret(client, cfgAbort); err != nil {
		t.Fatalf("RunDeleteSecret abort failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Deletion aborted.") {
		t.Errorf("expected Deletion aborted, got: %s", buf.String())
	}
}

func TestRunListSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/secret/metadata/services" && r.URL.Query().Get("list") == "true" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"keys": ["api/", "auth", "db"]
				}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := vault.NewClient(server.URL, "token", "secret")

	// Table format
	var buf bytes.Buffer
	OutputWriter = &buf
	cfgTable := &config.Config{
		Command: "list",
		Prefix:  "services",
		Format:  "table",
	}
	if err := RunListSecrets(client, cfgTable); err != nil {
		t.Fatalf("RunListSecrets table failed: %v", err)
	}
	outTable := buf.String()
	if !strings.Contains(outTable, "📁 api/") || !strings.Contains(outTable, "🔑 auth") {
		t.Errorf("unexpected table output: %s", outTable)
	}

	// JSON format
	buf.Reset()
	cfgJSON := &config.Config{
		Command: "list",
		Prefix:  "services",
		Format:  "json",
	}
	if err := RunListSecrets(client, cfgJSON); err != nil {
		t.Fatalf("RunListSecrets json failed: %v", err)
	}
	outJSON := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(outJSON), &parsed); err != nil {
		t.Fatalf("failed to unmarshal list json: %v", err)
	}
	if parsed["count"].(float64) != 3 {
		t.Errorf("expected count 3, got %v", parsed["count"])
	}
}


