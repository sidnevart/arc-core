package presets

import (
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/project"
)

func TestInstallPublishedDraftSelectsProjectDefaultProvider(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "claude",
		EnabledProviders: []string{"claude", "codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "install-me",
		Name:          "Install Me",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
		Providers:     []string{"codex", "claude"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := SimulateDraft(root, "install-me"); err != nil {
		t.Fatal(err)
	}
	if _, err := MarkDraftTested(root, "install-me"); err != nil {
		t.Fatal(err)
	}
	if _, err := PublishDraft(root, "install-me"); err != nil {
		t.Fatal(err)
	}
	result, err := InstallPublishedDraft(root, "install-me", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedProvider != "claude" {
		t.Fatalf("expected claude provider, got %#v", result)
	}
	if result.BundleID != "install-me-claude" {
		t.Fatalf("unexpected bundle id: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Fatalf("expected installed CLAUDE.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "overview.md")); err != nil {
		t.Fatalf("expected installed overview: %v", err)
	}
}
