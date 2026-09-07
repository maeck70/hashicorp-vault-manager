package tui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

func TestTUI_FilterLogic(t *testing.T) {
	cfg := &config.Config{
		Address: "http://10.0.0.180:8200",
		Mount:   "secret",
	}
	client := vault.NewClient(cfg.Address, "", cfg.Mount)
	m := NewModel(cfg, client)

	m.secrets = []string{"webapp/database", "webapp/redis", "payment/stripe", "auth/jwt"}
	m.applyFilter()
	if len(m.filteredList) != 4 {
		t.Fatalf("expected 4 items in unfiltered list, got %d", len(m.filteredList))
	}

	// Apply filter "redis"
	m.filterInput.SetValue("redis")
	m.applyFilter()
	if len(m.filteredList) != 1 || m.filteredList[0] != "webapp/redis" {
		t.Errorf("unexpected filtered result: %v", m.filteredList)
	}

	// Apply filter "webapp"
	m.filterInput.SetValue("webapp")
	m.applyFilter()
	if len(m.filteredList) != 2 {
		t.Errorf("expected 2 items matching 'webapp', got %d", len(m.filteredList))
	}

	// Clear filter
	m.filterInput.SetValue("")
	m.applyFilter()
	if len(m.filteredList) != 4 {
		t.Errorf("expected 4 items after clearing filter, got %d", len(m.filteredList))
	}
}

func TestTUI_HealthRouting(t *testing.T) {
	cfg := &config.Config{
		Address: "http://10.0.0.180:8200",
		Mount:   "secret",
	}
	client := vault.NewClient(cfg.Address, "", cfg.Mount)

	// 1. Uninitialized
	m := NewModel(cfg, client)
	updated, _ := m.Update(HealthMsg{
		Health: &vault.HealthResponse{
			Initialized: false,
			Sealed:      true,
			Version:     "1.18.4",
		},
	})
	m = updated.(Model)
	if m.state != StateInit {
		t.Errorf("expected StateInit when uninitialized, got %v", m.state)
	}

	// 2. Sealed
	m2 := NewModel(cfg, client)
	updated2, _ := m2.Update(HealthMsg{
		Health: &vault.HealthResponse{
			Initialized: true,
			Sealed:      true,
			Version:     "1.18.4",
		},
	})
	m2 = updated2.(Model)
	if m2.state != StateUnseal {
		t.Errorf("expected StateUnseal when sealed, got %v", m2.state)
	}

	// 3. Initialized & Unsealed
	m3 := NewModel(cfg, client)
	updated3, _ := m3.Update(HealthMsg{
		Health: &vault.HealthResponse{
			Initialized: true,
			Sealed:      false,
			Version:     "1.18.4",
		},
	})
	m3 = updated3.(Model)
	if m3.state != StateList {
		t.Errorf("expected StateList when unsealed, got %v", m3.state)
	}
}

func TestTUI_EditorDataCollection(t *testing.T) {
	cfg := &config.Config{
		Address: "http://10.0.0.180:8200",
		Mount:   "secret",
	}
	client := vault.NewClient(cfg.Address, "", cfg.Mount)
	m := NewModel(cfg, client)

	m.editPathInput.SetValue("myapp/config")
	m.initEditorRows(2)
	m.editKeyInputs[0].SetValue("DB_HOST")
	m.editValInputs[0].SetValue("localhost")
	m.editKeyInputs[1].SetValue("DB_PORT")
	m.editValInputs[1].SetValue("5432")

	data := m.collectEditorData()
	if data["DB_HOST"] != "localhost" || data["DB_PORT"] != "5432" {
		t.Errorf("unexpected collected data: %v", data)
	}

	// Test adding a row
	m.appendEditorRow()
	if len(m.editKeyInputs) != 3 {
		t.Errorf("expected 3 rows after append, got %d", len(m.editKeyInputs))
	}

	// Test removing a row
	m.activeField = 3 // selects row 1
	m.removeCurrentEditorRow()
	if len(m.editKeyInputs) != 2 {
		t.Errorf("expected 2 rows after remove, got %d", len(m.editKeyInputs))
	}
}

