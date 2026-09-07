package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveToEnv(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	initialContent := "# Initial comment\nEXISTING_KEY=initial_value\nKEEP_ME=yes\n"
	if err := os.WriteFile(envPath, []byte(initialContent), 0600); err != nil {
		t.Fatalf("failed to write initial env: %v", err)
	}

	updates := map[string]string{
		"EXISTING_KEY": "updated_value",
		"NEW_KEY":      "new_value_123",
	}

	if err := SaveToEnv(envPath, updates); err != nil {
		t.Fatalf("SaveToEnv failed: %v", err)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read updated env: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "EXISTING_KEY=updated_value") {
		t.Errorf("expected EXISTING_KEY=updated_value in content: %s", content)
	}
	if !strings.Contains(content, "NEW_KEY=new_value_123") {
		t.Errorf("expected NEW_KEY=new_value_123 in content: %s", content)
	}
	if !strings.Contains(content, "KEEP_ME=yes") {
		t.Errorf("expected KEEP_ME=yes to be preserved in content: %s", content)
	}
}

func TestLoadFromArgs(t *testing.T) {
	// Test 1: -get flag with format and field
	cfg, err := LoadFromArgs([]string{"-get", "services/auth", "-field", "token", "-format", "raw"})
	if err != nil {
		t.Fatalf("LoadFromArgs failed: %v", err)
	}
	if cfg.GetPath != "services/auth" {
		t.Errorf("expected GetPath 'services/auth', got %q", cfg.GetPath)
	}
	if cfg.Field != "token" {
		t.Errorf("expected Field 'token', got %q", cfg.Field)
	}
	if cfg.Format != "raw" {
		t.Errorf("expected Format 'raw', got %q", cfg.Format)
	}

	// Test 2: positional "get" argument with -json flag
	cfg2, err := LoadFromArgs([]string{"get", "databases/mysql", "-json"})
	if err != nil {
		t.Fatalf("LoadFromArgs failed: %v", err)
	}
	if cfg2.GetPath != "databases/mysql" {
		t.Errorf("expected GetPath 'databases/mysql', got %q", cfg2.GetPath)
	}
	if cfg2.Format != "json" {
		t.Errorf("expected Format 'json', got %q", cfg2.Format)
	}

	// Test 3: direct path argument
	cfg3, err := LoadFromArgs([]string{"api/stripe"})
	if err != nil {
		t.Fatalf("LoadFromArgs failed: %v", err)
	}
	if cfg3.GetPath != "api/stripe" {
		t.Errorf("expected GetPath 'api/stripe', got %q", cfg3.GetPath)
	}
	if cfg3.Command != "get" {
		t.Errorf("expected Command 'get', got %q", cfg3.Command)
	}
}

func TestLoadFromArgs_CRUD(t *testing.T) {
	// Test put with key-value pairs
	cfgPut, err := LoadFromArgs([]string{"put", "services/payment", "api_key=sk_test_123", "env=prod", "-json"})
	if err != nil {
		t.Fatalf("LoadFromArgs put failed: %v", err)
	}
	if cfgPut.Command != "put" {
		t.Errorf("expected Command 'put', got %q", cfgPut.Command)
	}
	if cfgPut.TargetPath != "services/payment" {
		t.Errorf("expected TargetPath 'services/payment', got %q", cfgPut.TargetPath)
	}
	if cfgPut.DataArgs["api_key"] != "sk_test_123" || cfgPut.DataArgs["env"] != "prod" {
		t.Errorf("unexpected DataArgs: %+v", cfgPut.DataArgs)
	}
	if cfgPut.Format != "json" {
		t.Errorf("expected Format 'json', got %q", cfgPut.Format)
	}

	// Test update with merge flag
	cfgUpdate, err := LoadFromArgs([]string{"update", "services/payment", "api_key=sk_live_999"})
	if err != nil {
		t.Fatalf("LoadFromArgs update failed: %v", err)
	}
	if cfgUpdate.Command != "put" || !cfgUpdate.Merge {
		t.Errorf("expected Command 'put' with Merge=true, got command=%q, merge=%v", cfgUpdate.Command, cfgUpdate.Merge)
	}
	if cfgUpdate.DataArgs["api_key"] != "sk_live_999" {
		t.Errorf("unexpected DataArgs: %+v", cfgUpdate.DataArgs)
	}

	// Test put with -data json string
	cfgData, err := LoadFromArgs([]string{"put", "infra/redis", "-data", `{"host":"localhost","port":"6379"}`})
	if err != nil {
		t.Fatalf("LoadFromArgs -data failed: %v", err)
	}
	if cfgData.Command != "put" || cfgData.RawData != `{"host":"localhost","port":"6379"}` {
		t.Errorf("unexpected RawData: %q", cfgData.RawData)
	}

	// Test delete with force and soft flags
	cfgDel, err := LoadFromArgs([]string{"delete", "legacy/creds", "-f", "-soft"})
	if err != nil {
		t.Fatalf("LoadFromArgs delete failed: %v", err)
	}
	if cfgDel.Command != "delete" {
		t.Errorf("expected Command 'delete', got %q", cfgDel.Command)
	}
	if cfgDel.TargetPath != "legacy/creds" {
		t.Errorf("expected TargetPath 'legacy/creds', got %q", cfgDel.TargetPath)
	}
	if !cfgDel.Force || !cfgDel.SoftDelete {
		t.Errorf("expected Force=true and SoftDelete=true, got force=%v, soft=%v", cfgDel.Force, cfgDel.SoftDelete)
	}

	// Test list with prefix
	cfgList, err := LoadFromArgs([]string{"list", "services/", "-json"})
	if err != nil {
		t.Fatalf("LoadFromArgs list failed: %v", err)
	}
	if cfgList.Command != "list" {
		t.Errorf("expected Command 'list', got %q", cfgList.Command)
	}
	if cfgList.Prefix != "services" {
		t.Errorf("expected Prefix 'services', got %q", cfgList.Prefix)
	}
	if cfgList.Format != "json" {
		t.Errorf("expected Format 'json', got %q", cfgList.Format)
	}

	// Test empty args defaults to TUI
	cfgEmpty, err := LoadFromArgs([]string{})
	if err != nil {
		t.Fatalf("LoadFromArgs empty failed: %v", err)
	}
	if cfgEmpty.Command != "" {
		t.Errorf("expected empty Command for TUI, got %q", cfgEmpty.Command)
	}
}
