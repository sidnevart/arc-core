package project

import (
	"os"
	"path/filepath"
	"strings"
)

func Repair(root string) ([]string, error) {
	proj, err := Load(root)
	if err != nil {
		return nil, err
	}
	scaffold := DefaultScaffold(root, proj.Config.DefaultProvider, proj.Config.EnabledProviders, proj.Mode.Mode, autonomyForMode(proj.Mode.Mode))
	created, err := ApplyScaffold(root, scaffold)
	if err != nil {
		return nil, err
	}

	rewritten := []string{}
	for rel, content := range map[string]string{
		filepath.Join(".arc", "provider", "AGENTS.md"): scaffold.Files[filepath.Join(".arc", "provider", "AGENTS.md")],
		filepath.Join(".arc", "provider", "CLAUDE.md"): scaffold.Files[filepath.Join(".arc", "provider", "CLAUDE.md")],
	} {
		path := filepath.Join(root, rel)
		if shouldRewriteScaffoldTemplate(path) {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return nil, err
			}
			rewritten = append(rewritten, path)
		}
	}
	return append(created, rewritten...), nil
}

func shouldRewriteScaffoldTemplate(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return true
	}
	return strings.Contains(strings.ToUpper(text), "TODO")
}
