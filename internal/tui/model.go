package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

// ViewState represents the active screen or dialog within the TUI state machine.
type ViewState int

const (
	// StateChecking indicates the initial startup phase while connecting to Vault.
	StateChecking ViewState = iota

	// StateInit indicates the Vault initialization wizard screen.
	StateInit

	// StateUnseal indicates the Vault unseal screen.
	StateUnseal

	// StateList indicates the main secret browser path listing screen.
	StateList

	// StateDetail indicates the secret key-value inspection screen.
	StateDetail

	// StateEditor indicates the secret creation and edit form screen.
	StateEditor

	// StateConfirm indicates the delete confirmation modal.
	StateConfirm

	// StateStructBuilder indicates the nested JSON structure builder wizard.
	StateStructBuilder
)

// StructPreset defines pre-configured templates for complex service connection strings.
type StructPreset int

const (
	// PresetRabbitMQ provides fields for host, port, username, password, and namespace.
	PresetRabbitMQ StructPreset = iota

	// PresetPostgres provides fields for host, port, database, username, and password.
	PresetPostgres

	// PresetRedis provides fields for host, port, password, and database.
	PresetRedis

	// PresetCustom provides a blank template with dynamic field addition.
	PresetCustom
)

// Messages
type (
	HealthMsg        struct{ Health *vault.HealthResponse }
	SecretsLoadedMsg struct{ Keys []string }
	SecretDetailMsg  struct{ Item *vault.SecretItem }
	SecretSavedMsg   struct{ Path string }
	SecretDeletedMsg struct{ Path string }
	VaultInitMsg     struct{ Resp *vault.InitResponse }
	VaultUnsealMsg   struct{ Resp *vault.UnsealResponse }
	ErrMsg           struct{ Err error }
	ToastClearMsg    struct{}
)

// ToastType defines visual toast style
type ToastType int

const (
	ToastInfo ToastType = iota
	ToastSuccess
	ToastError
)

type Toast struct {
	Message string
	Type    ToastType
}

// Model is the main Bubble Tea model for the Vault TUI.
type Model struct {
	cfg        *config.Config
	client     *vault.Client
	state      ViewState
	width      int
	height     int
	health     *vault.HealthResponse
	err        error
	toast      *Toast
	toastTimer *time.Timer

	// List View State
	secrets       []string
	filteredList  []string
	selectedIndex int
	filterInput   textinput.Model
	isFiltering   bool

	// Detail View State
	selectedSecret *vault.SecretItem
	maskValues     bool
	detailRowIndex int
	sortedKeys     []string

	// Editor View State
	isEditingExisting bool
	editPathInput     textinput.Model
	editKeyInputs     []textinput.Model
	editValInputs     []textinput.Model
	activeField       int // 0 = path, 1..2N = key/value fields, 2N+1 = save, 2N+2 = cancel
	isRawJSONMode     bool
	rawJSONContent    string

	// Init / Unseal Form State
	initSharesInput    textinput.Model
	initThresholdInput textinput.Model
	unsealKeyInput     textinput.Model
	initResult         *vault.InitResponse

	// Confirm Dialog State
	pendingDeletePath string

	// Struct Builder State
	structKeyInput    textinput.Model
	structPreset      StructPreset
	structKeyInputs   []textinput.Model
	structValInputs   []textinput.Model
	structActiveField int // 0: Key name, 1..2M: sub-field key/val pairs, 2M+1: Save, 2M+2: Cancel
	structTargetRow   int // Row index in parent editor to populate (-1 for new row)
}

// NewModel creates an initialized Bubble Tea model.
func NewModel(cfg *config.Config, client *vault.Client) Model {
	// Filter input
	fi := textinput.New()
	fi.Placeholder = "Filter secrets (e.g. app/db)..."
	fi.Prompt = "🔍 "
	fi.PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
	fi.CharLimit = 64

	// Path input
	pi := textinput.New()
	pi.Placeholder = "path/to/secret"
	pi.Prompt = "Path: "
	pi.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	pi.Focus()

	// Init inputs
	sh := textinput.New()
	sh.Placeholder = "1"
	sh.SetValue("1")
	sh.Prompt = "Key Shares: "
	sh.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	th := textinput.New()
	th.Placeholder = "1"
	th.SetValue("1")
	th.Prompt = "Key Threshold: "
	th.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	// Unseal input
	ui := textinput.New()
	ui.Placeholder = "Enter unseal key..."
	ui.Prompt = "Unseal Key: "
	ui.PromptStyle = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
	if cfg.UnsealKey != "" {
		ui.SetValue(cfg.UnsealKey)
	}

	// Struct Key input
	ski := textinput.New()
	ski.Placeholder = "key_name (e.g. rabbitmq)"
	ski.Prompt = "JSON Key: "
	ski.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)

	m := Model{
		cfg:                cfg,
		client:             client,
		state:              StateChecking,
		maskValues:         true,
		filterInput:        fi,
		editPathInput:      pi,
		initSharesInput:    sh,
		initThresholdInput: th,
		unsealKeyInput:     ui,
		structKeyInput:     ski,
		structTargetRow:    -1,
	}

	m.initEditorRows(1)
	return m
}

