package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"agent-os/internal/budget"
	"agent-os/internal/contextpack"
	"agent-os/internal/contexttool"
	"agent-os/internal/presets"
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
		"environment_resolution.json",
		"environment_resolution.md",
		"memory_policy.json",
		"memory_policy.md",
		"context_pack.md",
		"context_pack.json",
		"arc_context_pack.md",
		"arc_context_pack.json",
		"ctx_context_pack.md",
		"ctx_context_pack.json",
		"ctx_context_metadata.json",
		"context_selection.json",
		"budget_assessment.json",
		"budget_usage_event.json",
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

	if got := run.Metadata["context_source"]; got == "" {
		t.Fatal("expected context_source metadata to be populated")
	}
	if got := run.Metadata["context_selection_reason"]; got == "" {
		t.Fatal("expected context_selection_reason metadata to be populated")
	}
	if got := run.Metadata["context_ctx_candidate_total"]; got == "" {
		t.Fatal("expected context_ctx_candidate_total metadata to be populated")
	}
	if got := run.Metadata["context_ctx_selected_total"]; got == "" {
		t.Fatal("expected context_ctx_selected_total metadata to be populated")
	}
	if got := run.Metadata["context_ctx_index_source"]; got == "" {
		t.Fatal("expected context_ctx_index_source metadata to be populated")
	}
	if got := run.Metadata["context_ctx_reused_artifact_count"]; got == "" {
		t.Fatal("expected context_ctx_reused_artifact_count metadata to be populated")
	}
	if got := run.Metadata["context_ctx_source_diversity"]; got == "" {
		t.Fatal("expected context_ctx_source_diversity metadata to be populated")
	}
	if got := run.Metadata["context_ctx_diversity_bonus"]; got == "" {
		t.Fatal("expected context_ctx_diversity_bonus metadata to be populated")
	}
	if got := run.Metadata["context_ctx_doc_family_diversity"]; got == "" {
		t.Fatal("expected context_ctx_doc_family_diversity metadata to be populated")
	}
	if got := run.Metadata["context_ctx_code_family_diversity"]; got == "" {
		t.Fatal("expected context_ctx_code_family_diversity metadata to be populated")
	}
	if got := run.Metadata["context_ctx_doc_cluster_diversity"]; got == "" {
		t.Fatal("expected context_ctx_doc_cluster_diversity metadata to be populated")
	}
	if got := run.Metadata["context_ctx_code_cluster_diversity"]; got == "" {
		t.Fatal("expected context_ctx_code_cluster_diversity metadata to be populated")
	}
	if got := run.Metadata["context_ctx_doc_dominant_cluster_share"]; got == "" {
		t.Fatal("expected context_ctx_doc_dominant_cluster_share metadata to be populated")
	}
	if got := run.Metadata["context_ctx_code_dominant_cluster_share"]; got == "" {
		t.Fatal("expected context_ctx_code_dominant_cluster_share metadata to be populated")
	}

	var ctxMeta map[string]any
	if err := project.ReadJSON(run.Artifacts["ctx_context_metadata.json"], &ctxMeta); err != nil {
		t.Fatal(err)
	}
	if _, ok := ctxMeta["section_provenance"]; !ok {
		t.Fatalf("expected section_provenance in ctx metadata: %#v", ctxMeta)
	}
	if _, ok := ctxMeta["accounting"]; !ok {
		t.Fatalf("expected accounting in ctx metadata: %#v", ctxMeta)
	}
	if _, ok := ctxMeta["reuse"]; !ok {
		t.Fatalf("expected reuse in ctx metadata: %#v", ctxMeta)
	}
	var items []map[string]any
	if err := project.ReadJSON(project.ProjectFile(root, "memory", "entries.json"), &items); err != nil {
		t.Fatal(err)
	}
	foundRunScope := false
	for _, item := range items {
		if scope, _ := item["scope"].(string); scope == "runs/"+run.ID {
			foundRunScope = true
			break
		}
	}
	if !foundRunScope {
		t.Fatalf("expected memory entries to use runs/%s scope, got %#v", run.ID, items)
	}
}

func TestRunTaskBlocksHighRiskProviderWorkInUltraSafeMode(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "hero",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "deploy production database migration",
		Mode:        "hero",
		Provider:    "codex",
		BudgetMode:  "ultra_safe",
		DryRun:      false,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "blocked" {
		t.Fatalf("expected blocked status, got %s", run.Status)
	}
	if got := run.Metadata["blocked_reason"]; got != "budget_policy" {
		t.Fatalf("blocked_reason = %q, want budget_policy", got)
	}
	if got := run.Metadata["budget_classification"]; got != "premium_high_risk" {
		t.Fatalf("budget_classification = %q, want premium_high_risk", got)
	}
}

