package project

import (
	"strings"
	"testing"
)

func TestInitOverwritesMachineManagedConfigFiles(t *testing.T) {
	root := t.TempDir()

	if err := WriteString(ProjectFile(root, "project.yaml"), "default_provider: \"auto\"\n"); err != nil {
		t.Fatal(err)
	}
	if err := WriteString(ProjectFile(root, "mode.yaml"), "mode: \"work\"\n"); err != nil {
		t.Fatal(err)
	}

	_, err := Init(root, InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "hero",
	})
	if err != nil {
		t.Fatal(err)
	}

	projectContent, err := ReadString(ProjectFile(root, "project.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	modeContent, err := ReadString(ProjectFile(root, "mode.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(projectContent, `default_provider: "codex"`) {
		t.Fatalf("expected updated provider, got:\n%s", projectContent)
	}
	if !strings.Contains(projectContent, `enabled_providers: "codex,claude"`) {
		t.Fatalf("expected enabled providers, got:\n%s", projectContent)
	}
	if !strings.Contains(modeContent, `mode: "hero"`) || !strings.Contains(modeContent, `autonomy: "high"`) {
		t.Fatalf("expected updated mode config, got:\n%s", modeContent)
	}
}

func TestInitCreatesProjectSkillScaffold(t *testing.T) {
	root := t.TempDir()

	if _, err := Init(root, InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	content, err := ReadString(ProjectFile(root, "skills", "plan-task", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "Plan Task") {
		t.Fatalf("expected built-in project skill scaffold, got:\n%s", content)
	}
}
