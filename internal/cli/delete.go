package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"vault-experiment/internal/config"
	"vault-experiment/internal/vault"
)

// InputReader directs CLI input for confirmation prompts (defaults to os.Stdin).
var InputReader io.Reader = os.Stdin

// RunDeleteSecret removes a secret from Vault via the CLI.
func RunDeleteSecret(client *vault.Client, cfg *config.Config) error {
	if cfg.TargetPath == "" {
		return fmt.Errorf("secret path required for delete command (e.g. 'vault-tui delete <path>')")
	}

	mode := "permanently"
	action := "permanent_delete"
	if cfg.SoftDelete {
		mode = "soft-delete (latest version)"
		action = "soft_delete"
	}

	// 1. Interactive confirmation prompt unless forced
	if !cfg.Force {
		fmt.Fprintf(OutputWriter, "Are you sure you want to %s delete secret '%s/%s'? [y/N]: ", mode, client.Mount(), cfg.TargetPath)
		reader := bufio.NewReader(InputReader)
		answer, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(OutputWriter, "Deletion aborted.")
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 2. Perform deletion
	if cfg.SoftDelete {
		if err := client.SoftDeleteSecret(ctx, cfg.TargetPath); err != nil {
			return fmt.Errorf("failed to soft-delete secret '%s': %w", cfg.TargetPath, err)
		}
	} else {
		if err := client.DeleteSecret(ctx, cfg.TargetPath); err != nil {
			return fmt.Errorf("failed to delete secret '%s': %w", cfg.TargetPath, err)
		}
	}

	// 3. JSON output
	if cfg.Format == "json" {
		payload := map[string]any{
			"status": "success",
			"action": action,
			"mount":  client.Mount(),
			"path":   cfg.TargetPath,
		}
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode json: %w", err)
		}
		fmt.Fprintln(OutputWriter, string(encoded))
		return nil
	}

	// 4. Human-readable output
	if cfg.SoftDelete {
		fmt.Fprintf(OutputWriter, "✓ Secret '%s/%s' soft-deleted (latest version marked deleted)\n", client.Mount(), cfg.TargetPath)
	} else {
		fmt.Fprintf(OutputWriter, "✓ Secret '%s/%s' permanently deleted and purged from Vault\n", client.Mount(), cfg.TargetPath)
	}

	return nil
}
