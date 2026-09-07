package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

func TestRunGetSecret_Table(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/secret/data/my/db" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"data": {
					"data": {
						"username": "admin",
						"config": "{\"host\":\"localhost\",\"port\":5432}"
					},
					"metadata": {
						"version": 1,
						"created_time": "2026-09-07T00:00:00Z"
					}
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
		GetPath: "my/db",
		Format:  "table",
	}

	err := RunGetSecret(client, cfg)
	if err != nil {
		t.Fatalf("RunGetSecret failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "username") || !strings.Contains(out, "admin") {
		t.Errorf("expected username admin in output, got: %s", out)
	}
	if !strings.Contains(out, "host: localhost") {
		t.Errorf("expected nested JSON host in output, got: %s", out)
	}
}

func TestRunGetSecret_Field(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"data": {
					"password": "super-secret-password-123"
				},
				"metadata": {
					"version": 2,
					"created_time": "2026-09-07T00:00:00Z"
				}
			}
		}`))
	}))
	defer server.Close()

	var buf bytes.Buffer
	OutputWriter = &buf

	client := vault.NewClient(server.URL, "token", "secret")
	cfg := &config.Config{
		GetPath: "app/pass",
		Field:   "password",
	}

	err := RunGetSecret(client, cfg)
	if err != nil {
		t.Fatalf("RunGetSecret failed: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != "super-secret-password-123" {
		t.Errorf("expected 'super-secret-password-123', got: %q", out)
	}
}

func TestRunGetSecret_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"data": {
					"api_key": "xyz987"
				},
				"metadata": {
					"version": 1,
					"created_time": "2026-09-07T00:00:00Z"
				}
			}
		}`))
	}))
	defer server.Close()

	var buf bytes.Buffer
	OutputWriter = &buf

	client := vault.NewClient(server.URL, "token", "secret")
	cfg := &config.Config{
		GetPath: "app/keys",
		Format:  "json",
	}

	err := RunGetSecret(client, cfg)
	if err != nil {
		t.Fatalf("RunGetSecret failed: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"api_key": "xyz987"`) || !strings.Contains(out, `"path": "app/keys"`) {
		t.Errorf("expected json output with api_key, got: %s", out)
	}
}

func TestRunGetSecret_FieldNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": {
				"data": {
					"user": "alice"
				},
				"metadata": {
					"version": 1,
					"created_time": "2026-09-07T00:00:00Z"
				}
			}
		}`))
	}))
	defer server.Close()

	client := vault.NewClient(server.URL, "token", "secret")
	cfg := &config.Config{
		GetPath: "app/info",
		Field:   "non_existent",
	}

	err := RunGetSecret(client, cfg)
	if err == nil || !strings.Contains(err.Error(), "field 'non_existent' not found") {
		t.Errorf("expected field not found error, got: %v", err)
	}
}

func TestRunGetSecret_SoftDeleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{
			"data": {
				"data": null,
				"metadata": {
					"version": 1,
					"created_time": "2026-09-07T00:00:00Z",
					"deletion_time": "2026-09-07T08:00:00Z",
					"destroyed": false
				}
			}
		}`))
	}))
	defer server.Close()

	client := vault.NewClient(server.URL, "token", "secret")
	cfg := &config.Config{
		GetPath: "app/deleted",
	}

	err := RunGetSecret(client, cfg)
	if err == nil || !strings.Contains(err.Error(), "is soft-deleted in Vault") {
		t.Errorf("expected soft-deleted error message, got: %v", err)
	}
}

// Reset OutputWriter back to standard output
func init() {
	time.Sleep(1 * time.Millisecond)
}
