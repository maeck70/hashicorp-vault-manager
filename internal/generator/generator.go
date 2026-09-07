// Package generator creates standalone, runnable Go code snippets that demonstrate
// how to retrieve secrets from HashiCorp Vault using pure Go standard library (net/http).
package generator

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

// StructField represents a typed field within a generated Go struct.
type StructField struct {
	Name    string
	Type    string
	JSONKey string
}

// StructDef represents a generated Go struct definition for a nested JSON key.
type StructDef struct {
	KeyName  string
	TypeName string
	VarName  string
	Fields   []StructField
}

// TemplateData holds the parameters passed into the text/template code generator.
type TemplateData struct {
	VaultAddr string
	Mount     string
	Path      string
	Structs   []StructDef
	PlainKeys []string
}

//go:embed resources/retrieval.go.tmpl
var retrievalGoTemplate string

var (
	sanitizeRegex  = regexp.MustCompile(`[^a-zA-Z0-9_\-]+`)
	splitWordRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	retrievalTmpl = template.Must(template.New("retrieval").Funcs(template.FuncMap{
		"jsonTag": func(k string) string {
			return fmt.Sprintf("`json:\"%s\"`", k)
		},
	}).Parse(retrievalGoTemplate))
)

// GenerateGoRetrievalCode produces a complete, runnable Go source file that fetches
// and decodes a specific secret from Vault using text/template.
func GenerateGoRetrievalCode(vaultAddr, mount, path string, secretData map[string]string) string {
	if vaultAddr == "" {
		vaultAddr = "http://10.0.0.180:8200"
	}
	if mount == "" {
		mount = "secret"
	}

	cleanPath := strings.Trim(path, "/")
	structs, plainKeys := buildTemplateDataKeys(secretData)

	data := TemplateData{
		VaultAddr: vaultAddr,
		Mount:     mount,
		Path:      cleanPath,
		Structs:   structs,
		PlainKeys: plainKeys,
	}

	var buf bytes.Buffer
	if err := retrievalTmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("// Template execute error: %v", err)
	}

	return buf.String()
}

func buildTemplateDataKeys(secretData map[string]string) ([]StructDef, []string) {
	var structs []StructDef
	var plainKeys []string

	keys := make([]string, 0, len(secretData))
	for k := range secretData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		val := secretData[k]
		if isJSONString(val) {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(val), &parsed); err == nil && len(parsed) > 0 {
				typeName := toPascalCase(k) + "Config"
				varName := toCamelCase(k)

				subKeys := make([]string, 0, len(parsed))
				for sk := range parsed {
					subKeys = append(subKeys, sk)
				}
				sort.Strings(subKeys)

				fields := make([]StructField, 0, len(subKeys))
				for _, sk := range subKeys {
					fields = append(fields, StructField{
						Name:    toPascalCase(sk),
						Type:    inferGoType(parsed[sk]),
						JSONKey: sk,
					})
				}

				structs = append(structs, StructDef{
					KeyName:  k,
					TypeName: typeName,
					VarName:  varName,
					Fields:   fields,
				})
				continue
			}
		}

		plainKeys = append(plainKeys, k)
	}

	return structs, plainKeys
}

func inferGoType(val any) string {
	switch val := val.(type) {
	case float64:
		if val == float64(int(val)) {
			return "int"
		}
		return "float64"
	case bool:
		return "bool"
	default:
		return "string"
	}
}

func isJSONString(s string) bool {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return false
	}
	return json.Valid([]byte(trimmed))
}

// SanitizeFilename creates a safe filename from a secret path.
func SanitizeFilename(path string) string {
	clean := strings.Trim(path, "/")
	safe := sanitizeRegex.ReplaceAllString(clean, "_")
	if safe == "" {
		safe = "secret"
	}
	return "get_" + safe + ".go"
}

// SaveRetrievalFile writes the generated Go code to the target directory.
func SaveRetrievalFile(dir, filename, code string) (string, error) {
	if dir == "" {
		dir = "examples"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		return "", err
	}
	return filePath, nil
}

func toPascalCase(s string) string {
	parts := splitWordRegex.Split(s, -1)
	var sb strings.Builder
	sb.Grow(len(s))
	for _, p := range parts {
		if len(p) > 0 {
			sb.WriteString(strings.ToUpper(p[:1]))
			sb.WriteString(p[1:])
		}
	}
	res := sb.String()
	if res == "" {
		return "Field"
	}
	return res
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if len(pascal) == 0 {
		return "field"
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}
