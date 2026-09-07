# Project Guidelines

## Technology Stack & Constraints

- **Language**: Go (Golang)
- **Standard Library First**: Prioritize Go standard libraries for core logic, architecture, and application functionality.
- **Preferred Modules**:
  - **Flags**: Use `flag` / `flags` for CLI argument and option parsing.
  - **Environment Management**: Use `godotenv` (`github.com/joho/godotenv`) for loading `.env` configuration.
- **TUI Interface**: Use Bubblegum (Bubble Tea / Charm TUI ecosystem) as the terminal user interface (TUI).
- **Build Output**: All compiled Go binaries must be placed in the `bin/` directory (e.g. `bin/<binary-name>`).
- **Performance & Idiomatic Go**:
  - Avoid inefficient string concatenation inside `WriteString` or `strings.Builder` calls. Use sequential `WriteString` calls, `WriteByte`/`WriteRune`, or `fmt.Fprintf` rather than `+` string concatenation.
  - Assign the result of type assertion to a variable in type switches (e.g. `switch val := val.(type)`) to eliminate redundant type assertions in case branches.
  - Use `any` instead of `interface{}` for modern, idiomatic Go.
