// Package cli provides non-interactive, script-friendly command-line operations
// for HashiCorp Vault without launching the terminal user interface.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

// OutputWriter directs CLI output (defaults to os.Stdout, customizable for tests).
var OutputWriter io.Writer = os.Stdout

// RunGetSecret retrieves a secret from Vault and outputs it to stdout according to cfg settings.
func RunGetSecret(client *vault.Client, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	item, err := client.GetSecret(ctx, cfg.GetPath)
	if err != nil {
		return err
	}

	if item.IsDeleted {
		delTimeStr := ""
		if !item.DeletionTime.IsZero() {
			delTimeStr = fmt.Sprintf(" at %s", item.DeletionTime.Format(time.RFC3339))
		}
		return fmt.Errorf("secret '%s' is soft-deleted in Vault (version %d deleted%s)", item.Path, item.Version, delTimeStr)
	}

	// 1. Single Field Extraction (-field)
	if cfg.Field != "" {
		val, exists := item.Data[cfg.Field]
		if !exists {
			return fmt.Errorf("field '%s' not found in secret '%s'", cfg.Field, item.Path)
		}
		fmt.Fprintln(OutputWriter, val)
		return nil
	}

	// 2. JSON Format (-format=json or -json)
	if cfg.Format == "json" {
		payload := map[string]any{
			"path":         item.Path,
			"version":      item.Version,
			"created_time": item.CreatedTime.Format(time.RFC3339Nano),
			"data":         item.Data,
		}
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode json: %w", err)
		}
		fmt.Fprintln(OutputWriter, string(encoded))
		return nil
	}

	// 3. Table Format (Default)
	keys := make([]string, 0, len(item.Data))
	maxKeyLen := 3
	for k := range item.Data {
		keys = append(keys, k)
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}
	sort.Strings(keys)

	fmt.Fprintf(OutputWriter, "=== Vault Secret: %s/%s (v%d) ===\n", client.Mount(), item.Path, item.Version)
	fmt.Fprintf(OutputWriter, "Created: %s\n\n", item.CreatedTime.Format("2006-01-02 15:04:05 MST"))

	formatStr := fmt.Sprintf("%%-%ds   %%s\n", maxKeyLen)
	fmt.Fprintf(OutputWriter, formatStr, "KEY", "VALUE")
	fmt.Fprintf(OutputWriter, "%s   %s\n", strings.Repeat("-", maxKeyLen), strings.Repeat("-", 30))

	for _, k := range keys {
		val := item.Data[k]

		var parsedMap map[string]any
		trimmed := strings.TrimSpace(val)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") && json.Unmarshal([]byte(trimmed), &parsedMap) == nil && len(parsedMap) > 0 {
			fmt.Fprintf(OutputWriter, formatStr, k, "(JSON Object)")
			subKeys := make([]string, 0, len(parsedMap))
			for sk := range parsedMap {
				subKeys = append(subKeys, sk)
			}
			sort.Strings(subKeys)
			for _, sk := range subKeys {
				fmt.Fprintf(OutputWriter, "  ↳ %s: %v\n", sk, parsedMap[sk])
			}
		} else {
			fmt.Fprintf(OutputWriter, formatStr, k, val)
		}
	}

	return nil
}
