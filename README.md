# HashiCorp Vault TUI to manage secrets.

A fast, intuitive, and colorful Terminal User Interface (TUI) in Go for managing secrets on HashiCorp Vault. Designed specifically for experimental and rapid development workflows, with built-in in-terminal initialization, unsealing, and complete KV v2 secret lifecycle management.

---

## Features

- **Standard Library First**: Direct, high-performance HTTP REST client built purely with Go's `net/http` and `encoding/json` — zero heavy external Vault SDK dependencies.
- **Vibrant Charm TUI**: Built with Bubble Tea, Bubbles, and Lip Gloss featuring a cyber-purple/cyan/neon aesthetic with status badges and responsive layouts.
- **In-Terminal Init & Unseal**:
  - Automatically detects uninitialized or sealed Vault states.
  - Interactive wizard to initialize Vault (default 1 share / 1 threshold for dev/experimentation).
  - Automatically unseals and persists the generated root token and unseal keys directly into your local `.env`.
- **KV Version 2 Engine**: Full support for versioned secrets at any mount path (default: `secret/`).
- **JSON Structure Builder**:
  - Define complex structured secrets (e.g. `rabbitmq` with `host`, `port`, `username`, `password`, and `namespace`).
  - Built-in presets for **RabbitMQ**, **PostgreSQL**, **Redis**, and **Custom** objects.
  - Live preview card rendering formatted JSON in real time as data is captured.
  - Automatically serializes and saves to Vault as a JSON string.
  - Intuitive editing of existing structures (`Ctrl+E`).
- **Create & Maintain Secrets**:
  - Interactive multi-row Key-Value form with `[Tab]` field cycling.
  - Add (`Ctrl+N`) or remove (`Ctrl+D`) key-value pairs dynamically.
  - Structured JSON builder (`Ctrl+T`) and instant toggle (`Ctrl+J`) to switch to raw JSON mode.
  - Safe version deletion with an explicit confirmation dialog.
- **Retrieve & Masking**:
  - Automatic detection of JSON strings in secret values with formatted sub-field tree view.
  - Masked values and nested passwords by default (`••••••••`) to prevent shoulder-surfing.
  - Toggle masking on/off with `m`.
  - One-key copy to system clipboard (`y` or `c`) for secret values and JSON payloads.
- **Search & Filter**: Real-time filtering across secret paths with `/`.

---

## Architecture & Technology Stack

- **Language**: Go 1.27+
- **CLI Options**: Standard library `flag` package
- **Environment Management**: `godotenv` (`github.com/joho/godotenv`)
- **TUI Framework**: Charm ecosystem (`bubbletea`, `bubbles`, `lipgloss`)
- **Clipboard**: Cross-platform system clipboard integration (`github.com/atotto/clipboard`)

```
HashiCorp-Vault-Experiment/
├── .env.example              # Example environment configuration
├── go.mod                    # Go module dependencies
├── main.go                   # Application entry point
├── internal/
│   ├── config/               # Flag parsing, godotenv integration, .env persistence
│   ├── vault/                # Pure standard library Vault HTTP client & models
│   └── tui/                  # Bubble Tea state machine, views, and Lip Gloss styling
└── bin/
    └── vault-tui             # Compiled executable
```

---

## Getting Started

### Prerequisites

- Go 1.22+ installed
- Network access to your Vault server (default: `http://10.0.0.180:8200`)

### Installation & Build

Clone the repository and build the binary:

```bash
git clone <repo-url>
cd HashiCorp-Vault-Experiment
go build -o bin/vault-tui .
```

### Running the TUI

Run with defaults (connects to `http://10.0.0.180:8200`):

```bash
./bin/vault-tui
```

Or specify custom flags:

```bash
./bin/vault-tui -addr http://10.0.0.180:8200 -mount secret -token <your-token>
```

---

## Configuration

Configuration is loaded with the following precedence:
1. Command-line flags
2. Environment variables / `.env` file
3. Built-in defaults

### Command-Line Flags

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-addr` | Vault server URL | `http://10.0.0.180:8200` (or `$VAULT_ADDR`) |
| `-token` | Vault authentication token | `$VAULT_TOKEN` |
| `-mount` | KV v2 secret engine mount path | `secret` (or `$VAULT_MOUNT`) |
| `-unseal-key` | Vault unseal shard key | `$VAULT_UNSEAL_KEY` |
| `-env` | Path to `.env` file | `.env` |

### Environment Variables (`.env`)

Copy `.env.example` to `.env`:

```bash
cp .env.example .env
```

```env
VAULT_ADDR=http://10.0.0.180:8200
VAULT_TOKEN=
VAULT_MOUNT=secret
VAULT_UNSEAL_KEY=
```

> **Note**: When you initialize or unseal Vault via the TUI wizard, your `VAULT_TOKEN` and `VAULT_UNSEAL_KEY` are automatically saved to your `.env` file!

---

## Keyboard Shortcuts Reference

### Global
- `Ctrl+C`: Quit application

### Secret List View
- `↑` / `k`: Move up in secret list
- `↓` / `j`: Move down in secret list
- `Enter`: View secret details
- `n`: Create new secret
- `e`: Edit selected secret
- `d`: Delete selected secret
- `/`: Search and filter secrets
- `r`: Refresh secret list from Vault
- `q`: Quit

### Secret Detail View
- `m`: Toggle value masking (`••••••••` vs plain text)
- `y` / `c`: Copy selected key value to clipboard
- `↑` / `k`: Select previous key-value row
- `↓` / `j`: Select next key-value row
- `e`: Edit this secret
- `d`: Delete this secret version
- `Esc` / `Backspace` / `h`: Back to secret list

### Secret Editor View
- `Tab`: Jump to next input field / Save button
- `Shift+Tab`: Jump to previous input field
- `Ctrl+T`: Open JSON Structure Builder wizard (RabbitMQ, Postgres, Redis, Custom)
- `Ctrl+E`: Edit existing JSON structure for focused key
- `Ctrl+N` / `Ctrl+A`: Add a new key-value row
- `Ctrl+D`: Remove currently focused row
- `Ctrl+J`: Toggle between Key-Value Form and raw JSON editor
- `Ctrl+S` / `Enter` on Save: Save secret to Vault
- `Esc`: Cancel and discard changes

### JSON Structure Builder Wizard
- `Tab` / `Shift+Tab`: Navigate between Key name, sub-fields, and Apply button
- `Ctrl+P`: Cycle preset template (RabbitMQ → PostgreSQL → Redis → Custom)
- `Ctrl+N`: Add an additional sub-field
- `Ctrl+D`: Remove currently selected sub-field
- `Ctrl+S` / `Enter` on Apply: Serialize to JSON string and apply to secret editor
- `Esc`: Cancel and return to secret editor

### Initialization & Unseal Wizard
- `Tab`: Switch between Key Shares and Threshold fields
- `Enter`: Submit initialization / unseal shard
- `q` / `Esc`: Quit

---

## Running Tests

Run all unit tests across the entire codebase:

```bash
go test -v ./...
```

The test suite covers:
- Vault client health parsing (200, 501 uninitialized, 503 sealed)
- Vault initialization and unseal API flows
- Full KV v2 lifecycle (List, Put, Get, Delete)
- Dynamic `.env` updating and persistence
- TUI state routing, real-time filtering, row manipulation, and secret masking
