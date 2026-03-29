package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-os/internal/project"
)

func TestGenerateDocsWithApplyUpdatesProjectMaps(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "hero",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/docs\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := Plan(root, TaskOptions{
		Root:     root,
		Task:     "build docs",
		Mode:     "hero",
		Provider: "codex",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := GenerateDocsWithApply(root, run.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if result["applied"] == "" {
		t.Fatal("expected applied paths to be reported")
	}

	repoMap, err := project.ReadString(project.ProjectFile(root, "maps", "REPO_MAP.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(repoMap, "Important code surfaces") {
		t.Fatalf("expected repo map to be updated, got:\n%s", repoMap)
	}

	cliMap, err := project.ReadString(project.ProjectFile(root, "maps", "CLI_MAP.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cliMap, "arc task run") {
		t.Fatalf("expected CLI map to be updated, got:\n%s", cliMap)
	}

	runtimeStatus, err := project.ReadString(project.ProjectFile(root, "maps", "RUNTIME_STATUS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(runtimeStatus, "Provider status") {
		t.Fatalf("expected runtime status doc to be updated, got:\n%s", runtimeStatus)
	}
}
