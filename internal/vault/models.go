// Package vault provides a lightweight, pure Go standard-library HTTP client
// and data models for HashiCorp Vault's REST API.
//
// It supports Vault system operations (health checking, Shamir secret sharing
// initialization, and unsealing) as well as full KV Version 2 secret lifecycle
// management (list, read, write, soft delete, and permanent metadata destroy).
package vault

import "time"

// HealthResponse represents the response payload from /v1/sys/health.
type HealthResponse struct {
	// Initialized reports whether the Vault has been initialized with Shamir keys.
	Initialized bool `json:"initialized"`

	// Sealed reports whether the Vault storage engine is currently locked.
	Sealed bool `json:"sealed"`

	// Standby indicates whether this Vault instance is in standby mode in a cluster.
	Standby bool `json:"standby"`

	// PerformanceStandby indicates whether this is a performance standby node.
	PerformanceStandby bool `json:"performance_standby"`

	// Version is the Vault server release version (e.g. "1.18.4").
	Version string `json:"version"`

	// ServerTimeUTC is the Unix timestamp from the Vault server clock.
	ServerTimeUTC int64 `json:"server_time_utc"`
}

// InitRequest represents the JSON payload sent to /v1/sys/init to initialize Vault.
type InitRequest struct {
	// SecretShares is the total number of unseal key shares to generate.
	SecretShares int `json:"secret_shares"`

	// SecretThreshold is the minimum number of key shares required to reconstruct the master key.
	SecretThreshold int `json:"secret_threshold"`
}

// InitResponse represents the response returned by /v1/sys/init upon successful initialization.
type InitResponse struct {
	// Keys contains the generated hex-encoded unseal key shards.
	Keys []string `json:"keys"`

	// KeysBase64 contains base64-encoded unseal key shards.
	KeysBase64 []string `json:"keys_base64"`

	// RootToken is the initial superuser administrative token.
	RootToken string `json:"root_token"`
}

// UnsealRequest represents the JSON payload sent to /v1/sys/unseal.
type UnsealRequest struct {
	// Key is an unseal shard key entered by the operator.
	Key string `json:"key"`

	// Reset resets the unsealing progress if true.
	Reset bool `json:"reset,omitempty"`
}

// UnsealResponse represents the response returned by /v1/sys/unseal.
type UnsealResponse struct {
	// Sealed indicates if the Vault remains sealed after applying the shard.
	Sealed bool `json:"sealed"`

	// T is the unseal threshold (shares required).
	T int `json:"t"`

	// N is the total number of unseal shares.
	N int `json:"n"`

	// Progress is the count of valid shards submitted so far toward threshold T.
	Progress int `json:"progress"`

	// Version is the Vault server version.
	Version string `json:"version"`
}

// KVV2ReadResponse represents the raw HTTP response from /v1/{mount}/data/{path}.
type KVV2ReadResponse struct {
	RequestID string `json:"request_id"`
	Data      struct {
		Data     map[string]any `json:"data"`
		Metadata struct {
			CreatedTime  string `json:"created_time"`
			DeletionTime string `json:"deletion_time"`
			Destroyed    bool   `json:"destroyed"`
			Version      int    `json:"version"`
		} `json:"metadata"`
	} `json:"data"`
}

// KVV2WriteRequest represents the JSON payload for creating or updating a secret at /v1/{mount}/data/{path}.
type KVV2WriteRequest struct {
	Data map[string]any `json:"data"`
}

// KVV2ListResponse represents the keys list response from /v1/{mount}/metadata/{path}?list=true.
type KVV2ListResponse struct {
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
}

// SecretItem represents a clean, application-level view of a retrieved Vault secret.
type SecretItem struct {
	// Path is the relative secret path within the mount (e.g. "webapp/database").
	Path string

	// Data contains the key-value pairs stored in the secret.
	Data map[string]string

	// Version is the KV v2 revision number (increments on every write).
	Version int

	// CreatedTime is the parsed timestamp when this version was committed.
	CreatedTime time.Time

	// DeletionTime is the timestamp when this version was soft-deleted, if applicable.
	DeletionTime time.Time

	// Destroyed indicates if this specific version has been destroyed.
	Destroyed bool

	// IsDeleted indicates whether the current version is soft-deleted.
	IsDeleted bool
}

// VaultErrorResponse represents standard error output format from Vault API.
type VaultErrorResponse struct {
	Errors []string `json:"errors"`
}
