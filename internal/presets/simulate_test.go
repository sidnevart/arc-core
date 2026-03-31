package presets

import (
	"os"
	"testing"

	"agent-os/internal/project"
)

func TestSimulateDraftWritesArtifactsAndPassesForValidDraft(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "simulate-pass",
		Name:          "Simulate Pass",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
	}); err != nil {
		t.Fatal(err)
	}
	report, err := SimulateDraft(root, "simulate-pass")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if report.Paths.JSONPath == "" || report.Paths.MarkdownPath == "" {
		t.Fatalf("expected simulation artifact paths, got %#v", report.Paths)
	}
}

func TestSimulateDraftBlocksOnContradictions(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "simulate-blocked",
		Name:          "Simulate Blocked",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateDraft(DraftUpdateOptions{
		WorkspaceRoot: root,
		ID:            "simulate-blocked",
		Outputs:       []string{"evaluation_pack"},
		Workflow:      []string{"interview", "refine", "save"},
	}); err != nil {
		t.Fatal(err)
	}
	report, err := SimulateDraft(root, "simulate-blocked")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "blocked" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if len(report.Contradictions) == 0 {
		t.Fatalf("expected contradictions, got %#v", report)
	}
}

func TestMarkDraftTestedRequiresPassingSimulation(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "tested-pass",
		Name:          "Tested Pass",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := MarkDraftTested(root, "tested-pass"); err == nil {
		t.Fatal("expected mark-tested to fail without simulation")
	}
	if _, err := SimulateDraft(root, "tested-pass"); err != nil {
		t.Fatal(err)
	}
	result, err := MarkDraftTested(root, "tested-pass")
	if err != nil {
		t.Fatal(err)
	}
	if result.NewStatus != "tested" {
		t.Fatalf("expected tested status, got %#v", result)
	}
	draft, err := LoadDraft(root, "tested-pass")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Profile.Status != "tested" || draft.EvaluationPack.Status != "tested" {
		t.Fatalf("expected tested draft bundle, got %#v", draft)
	}
}

func TestPublishDraftRequiresTestedStatusAndWritesArtifacts(t *testing.T) {
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
		ID:            "publish-pass",
		Name:          "Publish Pass",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
		Providers:     []string{"codex", "claude"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := PublishDraft(root, "publish-pass"); err == nil {
		t.Fatal("expected publish to fail before tested status")
	}
	if _, err := SimulateDraft(root, "publish-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := MarkDraftTested(root, "publish-pass"); err != nil {
		t.Fatal(err)
	}
	result, err := PublishDraft(root, "publish-pass")
	if err != nil {
		t.Fatal(err)
	}
	if result.NewStatus != "published" {
		t.Fatalf("expected published status, got %#v", result)
	}
	if result.Paths.JSONPath == "" || result.Paths.MarkdownPath == "" {
		t.Fatalf("expected publish artifact paths, got %#v", result.Paths)
	}
	if result.Paths.EnvelopeJSONPath == "" || result.Paths.EnvelopeMDPath == "" {
		t.Fatalf("expected publish envelope paths, got %#v", result.Paths)
	}
	if _, err := os.Stat(result.Paths.EnvelopeJSONPath); err != nil {
		t.Fatalf("expected publish envelope json: %v", err)
	}
	if _, err := os.Stat(result.Paths.EnvelopeMDPath); err != nil {
		t.Fatalf("expected publish envelope markdown: %v", err)
	}
	if len(result.Envelope.Bundles) != 2 {
		t.Fatalf("expected publish envelope bundles, got %#v", result.Envelope)
	}
	for _, bundle := range result.Envelope.Bundles {
		if bundle.ManifestSHA256 == "" || bundle.ReadmeSHA256 == "" || bundle.OverviewSHA256 == "" || bundle.ProviderDocSHA256 == "" {
			t.Fatalf("expected populated bundle hashes, got %#v", bundle)
		}
	}
	draft, err := LoadDraft(root, "publish-pass")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Profile.Status != "published" || draft.EvaluationPack.Status != "published" {
		t.Fatalf("expected published draft bundle, got %#v", draft)
	}
}