func (m *Model) initEditorRows(count int) {
	m.editKeyInputs = make([]textinput.Model, count)
	m.editValInputs = make([]textinput.Model, count)
	for i := 0; i < count; i++ {
		ki := textinput.New()
		ki.Placeholder = fmt.Sprintf("Key #%d", i+1)
		ki.Prompt = "  Key: "
		ki.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)

		vi := textinput.New()
		vi.Placeholder = fmt.Sprintf("Value #%d", i+1)
		vi.Prompt = "Value: "
		vi.PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)

		m.editKeyInputs[i] = ki
		m.editValInputs[i] = vi
	}
}

// Init starts initial health checking command.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.checkHealthCmd(),
	)
}

// Commands
func (m Model) checkHealthCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h, err := m.client.GetHealth(ctx)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return HealthMsg{Health: h}
	}
}

func (m Model) loadSecretsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		keys, err := m.client.ListSecrets(ctx, "")
		if err != nil {
			return ErrMsg{Err: err}
		}
		return SecretsLoadedMsg{Keys: keys}
	}
}

func (m Model) loadSecretDetailCmd(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		item, err := m.client.GetSecret(ctx, path)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return SecretDetailMsg{Item: item}
	}
}

func (m Model) initializeVaultCmd(shares, threshold int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := m.client.Initialize(ctx, shares, threshold)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return VaultInitMsg{Resp: resp}
	}
}

func (m Model) unsealVaultCmd(key string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		resp, err := m.client.Unseal(ctx, key)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return VaultUnsealMsg{Resp: resp}
	}
}

func (m Model) saveSecretCmd(path string, data map[string]string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := m.client.PutSecret(ctx, path, data); err != nil {
			return ErrMsg{Err: err}
		}
		return SecretSavedMsg{Path: path}
	}
}

func (m Model) deleteSecretCmd(path string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := m.client.DeleteSecret(ctx, path); err != nil {
			return ErrMsg{Err: err}
		}
		return SecretDeletedMsg{Path: path}
	}
}

func clearToastCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return ToastClearMsg{}
	})
}

