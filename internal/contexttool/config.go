package contexttool

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const humanConfigFile = ".context-tool.yaml"

type HumanConfig struct {
	ProjectName    string   `json:"project_name,omitempty"`
	IncludePaths   []string `json:"include_paths,omitempty"`
	ExcludePaths   []string `json:"exclude_paths,omitempty"`
	DocsPaths      []string `json:"docs_paths,omitempty"`
	MemoryPaths    []string `json:"memory_paths,omitempty"`
	LanguageHints  []string `json:"language_hints,omitempty"`
	MetricsEnabled bool     `json:"metrics_enabled"`
	metricsSet     bool
}

func HumanConfigPath(root string) string {
	return filepath.Join(root, humanConfigFile)
}

func DefaultHumanConfig(root string) HumanConfig {
	return HumanConfig{
		ProjectName:    filepath.Base(root),
		IncludePaths:   []string{"."},
		ExcludePaths:   []string{".context", ".arc/runs", ".arc/evals", ".arc/cache", "node_modules", "vendor"},
		DocsPaths:      []string{"docs", "apps/docs/docs", "."},
		MemoryPaths:    []string{".context/memory"},
		LanguageHints:  []string{"go"},
		MetricsEnabled: true,
	}
}

func LoadHumanConfig(root string) (HumanConfig, error) {
	cfg := DefaultHumanConfig(root)
	path := HumanConfigPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return HumanConfig{}, err
	}
	parsed, err := parseHumanConfig(string(data))
	if err != nil {
		return HumanConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if strings.TrimSpace(parsed.ProjectName) != "" {
		cfg.ProjectName = strings.TrimSpace(parsed.ProjectName)
	}
	if len(parsed.IncludePaths) > 0 {
		cfg.IncludePaths = normalizeConfigPaths(parsed.IncludePaths)
	}
	if len(parsed.ExcludePaths) > 0 {
		cfg.ExcludePaths = normalizeConfigPaths(parsed.ExcludePaths)
	}
	if len(parsed.DocsPaths) > 0 {
		cfg.DocsPaths = normalizeConfigPaths(parsed.DocsPaths)
	}
	if len(parsed.MemoryPaths) > 0 {
		cfg.MemoryPaths = normalizeConfigPaths(parsed.MemoryPaths)
	}
	if len(parsed.LanguageHints) > 0 {
		cfg.LanguageHints = normalizeStringList(parsed.LanguageHints)
	}
	if parsed.metricsSet {
		cfg.MetricsEnabled = parsed.MetricsEnabled
	}
	return cfg, nil
}

func RenderHumanConfig(root string, cfg HumanConfig) string {
	if strings.TrimSpace(cfg.ProjectName) == "" {
		cfg.ProjectName = filepath.Base(root)
	}
	if len(cfg.IncludePaths) == 0 {
		cfg.IncludePaths = []string{"."}
	}
	if len(cfg.ExcludePaths) == 0 {
		cfg.ExcludePaths = []string{".context"}
	}
	if len(cfg.DocsPaths) == 0 {
		cfg.DocsPaths = []string{"docs"}
	}
	if len(cfg.MemoryPaths) == 0 {
		cfg.MemoryPaths = []string{".context/memory"}
	}
	if len(cfg.LanguageHints) == 0 {
		cfg.LanguageHints = []string{"go"}
	}
	var b strings.Builder
	b.WriteString("# Context Tool Config\n\n")
	b.WriteString("project_name: \"" + escapeYAMLString(cfg.ProjectName) + "\"\n")
	b.WriteString(renderStringList("include_paths", cfg.IncludePaths))
	b.WriteString(renderStringList("exclude_paths", cfg.ExcludePaths))
	b.WriteString(renderStringList("docs_paths", cfg.DocsPaths))
	b.WriteString(renderStringList("memory_paths", cfg.MemoryPaths))
	b.WriteString(renderStringList("language_hints", cfg.LanguageHints))
	b.WriteString("metrics_enabled: " + strconv.FormatBool(cfg.MetricsEnabled) + "\n")
	return b.String()
}

func parseHumanConfig(content string) (HumanConfig, error) {
	var cfg HumanConfig
	var currentList string
	for lineNo, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			if currentList == "" {
				return HumanConfig{}, fmt.Errorf("line %d: list item without key", lineNo+1)
			}
			value := normalizeConfigValue(strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			switch currentList {
			case "include_paths":
				cfg.IncludePaths = append(cfg.IncludePaths, value)
			case "exclude_paths":
				cfg.ExcludePaths = append(cfg.ExcludePaths, value)
			case "docs_paths":
				cfg.DocsPaths = append(cfg.DocsPaths, value)
			case "memory_paths":
				cfg.MemoryPaths = append(cfg.MemoryPaths, value)
			case "language_hints":
				cfg.LanguageHints = append(cfg.LanguageHints, value)
			default:
				return HumanConfig{}, fmt.Errorf("line %d: unsupported list key %q", lineNo+1, currentList)
			}
			continue
		}
		currentList = ""
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return HumanConfig{}, fmt.Errorf("line %d: invalid entry", lineNo+1)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if value == "" {
			switch key {
			case "include_paths", "exclude_paths", "docs_paths", "memory_paths", "language_hints":
				currentList = key
				continue
			default:
				return HumanConfig{}, fmt.Errorf("line %d: key %q requires a value", lineNo+1, key)
			}
		}
		value = normalizeConfigValue(value)
		switch key {
		case "project_name":
			cfg.ProjectName = value
		case "metrics_enabled":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return HumanConfig{}, fmt.Errorf("line %d: invalid metrics_enabled %q", lineNo+1, value)
			}
			cfg.MetricsEnabled = parsed
			cfg.metricsSet = true
		case "include_paths":
			cfg.IncludePaths = append(cfg.IncludePaths, splitInlineList(value)...)
		case "exclude_paths":
			cfg.ExcludePaths = append(cfg.ExcludePaths, splitInlineList(value)...)
		case "docs_paths":
			cfg.DocsPaths = append(cfg.DocsPaths, splitInlineList(value)...)
		case "memory_paths":
			cfg.MemoryPaths = append(cfg.MemoryPaths, splitInlineList(value)...)
		case "language_hints":
			cfg.LanguageHints = append(cfg.LanguageHints, splitInlineList(value)...)
		default:
			return HumanConfig{}, fmt.Errorf("line %d: unsupported key %q", lineNo+1, key)
		}
	}
	return cfg, nil
}

func normalizeConfigPaths(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = filepath.Clean(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if value == "." {
			value = "."
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeConfigValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	return strings.TrimSpace(value)
}

func splitInlineList(value string) []string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeConfigValue(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func escapeYAMLString(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func renderStringList(key string, values []string) string {
	var b strings.Builder
	b.WriteString(key + ":\n")
	for _, value := range values {
		b.WriteString("  - \"" + escapeYAMLString(value) + "\"\n")
	}
	return b.String()
}
