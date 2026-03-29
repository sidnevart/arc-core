package presets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "manifest.yaml")
	content := `{
  "id": "incident-investigator-codex",
  "name": "Incident Investigator",
  "tagline": "Investigate alerts with evidence-first output.",
  "goal": "Investigate incidents safely.",
  "adapter": "codex",
  "category": "engineering",
  "version": "0.1.0",
  "files": ["AGENTS.md", ".codex/config.toml"],
  "author": {"name": "ARC Team", "handle": "arc"}
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ID != "incident-investigator-codex" {
		t.Fatalf("unexpected id: %s", manifest.ID)
	}
	if len(manifest.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(manifest.Files))
	}
}
