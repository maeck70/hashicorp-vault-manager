# Project Guidelines

## Technology Stack & Constraints

- **Language**: Go (Golang)
- **Standard Library First**: Prioritize Go standard libraries for core logic, architecture, and application functionality.
- **Preferred Modules**:
  - **Flags**: Use `flag` / `flags` for CLI argument and option parsing.
  - **Environment Management**: Use `godotenv` (`github.com/joho/godotenv`) for loading `.env` configuration.
- **TUI Interface**: Use Bubblegum (Bubble Tea / Charm TUI ecosystem) as the terminal user interface (TUI).
- **Build Output**: All compiled Go binaries must be placed in the `bin/` directory (e.g. `bin/<binary-name>`).