func TestRunTaskRoutesLocalFirstProviderWorkWithoutCallingProvider(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "inspect context tool budget schema",
		Mode:        "work",
		Provider:    "codex",
		BudgetMode:  "balanced",
		DryRun:      false,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "done" {
		t.Fatalf("expected done status, got %s", run.Status)
	}
	if got := run.Metadata["budget_classification"]; got != "local_first" {
		t.Fatalf("budget_classification = %q, want local_first", got)
	}
	if got := run.Metadata["budget_route_locally"]; got != "true" {
		t.Fatalf("budget_route_locally = %q, want true", got)
	}
	if got := run.Metadata["provider_execution_mode"]; got != "local_routed" {
		t.Fatalf("provider_execution_mode = %q, want local_routed", got)
	}
	if run.ProviderResult != nil {
		t.Fatalf("expected provider result to be nil for local-routed task")
	}
}

func TestRunTaskBlocksPremiumRequiredInEmergencyLowLimit(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "implement the budget schema",
		Mode:        "work",
		Provider:    "codex",
		BudgetMode:  "emergency_low_limit",
		DryRun:      false,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "blocked" {
		t.Fatalf("expected blocked status, got %s", run.Status)
	}
	if got := run.Metadata["budget_low_limit_state"]; got != "emergency" {
		t.Fatalf("budget_low_limit_state = %q, want emergency", got)
	}
	if got := run.Metadata["budget_classification"]; got != "premium_required" {
		t.Fatalf("budget_classification = %q, want premium_required", got)
	}
}

func TestRunTaskRoutesCheapProviderWorkLocallyInEmergencyLowLimit(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "brainstorm three friendly names for the budget modes",
		Mode:        "work",
		Provider:    "codex",
		BudgetMode:  "emergency_low_limit",
		DryRun:      false,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "done" {
		t.Fatalf("expected done status, got %s", run.Status)
	}
	if got := run.Metadata["budget_classification"]; got != "cheap_provider_ok" {
		t.Fatalf("budget_classification = %q, want cheap_provider_ok", got)
	}
	if got := run.Metadata["budget_route_locally"]; got != "true" {
		t.Fatalf("budget_route_locally = %q, want true", got)
	}
	if got := run.Metadata["provider_execution_mode"]; got != "local_routed" {
		t.Fatalf("provider_execution_mode = %q, want local_routed", got)
	}
	if got := run.Metadata["budget_low_limit_state"]; got != "emergency" {
		t.Fatalf("budget_low_limit_state = %q, want emergency", got)
	}
	if got := run.Metadata["budget_confidence"]; got == "" {
		t.Fatalf("expected budget_confidence metadata")
	}
	if got := run.Metadata["budget_confidence_tier"]; got != "low" {
		t.Fatalf("budget_confidence_tier = %q, want low", got)
	}
	if got := run.Metadata["budget_signal_breakdown"]; got == "" {
		t.Fatalf("expected budget_signal_breakdown metadata")
	}
}

func TestRunTaskUsesPresetBudgetProfileWhenFlagIsAbsent(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	records := []presets.InstalledRecord{
		{
			InstallID:   "test-domain-budget",
			PresetID:    "study-budget",
			Version:     "0.1.0",
			Status:      "installed",
			InstalledAt: "2026-03-30T00:00:00Z",
			Manifest: presets.Manifest{
				ID:            "study-budget",
				Name:          "Study Budget",
				Tagline:       "Budget-biased preset",
				Goal:          "bias budget profile",
				Adapter:       "arc",
				Category:      "study",
				PresetType:    "domain",
				Version:       "0.1.0",
				BudgetProfile: "deep_work",
			},
		},
	}
	if err := project.WriteJSON(project.ProjectFile(root, "presets", "installed.json"), records); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "brainstorm three friendly names for the budget modes",
		Mode:        "work",
		Provider:    "codex",
		DryRun:      true,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := run.Metadata["environment_budget_profile"]; got != "deep_work" {
		t.Fatalf("environment_budget_profile = %q, want deep_work", got)
	}
	if got := run.Metadata["budget_mode"]; got != "deep_work" {
		t.Fatalf("budget_mode = %q, want deep_work", got)
	}
	if got := run.Metadata["budget_mode_source"]; got != "preset_profile" {
		t.Fatalf("budget_mode_source = %q, want preset_profile", got)
	}
	if got := run.Metadata["budget_confidence_tier"]; got != "high" {
		t.Fatalf("budget_confidence_tier = %q, want high", got)
	}
}

