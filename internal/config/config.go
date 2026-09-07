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

// Config holds runtime configuration for the Vault TUI and CLI operations.
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

	// Command specifies the CLI operation ("get", "put", "delete", "list", or "" for interactive TUI).
	Command string

	// TargetPath specifies the target secret path in Vault.
	TargetPath string

	// DataArgs holds key=value pairs supplied via CLI arguments for put/update.
	DataArgs map[string]string

	// RawData holds JSON string or @filepath supplied via -data flag.
	RawData string

	// Field specifies a single secret field to extract and print to stdout (for get).
	Field string

	// Format specifies the CLI output format ("table", "json", or "raw").
	Format string

	// Force skips interactive confirmation prompts (for delete).
	Force bool

	// SoftDelete performs a KV v2 soft delete rather than permanent metadata destruction.
	SoftDelete bool

	// Merge preserves existing keys and merges new key=values during put/update.
	Merge bool

	// Prefix specifies the path prefix to list secrets from.
	Prefix string

	// GetPath is preserved for backwards compatibility with existing code and tests.
	GetPath string
}

// Load parses command-line flags and merges with environment variables / .env file.
func Load() (*Config, error) {
	return LoadFromArgs(os.Args[1:])
}

// LoadFromArgs parses command-line arguments and configuration settings.
// It initializes a custom FlagSet to prevent interference during testing.
func LoadFromArgs(args []string) (*Config, error) {
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

	var getFlag string
	fs.StringVar(&getFlag, "get", "", "Retrieve a secret path directly via CLI")
	fs.StringVar(&getFlag, "read", "", "Alias for -get")

	var putFlag string
	fs.StringVar(&putFlag, "put", "", "Write/create/update a secret path directly via CLI")
	fs.StringVar(&putFlag, "set", "", "Alias for -put")
	fs.StringVar(&putFlag, "write", "", "Alias for -put")

	var deleteFlag string
	fs.StringVar(&deleteFlag, "delete", "", "Delete a secret path directly via CLI")
	fs.StringVar(&deleteFlag, "del", "", "Alias for -delete")
	fs.StringVar(&deleteFlag, "rm", "", "Alias for -delete")
	fs.StringVar(&deleteFlag, "destroy", "", "Alias for -delete")

	var listFlag string
	var listSet bool
	fs.StringVar(&listFlag, "list", "", "List secrets under the specified prefix")
	fs.StringVar(&listFlag, "ls", "", "Alias for -list")

	var dataFlag string
	fs.StringVar(&dataFlag, "data", "", "Secret data payload as JSON string or @filepath")

	var fieldFlag string
	fs.StringVar(&fieldFlag, "field", "", "Extract and print only the specified field value")

	var formatFlag string
	fs.StringVar(&formatFlag, "format", "table", "Output format: table, json, or raw")

	var jsonFlag bool
	fs.BoolVar(&jsonFlag, "json", false, "Output secret data as JSON (shorthand for -format=json)")

	var forceFlag bool
	fs.BoolVar(&forceFlag, "f", false, "Force operation without confirmation prompt")
	fs.BoolVar(&forceFlag, "force", false, "Alias for -f")
	fs.BoolVar(&forceFlag, "y", false, "Alias for -f")
	fs.BoolVar(&forceFlag, "yes", false, "Alias for -f")

	var softFlag bool
	fs.BoolVar(&softFlag, "soft", false, "Perform soft-delete instead of permanent metadata destruction")

	var mergeFlag bool
	fs.BoolVar(&mergeFlag, "merge", false, "Merge with existing secret keys during put/update")

	// Set of known flags that consume an argument if provided without '='
	takesArg := map[string]bool{
		"env":        true,
		"addr":       true,
		"token":      true,
		"mount":      true,
		"unseal-key": true,
		"get":        true,
		"read":       true,
		"put":        true,
		"set":        true,
		"write":      true,
		"delete":     true,
		"del":        true,
		"rm":         true,
		"destroy":    true,
		"list":       true,
		"ls":         true,
		"data":       true,
		"field":      true,
		"format":     true,
	}

	// Partition arguments into flags and positional args
	var flagArgs []string
	var positionalArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagName := strings.TrimLeft(arg, "-")
			eqIdx := strings.Index(flagName, "=")
			if eqIdx != -1 {
				flagArgs = append(flagArgs, arg)
				continue
			}

			// Check special case for -list / -ls when passed without a value
			if flagName == "list" || flagName == "ls" {
				listSet = true
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.Contains(args[i+1], "=") {
					flagArgs = append(flagArgs, arg, args[i+1])
					i++
				} else {
					flagArgs = append(flagArgs, arg, "")
				}
				continue
			}

			if takesArg[flagName] {
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					flagArgs = append(flagArgs, arg, args[i+1])
					i++
				} else {
					flagArgs = append(flagArgs, arg)
				}
			} else {
				flagArgs = append(flagArgs, arg)
			}
		} else {
			positionalArgs = append(positionalArgs, arg)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		return nil, err
	}

	var command string
	var targetPath string
	var prefix string
	dataArgs := make(map[string]string)
	isMerge := mergeFlag

	// Determine command and target from positional arguments or flags
	posIdx := 0
	if len(positionalArgs) > 0 {
		first := strings.ToLower(positionalArgs[0])
		switch first {
		case "get", "read":
			command = "get"
			posIdx = 1
		case "put", "set", "write", "create":
			command = "put"
			posIdx = 1
		case "update":
			command = "put"
			isMerge = true
			posIdx = 1
		case "delete", "del", "rm", "destroy":
			command = "delete"
			posIdx = 1
		case "list", "ls":
			command = "list"
			posIdx = 1
		}
	}

	// Check if command was triggered via flag
	if command == "" {
		if getFlag != "" {
			command = "get"
			targetPath = getFlag
		} else if putFlag != "" {
			command = "put"
			targetPath = putFlag
		} else if deleteFlag != "" {
			command = "delete"
			targetPath = deleteFlag
		} else if listSet || listFlag != "" {
			command = "list"
			prefix = listFlag
		}
	}

	// Process remaining positional arguments
	for ; posIdx < len(positionalArgs); posIdx++ {
		pArg := positionalArgs[posIdx]
		if strings.Contains(pArg, "=") {
			eqIdx := strings.Index(pArg, "=")
			k := pArg[:eqIdx]
			v := pArg[eqIdx+1:]
			dataArgs[k] = v
		} else if targetPath == "" && command != "list" {
			targetPath = pArg
		} else if prefix == "" && command == "list" {
			prefix = pArg
		}
	}

	// If no explicit command, but targetPath provided with no dataArgs, default to "get"
	if command == "" && targetPath != "" && len(dataArgs) == 0 && dataFlag == "" {
		command = "get"
	} else if command == "" && (len(dataArgs) > 0 || dataFlag != "") && targetPath != "" {
		command = "put"
	}

	// Clean mount name (remove leading/trailing slashes)
	mount := strings.Trim(*mountFlag, "/")
	if mount == "" {
		mount = "secret"
	}

	// Clean address (remove trailing slash)
	addr := strings.TrimRight(*addrFlag, "/")

	fmtChoice := strings.ToLower(strings.TrimSpace(formatFlag))
	if jsonFlag {
		fmtChoice = "json"
	}

	cleanTargetPath := strings.Trim(targetPath, "/")
	cleanPrefix := strings.Trim(prefix, "/")

	return &Config{
		Address:    addr,
		Token:      *tokenFlag,
		Mount:      mount,
		UnsealKey:  *unsealFlag,
		EnvFile:    envFile,
		Command:    command,
		TargetPath: cleanTargetPath,
		DataArgs:   dataArgs,
		RawData:    dataFlag,
		Field:      fieldFlag,
		Format:     fmtChoice,
		Force:      forceFlag,
		SoftDelete: softFlag,
		Merge:      isMerge,
		Prefix:     cleanPrefix,
		GetPath:    cleanTargetPath,
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