// Update handles message dispatching
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ToastClearMsg:
		m.toast = nil
		return m, nil

	case HealthMsg:
		m.health = msg.Health
		if !msg.Health.Initialized {
			m.state = StateInit
			m.initSharesInput.Focus()
			return m, textinput.Blink
		}
		if msg.Health.Sealed {
			m.state = StateUnseal
			m.unsealKeyInput.Focus()
			// If we already have an unseal key from .env / flag, auto-try unseal
			if m.cfg.UnsealKey != "" {
				return m, m.unsealVaultCmd(m.cfg.UnsealKey)
			}
			return m, textinput.Blink
		}
		// Initialized and unsealed!
		m.state = StateList
		return m, m.loadSecretsCmd()

	case VaultInitMsg:
		m.initResult = msg.Resp
		m.toast = &Toast{Message: "✓ Vault successfully initialized!", Type: ToastSuccess}

		// Save root token and first unseal key to .env
		updates := map[string]string{
			"VAULT_ADDR":  m.client.Address(),
			"VAULT_TOKEN": msg.Resp.RootToken,
		}
		if len(msg.Resp.Keys) > 0 {
			updates["VAULT_UNSEAL_KEY"] = msg.Resp.Keys[0]
			m.cfg.UnsealKey = msg.Resp.Keys[0]
		}
		_ = config.SaveToEnv(m.cfg.EnvFile, updates)

		m.client.SetToken(msg.Resp.RootToken)
		m.cfg.Token = msg.Resp.RootToken

		// If shares > 0, unseal with the first key immediately
		if len(msg.Resp.Keys) > 0 {
			return m, tea.Batch(
				m.unsealVaultCmd(msg.Resp.Keys[0]),
				clearToastCmd(),
			)
		}
		m.state = StateUnseal
		return m, clearToastCmd()

	case VaultUnsealMsg:
		if !msg.Resp.Sealed {
			m.toast = &Toast{Message: "✓ Vault unsealed successfully!", Type: ToastSuccess}
			if m.health != nil {
				m.health.Sealed = false
			}
			// Attempt to ensure KV v2 is enabled on default mount
			_ = m.client.EnableKVv2(context.Background(), m.client.Mount())
			m.state = StateList
			return m, tea.Batch(m.loadSecretsCmd(), clearToastCmd())
		}
		m.toast = &Toast{
			Message: fmt.Sprintf("Unseal progress: %d/%d shards entered", msg.Resp.Progress, msg.Resp.T),
			Type:    ToastInfo,
		}
		return m, clearToastCmd()

	case SecretsLoadedMsg:
		m.secrets = msg.Keys
		m.applyFilter()
		return m, nil

	case SecretDetailMsg:
		m.selectedSecret = msg.Item
		m.state = StateDetail
		m.detailRowIndex = 0
		m.sortedKeys = nil
		for k := range msg.Item.Data {
			m.sortedKeys = append(m.sortedKeys, k)
		}
		return m, nil

	case SecretSavedMsg:
		m.toast = &Toast{Message: fmt.Sprintf("✓ Secret '%s' saved!", msg.Path), Type: ToastSuccess}
		m.state = StateList
		return m, tea.Batch(m.loadSecretsCmd(), clearToastCmd())

	case SecretDeletedMsg:
		m.toast = &Toast{Message: fmt.Sprintf("✓ Secret '%s' deleted!", msg.Path), Type: ToastSuccess}
		m.state = StateList
		return m, tea.Batch(m.loadSecretsCmd(), clearToastCmd())

	case ErrMsg:
		m.err = msg.Err
		m.toast = &Toast{Message: fmt.Sprintf("Error: %v", msg.Err), Type: ToastError}
		return m, clearToastCmd()

	case tea.KeyMsg:
		// Global Quit key
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Route input based on state
		switch m.state {
		case StateInit:
			return m.handleInitKeys(msg)
		case StateUnseal:
			return m.handleUnsealKeys(msg)
		case StateList:
			return m.handleListKeys(msg)
		case StateDetail:
			return m.handleDetailKeys(msg)
		case StateEditor:
			return m.handleEditorKeys(msg)
		case StateConfirm:
			return m.handleConfirmKeys(msg)
		case StateStructBuilder:
			return m.handleStructBuilderKeys(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	if query == "" {
		m.filteredList = m.secrets
	} else {
		var filtered []string
		for _, s := range m.secrets {
			if strings.Contains(strings.ToLower(s), query) {
				filtered = append(filtered, s)
			}
		}
		m.filteredList = filtered
	}
	if m.selectedIndex >= len(m.filteredList) {
		m.selectedIndex = max(0, len(m.filteredList)-1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m Model) copyToClipboard(text string) (Model, tea.Cmd) {
	err := clipboard.WriteAll(text)
	if err != nil {
		m.toast = &Toast{Message: "Failed to copy to clipboard", Type: ToastError}
	} else {
		m.toast = &Toast{Message: "✓ Copied to clipboard!", Type: ToastSuccess}
	}
	return m, clearToastCmd()
}

// openStructBuilder initializes the Struct Builder sub-form
func (m *Model) openStructBuilder(targetRow int, initialKey string, preset StructPreset) {
	m.structTargetRow = targetRow
	m.structPreset = preset
	m.structActiveField = 0

	if initialKey != "" {
		m.structKeyInput.SetValue(initialKey)
	} else {
		switch preset {
		case PresetRabbitMQ:
			m.structKeyInput.SetValue("rabbitmq")
		case PresetPostgres:
			m.structKeyInput.SetValue("postgres")
		case PresetRedis:
			m.structKeyInput.SetValue("redis")
		default:
			m.structKeyInput.SetValue("")
		}
	}
	m.structKeyInput.Focus()

	// If targetRow has existing valid JSON, parse it to prefill
	if targetRow >= 0 && targetRow < len(m.editValInputs) {
		valStr := m.editValInputs[targetRow].Value()
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(valStr), &parsed); err == nil && len(parsed) > 0 {
			m.initStructRowsFromMap(parsed)
			m.state = StateStructBuilder
			return
		}
	}

	m.applyStructPreset(preset)
	m.state = StateStructBuilder
}

func (m *Model) applyStructPreset(preset StructPreset) {
	m.structPreset = preset
	switch preset {
	case PresetRabbitMQ:
		fields := []struct{ k, v string }{
			{"host", "10.0.0.180"},
			{"port", "5672"},
			{"username", "guest"},
			{"password", "guest"},
			{"namespace", "default"},
		}
		m.initStructRows(fields)
	case PresetPostgres:
		fields := []struct{ k, v string }{
			{"host", "10.0.0.180"},
			{"port", "5432"},
			{"database", "app_db"},
			{"username", "postgres"},
			{"password", ""},
		}
		m.initStructRows(fields)
	case PresetRedis:
		fields := []struct{ k, v string }{
			{"host", "10.0.0.180"},
			{"port", "6379"},
			{"password", ""},
			{"database", "0"},
		}
		m.initStructRows(fields)
	case PresetCustom:
		fields := []struct{ k, v string }{
			{"field1", ""},
		}
		m.initStructRows(fields)
	}
}

func (m *Model) initStructRows(fields []struct{ k, v string }) {
	m.structKeyInputs = make([]textinput.Model, len(fields))
	m.structValInputs = make([]textinput.Model, len(fields))
	for i, f := range fields {
		ki := textinput.New()
		ki.Placeholder = "sub-key"
		ki.Prompt = "  Sub-Key: "
		ki.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
		ki.SetValue(f.k)

		vi := textinput.New()
		vi.Placeholder = "sub-value"
		vi.Prompt = "Sub-Value: "
		vi.PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		vi.SetValue(f.v)

		m.structKeyInputs[i] = ki
		m.structValInputs[i] = vi
	}
}

func (m *Model) initStructRowsFromMap(data map[string]interface{}) {
	count := len(data)
	if count == 0 {
		m.applyStructPreset(PresetCustom)
		return
	}
	m.structKeyInputs = make([]textinput.Model, count)
	m.structValInputs = make([]textinput.Model, count)
	i := 0
	for k, v := range data {
		ki := textinput.New()
		ki.Placeholder = "sub-key"
		ki.Prompt = "  Sub-Key: "
		ki.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
		ki.SetValue(k)

		vi := textinput.New()
		vi.Placeholder = "sub-value"
		vi.Prompt = "Sub-Value: "
		vi.PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		vi.SetValue(fmt.Sprintf("%v", v))

		m.structKeyInputs[i] = ki
		m.structValInputs[i] = vi
		i++
	}
}

func (m *Model) appendStructRow() {
	count := len(m.structKeyInputs) + 1
	newKeys := make([]textinput.Model, count)
	newVals := make([]textinput.Model, count)
	copy(newKeys, m.structKeyInputs)
	copy(newVals, m.structValInputs)

	ki := textinput.New()
	ki.Placeholder = fmt.Sprintf("Sub-Key #%d", count)
	ki.Prompt = "  Sub-Key: "
	ki.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)

	vi := textinput.New()
	vi.Placeholder = fmt.Sprintf("Sub-Value #%d", count)
	vi.Prompt = "Sub-Value: "
	vi.PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)

	newKeys[count-1] = ki
	newVals[count-1] = vi

	m.structKeyInputs = newKeys
	m.structValInputs = newVals
}