func TestRunTaskExplicitBudgetModeOverridesPresetBudgetProfile(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	records := []presets.InstalledRecord{
		{
			InstallID:   "test-domain-budget",
			PresetID:    "study-budget",
			Version:     "0.1.0",
			Status:      "installed",
			InstalledAt: "2026-03-30T00:00:00Z",
			Manifest: presets.Manifest{
				ID:            "study-budget",
				Name:          "Study Budget",
				Tagline:       "Budget-biased preset",
				Goal:          "bias budget profile",
				Adapter:       "arc",
				Category:      "study",
				PresetType:    "domain",
				Version:       "0.1.0",
				BudgetProfile: "deep_work",
			},
		},
	}
	if err := project.WriteJSON(project.ProjectFile(root, "presets", "installed.json"), records); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "brainstorm three friendly names for the budget modes",
		Mode:        "work",
		Provider:    "codex",
		BudgetMode:  "balanced",
		DryRun:      true,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := run.Metadata["environment_budget_profile"]; got != "deep_work" {
		t.Fatalf("environment_budget_profile = %q, want deep_work", got)
	}
	if got := run.Metadata["budget_mode"]; got != "balanced" {
		t.Fatalf("budget_mode = %q, want balanced", got)
	}
	if got := run.Metadata["budget_mode_source"]; got != "requested" {
		t.Fatalf("budget_mode_source = %q, want requested", got)
	}
	if got := run.Metadata["budget_signal_breakdown"]; got == "" {
		t.Fatalf("expected budget_signal_breakdown metadata")
	}
}

func TestRunTaskAppliesProjectBudgetOverride(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	preferLocal := false
	if err := project.WriteJSON(project.ProjectFile(root, "budget", "project_override.json"), budget.PolicyOverride{
		Mode:          "deep_work",
		PreferLocal:   &preferLocal,
		LowLimitState: "warning",
		Notes:         []string{"project override"},
	}); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "brainstorm three friendly names for the budget modes",
		Mode:        "work",
		Provider:    "codex",
		DryRun:      true,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := run.Metadata["budget_mode"]; got != "deep_work" {
		t.Fatalf("budget_mode = %q, want deep_work", got)
	}
	if got := run.Metadata["budget_mode_source"]; got != "project_override" {
		t.Fatalf("budget_mode_source = %q, want project_override", got)
	}
	if got := run.Metadata["budget_project_override_present"]; got != "true" {
		t.Fatalf("budget_project_override_present = %q, want true", got)
	}
	if got := run.Metadata["budget_override_sources"]; got != "project_override" {
		t.Fatalf("budget_override_sources = %q, want project_override", got)
	}
	if _, ok := run.Artifacts["budget_policy_resolution.json"]; !ok {
		t.Fatalf("expected budget_policy_resolution.json artifact")
	}
}

func TestRunTaskAppliesSessionBudgetOverrideFile(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	overridePath := filepath.Join(root, "session-budget-override.json")
	blockRequired := true
	if err := project.WriteJSON(overridePath, budget.PolicyOverride{
		Mode:                              "emergency_low_limit",
		BlockPremiumRequired:              &blockRequired,
		RequireApprovalForPremiumRequired: &blockRequired,
		Notes:                             []string{"session override"},
	}); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:           root,
		Task:           "implement the budget schema",
		Mode:           "work",
		Provider:       "codex",
		DryRun:         false,
		RunChecks:      false,
		UseProvider:    true,
		BudgetOverride: overridePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := run.Metadata["budget_mode"]; got != "emergency_low_limit" {
		t.Fatalf("budget_mode = %q, want emergency_low_limit", got)
	}
	if got := run.Metadata["budget_mode_source"]; got != "session_override" {
		t.Fatalf("budget_mode_source = %q, want session_override", got)
	}
	if got := run.Metadata["budget_session_override_present"]; got != "true" {
		t.Fatalf("budget_session_override_present = %q, want true", got)
	}
	if got := run.Metadata["blocked_reason"]; got != "budget_policy" {
		t.Fatalf("blocked_reason = %q, want budget_policy", got)
	}
	if _, ok := run.Artifacts["budget_policy_resolution.json"]; !ok {
		t.Fatalf("expected budget_policy_resolution.json artifact")
	}
}

