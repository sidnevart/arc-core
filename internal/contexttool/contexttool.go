package contexttool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

const schemaVersion = "v1"

type WorkspaceConfig struct {
	SchemaVersion string `json:"schema_version"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type DoctorReport struct {
	Root          string        `json:"root"`
	WorkspaceDir  string        `json:"workspace_dir"`
	SchemaVersion string        `json:"schema_version,omitempty"`
	ConfigPath    string        `json:"config_path,omitempty"`
	HumanConfig   HumanConfig   `json:"human_config"`
	Checks        []DoctorCheck `json:"checks"`
	Warnings      []string      `json:"warnings,omitempty"`
}

func WorkspaceDir(root string) string {
	return filepath.Join(root, ".context")
}

func WorkspaceFile(root string, elems ...string) string {
	parts := append([]string{WorkspaceDir(root)}, elems...)
	return filepath.Join(parts...)
}

func DiscoverRoot(start string) (string, error) {
	root, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(WorkspaceFile(root, "config.json")); err == nil {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", fmt.Errorf("no ctx workspace found in current path or parents")
		}
		root = parent
	}
}

func ResolveRoot(start string) (string, error) {
	if root, err := DiscoverRoot(start); err == nil {
		return root, nil
	}
	return filepath.Abs(start)
}

func Init(root string) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	created := []string{}
	dirs := []string{
		WorkspaceDir(root),
		WorkspaceFile(root, "index"),
		WorkspaceFile(root, "maps"),
		WorkspaceFile(root, "memory"),
		WorkspaceFile(root, "artifacts", "assemble"),
		WorkspaceFile(root, "runs"),
		WorkspaceFile(root, "benchmarks"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			created = append(created, dir)
		} else if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	configPath := WorkspaceFile(root, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := WorkspaceConfig{
			SchemaVersion: schemaVersion,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := project.WriteJSON(configPath, cfg); err != nil {
			return nil, err
		}
		created = append(created, configPath)
	} else if err != nil {
		return nil, err
	}

	files := map[string]string{
		HumanConfigPath(root):                         RenderHumanConfig(root, DefaultHumanConfig(root)),
		WorkspaceFile(root, "memory", "entries.json"): "[]\n",
		WorkspaceFile(root, "memory", "README.md"): `# Context Tool Memory

Use ` + "`ctx memory add`" + ` to record durable human-authored context that should bias future assembly.
`,
		WorkspaceFile(root, "memory", "MEMORY_ACTIVE.md"):  "# CTX MEMORY ACTIVE\n\nNo items.\n",
		WorkspaceFile(root, "memory", "MEMORY_ARCHIVE.md"): "# CTX MEMORY ARCHIVE\n\nNo items.\n",
		WorkspaceFile(root, "memory", "OPEN_QUESTIONS.md"): "# CTX OPEN QUESTIONS\n\nNo items.\n",
		WorkspaceFile(root, "README.md"): "# Context Tool Workspace\n\n" +
			"This directory is owned by the standalone-first `ctx` tool.\n\n" +
			"- `index/` stores machine-generated project indexes.\n" +
			"- `memory/` stores context-tool memory, separate from ARC `.arc/memory`.\n" +
			"- `artifacts/assemble/` stores generated context bundles and provenance.\n",
		WorkspaceFile(root, "maps", "README.md"): `# Context Maps

Human-authored context maps can live here as the standalone context tool grows.
`,
	}
	for path, content := range files {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := project.WriteString(path, content); err != nil {
				return nil, err
			}
			created = append(created, path)
		} else if err != nil {
			return nil, err
		}
	}

	return created, nil
}

func Doctor(root string) (DoctorReport, error) {
	root, err := ResolveRoot(root)
	if err != nil {
		return DoctorReport{}, err
	}
	report := DoctorReport{
		Root:         root,
		WorkspaceDir: WorkspaceDir(root),
		ConfigPath:   HumanConfigPath(root),
		Checks:       []DoctorCheck{},
		Warnings:     []string{},
	}
	check := func(name, status, details string) {
		report.Checks = append(report.Checks, DoctorCheck{Name: name, Status: status, Details: details})
	}

	if _, err := Init(root); err != nil {
		return DoctorReport{}, err
	}

	humanCfg, err := LoadHumanConfig(root)
	if err != nil {
		check("human_config", "error", err.Error())
	} else {
		report.HumanConfig = humanCfg
		check("human_config", "ok", report.ConfigPath)
	}

	cfgPath := WorkspaceFile(root, "config.json")
	var cfg WorkspaceConfig
	if err := project.ReadJSON(cfgPath, &cfg); err != nil {
		check("workspace_config", "error", err.Error())
	} else {
		report.SchemaVersion = cfg.SchemaVersion
		check("workspace_config", "ok", cfgPath)
	}

	requiredDirs := []string{
		WorkspaceFile(root, "index"),
		WorkspaceFile(root, "maps"),
		WorkspaceFile(root, "memory"),
		WorkspaceFile(root, "artifacts", "assemble"),
		WorkspaceFile(root, "benchmarks"),
		WorkspaceFile(root, "runs"),
	}
	for _, dir := range requiredDirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			check("required_dir", "error", dir)
		} else {
			check("required_dir", "ok", dir)
		}
	}

	items, err := ListMemory(root)
	if err != nil {
		check("memory_entries", "error", err.Error())
	} else {
		check("memory_entries", "ok", fmt.Sprintf("%d entries", len(items)))
	}

	idx, err := LoadIndex(root)
	if err != nil {
		check("index_bundle", "warning", "index bundle missing; run `ctx index build`")
		report.Warnings = append(report.Warnings, "index bundle missing; run `ctx index build`")
	} else {
		check("index_bundle", "ok", fmt.Sprintf("files=%d symbols=%d docs=%d", len(idx.Files), len(idx.Symbols), len(idx.Docs)))
		for _, file := range idx.Files {
			if strings.HasPrefix(file.Path, ".context/") {
				report.Warnings = append(report.Warnings, "index bundle contains .context artifacts; rebuild after ignore rules change")
				check("self_indexing", "warning", file.Path)
				return report, nil
			}
		}
		check("self_indexing", "ok", ".context artifacts are excluded from the index")
	}

	return report, nil
}
