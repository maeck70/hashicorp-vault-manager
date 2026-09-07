// Package config manages application configuration parsing, environment
// variable loading, and local .env persistence for the Vault TUI.
//
// Configuration precedence follows:
//  1. Explicit command-line flags (e.g. -addr, -token, -mount)
//  2. Environment variables from active shell and .env file (via godotenv)
//  3. Default fallback values (e.g. http://10.0.0.180:8200)
package config

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration for the Vault TUI.
type Config struct {
	// Address is the base URL of the HashiCorp Vault server (e.g. "http://10.0.0.180:8200").
	Address string

	// Token is the Vault client authentication token (e.g. root token or service token).
	Token string

	// Mount is the KV v2 secret engine mount path (default: "secret").
	Mount string

	// UnsealKey is an optional unseal shard key supplied via CLI or .env.
	UnsealKey string

	// EnvFile is the path to the environment file (default: ".env").
	EnvFile string
}

// Load parses command-line flags and merges with environment variables / .env file.
// It initializes a custom FlagSet to prevent interference during testing.
func Load() (*Config, error) {
	fs := flag.NewFlagSet("vault-tui", flag.ContinueOnError)

	var envFile string
	fs.StringVar(&envFile, "env", ".env", "Path to .env configuration file")

	// Pre-load .env file if available (ignores error if file doesn't exist yet)
	_ = godotenv.Load(envFile)

	defaultAddr := os.Getenv("VAULT_ADDR")
	if defaultAddr == "" {
		defaultAddr = "http://10.0.0.180:8200"
	}

	defaultToken := os.Getenv("VAULT_TOKEN")

	defaultMount := os.Getenv("VAULT_MOUNT")
	if defaultMount == "" {
		defaultMount = "secret"
	}

	defaultUnsealKey := os.Getenv("VAULT_UNSEAL_KEY")

	addrFlag := fs.String("addr", defaultAddr, "Vault server address (e.g. http://10.0.0.180:8200)")
	tokenFlag := fs.String("token", defaultToken, "Vault authentication token")
	mountFlag := fs.String("mount", defaultMount, "KV v2 mount path")
	unsealFlag := fs.String("unseal-key", defaultUnsealKey, "Vault unseal key (if unsealing via flag)")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	// Clean mount name (remove leading/trailing slashes)
	mount := strings.Trim(*mountFlag, "/")
	if mount == "" {
		mount = "secret"
	}

	// Clean address (remove trailing slash)
	addr := strings.TrimRight(*addrFlag, "/")

	return &Config{
		Address:   addr,
		Token:     *tokenFlag,
		Mount:     mount,
		UnsealKey: *unsealFlag,
		EnvFile:   envFile,
	}, nil
}

// SaveToEnv updates or appends key-values into the specified .env file.
// Comments and formatting in existing files are preserved, and updated
// files are written with restricted permissions (0600) to protect secrets.
func SaveToEnv(envFile string, updates map[string]string) error {
	if envFile == "" {
		envFile = ".env"
	}

	var lines []string

	if file, err := os.Open(envFile); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		scanErr := scanner.Err()
		_ = file.Close()
		if scanErr != nil {
			return fmt.Errorf("error reading %s: %w", envFile, scanErr)
		}
	}

	// Update existing lines or append new ones
	updatedKeys := make(map[string]bool, len(updates))
	newLines := make([]string, 0, len(lines)+len(updates))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			if newVal, ok := updates[key]; ok {
				newLines = append(newLines, fmt.Sprintf("%s=%s", key, newVal))
				updatedKeys[key] = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	// Append keys that weren't in existing lines
	for k, v := range updates {
		if !updatedKeys[k] {
			newLines = append(newLines, fmt.Sprintf("%s=%s", k, v))
		}
	}

	output := strings.Join(newLines, "\n") + "\n"
	return os.WriteFile(envFile, []byte(output), 0600)
}