func TestTUI_DetailViewRendering(t *testing.T) {
	cfg := &config.Config{
		Address: "http://10.0.0.180:8200",
		Mount:   "secret",
	}
	client := vault.NewClient(cfg.Address, "", cfg.Mount)
	m := NewModel(cfg, client)

	item := &vault.SecretItem{
		Path: "test/secret",
		Data: map[string]string{
			"api_token": "super-secret-value-12345",
		},
		Version:     1,
		CreatedTime: time.Now(),
	}

	updated, _ := m.Update(SecretDetailMsg{Item: item})
	m = updated.(Model)

	if m.state != StateDetail {
		t.Fatalf("expected StateDetail, got %v", m.state)
	}

	view := m.View()
	if len(view) == 0 {
		t.Errorf("expected non-empty view output")
	}

	// Should be masked initially
	if contains(view, "super-secret-value-12345") {
		t.Errorf("expected secret value to be masked in initial view, but plaintext was found")
	}

	// Toggle mask
	m.maskValues = false
	unmaskedView := m.View()
	if !contains(unmaskedView, "super-secret-value-12345") {
		t.Errorf("expected plaintext in unmasked view")
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

func TestTUI_StructBuilder_RabbitMQ(t *testing.T) {
	cfg := &config.Config{
		Address: "http://10.0.0.180:8200",
		Mount:   "secret",
	}
	client := vault.NewClient(cfg.Address, "", cfg.Mount)
	m := NewModel(cfg, client)

	// Open Struct Builder with RabbitMQ preset
	m.openStructBuilder(-1, "rabbitmq", PresetRabbitMQ)

	if m.state != StateStructBuilder {
		t.Fatalf("expected StateStructBuilder, got %v", m.state)
	}
	if m.structKeyInput.Value() != "rabbitmq" {
		t.Errorf("expected key 'rabbitmq', got '%s'", m.structKeyInput.Value())
	}
	if len(m.structKeyInputs) != 5 {
		t.Fatalf("expected 5 fields for RabbitMQ preset, got %d", len(m.structKeyInputs))
	}

	// Customize host & namespace
	m.structValInputs[0].SetValue("10.0.0.50")  // host
	m.structValInputs[4].SetValue("production") // namespace

	// Check JSON serialization
	jsonStr := m.buildStructJSONString()
	if !contains(jsonStr, `"host":"10.0.0.50"`) {
		t.Errorf("expected host in json: %s", jsonStr)
	}
	if !contains(jsonStr, `"port":5672`) {
		t.Errorf("expected port in json: %s", jsonStr)
	}
	if !contains(jsonStr, `"namespace":"production"`) {
		t.Errorf("expected namespace in json: %s", jsonStr)
	}

	// Apply to editor
	updated, _ := m.commitStructApply()
	m = updated.(Model)

	if m.state != StateEditor {
		t.Fatalf("expected StateEditor after apply, got %v", m.state)
	}
	if m.editKeyInputs[0].Value() != "rabbitmq" {
		t.Errorf("expected editor key 'rabbitmq', got '%s'", m.editKeyInputs[0].Value())
	}
	if m.editValInputs[0].Value() != jsonStr {
		t.Errorf("expected editor value to match jsonStr")
	}
}

func TestTUI_DetailView_JSONRendering(t *testing.T) {
	cfg := &config.Config{
		Address: "http://10.0.0.180:8200",
		Mount:   "secret",
	}
	client := vault.NewClient(cfg.Address, "", cfg.Mount)
	m := NewModel(cfg, client)

	rabbitJSON := `{"host":"10.0.0.180","namespace":"prod","password":"secretpass","port":5672,"username":"guest"}`
	item := &vault.SecretItem{
		Path: "infra/rabbitmq",
		Data: map[string]string{
			"rabbitmq": rabbitJSON,
		},
		Version:     1,
		CreatedTime: time.Now(),
	}

	updated, _ := m.Update(SecretDetailMsg{Item: item})
	m = updated.(Model)

	view := m.View()
	if !contains(view, "rabbitmq") {
		t.Errorf("expected 'rabbitmq' in detail view: %s", view)
	}
	if !contains(view, "JSON") {
		t.Errorf("expected 'JSON' badge in detail view: %s", view)
	}
	if !contains(view, "host:") || !contains(view, "10.0.0.180") {
		t.Errorf("expected sub-field host in detail view: %s", view)
	}
	if !contains(view, "port:") || !contains(view, "5672") {
		t.Errorf("expected sub-field port in detail view: %s", view)
	}

	// Masked password verification
	if contains(view, "secretpass") {
		t.Errorf("expected password to be masked, but found 'secretpass'")
	}
	if !contains(view, "••••••••") {
		t.Errorf("expected masked bullets for password field")
	}

	// Unmasked password verification
	m.maskValues = false
	unmasked := m.View()
	if !contains(unmasked, "secretpass") {
		t.Errorf("expected 'secretpass' in unmasked view: %s", unmasked)
	}
}

func TestTUI_DetailView_GenerateGoCode(t *testing.T) {
	cfg := &config.Config{
		Address: "http://10.0.0.180:8200",
		Mount:   "secret",
	}
	client := vault.NewClient(cfg.Address, "", cfg.Mount)
	m := NewModel(cfg, client)

	item := &vault.SecretItem{
		Path: "test/service",
		Data: map[string]string{
			"api_key": "123456",
		},
		Version:     1,
		CreatedTime: time.Now(),
	}

	updated, _ := m.Update(SecretDetailMsg{Item: item})
	m = updated.(Model)

	// Press 'g'
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)

	if m.toast == nil || !contains(m.toast.Message, "Generated Go file") {
		t.Errorf("expected toast confirming generated Go file, got: %v", m.toast)
	}

	expectedFile := "examples/get_test_service.go"
	data, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("expected generated file %s to exist: %v", expectedFile, err)
	}
	defer os.Remove(expectedFile)

	if !contains(string(data), "test/service") {
		t.Errorf("expected secret path in generated file: %s", string(data))
	}
}
