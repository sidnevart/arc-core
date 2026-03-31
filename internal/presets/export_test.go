package presets

import (
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/project"
)

func TestExportDraftRequiresTestedStatus(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "export-me",
		Name:          "Export Me",
		Summary:       "Test export preset",
		Goal:          "Export a tested draft into catalog form.",
		TargetAgent:   "codex",
		Providers:     []string{"codex", "claude"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportDraft(root, "export-me"); err == nil {
		t.Fatal("expected export to fail before tested status")
	}
}

func TestExportDraftWritesProviderBundles(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "export-me",
		Name:          "Export Me",
		Summary:       "Collaborative preset export.",
		Goal:          "Export a tested draft into installable provider bundles.",
		TargetAgent:   "codex",
		Providers:     []string{"codex", "claude"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := SimulateDraft(root, "export-me"); err != nil {
		t.Fatal(err)
	}
	if _, err := MarkDraftTested(root, "export-me"); err != nil {
		t.Fatal(err)
	}
	result, err := ExportDraft(root, "export-me")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Bundles) != 2 {
		t.Fatalf("expected 2 bundles, got %d", len(result.Bundles))
	}
	for _, bundle := range result.Bundles {
		if _, err := os.Stat(bundle.Paths.ManifestPath); err != nil {
			t.Fatalf("manifest missing for %s: %v", bundle.Provider, err)
		}
		if _, err := os.Stat(bundle.Paths.ProviderPath); err != nil {
			t.Fatalf("provider guide missing for %s: %v", bundle.Provider, err)
		}
		if _, err := os.Stat(bundle.Paths.OverviewPath); err != nil {
			t.Fatalf("overview missing for %s: %v", bundle.Provider, err)
		}
		manifest, err := LoadManifest(bundle.Paths.ManifestPath)
		if err != nil {
			t.Fatalf("load manifest for %s: %v", bundle.Provider, err)
		}
		if manifest.Adapter != bundle.Provider {
			t.Fatalf("expected adapter %s, got %s", bundle.Provider, manifest.Adapter)
		}
		if len(manifest.Files) != 2 {
			t.Fatalf("expected installable files list for %s, got %#v", bundle.Provider, manifest.Files)
		}
	}
	if _, err := LoadByID(exportsRoot(root), "export-me-codex"); err != nil {
		t.Fatalf("load exported codex bundle: %v", err)
	}
	if _, err := LoadByID(exportsRoot(root), "export-me-claude"); err != nil {
		t.Fatalf("load exported claude bundle: %v", err)
	}
	if _, err := os.Stat(filepath.Join(exportsRoot(root), "export-me-codex", "simulation.json")); err != nil {
		t.Fatalf("expected exported simulation sidecar: %v", err)
	}
}
