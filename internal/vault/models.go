package vault

import "time"

// HealthResponse represents the response from /v1/sys/health
type HealthResponse struct {
	Initialized        bool   `json:"initialized"`
	Sealed             bool   `json:"sealed"`
	Standby            bool   `json:"standby"`
	PerformanceStandby bool   `json:"performance_standby"`
	Version            string `json:"version"`
	ServerTimeUTC      int64  `json:"server_time_utc"`
}

// InitRequest represents the payload for /v1/sys/init
type InitRequest struct {
	SecretShares    int `json:"secret_shares"`
	SecretThreshold int `json:"secret_threshold"`
}

// InitResponse represents the response from /v1/sys/init
type InitResponse struct {
	Keys       []string `json:"keys"`
	KeysBase64 []string `json:"keys_base64"`
	RootToken  string   `json:"root_token"`
}

// UnsealRequest represents the payload for /v1/sys/unseal
type UnsealRequest struct {
	Key   string `json:"key"`
	Reset bool   `json:"reset,omitempty"`
}

// UnsealResponse represents the response from /v1/sys/unseal
type UnsealResponse struct {
	Sealed   bool   `json:"sealed"`
	T        int    `json:"t"`
	N        int    `json:"n"`
	Progress int    `json:"progress"`
	Version  string `json:"version"`
}

// KVV2ReadResponse represents a secret read response from /v1/{mount}/data/{path}
type KVV2ReadResponse struct {
	RequestID string `json:"request_id"`
	Data      struct {
		Data     map[string]interface{} `json:"data"`
		Metadata struct {
			CreatedTime  string `json:"created_time"`
			DeletionTime string `json:"deletion_time"`
			Destroyed    bool   `json:"destroyed"`
			Version      int    `json:"version"`
		} `json:"metadata"`
	} `json:"data"`
}

// KVV2WriteRequest represents the payload for /v1/{mount}/data/{path}
type KVV2WriteRequest struct {
	Data map[string]interface{} `json:"data"`
}

// KVV2ListResponse represents the response from /v1/{mount}/metadata/{path}?list=true
type KVV2ListResponse struct {
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
}

// SecretItem represents a clean, application-level secret view
type SecretItem struct {
	Path        string
	Data        map[string]string
	Version     int
	CreatedTime time.Time
	Destroyed   bool
}

// VaultErrorResponse represents standard Vault error output
type VaultErrorResponse struct {
	Errors []string `json:"errors"`
}