func TestRunTaskRecordsRoutingTriggerForUltraSafeCheapProviderWork(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "brainstorm three friendly names for the budget modes",
		Mode:        "work",
		Provider:    "codex",
		BudgetMode:  "ultra_safe",
		DryRun:      false,
		RunChecks:   false,
		UseProvider: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := run.Metadata["budget_route_locally"]; got != "true" {
		t.Fatalf("budget_route_locally = %q, want true", got)
	}
	if got := run.Metadata["budget_routing_trigger"]; got != "constrained_low_confidence" {
		t.Fatalf("budget_routing_trigger = %q, want constrained_low_confidence", got)
	}
}

func TestChooseContextPackPrefersCtxWhenItIsSmaller(t *testing.T) {
	arcPack := contextpack.Pack{ApproxTokens: 500}
	ctxResult := contexttool.AssembleResult{Pack: contextpack.Pack{ApproxTokens: 300}, QualityScore: 10}
	selection := chooseContextPack(arcPack, ctxResult)
	if selection.SelectedSource != "ctx" {
		t.Fatalf("SelectedSource = %q, want ctx", selection.SelectedSource)
	}
	if selection.Selected.ApproxTokens != 300 {
		t.Fatalf("selected pack tokens = %d, want 300", selection.Selected.ApproxTokens)
	}
}

func TestChooseContextPackKeepsArcWhenCtxIsLarger(t *testing.T) {
	arcPack := contextpack.Pack{ApproxTokens: 300}
	ctxResult := contexttool.AssembleResult{Pack: contextpack.Pack{ApproxTokens: 500}, QualityScore: 10}
	selection := chooseContextPack(arcPack, ctxResult)
	if selection.SelectedSource != "arc" {
		t.Fatalf("SelectedSource = %q, want arc", selection.SelectedSource)
	}
	if selection.Selected.ApproxTokens != 300 {
		t.Fatalf("selected pack tokens = %d, want 300", selection.Selected.ApproxTokens)
	}
}

func TestChooseContextPackPrefersCtxOnQualityWithinTokenWindow(t *testing.T) {
	arcPack := contextpack.Pack{
		Task:         "explain context selection",
		ApproxTokens: 1000,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain context selection"},
			{Title: "Mode Policy", Content: "generic work mode"},
		},
	}
	ctxPack := contextpack.Pack{
		Task:         "explain context selection",
		ApproxTokens: 1080,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain context selection"},
			{Title: "Query Signals", Content: "context\nselection"},
			{Title: "Relevant Docs", Content: "context selection"},
			{Title: "Relevant Code Surfaces", Content: "context selection"},
		},
	}
	selection := chooseContextPack(arcPack, contexttool.AssembleResult{Pack: ctxPack, QualityScore: 474})
	if selection.SelectedSource != "ctx" {
		t.Fatalf("SelectedSource = %q, want ctx", selection.SelectedSource)
	}
	if selection.SelectionReason != "ctx_higher_quality_within_token_window" {
		t.Fatalf("SelectionReason = %q, want ctx_higher_quality_within_token_window", selection.SelectionReason)
	}
}

func TestChooseContextPackPrefersCtxForMemoryMatchesWithinExtendedWindow(t *testing.T) {
	arcPack := contextpack.Pack{
		Task:         "explain preset memory rules",
		ApproxTokens: 1000,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain preset memory rules"},
			{Title: "Relevant Docs", Content: "generic preset docs"},
		},
	}
	ctxPack := contextpack.Pack{
		Task:         "explain preset memory rules",
		ApproxTokens: 1180,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain preset memory rules"},
			{Title: "Relevant Memory", Content: "memory scope rules from human-authored decision"},
			{Title: "Memory Summary", Content: "same decision entry"},
		},
	}
	selection := chooseContextPack(arcPack, contexttool.AssembleResult{
		Pack:             ctxPack,
		QualityScore:     430,
		MemoryMatchCount: 1,
		MemoryBoost:      40,
	})
	if selection.SelectedSource != "ctx" {
		t.Fatalf("SelectedSource = %q, want ctx", selection.SelectedSource)
	}
	if selection.SelectionReason != "ctx_memory_match_within_extended_token_window" {
		t.Fatalf("SelectionReason = %q, want ctx_memory_match_within_extended_token_window", selection.SelectionReason)
	}
	if selection.CtxMemoryMatches != 1 {
		t.Fatalf("CtxMemoryMatches = %d, want 1", selection.CtxMemoryMatches)
	}
}

