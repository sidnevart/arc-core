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
  "short_description": [
    "Best for incident work.",
    "Use it with concrete symptoms.",
    "It prefers evidence over guesses."
  ],
  "goal": "Investigate incidents safely.",
  "adapter": "codex",
  "category": "engineering",
  "preset_type": "domain",
  "version": "0.1.0",
  "files": ["AGENTS.md", ".codex/config.toml"],
  "compatible_providers": ["codex"],
  "permissions": {"runtime": "read_only"},
  "memory_scopes": ["project", "session", "presets/incident-investigator-codex"],
  "hooks": [
    {"name": "collect-evidence", "lifecycle": "before_run", "timeout_seconds": 30, "permission_scope": "read_only"}
  ],
  "commands": [
    {"name": "incident-summary", "summary": "Generate an incident summary"}
  ],
  "budget_profile": "balanced",
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
	if manifest.PresetType != "domain" {
		t.Fatalf("unexpected preset_type: %s", manifest.PresetType)
	}
	if manifest.Permissions.Runtime != "read_only" {
		t.Fatalf("unexpected runtime permission: %s", manifest.Permissions.Runtime)
	}
	if len(manifest.Hooks) != 1 || manifest.Hooks[0].Lifecycle != "before_run" {
		t.Fatalf("expected validated hooks, got %#v", manifest.Hooks)
	}
	if len(manifest.ShortDescription) != 3 {
		t.Fatalf("expected short description to load, got %#v", manifest.ShortDescription)
	}
}

func TestLoadManifestDefaultsPresetType(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "manifest.yaml")
	content := `{
  "id": "study",
  "name": "Study",
  "tagline": "Teach first",
  "goal": "Help the user learn.",
  "adapter": "arc",
  "category": "built-in",
  "version": "1.0.0",
  "files": [],
  "author": {"name": "ARC Team", "handle": "arc"}
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.PresetType != "domain" {
		t.Fatalf("expected default preset_type domain, got %q", manifest.PresetType)
	}
}

func TestLoadManifestRejectsInvalidEnvironmentRules(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "manifest.yaml")
	content := `{
  "id": "broken-preset",
  "name": "Broken",
  "tagline": "Broken",
  "goal": "Broken",
  "adapter": "arc",
  "category": "test",
  "preset_type": "domain",
  "version": "1.0.0",
  "files": [],
  "permissions": {"runtime": "dangerous"},
  "hooks": [
    {"name": "bad-hook", "lifecycle": "after_everything", "timeout_seconds": 0}
  ],
  "memory_scopes": ["", "project"],
  "author": {"name": "ARC Team", "handle": "arc"}
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadManifest(path); err == nil {
		t.Fatal("expected invalid manifest to fail validation")
	}
}

func TestLoadManifestRejectsTooManyShortDescriptionLines(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "manifest.yaml")
	content := `{
  "id": "broken-short-description",
  "name": "Broken",
  "tagline": "Broken",
  "short_description": ["one", "two", "three", "four"],
  "goal": "Broken",
  "adapter": "arc",
  "category": "test",
  "preset_type": "domain",
  "version": "1.0.0",
  "files": [],
  "author": {"name": "ARC Team", "handle": "arc"}
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadManifest(path); err == nil {
		t.Fatal("expected manifest with too many short_description lines to fail")
	}
}

func TestLoadManifestRejectsNonInfrastructureClaims(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "manifest.yaml")
	content := `{
  "id": "bad-domain",
  "name": "Bad Domain",
  "tagline": "Broken",
  "goal": "Broken",
  "adapter": "codex",
  "category": "test",
  "preset_type": "domain",
  "version": "1.0.0",
  "files": [],
  "required_modules": ["runtime/budget"],
  "permissions": {"runtime": "sandboxed_exec"},
  "memory_scopes": ["project", "system"],
  "author": {"name": "ARC Team", "handle": "arc"}
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadManifest(path); err == nil {
		t.Fatal("expected non-infrastructure manifest with elevated claims to fail")
	}
}
