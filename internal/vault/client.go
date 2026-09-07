package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Client interacts with the HashiCorp Vault HTTP API using Go standard library.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	mount      string
	mu         sync.RWMutex
}

// NewClient constructs a new Vault client.
func NewClient(baseURL, token, mount string) *Client {
	if mount == "" {
		mount = "secret"
	}
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		mount:   strings.Trim(mount, "/"),
	}
}

// SetToken safely updates the Vault authentication token.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// Token returns the current Vault token.
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// SetAddress safely updates the Vault base URL.
func (c *Client) SetAddress(addr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = strings.TrimRight(addr, "/")
}

// Address returns the configured Vault base URL.
func (c *Client) Address() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.baseURL
}

// SetMount safely updates the KV v2 mount path.
func (c *Client) SetMount(mount string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mount = strings.Trim(mount, "/")
}

// Mount returns the configured mount path.
func (c *Client) Mount() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mount
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	c.mu.RLock()
	base := c.baseURL
	token := c.token
	c.mu.RUnlock()

	url := fmt.Sprintf("%s%s", base, path)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}

	return req, nil
}

func (c *Client) parseError(resp *http.Response) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var errResp VaultErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && len(errResp.Errors) > 0 {
		return fmt.Errorf("vault error (%d): %s", resp.StatusCode, strings.Join(errResp.Errors, ", "))
	}

	if len(body) > 0 {
		return fmt.Errorf("vault error (%d): %s", resp.StatusCode, string(body))
	}
	return fmt.Errorf("vault error (%d): %s", resp.StatusCode, resp.Status)
}

// GetHealth queries /v1/sys/health.
// Note: Vault returns 200 (healthy), 429 (standby), 472/473 (disaster recovery), 501 (not initialized), 503 (sealed).
// The JSON payload is present across these codes.
func (c *Client) GetHealth(ctx context.Context) (*HealthResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/v1/sys/health", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response (status %d): %w", resp.StatusCode, err)
	}

	return &health, nil
}

// Initialize sends a PUT request to /v1/sys/init to initialize Vault.
func (c *Client) Initialize(ctx context.Context, shares, threshold int) (*InitResponse, error) {
	payload := InitRequest{
		SecretShares:    shares,
		SecretThreshold: threshold,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPut, "/v1/sys/init", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var initResp InitResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		return nil, err
	}

	return &initResp, nil
}

// Unseal sends a PUT request to /v1/sys/unseal with a key shard.
func (c *Client) Unseal(ctx context.Context, key string) (*UnsealResponse, error) {
	payload := UnsealRequest{
		Key: strings.TrimSpace(key),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPut, "/v1/sys/unseal", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var unsealResp UnsealResponse
	if err := json.NewDecoder(resp.Body).Decode(&unsealResp); err != nil {
		return nil, err
	}

	return &unsealResp, nil
}

// EnableKVv2 enables a KV v2 engine at the given mount if it doesn't already exist.
func (c *Client) EnableKVv2(ctx context.Context, mount string) error {
	payload := map[string]interface{}{
		"type": "kv",
		"options": map[string]string{
			"version": "2",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := c.newRequest(ctx, http.MethodPost, fmt.Sprintf("/v1/sys/mounts/%s", strings.Trim(mount, "/")), bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	// If already mounted, ignore error
	errText := c.parseError(resp).Error()
	if strings.Contains(errText, "path is already in use") {
		return nil
	}
	return errors.New(errText)
}

// ListSecrets lists keys under the given path in the KV v2 engine.
// Query: GET /v1/{mount}/metadata/{prefix}?list=true
func (c *Client) ListSecrets(ctx context.Context, prefix string) ([]string, error) {
	c.mu.RLock()
	mount := c.mount
	c.mu.RUnlock()

	cleanPrefix := strings.Trim(prefix, "/")
	path := fmt.Sprintf("/v1/%s/metadata", mount)
	if cleanPrefix != "" {
		path = fmt.Sprintf("%s/%s", path, cleanPrefix)
	}
	path += "?list=true"

	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var listResp KVV2ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	return listResp.Data.Keys, nil
}

// GetSecret reads a secret from /v1/{mount}/data/{path}.
func (c *Client) GetSecret(ctx context.Context, path string) (*SecretItem, error) {
	c.mu.RLock()
	mount := c.mount
	c.mu.RUnlock()

	cleanPath := strings.Trim(path, "/")
	reqURL := fmt.Sprintf("/v1/%s/data/%s", mount, cleanPath)

	req, err := c.newRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var readResp KVV2ReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&readResp); err != nil {
		return nil, err
	}

	// Convert data map[string]interface{} to map[string]string
	dataMap := make(map[string]string)
	for k, v := range readResp.Data.Data {
		dataMap[k] = fmt.Sprintf("%v", v)
	}

	createdTime, _ := time.Parse(time.RFC3339Nano, readResp.Data.Metadata.CreatedTime)
	if createdTime.IsZero() {
		createdTime, _ = time.Parse(time.RFC3339, readResp.Data.Metadata.CreatedTime)
	}

	return &SecretItem{
		Path:        cleanPath,
		Data:        dataMap,
		Version:     readResp.Data.Metadata.Version,
		CreatedTime: createdTime,
		Destroyed:   readResp.Data.Metadata.Destroyed,
	}, nil
}

// PutSecret writes or updates a secret at /v1/{mount}/data/{path}.
func (c *Client) PutSecret(ctx context.Context, path string, data map[string]string) error {
	c.mu.RLock()
	mount := c.mount
	c.mu.RUnlock()

	cleanPath := strings.Trim(path, "/")
	reqURL := fmt.Sprintf("/v1/%s/data/%s", mount, cleanPath)

	rawMap := make(map[string]interface{})
	for k, v := range data {
		rawMap[k] = v
	}

	payload := KVV2WriteRequest{
		Data: rawMap,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := c.newRequest(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// DeleteSecret soft-deletes the current version of the secret at /v1/{mount}/data/{path}.
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	c.mu.RLock()
	mount := c.mount
	c.mu.RUnlock()

	cleanPath := strings.Trim(path, "/")
	reqURL := fmt.Sprintf("/v1/%s/data/%s", mount, cleanPath)

	req, err := c.newRequest(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// DestroySecretMetadata permanently deletes a secret and its entire version history.
func (c *Client) DestroySecretMetadata(ctx context.Context, path string) error {
	c.mu.RLock()
	mount := c.mount
	c.mu.RUnlock()

	cleanPath := strings.Trim(path, "/")
	reqURL := fmt.Sprintf("/v1/%s/metadata/%s", mount, cleanPath)

	req, err := c.newRequest(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}
