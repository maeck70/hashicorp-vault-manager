package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

// RunListSecrets lists keys and folders under the specified prefix in Vault via the CLI.
func RunListSecrets(client *vault.Client, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	keys, err := client.ListSecrets(ctx, cfg.Prefix)
	if err != nil {
		return fmt.Errorf("failed to list secrets at '%s': %w", cfg.Prefix, err)
	}

	sort.Strings(keys)

	// 1. JSON format
	if cfg.Format == "json" {
		payload := map[string]any{
			"mount":  client.Mount(),
			"prefix": cfg.Prefix,
			"count":  len(keys),
			"keys":   keys,
		}
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode json: %w", err)
		}
		fmt.Fprintln(OutputWriter, string(encoded))
		return nil
	}

	// 2. Table / human-readable format
	prefixDisplay := cfg.Prefix
	if prefixDisplay == "" {
		prefixDisplay = "(root)"
	}

	if len(keys) == 0 {
		fmt.Fprintf(OutputWriter, "No secrets found under %s/%s\n", client.Mount(), prefixDisplay)
		return nil
	}

	fmt.Fprintf(OutputWriter, "=== Vault Secrets: %s/%s (%d items) ===\n\n", client.Mount(), prefixDisplay, len(keys))
	for _, key := range keys {
		if strings.HasSuffix(key, "/") {
			fmt.Fprintf(OutputWriter, "  📁 %s\n", key)
		} else {
			fmt.Fprintf(OutputWriter, "  🔑 %s\n", key)
		}
	}

	return nil
}
