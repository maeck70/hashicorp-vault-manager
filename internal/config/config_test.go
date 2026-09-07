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
