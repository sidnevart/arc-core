package presets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-os/internal/project"
)

func writePreset(t *testing.T, catalog string, id string, manifest string, files map[string]string) {
	t.Helper()
	presetDir := filepath.Join(catalog, id)
	if err := os.MkdirAll(filepath.Join(presetDir, "payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(presetDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, body := range files {
		target := filepath.Join(presetDir, "payload", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

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
	writePreset(t, catalog, "sample", manifest, map[string]string{
		"AGENTS.md":          "preset agents",
		".codex/config.toml": "mode = \"safe\"",
	})

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

func TestPreviewInstallDetectsCommandCollisionWithInstalledPreset(t *testing.T) {
	workspace := t.TempDir()
	if _, err := project.Init(workspace, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	catalog := filepath.Join(t.TempDir(), "catalog")
	writePreset(t, catalog, "infra", `{
  "id": "infra-runtime",
  "name": "Infra Runtime",
  "tagline": "Base infra layer",
  "goal": "Provide infra hooks",
  "adapter": "arc",
  "category": "infra",
  "preset_type": "infrastructure",
  "version": "0.1.0",
  "files": ["AGENTS.md"],
  "commands": [{"name": "shared-audit"}],
  "memory_scopes": ["system", "presets/infra-runtime"],
  "permissions": {"runtime": "sandboxed_exec"},
  "author": {"name": "ARC Team", "handle": "arc"}
}`, map[string]string{"AGENTS.md": "infra"})
	writePreset(t, catalog, "domain", `{
  "id": "domain-preset",
  "name": "Domain",
  "tagline": "Domain layer",
  "goal": "Do task work",
  "adapter": "codex",
  "category": "domain",
  "preset_type": "domain",
  "version": "0.1.0",
  "files": [".codex/config.toml"],
  "commands": [{"name": "shared-audit"}],
  "memory_scopes": ["project", "presets/domain-preset"],
  "permissions": {"runtime": "read_only"},
  "author": {"name": "ARC Team", "handle": "arc"}
}`, map[string]string{".codex/config.toml": "mode = \"safe\""})

	if _, err := Install(InstallOptions{
		WorkspaceRoot:  workspace,
		CatalogRoot:    catalog,
		PresetID:       "infra-runtime",
		AllowOverwrite: true,
	}); err != nil {
		t.Fatal(err)
	}

	preview, err := PreviewInstall(PreviewOptions{
		WorkspaceRoot: workspace,
		CatalogRoot:   catalog,
		PresetID:      "domain-preset",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.HasEnvironmentConflicts {
		t.Fatalf("expected environment conflicts, got %#v", preview)
	}
}

func TestInstallRejectsInvalidManifestEvenWithForce(t *testing.T) {
	workspace := t.TempDir()
	if _, err := project.Init(workspace, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	catalog := filepath.Join(t.TempDir(), "catalog")
	writePreset(t, catalog, "safe-infra", `{
  "id": "safe-infra",
  "name": "Safe Infra",
  "tagline": "Infra",
  "goal": "Infra",
  "adapter": "arc",
  "category": "infra",
  "preset_type": "infrastructure",
  "version": "0.1.0",
  "files": [],
  "memory_scopes": ["system", "presets/safe-infra"],
  "permissions": {"runtime": "sandboxed_exec"},
  "author": {"name": "ARC Team", "handle": "arc"}
}`, nil)

	if _, err := Install(InstallOptions{
		WorkspaceRoot:  workspace,
		CatalogRoot:    catalog,
		PresetID:       "safe-infra",
		AllowOverwrite: true,
	}); err != nil {
		t.Fatal(err)
	}

	writePreset(t, catalog, "bad-domain", `{
  "id": "bad-domain",
  "name": "Bad Domain",
  "tagline": "Bad",
  "goal": "Bad",
  "adapter": "arc",
  "category": "domain",
  "preset_type": "domain",
  "version": "0.1.0",
  "files": [],
  "memory_scopes": ["system", "presets/bad-domain"],
  "permissions": {"runtime": "risky_exec_requires_approval"},
  "author": {"name": "ARC Team", "handle": "arc"}
}`, nil)

	if _, err := Install(InstallOptions{
		WorkspaceRoot:  workspace,
		CatalogRoot:    catalog,
		PresetID:       "bad-domain",
		AllowOverwrite: true,
	}); err == nil {
		t.Fatal("expected install to reject invalid manifest even with force")
	}
}

func TestPreviewInstallIncludesEnvironmentResolution(t *testing.T) {
	workspace := t.TempDir()
	if _, err := project.Init(workspace, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	catalog := filepath.Join(t.TempDir(), "catalog")
	writePreset(t, catalog, "infra", `{
  "id": "infra-runtime",
  "name": "Infra Runtime",
  "tagline": "Infra",
  "goal": "Infra",
  "adapter": "arc",
  "category": "infra",
  "preset_type": "infrastructure",
  "version": "0.1.0",
  "files": [],
  "memory_scopes": ["system", "presets/infra-runtime"],
  "permissions": {"runtime": "sandboxed_exec"},
  "hooks": [{"name": "prepare-runtime", "lifecycle": "before_launch_runtime", "timeout_seconds": 15, "permission_scope": "sandboxed_exec"}],
  "commands": [{"name": "infra-check"}],
  "budget_profile": "deep_work",
  "author": {"name": "ARC Team", "handle": "arc"}
}`, nil)
	writePreset(t, catalog, "domain", `{
  "id": "domain-preset",
  "name": "Domain",
  "tagline": "Domain",
  "goal": "Domain",
  "adapter": "codex",
  "category": "domain",
  "preset_type": "domain",
  "version": "0.1.0",
  "files": [".codex/config.toml"],
  "memory_scopes": ["project", "presets/domain-preset"],
  "permissions": {"runtime": "read_only"},
  "commands": [{"name": "domain-summary"}],
  "budget_profile": "balanced",
  "author": {"name": "ARC Team", "handle": "arc"}
}`, map[string]string{".codex/config.toml": "mode = \"safe\""})

	if _, err := Install(InstallOptions{
		WorkspaceRoot: workspace,
		CatalogRoot:   catalog,
		PresetID:      "infra-runtime",
	}); err != nil {
		t.Fatal(err)
	}

	preview, err := PreviewInstall(PreviewOptions{
		WorkspaceRoot: workspace,
		CatalogRoot:   catalog,
		PresetID:      "domain-preset",
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Resolution.CandidatePresetID != "domain-preset" {
		t.Fatalf("expected candidate preset in resolution, got %#v", preview.Resolution)
	}
	if preview.Resolution.EffectiveRuntimeCeiling != "sandboxed_exec" {
		t.Fatalf("expected sandboxed_exec ceiling, got %q", preview.Resolution.EffectiveRuntimeCeiling)
	}
	if preview.Resolution.EffectiveBudgetProfile != "balanced" {
		t.Fatalf("expected candidate budget profile to win, got %q", preview.Resolution.EffectiveBudgetProfile)
	}
	if len(preview.Resolution.Layers) < 5 {
		t.Fatalf("expected layered resolution, got %#v", preview.Resolution.Layers)
	}
	if preview.Resolution.Layers[0].Kind != "arc_base" || preview.Resolution.Layers[1].Kind != "provider_adapter" {
		t.Fatalf("unexpected leading layers: %#v", preview.Resolution.Layers[:2])
	}
}

func TestInstallWritesEnvironmentArtifacts(t *testing.T) {
	workspace := t.TempDir()
	if _, err := project.Init(workspace, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	catalog := filepath.Join(t.TempDir(), "catalog")
	writePreset(t, catalog, "sample", `{
  "id": "sample-codex",
  "name": "Sample Codex",
  "tagline": "Sample preset",
  "goal": "Test install",
  "adapter": "codex",
  "category": "engineering",
  "preset_type": "domain",
  "version": "0.1.0",
  "files": ["AGENTS.md"],
  "memory_scopes": ["project", "presets/sample-codex"],
  "permissions": {"runtime": "read_only"},
  "budget_profile": "balanced",
  "author": {"name": "ARC Team", "handle": "arc"}
}`, map[string]string{
		"AGENTS.md": "preset agents",
	})

	result, err := Install(InstallOptions{
		WorkspaceRoot: workspace,
		CatalogRoot:   catalog,
		PresetID:      "sample-codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(result.Record.EnvironmentReportPath) == "" || strings.TrimSpace(result.Record.EnvironmentJSONPath) == "" {
		t.Fatalf("expected environment artifact paths, got %#v", result.Record)
	}
	if _, err := os.Stat(result.Record.EnvironmentReportPath); err != nil {
		t.Fatalf("expected environment markdown report: %v", err)
	}
	if _, err := os.Stat(result.Record.EnvironmentJSONPath); err != nil {
		t.Fatalf("expected environment json report: %v", err)
	}
}
