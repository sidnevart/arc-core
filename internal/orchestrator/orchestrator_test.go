package orchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/project"
)

func TestRunTaskDryRunProducesArtifacts(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel string, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "hero",
	})
	if err != nil {
		t.Fatal(err)
	}

	mustWrite("go.mod", "module example.com/task\n\ngo 1.23.6\n")
	mustWrite("main.go", "package main\n\nfunc main() {}\n")
	mustWrite("main_test.go", "package main\n\nimport \"testing\"\n\nfunc TestSmoke(t *testing.T) {}\n")

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "verify dry-run orchestration",
		Mode:        "hero",
		Provider:    "codex",
		DryRun:      true,
		RunChecks:   true,
		UseProvider: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if run.Status != "done" {
		t.Fatalf("expected done status, got %s", run.Status)
	}

	required := []string{
		"context_pack.md",
		"active_roles.md",
		"active_skills.md",
		"ticket_spec.md",
		"business_spec.md",
		"tech_spec.md",
		"implementation_log.md",
		"anti_hallucination_report.md",
		"verification_report.md",
		"review_report.md",
		"docs_delta.md",
	}
	for _, name := range required {
		path, ok := run.Artifacts[name]
		if !ok {
			t.Fatalf("expected artifact %s", name)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact file %s to exist: %v", path, err)
		}
	}
}
