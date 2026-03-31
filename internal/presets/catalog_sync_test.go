package presets

import (
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/project"
)

func TestSyncDraftToCatalogRequiresPublishedStatus(t *testing.T) {
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
		ID:            "catalog-me",
		Name:          "Catalog Me",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
		Providers:     []string{"codex", "claude"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncDraftToCatalog(root, "catalog-me", filepath.Join(t.TempDir(), "catalog")); err == nil {
		t.Fatal("expected catalog sync to fail before published status")
	}
}

func TestSyncDraftToCatalogCopiesPublishedBundles(t *testing.T) {
	root := t.TempDir()
	catalogRoot := filepath.Join(t.TempDir(), "catalog")
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "catalog-me",
		Name:          "Catalog Me",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
		Providers:     []string{"codex", "claude"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := SimulateDraft(root, "catalog-me"); err != nil {
		t.Fatal(err)
	}
	if _, err := MarkDraftTested(root, "catalog-me"); err != nil {
		t.Fatal(err)
	}
	if _, err := PublishDraft(root, "catalog-me"); err != nil {
		t.Fatal(err)
	}
	result, err := SyncDraftToCatalog(root, "catalog-me", catalogRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Bundles) != 2 {
		t.Fatalf("expected two synced bundles, got %d", len(result.Bundles))
	}
	for _, bundle := range result.Bundles {
		if _, err := os.Stat(bundle.Paths.ManifestPath); err != nil {
			t.Fatalf("manifest missing for %s: %v", bundle.BundleID, err)
		}
		if _, err := LoadByID(catalogRoot, bundle.BundleID); err != nil {
			t.Fatalf("expected synced catalog bundle %s to validate: %v", bundle.BundleID, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".arc", "presets", "platform", "catalog-me", "catalog_sync.json")); err != nil {
		t.Fatalf("expected catalog sync report: %v", err)
	}
}
