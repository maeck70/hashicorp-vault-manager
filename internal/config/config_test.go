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
}