func TestChooseContextPackPrefersCtxForDiverseSourcesWithinExtendedWindow(t *testing.T) {
	arcPack := contextpack.Pack{
		Task:         "explain context selection",
		ApproxTokens: 1000,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain context selection"},
			{Title: "Mode Policy", Content: "generic work mode"},
		},
	}
	ctxPack := contextpack.Pack{
		Task:         "explain context selection",
		ApproxTokens: 1170,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain context selection"},
			{Title: "Relevant Docs", Content: "context selection docs"},
			{Title: "Relevant Code Surfaces", Content: "context selection code"},
			{Title: "Relevant Memory", Content: "context selection memory"},
		},
	}
	selection := chooseContextPack(arcPack, contexttool.AssembleResult{
		Pack:            ctxPack,
		QualityScore:    430,
		SourceDiversity: 3,
		DiversityBonus:  70,
	})
	if selection.SelectedSource != "ctx" {
		t.Fatalf("SelectedSource = %q, want ctx", selection.SelectedSource)
	}
	if selection.SelectionReason != "ctx_diverse_sources_within_extended_token_window" {
		t.Fatalf("SelectionReason = %q, want ctx_diverse_sources_within_extended_token_window", selection.SelectionReason)
	}
	if selection.CtxSourceDiversity != 3 {
		t.Fatalf("CtxSourceDiversity = %d, want 3", selection.CtxSourceDiversity)
	}
	if selection.CtxDocFamilyDiversity != 0 || selection.CtxCodeFamilyDiversity != 0 {
		t.Fatalf("unexpected family diversity mirrors in synthetic selection: %#v", selection)
	}
	if selection.CtxDocClusterDiversity != 0 || selection.CtxCodeClusterDiversity != 0 {
		t.Fatalf("unexpected cluster diversity mirrors in synthetic selection: %#v", selection)
	}
}

func TestChooseContextPackPrefersCtxForBalancedClustersWithinExtendedWindow(t *testing.T) {
	arcPack := contextpack.Pack{
		Task:         "explain context selection",
		ApproxTokens: 1000,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain context selection"},
			{Title: "Mode Policy", Content: "generic work mode"},
		},
	}
	ctxPack := contextpack.Pack{
		Task:         "explain context selection",
		ApproxTokens: 1160,
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain context selection"},
			{Title: "Relevant Docs", Content: "cluster-balanced docs"},
			{Title: "Relevant Code Surfaces", Content: "cluster-balanced code"},
		},
	}
	selection := chooseContextPack(arcPack, contexttool.AssembleResult{
		Pack:                     ctxPack,
		QualityScore:             340,
		DocClusterDiversity:      3,
		CodeClusterDiversity:     4,
		DocDominantClusterShare:  50,
		CodeDominantClusterShare: 40,
	})
	if selection.SelectedSource != "ctx" {
		t.Fatalf("SelectedSource = %q, want ctx", selection.SelectedSource)
	}
	if selection.SelectionReason != "ctx_cluster_balanced_within_extended_token_window" {
		t.Fatalf("SelectionReason = %q, want ctx_cluster_balanced_within_extended_token_window", selection.SelectionReason)
	}
}