func (m *Model) removeCurrentStructRow() {
	if len(m.structKeyInputs) <= 1 {
		return
	}
	rowIdx := max(0, (m.structActiveField-1)/2)
	if rowIdx >= len(m.structKeyInputs) {
		rowIdx = len(m.structKeyInputs) - 1
	}

	var newKeys []textinput.Model
	var newVals []textinput.Model
	for i := range m.structKeyInputs {
		if i != rowIdx {
			newKeys = append(newKeys, m.structKeyInputs[i])
			newVals = append(newVals, m.structValInputs[i])
		}
	}
	m.structKeyInputs = newKeys
	m.structValInputs = newVals
	m.structActiveField = min(m.structActiveField, len(m.structKeyInputs)*2)
}

func (m Model) collectStructData() map[string]interface{} {
	data := make(map[string]interface{})
	for i := range m.structKeyInputs {
		k := strings.TrimSpace(m.structKeyInputs[i].Value())
		v := m.structValInputs[i].Value()
		if k != "" {
			// If integer, store as number in JSON
			if intVal, err := strconv.Atoi(v); err == nil {
				data[k] = intVal
			} else if boolVal, err := strconv.ParseBool(v); err == nil {
				data[k] = boolVal
			} else {
				data[k] = v
			}
		}
	}
	return data
}

func (m Model) buildStructJSONString() string {
	data := m.collectStructData()
	b, _ := json.Marshal(data)
	return string(b)
}

