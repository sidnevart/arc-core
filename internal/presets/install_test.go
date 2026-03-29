package presets

import (
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/project"
)

func TestPreviewInstallAndRollback(t *testing.T) {
	workspace := t.TempDir()
	if _, err := project.Init(workspace, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	catalog := filepath.Join(t.TempDir(), "catalog")
	presetDir := filepath.Join(catalog, "sample")
	if err := os.MkdirAll(filepath.Join(presetDir, "payload", ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "id": "sample-codex",
  "name": "Sample Codex",
  "tagline": "Sample preset",
  "goal": "Test install",
  "adapter": "codex",
  "category": "engineering",
  "version": "0.1.0",
  "files": ["AGENTS.md", ".codex/config.toml"],
  "author": {"name": "ARC Team", "handle": "arc"}
}`
	if err := os.WriteFile(filepath.Join(presetDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(presetDir, "payload", "AGENTS.md"), []byte("preset agents"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(presetDir, "payload", ".codex", "config.toml"), []byte("mode = \"safe\""), 0o644); err != nil {
		t.Fatal(err)
	}

	preview, err := PreviewInstall(PreviewOptions{
		WorkspaceRoot: workspace,
		CatalogRoot:   catalog,
		PresetID:      "sample-codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.HasConflicts {
		t.Fatal("expected no conflicts on fresh workspace")
	}

	result, err := Install(InstallOptions{
		WorkspaceRoot: workspace,
		CatalogRoot:   catalog,
		PresetID:      "sample-codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}

	record, err := Rollback(RollbackOptions{
		WorkspaceRoot: workspace,
		InstallID:     result.Record.InstallID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != "rolled_back" {
		t.Fatalf("expected rolled_back, got %s", record.Status)
	}
	if _, err := os.Stat(filepath.Join(workspace, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected AGENTS.md to be removed, got err=%v", err)
	}
}
