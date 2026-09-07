package config

import (
	"os"
	"path/filepath"
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
	if !contains(content, "EXISTING_KEY=updated_value") {
		t.Errorf("expected EXISTING_KEY=updated_value in content: %s", content)
	}
	if !contains(content, "NEW_KEY=new_value_123") {
		t.Errorf("expected NEW_KEY=new_value_123 in content: %s", content)
	}
	if !contains(content, "KEEP_ME=yes") {
		t.Errorf("expected KEEP_ME=yes to be preserved in content: %s", content)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
