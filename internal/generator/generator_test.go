package generator

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateGoRetrievalCode_SyntaxValidity(t *testing.T) {
	data := map[string]string{
		"apiKey":   "abcdef12345",
		"endpoint": "https://api.example.com",
	}

	code := GenerateGoRetrievalCode("http://10.0.0.180:8200", "secret", "services/api", data)

	// Verify that the generated code is valid Go syntax using the standard library go/parser
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "main.go", code, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated code failed Go syntax parsing: %v\nCode:\n%s", err, code)
	}

	if !strings.Contains(code, "services/api") {
		t.Errorf("expected secret path in generated code")
	}
	if !strings.Contains(code, `http://10.0.0.180:8200`) {
		t.Errorf("expected vault address in generated code")
	}
	if !strings.Contains(code, `"apiKey"`) {
		t.Errorf("expected apiKey in generated code")
	}
}

func TestGenerateGoRetrievalCode_WithStructuredJSON(t *testing.T) {
	rabbitJSON := `{"host":"10.0.0.180","namespace":"default","password":"guest","port":5672,"username":"guest"}`
	data := map[string]string{
		"rabbitmq": rabbitJSON,
		"env":      "staging",
	}

	code := GenerateGoRetrievalCode("http://10.0.0.180:8200", "secret", "infra/rabbitmq", data)

	// Verify Go syntax
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "main.go", code, parser.AllErrors)
	if err != nil {
		t.Fatalf("generated code failed Go syntax parsing: %v\nCode:\n%s", err, code)
	}

	// Verify struct definition
	if !strings.Contains(code, "type RabbitmqConfig struct") {
		t.Errorf("expected RabbitmqConfig struct in code:\n%s", code)
	}
	if !strings.Contains(code, "Host string `json:\"host\"`") {
		t.Errorf("expected Host field in struct")
	}
	if !strings.Contains(code, "Port int `json:\"port\"`") {
		t.Errorf("expected Port field with int type in struct")
	}
	if !strings.Contains(code, "json.Unmarshal") {
		t.Errorf("expected json.Unmarshal in code")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"webapp/database", "get_webapp_database.go"},
		{"services/rabbitmq/prod", "get_services_rabbitmq_prod.go"},
		{"app@key#1", "get_app_key_1.go"},
		{"", "get_secret.go"},
	}

	for _, tt := range tests {
		res := SanitizeFilename(tt.input)
		if res != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q, expected %q", tt.input, res, tt.expected)
		}
	}
}

func TestSaveRetrievalFile(t *testing.T) {
	tempDir := t.TempDir()
	code := "package main\n\nfunc main() {}\n"

	filePath, err := SaveRetrievalFile(tempDir, "test.go", code)
	if err != nil {
		t.Fatalf("SaveRetrievalFile failed: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	if string(data) != code {
		t.Errorf("saved content mismatch: got %q, expected %q", string(data), code)
	}
	if filepath.Base(filePath) != "test.go" {
		t.Errorf("unexpected file path: %s", filePath)
	}
}
