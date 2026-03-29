package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairCreatesMissingScaffoldAndRewritesTODOGuidance(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root, InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(ProjectFile(root, "skills")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ProjectFile(root, "provider", "AGENTS.md"), []byte("# AGENTS\n\n[TODO] replace me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Repair(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) == 0 {
		t.Fatal("expected repair to create or rewrite files")
	}
	if _, err := os.Stat(ProjectFile(root, "skills", "plan-task", "SKILL.md")); err != nil {
		t.Fatalf("expected repaired skills scaffold: %v", err)
	}
	data, err := os.ReadFile(ProjectFile(root, "provider", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToUpper(string(data)), "TODO") {
		t.Fatalf("expected TODO guidance to be rewritten, got:\n%s", string(data))
	}
	if _, err := os.Stat(filepath.Join(root, ".arc", "hooks", "README.md")); err != nil {
		t.Fatalf("expected hooks scaffold: %v", err)
	}
}
