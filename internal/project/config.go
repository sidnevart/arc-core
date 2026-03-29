package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func DiscoverRoot(start string) (string, error) {
	root, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(root, ".arc", "project.yaml")); err == nil {
			return root, nil
		}

		parent := filepath.Dir(root)
		if parent == root {
			return "", errors.New("no arc project found in current path or parents")
		}
		root = parent
	}
}

func Load(root string) (Project, error) {
	if err := RequireProject(root); err != nil {
		return Project{}, err
	}

	projectData, err := readSimpleYAML(filepath.Join(root, ".arc", "project.yaml"))
	if err != nil {
		return Project{}, err
	}
	modeData, err := readSimpleYAML(filepath.Join(root, ".arc", "mode.yaml"))
	if err != nil {
		return Project{}, err
	}

	cfg := ProjectConfig{
		Name:            projectData["name"],
		DefaultProvider: projectData["default_provider"],
		CreatedAt:       projectData["created_at"],
	}
	if raw := projectData["enabled_providers"]; raw != "" {
		cfg.EnabledProviders = splitCSV(raw)
	}
	if len(cfg.EnabledProviders) == 0 && cfg.DefaultProvider != "" {
		cfg.EnabledProviders = []string{cfg.DefaultProvider}
	}

	mode := ModeConfig{
		Mode:       modeData["mode"],
		Autonomy:   modeData["autonomy"],
		UpdatedAt:  modeData["updated_at"],
		Definition: filepath.Join(root, ".arc", "mode.yaml"),
	}

	return Project{
		Root:   root,
		ArcDir: filepath.Join(root, ".arc"),
		Config: cfg,
		Mode:   mode,
	}, nil
}

func Init(root string, opts InitOptions) ([]string, error) {
	if len(opts.EnabledProviders) == 0 {
		opts.EnabledProviders = splitCSV(opts.Provider)
	}
	if len(opts.EnabledProviders) == 0 {
		opts.EnabledProviders = []string{"codex"}
	}
	if opts.Provider == "" || opts.Provider == "auto" {
		opts.Provider = opts.EnabledProviders[0]
	}

	autonomy := autonomyForMode(opts.Mode)
	scaffold := DefaultScaffold(root, opts.Provider, opts.EnabledProviders, opts.Mode, autonomy)
	created, err := ApplyScaffold(root, scaffold)
	if err != nil {
		return nil, err
	}

	projectPath := ProjectFile(root, "project.yaml")
	modePath := ProjectFile(root, "mode.yaml")
	if err := WriteString(projectPath, projectYAML(root, opts.Provider, strings.Join(opts.EnabledProviders, ","))); err != nil {
		return nil, err
	}
	if err := WriteString(modePath, modeYAML(opts.Mode, autonomy)); err != nil {
		return nil, err
	}

	return created, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func readSimpleYAML(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"`)
		result[key] = value
	}
	return result, nil
}

func autonomyForMode(mode string) string {
	switch mode {
	case "study":
		return "low"
	case "work":
		return "medium"
	case "hero":
		return "high"
	default:
		return "medium"
	}
}

func ProjectFile(root string, elems ...string) string {
	parts := append([]string{root, ".arc"}, elems...)
	return filepath.Join(parts...)
}

func EnsureRunDir(root string, runID string) (string, error) {
	path := ProjectFile(root, "runs", runID)
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func LatestRunDir(root string) (string, error) {
	runsDir := ProjectFile(root, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return "", err
	}

	latest := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() > latest {
			latest = entry.Name()
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no runs found in %s", runsDir)
	}

	return filepath.Join(runsDir, latest), nil
}
