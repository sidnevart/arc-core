package cli

import (
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/presets"
	"agent-os/internal/project"
)

func TestPresetDraftCLIInitShowValidateAndList(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	if err := Run([]string{
		"preset", "draft", "init",
		"--path", root,
		"--id", "literature-guide",
		"--name", "Literature Guide",
		"--summary", "Guided preset draft for literature topics.",
		"--goal", "Help the user understand a literature topic with explanations and examples.",
		"--target-agent", "codex",
		"--providers", "codex,claude",
	}); err != nil {
		t.Fatal(err)
	}

	if err := Run([]string{"preset", "draft", "show", "--path", root, "--json", "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{
		"preset", "draft", "update",
		"--path", root,
		"--id", "literature-guide",
		"--summary", "Updated preset summary.",
		"--goal", "Help the user understand a literature topic through sharper structured explanations.",
		"--outputs", "preset_brief,preset_manifest,evaluation_pack,lesson_plan",
		"--workflow", "interview,simulate,refine,save",
		"--quality-gates", "profile_complete,simulation_ready,validation_ready,brief_reviewed",
		"--json",
	}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "validate", "--path", root, "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	session, err := presets.StartInterview(presets.StartInterviewOptions{
		WorkspaceRoot: root,
		DraftID:       "literature-guide",
		Mode:          "quick",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "interview", "remediate", "--path", root, "--json", session.ID}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "interview", "remediate", "--path", root, "--json", "literature-guide-00000000"}); err == nil {
		t.Fatal("expected remediation against a missing session to fail")
	}
	if err := Run([]string{"preset", "draft", "simulate", "--path", root, "--json", "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "mark-tested", "--path", root, "--json", "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "export", "--path", root, "--json", "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "publish", "--path", root, "--json", "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	catalogRoot := filepath.Join(root, "local-catalog")
	if err := Run([]string{"preset", "draft", "catalog-sync", "--path", root, "--root", catalogRoot, "--json", "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "install", "--path", root, "--json", "literature-guide"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"preset", "draft", "list", "--path", root, "--json"}); err != nil {
		t.Fatal(err)
	}

	draft, err := presets.LoadDraft(root, "literature-guide")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Profile.TargetAgent != "codex" {
		t.Fatalf("unexpected target agent: %q", draft.Profile.TargetAgent)
	}
	if draft.Profile.Summary != "Updated preset summary." {
		t.Fatalf("unexpected updated summary: %q", draft.Profile.Summary)
	}
	if draft.Profile.Status != "published" {
		t.Fatalf("expected published status, got %q", draft.Profile.Status)
	}
	if _, err := presets.LoadByID(project.ProjectFile(root, "presets", "exports"), "literature-guide-codex"); err != nil {
		t.Fatalf("expected exported codex bundle: %v", err)
	}
	if _, err := presets.LoadByID(project.ProjectFile(root, "presets", "exports"), "literature-guide-claude"); err != nil {
		t.Fatalf("expected exported claude bundle: %v", err)
	}
	if _, err := presets.LoadByID(catalogRoot, "literature-guide-codex"); err != nil {
		t.Fatalf("expected synced catalog codex bundle: %v", err)
	}
	if _, err := presets.LoadByID(catalogRoot, "literature-guide-claude"); err != nil {
		t.Fatalf("expected synced catalog claude bundle: %v", err)
	}
	if draft.EvaluationPack.Status != "published" {
		t.Fatalf("expected published evaluation pack, got %q", draft.EvaluationPack.Status)
	}
	if _, err := os.Stat(project.ProjectFile(root, "presets", "installed.json")); err != nil {
		t.Fatalf("expected installed preset registry: %v", err)
	}
}
