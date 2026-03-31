package presets

import (
	"path/filepath"
	"testing"

	"agent-os/internal/project"
)

func TestInitDraftCreatesAndLoadsCanonicalPresetDraft(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	created, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "literature-study",
		Name:          "Literature Study",
		Summary:       "Guided preset draft for literature explanations and examples.",
		Goal:          "Help the user understand a literature topic through guided explanations and examples.",
		PresetType:    "domain",
		TargetAgent:   "codex",
		Providers:     []string{"codex", "claude"},
		Version:       "0.1.0",
		BudgetProfile: "balanced",
		AutonomyLevel: "low",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Profile.ID != "literature-study" {
		t.Fatalf("unexpected profile id: %q", created.Profile.ID)
	}
	if created.Manifest.ID != created.Profile.ID {
		t.Fatalf("manifest id mismatch: %#v", created.Manifest)
	}
	if len(created.EvaluationPack.RepresentativeScenarios) != 3 {
		t.Fatalf("expected 3 representative scenarios, got %#v", created.EvaluationPack.RepresentativeScenarios)
	}
	for _, path := range []string{
		created.Paths.ProfilePath,
		created.Paths.BriefJSONPath,
		created.Paths.BriefMDPath,
		created.Paths.ManifestPath,
		created.Paths.EvaluationPath,
		created.Paths.EvaluationMD,
	} {
		if path == "" {
			t.Fatalf("expected non-empty path in draft paths: %#v", created.Paths)
		}
	}

	loaded, err := LoadDraft(root, "literature-study")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Profile.Name != "Literature Study" {
		t.Fatalf("unexpected loaded name: %q", loaded.Profile.Name)
	}
	if loaded.Brief.Markdown == "" {
		t.Fatalf("expected markdown brief, got %#v", loaded.Brief)
	}
	if loaded.Paths.ProfilePath != filepath.Join(root, ".arc", "presets", "platform", "literature-study", "profile.json") {
		t.Fatalf("unexpected stored profile path: %q", loaded.Paths.ProfilePath)
	}
}

func TestValidateProfileRequiresApprovalRulesForHigherAutonomy(t *testing.T) {
	profile := PresetProfile{
		ID:          "riskier",
		Name:        "Riskier",
		Summary:     "summary",
		PresetType:  "domain",
		TargetAgent: "codex",
		Goal:        "goal",
		Behavior: PresetBehavior{
			AutonomyLevel: "high",
		},
		Environment: PresetEnvironment{
			CompatibleProviders: []string{"codex"},
		},
		Outputs:      []string{"preset_manifest"},
		Workflow:     []string{"interview"},
		QualityGates: []string{"profile_complete"},
		MemoryPolicy: PresetMemoryPolicy{
			AllowedScopes: []string{"project", "presets/riskier"},
		},
		RuntimePolicy: PresetRuntimePolicy{
			ExecutionMode: "preview_only",
		},
		BudgetProfile: "balanced",
		Version:       "0.1.0",
		Status:        "draft",
	}
	if err := ValidateProfile(profile); err == nil {
		t.Fatal("expected approval policy requirement for high autonomy")
	}
}

func TestListDraftsReturnsSortedSummaries(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	_, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "alpha",
		Name:          "Alpha",
		Summary:       "alpha summary",
		Goal:          "alpha goal",
		TargetAgent:   "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "beta",
		Name:          "Beta",
		Summary:       "beta summary",
		Goal:          "beta goal",
		TargetAgent:   "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	items, err := ListDrafts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 draft summaries, got %#v", items)
	}
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.ID] = true
	}
	if !seen["alpha"] || !seen["beta"] {
		t.Fatalf("expected alpha and beta drafts, got %#v", items)
	}
}

func TestUpdateDraftRebuildsDerivedArtifacts(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	_, err := InitDraft(DraftInitOptions{
		WorkspaceRoot: root,
		ID:            "study-helper",
		Name:          "Study Helper",
		Summary:       "Initial summary",
		Goal:          "Initial goal",
		TargetAgent:   "codex",
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := UpdateDraft(DraftUpdateOptions{
		WorkspaceRoot: root,
		ID:            "study-helper",
		Summary:       "Updated summary",
		Goal:          "Teach literature topics through concise guided explanations.",
		Providers:     []string{"codex", "claude"},
		Outputs:       []string{"preset_brief", "preset_manifest", "evaluation_pack", "simulation_report"},
		Workflow:      []string{"interview", "simulate", "refine", "save"},
		QualityGates:  []string{"profile_complete", "simulation_ready", "validation_ready", "brief_reviewed"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Profile.Summary != "Updated summary" {
		t.Fatalf("expected updated summary, got %#v", updated.Profile)
	}
	if updated.Brief.Summary != "Updated summary" {
		t.Fatalf("expected rebuilt brief summary, got %#v", updated.Brief)
	}
	if len(updated.Manifest.CompatibleProviders) != 2 {
		t.Fatalf("expected rebuilt manifest providers, got %#v", updated.Manifest)
	}
	if len(updated.EvaluationPack.AcceptanceChecklist) == 0 {
		t.Fatalf("expected rebuilt evaluation pack, got %#v", updated.EvaluationPack)
	}
}

func TestUpdateDraftNormalizesAliasesAndPreservesWorkflowOrder(t *testing.T) {
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
		ID:            "normalize-me",
		Name:          "Normalize Me",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := UpdateDraft(DraftUpdateOptions{
		WorkspaceRoot: root,
		ID:            "normalize-me",
		Outputs:       []string{"brief, evaluation_pack, manifest"},
		Workflow:      []string{"refine, simulation, publish, interview"},
		QualityGates:  []string{"validated, brief-reviewed, simulation"},
	})
	if err != nil {
		t.Fatal(err)
	}
	expectedOutputs := []string{"preset_brief", "evaluation_pack", "preset_manifest"}
	for i, want := range expectedOutputs {
		if updated.Profile.Outputs[i] != want {
			t.Fatalf("expected output %d to be %q, got %#v", i, want, updated.Profile.Outputs)
		}
	}
	expectedWorkflow := []string{"refine", "simulate", "publish", "interview"}
	for i, want := range expectedWorkflow {
		if updated.Profile.Workflow[i] != want {
			t.Fatalf("expected workflow %d to be %q, got %#v", i, want, updated.Profile.Workflow)
		}
	}
	expectedGates := []string{"validation_ready", "brief_reviewed", "simulation_ready"}
	for i, want := range expectedGates {
		if updated.Profile.QualityGates[i] != want {
			t.Fatalf("expected quality gate %d to be %q, got %#v", i, want, updated.Profile.QualityGates)
		}
	}
}
