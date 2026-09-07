package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

// RunPutSecret creates or updates a secret in Vault via the CLI.
func RunPutSecret(client *vault.Client, cfg *config.Config) error {
	if cfg.TargetPath == "" {
		return fmt.Errorf("secret path required for put/update command (e.g. 'vault-tui put <path> key=val')")
	}

	data := make(map[string]string)

	// 1. Process -data flag (inline JSON or @filepath)
	if cfg.RawData != "" {
		rawJSON := cfg.RawData
		if strings.HasPrefix(rawJSON, "@") {
			filePath := strings.TrimPrefix(rawJSON, "@")
			fileBytes, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read data file '%s': %w", filePath, err)
			}
			rawJSON = string(fileBytes)
		}

		var parsedMap map[string]any
		if err := json.Unmarshal([]byte(rawJSON), &parsedMap); err != nil {
			return fmt.Errorf("failed to parse JSON data: %w", err)
		}

		for k, v := range parsedMap {
			switch val := v.(type) {
			case string:
				data[k] = val
			case map[string]any, []any:
				bytes, err := json.Marshal(val)
				if err != nil {
					data[k] = fmt.Sprintf("%v", val)
				} else {
					data[k] = string(bytes)
				}
			default:
				data[k] = fmt.Sprintf("%v", val)
			}
		}
	}

	// 2. Process command-line key=value pairs (overwrites -data if key conflicts)
	for k, v := range cfg.DataArgs {
		data[k] = v
	}

	if len(data) == 0 {
		return fmt.Errorf("no secret data provided (specify key=value pairs or use -data '<json>')")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 3. If merge mode is requested, fetch existing keys and merge
	if cfg.Merge {
		existing, err := client.GetSecret(ctx, cfg.TargetPath)
		if err == nil && existing != nil && !existing.IsDeleted {
			merged := make(map[string]string, len(existing.Data)+len(data))
			for ek, ev := range existing.Data {
				merged[ek] = ev
			}
			for nk, nv := range data {
				merged[nk] = nv
			}
			data = merged
		}
	}

	// 4. Write data to Vault KV v2 engine
	if err := client.PutSecret(ctx, cfg.TargetPath, data); err != nil {
		return fmt.Errorf("failed to write secret '%s': %w", cfg.TargetPath, err)
	}

	// 5. Read back metadata to display updated version and timestamp
	written, err := client.GetSecret(ctx, cfg.TargetPath)
	var version int
	var createdTime time.Time
	if err == nil && written != nil {
		version = written.Version
		createdTime = written.CreatedTime
	}

	keysWritten := make([]string, 0, len(data))
	for k := range data {
		keysWritten = append(keysWritten, k)
	}
	sort.Strings(keysWritten)

	// 6. JSON output
	if cfg.Format == "json" {
		payload := map[string]any{
			"status":       "success",
			"mount":        client.Mount(),
			"path":         cfg.TargetPath,
			"version":      version,
			"keys_written": keysWritten,
		}
		if !createdTime.IsZero() {
			payload["created_time"] = createdTime.Format(time.RFC3339Nano)
		}
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode json: %w", err)
		}
		fmt.Fprintln(OutputWriter, string(encoded))
		return nil
	}

	// 7. Human-readable output
	fmt.Fprintf(OutputWriter, "✓ Secret successfully written to Vault\n")
	fmt.Fprintf(OutputWriter, "  Path:    %s/%s\n", client.Mount(), cfg.TargetPath)
	if version > 0 {
		fmt.Fprintf(OutputWriter, "  Version: %d\n", version)
	}
	if !createdTime.IsZero() {
		fmt.Fprintf(OutputWriter, "  Created: %s\n", createdTime.Format("2006-01-02 15:04:05 MST"))
	}
	fmt.Fprintf(OutputWriter, "  Keys (%d):\n", len(keysWritten))
	for _, k := range keysWritten {
		fmt.Fprintf(OutputWriter, "    • %s\n", k)
	}

	return nil
}