func TestRunTaskWritesEnvironmentArtifactsAndExecutesHooks(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project.ProjectFile(root, "hooks", "prepare-context.sh"), []byte("#!/bin/sh\necho \"$ARC_HOOK_LIFECYCLE:$ARC_HOOK_NAME:$ARC_HOOK_OWNER_PRESET\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project.ProjectFile(root, "hooks", "before-run.sh"), []byte("#!/bin/sh\necho \"$ARC_HOOK_LIFECYCLE:$ARC_HOOK_NAME:$ARC_HOOK_OWNER_PRESET\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	records := []presets.InstalledRecord{
		{
			InstallID:   "test-infra",
			PresetID:    "infra-hooks",
			Version:     "0.1.0",
			Status:      "installed",
			InstalledAt: "2026-03-30T00:00:00Z",
			Manifest: presets.Manifest{
				ID:            "infra-hooks",
				Name:          "Infra Hooks",
				Tagline:       "Infra hooks",
				Goal:          "Provide runtime hooks",
				Adapter:       "arc",
				Category:      "infra",
				PresetType:    "infrastructure",
				Version:       "0.1.0",
				Files:         []string{},
				MemoryScopes:  []string{"system", "presets/infra-hooks"},
				BudgetProfile: "balanced",
				Permissions:   presets.Permissions{Runtime: "preview_only"},
				Hooks: []presets.Hook{
					{Name: "prepare-context", Lifecycle: "before_context_assembly", TimeoutSeconds: 5, PermissionScope: "read_only"},
					{Name: "before-run", Lifecycle: "before_run", TimeoutSeconds: 5, PermissionScope: "read_only"},
				},
				Author: presets.Author{Name: "ARC Team", Handle: "arc"},
			},
		},
	}
	if err := project.WriteJSON(project.ProjectFile(root, "presets", "installed.json"), records); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "exercise installed hooks",
		Mode:        "work",
		Provider:    "codex",
		DryRun:      true,
		RunChecks:   false,
		UseProvider: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "done" {
		t.Fatalf("expected done status, got %s", run.Status)
	}
	for _, name := range []string{"environment_resolution.json", "environment_resolution.md", "hook_execution.json", "hook_execution.md"} {
		path := run.Artifacts[name]
		if path == "" {
			t.Fatalf("expected artifact %s", name)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s to exist: %v", name, err)
		}
	}
	var summary presets.HookExecutionSummary
	if err := project.ReadJSON(run.Artifacts["hook_execution.json"], &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Executions) != 2 {
		t.Fatalf("expected 2 hook executions, got %#v", summary.Executions)
	}
	for _, execution := range summary.Executions {
		if execution.Status != "executed" {
			t.Fatalf("expected executed hook, got %#v", execution)
		}
	}
	var policy presets.MemoryPolicy
	if err := project.ReadJSON(run.Artifacts["memory_policy.json"], &policy); err != nil {
		t.Fatal(err)
	}
	foundRunScope := false
	for _, scope := range policy.AllowedScopes {
		if scope == "runs/"+run.ID {
			foundRunScope = true
			break
		}
	}
	if !foundRunScope {
		t.Fatalf("expected run-specific memory scope in policy, got %#v", policy.AllowedScopes)
	}
}

func TestRunTaskBlocksWhenHookRequiresApproval(t *testing.T) {
	root := t.TempDir()
	_, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/task\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project.ProjectFile(root, "hooks", "dangerous.sh"), []byte("#!/bin/sh\necho dangerous\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	records := []presets.InstalledRecord{
		{
			InstallID:   "test-risky",
			PresetID:    "infra-risky",
			Version:     "0.1.0",
			Status:      "installed",
			InstalledAt: "2026-03-30T00:00:00Z",
			Manifest: presets.Manifest{
				ID:           "infra-risky",
				Name:         "Infra Risky",
				Tagline:      "Risky hooks",
				Goal:         "Test approval gate",
				Adapter:      "arc",
				Category:     "infra",
				PresetType:   "infrastructure",
				Version:      "0.1.0",
				Files:        []string{},
				MemoryScopes: []string{"system", "presets/infra-risky"},
				Permissions:  presets.Permissions{Runtime: "risky_exec_requires_approval"},
				Hooks: []presets.Hook{
					{Name: "dangerous", Lifecycle: "before_run", TimeoutSeconds: 5, PermissionScope: "risky_exec_requires_approval"},
				},
				Author: presets.Author{Name: "ARC Team", Handle: "arc"},
			},
		},
	}
	if err := project.WriteJSON(project.ProjectFile(root, "presets", "installed.json"), records); err != nil {
		t.Fatal(err)
	}

	run, err := RunTask(root, TaskOptions{
		Root:        root,
		Task:        "exercise risky hook",
		Mode:        "work",
		Provider:    "codex",
		DryRun:      false,
		RunChecks:   false,
		UseProvider: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "blocked" {
		t.Fatalf("expected blocked status, got %s", run.Status)
	}
	if got := run.Metadata["blocked_reason"]; got != "hook_approval_required" {
		t.Fatalf("blocked_reason = %q, want hook_approval_required", got)
	}
	var summary presets.HookExecutionSummary
	if err := project.ReadJSON(run.Artifacts["hook_execution.json"], &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Executions) != 1 || summary.Executions[0].Status != "blocked_requires_approval" {
		raw, _ := json.Marshal(summary)
		t.Fatalf("unexpected hook execution summary: %s", string(raw))
	}
}
