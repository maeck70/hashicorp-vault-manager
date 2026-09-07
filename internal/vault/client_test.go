package vault

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_GetHealth(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		response    string
		expectInit  bool
		expectSeal  bool
		expectError bool
	}{
		{
			name:        "healthy unsealed",
			statusCode:  http.StatusOK,
			response:    `{"initialized":true,"sealed":false,"version":"1.18.4"}`,
			expectInit:  true,
			expectSeal:  false,
			expectError: false,
		},
		{
			name:        "uninitialized (501)",
			statusCode:  http.StatusNotImplemented,
			response:    `{"initialized":false,"sealed":true,"version":"1.18.4"}`,
			expectInit:  false,
			expectSeal:  true,
			expectError: false,
		},
		{
			name:        "sealed (503)",
			statusCode:  http.StatusServiceUnavailable,
			response:    `{"initialized":true,"sealed":true,"version":"1.18.4"}`,
			expectInit:  true,
			expectSeal:  true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/sys/health" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := NewClient(server.URL, "test-token", "secret")
			health, err := client.GetHealth(context.Background())
			if tt.expectError && err == nil {
				t.Fatalf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if health.Initialized != tt.expectInit {
				t.Errorf("expected initialized=%v, got %v", tt.expectInit, health.Initialized)
			}
			if health.Sealed != tt.expectSeal {
				t.Errorf("expected sealed=%v, got %v", tt.expectSeal, health.Sealed)
			}
		})
	}
}

func TestClient_Initialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1/sys/init" {
			t.Errorf("unexpected method/path: %s %s", r.Method, r.URL.Path)
		}
		var req InitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode init request: %v", err)
		}
		if req.SecretShares != 1 || req.SecretThreshold != 1 {
			t.Errorf("expected shares=1, threshold=1, got %d, %d", req.SecretShares, req.SecretThreshold)
		}

		resp := InitResponse{
			Keys:       []string{"key123"},
			KeysBase64: []string{"a2V5MTIz"},
			RootToken:  "root-token-abc",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "secret")
	resp, err := client.Initialize(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RootToken != "root-token-abc" {
		t.Errorf("expected root token root-token-abc, got %s", resp.RootToken)
	}
	if len(resp.Keys) != 1 || resp.Keys[0] != "key123" {
		t.Errorf("unexpected keys: %v", resp.Keys)
	}
}

func TestClient_Unseal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1/sys/unseal" {
			t.Errorf("unexpected method/path: %s %s", r.Method, r.URL.Path)
		}
		var req UnsealRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode unseal request: %v", err)
		}
		if req.Key != "unseal-key-1" {
			t.Errorf("unexpected unseal key: %s", req.Key)
		}

		resp := UnsealResponse{
			Sealed:   false,
			T:        1,
			N:        1,
			Progress: 0,
			Version:  "1.18.4",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "secret")
	resp, err := client.Unseal(context.Background(), "unseal-key-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Sealed {
		t.Errorf("expected sealed=false, got true")
	}
}

func TestClient_KVv2_Lifecycle(t *testing.T) {
	mockStore := make(map[string]map[string]string)
	mockVersions := make(map[string]int)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Vault-Token")
		if token != "my-vault-token" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"errors":["permission denied"]}`))
			return
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/secret/metadata":
			// List keys
			var keys []string
			for k := range mockStore {
				keys = append(keys, k)
			}
			resp := KVV2ListResponse{}
			resp.Data.Keys = keys
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && r.URL.Path == "/v1/secret/data/webapp/db":
			// Put secret
			var body KVV2WriteRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode write request failed: %v", err)
			}
			kv := make(map[string]string)
			for k, v := range body.Data {
				kv[k] = v.(string)
			}
			mockStore["webapp/db"] = kv
			mockVersions["webapp/db"]++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data":{"version":1}}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/secret/data/webapp/db":
			// Get secret
			data, exists := mockStore["webapp/db"]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"errors":[]}`))
				return
			}
			rawMap := make(map[string]interface{})
			for k, v := range data {
				rawMap[k] = v
			}
			readResp := KVV2ReadResponse{}
			readResp.Data.Data = rawMap
			readResp.Data.Metadata.Version = mockVersions["webapp/db"]
			readResp.Data.Metadata.CreatedTime = "2026-09-07T00:00:00Z"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(readResp)

		case r.Method == http.MethodDelete && r.URL.Path == "/v1/secret/data/webapp/db":
			// Delete secret version
			delete(mockStore, "webapp/db")
			w.WriteHeader(http.StatusNoContent)

		default:
			t.Fatalf("unhandled mock request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "my-vault-token", "secret")

	// 1. Put Secret
	err := client.PutSecret(context.Background(), "webapp/db", map[string]string{
		"host": "10.0.0.10",
		"port": "5432",
	})
	if err != nil {
		t.Fatalf("PutSecret failed: %v", err)
	}

	// 2. Get Secret
	item, err := client.GetSecret(context.Background(), "webapp/db")
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}
	if item.Data["host"] != "10.0.0.10" || item.Data["port"] != "5432" {
		t.Errorf("unexpected secret data: %v", item.Data)
	}
	if item.Version != 1 {
		t.Errorf("expected version 1, got %d", item.Version)
	}

	// 3. List Secrets
	keys, err := client.ListSecrets(context.Background(), "")
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}
	if len(keys) != 1 || keys[0] != "webapp/db" {
		t.Errorf("unexpected list result: %v", keys)
	}

	// 4. Delete Secret
	err = client.DeleteSecret(context.Background(), "webapp/db")
	if err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	// 5. Verify deleted
	_, err = client.GetSecret(context.Background(), "webapp/db")
	if err == nil {
		t.Fatalf("expected error getting deleted secret, but got none")
	}
}
