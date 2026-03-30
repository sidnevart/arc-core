package budget

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-os/internal/project"
)

func TestAssessLocalFirstWhenProviderDisabled(t *testing.T) {
	got := Assess(Request{
		Task:        "inspect the repo status",
		Provider:    "codex",
		UseProvider: false,
		DryRun:      false,
		BudgetMode:  "balanced",
	})
	if got.Classification != ClassNoProvider {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassNoProvider)
	}
	if !got.RouteLocally {
		t.Fatalf("expected no-provider request to stay local")
	}
}

func TestAssessRoutesLocalFirstInBalancedMode(t *testing.T) {
	got := Assess(Request{
		Task:        "inspect context tool budget schema",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "balanced",
	})
	if got.Classification != ClassLocalFirst {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassLocalFirst)
	}
	if !got.RouteLocally {
		t.Fatalf("expected local_first request to route locally in balanced mode")
	}
	if got.RoutingReason == "" {
		t.Fatalf("expected routing reason to be populated")
	}
}

func TestAssessPrefersPremiumRequiredOverWeakLocalSignal(t *testing.T) {
	got := Assess(Request{
		Task:        "inspect and implement the budget schema",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "balanced",
	})
	if got.Classification != ClassPremiumRequired {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassPremiumRequired)
	}
	if got.RouteLocally {
		t.Fatalf("expected premium_required request to avoid local routing")
	}
}

func TestAssessPrefersHighRiskOverLocalSignals(t *testing.T) {
	got := Assess(Request{
		Task:        "inspect and deploy production database migration",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "balanced",
	})
	if got.Classification != ClassPremiumHighRisk {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassPremiumHighRisk)
	}
}

func TestAssessBlocksPremiumHighRiskInUltraSafe(t *testing.T) {
	got := Assess(Request{
		Task:        "deploy production database migration",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "ultra_safe",
	})
	if got.Classification != ClassPremiumHighRisk {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassPremiumHighRisk)
	}
	if !got.ShouldBlock {
		t.Fatalf("expected ultra_safe premium_high_risk request to block")
	}
}

func TestAssessBlocksPremiumRequiredInEmergencyLowLimit(t *testing.T) {
	got := Assess(Request{
		Task:        "implement the budget schema",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "emergency_low_limit",
	})
	if got.Classification != ClassPremiumRequired {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassPremiumRequired)
	}
	if got.LowLimitState != LowLimitEmergency {
		t.Fatalf("low limit state = %s, want %s", got.LowLimitState, LowLimitEmergency)
	}
	if !got.ShouldBlock {
		t.Fatalf("expected emergency_low_limit premium_required request to block")
	}
}

func TestAssessRoutesCheapProviderWorkLocallyInEmergencyLowLimit(t *testing.T) {
	got := Assess(Request{
		Task:        "brainstorm three friendly names for the budget modes",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "emergency_low_limit",
	})
	if got.Classification != ClassCheapProviderOK {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassCheapProviderOK)
	}
	if !got.RouteLocally {
		t.Fatalf("expected cheap provider task to route locally in emergency low-limit mode")
	}
	if got.RoutingReason == "" {
		t.Fatalf("expected routing reason to be populated")
	}
	if got.ConfidenceTier != "low" {
		t.Fatalf("confidence tier = %q, want low", got.ConfidenceTier)
	}
	if got.RoutingTrigger != "emergency_low_confidence" {
		t.Fatalf("routing trigger = %q, want emergency_low_confidence", got.RoutingTrigger)
	}
}

func TestAssessRoutesCheapProviderWorkLocallyInUltraSafe(t *testing.T) {
	got := Assess(Request{
		Task:        "brainstorm three friendly names for the budget modes",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "ultra_safe",
	})
	if got.Classification != ClassCheapProviderOK {
		t.Fatalf("classification = %s, want %s", got.Classification, ClassCheapProviderOK)
	}
	if !got.RouteLocally {
		t.Fatalf("expected cheap provider task to route locally in ultra_safe mode")
	}
	if got.RoutingTrigger != "constrained_low_confidence" {
		t.Fatalf("routing trigger = %q, want constrained_low_confidence", got.RoutingTrigger)
	}
}

func TestAssessIncludesConfidenceAndMatchedSignals(t *testing.T) {
	got := Assess(Request{
		Task:        "inspect and implement the budget schema",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "balanced",
	})
	if got.Confidence <= 0 {
		t.Fatalf("expected positive confidence")
	}
	if len(got.MatchedSignals) == 0 {
		t.Fatalf("expected matched signals")
	}
	if got.ConfidenceTier != "high" {
		t.Fatalf("confidence tier = %q, want high", got.ConfidenceTier)
	}
	if got.SignalBreakdown["premium"] == 0 {
		t.Fatalf("expected premium score in signal breakdown")
	}
}

func TestAssessWithPolicyHonorsOverridePolicy(t *testing.T) {
	preferLocal := false
	requirePremium := true
	policy := ApplyOverride(DefaultPolicy(ModeBalanced), PolicyOverride{
		PreferLocal:                       &preferLocal,
		RequireApprovalForPremiumRequired: &requirePremium,
	}, "test")

	local := AssessWithPolicy(Request{
		Task:        "inspect context tool budget schema",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "balanced",
	}, policy)
	if local.RouteLocally {
		t.Fatalf("expected prefer_local=false override to disable local routing")
	}

	premium := AssessWithPolicy(Request{
		Task:        "implement the budget schema",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "balanced",
	}, policy)
	if !premium.RequiresApproval {
		t.Fatalf("expected override to require approval for premium_required work")
	}
}

func TestEnsurePolicyWritesDefaultPolicy(t *testing.T) {
	root := t.TempDir()
	policy, err := EnsurePolicy(root, ModeBalanced)
	if err != nil {
		t.Fatalf("EnsurePolicy() error = %v", err)
	}
	if policy.Mode != ModeBalanced {
		t.Fatalf("policy mode = %s, want %s", policy.Mode, ModeBalanced)
	}
	if _, err := os.Stat(filepath.Join(root, ".arc", "budget", "policy.json")); err != nil {
		t.Fatalf("expected policy file to exist: %v", err)
	}
}

func TestResolveEffectiveModeFallsBackToPresetProfile(t *testing.T) {
	mode, source := ResolveEffectiveMode("", "deep_work")
	if mode != ModeDeepWork {
		t.Fatalf("mode = %s, want %s", mode, ModeDeepWork)
	}
	if source != "preset_profile" {
		t.Fatalf("source = %q, want preset_profile", source)
	}
}

func TestResolveEffectiveModePrefersRequestedOverPresetProfile(t *testing.T) {
	mode, source := ResolveEffectiveMode("balanced", "deep_work")
	if mode != ModeBalanced {
		t.Fatalf("mode = %s, want %s", mode, ModeBalanced)
	}
	if source != "requested" {
		t.Fatalf("source = %q, want requested", source)
	}
}

func TestResolveEffectiveModeUsesSessionAndProjectOverrides(t *testing.T) {
	mode, source := ResolveEffectiveModeWithOverrides("", PolicyOverride{Mode: "ultra_safe"}, PolicyOverride{Mode: "deep_work"}, "balanced")
	if mode != ModeUltraSafe {
		t.Fatalf("mode = %s, want %s", mode, ModeUltraSafe)
	}
	if source != "session_override" {
		t.Fatalf("source = %q, want session_override", source)
	}

	mode, source = ResolveEffectiveModeWithOverrides("", PolicyOverride{}, PolicyOverride{Mode: "deep_work"}, "balanced")
	if mode != ModeDeepWork {
		t.Fatalf("mode = %s, want %s", mode, ModeDeepWork)
	}
	if source != "project_override" {
		t.Fatalf("source = %q, want project_override", source)
	}
}

func TestEnsurePolicyRewritesStoredModeToRequestedMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".arc", "budget")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := project.WriteJSON(filepath.Join(path, "policy.json"), Policy{
		Mode:                              ModeUltraSafe,
		PreferLocal:                       true,
		BlockPremiumHighRisk:              true,
		RequireApprovalForPremiumHighRisk: true,
		RequireApprovalForPremiumRequired: true,
	}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	policy, err := EnsurePolicy(root, ModeBalanced)
	if err != nil {
		t.Fatalf("EnsurePolicy() error = %v", err)
	}
	if policy.Mode != ModeBalanced {
		t.Fatalf("policy mode = %s, want %s", policy.Mode, ModeBalanced)
	}
	if policy.BlockPremiumHighRisk {
		t.Fatalf("expected balanced policy to stop blocking premium_high_risk by default")
	}
	var persisted Policy
	if err := project.ReadJSON(filepath.Join(path, "policy.json"), &persisted); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if persisted.Mode != ModeBalanced {
		t.Fatalf("persisted policy mode = %s, want %s", persisted.Mode, ModeBalanced)
	}
}

func TestResolvePolicyAppliesProjectAndSessionOverrides(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".arc", "budget"), 0o755); err != nil {
		t.Fatal(err)
	}
	preferLocal := false
	requirePremium := true
	if err := project.WriteJSON(filepath.Join(root, ".arc", "budget", "project_override.json"), PolicyOverride{
		Mode:          "deep_work",
		PreferLocal:   &preferLocal,
		LowLimitState: "warning",
		Notes:         []string{"project override"},
	}); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(root, "session_budget_override.json")
	if err := project.WriteJSON(sessionPath, PolicyOverride{
		LowLimitState:                     "emergency",
		RequireApprovalForPremiumRequired: &requirePremium,
		Notes:                             []string{"session override"},
	}); err != nil {
		t.Fatal(err)
	}

	resolution, err := ResolvePolicy(root, "", "", sessionPath)
	if err != nil {
		t.Fatalf("ResolvePolicy() error = %v", err)
	}
	if resolution.EffectiveMode != ModeDeepWork {
		t.Fatalf("effective mode = %s, want %s", resolution.EffectiveMode, ModeDeepWork)
	}
	if resolution.EffectiveModeSource != "project_override" {
		t.Fatalf("mode source = %q, want project_override", resolution.EffectiveModeSource)
	}
	if resolution.EffectivePolicy.LowLimitState != LowLimitEmergency {
		t.Fatalf("low limit state = %s, want %s", resolution.EffectivePolicy.LowLimitState, LowLimitEmergency)
	}
	if resolution.EffectivePolicy.PreferLocal {
		t.Fatalf("prefer_local = true, want false from project override")
	}
	if !resolution.EffectivePolicy.RequireApprovalForPremiumRequired {
		t.Fatalf("expected session override to require approval for premium_required")
	}
	if len(resolution.AppliedOverrideSources) != 2 {
		t.Fatalf("applied override sources = %v, want 2 entries", resolution.AppliedOverrideSources)
	}
	var persisted Policy
	if err := project.ReadJSON(filepath.Join(root, ".arc", "budget", "policy.json"), &persisted); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}
	if persisted.LowLimitState != LowLimitEmergency {
		t.Fatalf("persisted low limit state = %s, want %s", persisted.LowLimitState, LowLimitEmergency)
	}
}

func TestAppendUsageEventWritesJSONL(t *testing.T) {
	root := t.TempDir()
	event := UsageEvent{RunID: "run-1", Task: "implement budget layer", Classification: ClassPremiumRequired, Status: "done"}
	if err := AppendUsageEvent(root, event); err != nil {
		t.Fatalf("AppendUsageEvent() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".arc", "budget", "usage_events.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), `"run_id":"run-1"`) {
		t.Fatalf("expected usage event to be written, got:\n%s", string(data))
	}
}

func TestNewUsageEventCarriesPromptMinimizationAttribution(t *testing.T) {
	event := NewUsageEvent("run-ctx", Request{
		Task:        "inspect context tool budget schema",
		Provider:    "codex",
		UseProvider: true,
		DryRun:      false,
		BudgetMode:  "balanced",
	}, Assessment{
		Mode:           ModeBalanced,
		LowLimitState:  LowLimitNormal,
		Classification: ClassLocalFirst,
		Confidence:     75,
		ConfidenceTier: "high",
		RouteLocally:   true,
	}, UsageContext{
		ProjectRoot:                  "/tmp/demo",
		ProviderModel:                "gpt-5.4",
		ProviderSessionID:            "session-ctx-1",
		BudgetModeSource:             "requested",
		EnvironmentBudgetProfile:     "balanced",
		ContextSource:                "ctx",
		ContextSelectionReason:       "ctx_smaller_or_equal_and_quality_not_worse",
		ContextArcTokens:             2000,
		ContextCtxTokens:             700,
		ContextSelectedTokens:        700,
		ContextTokenReduction:        1300,
		ContextTokenReductionPercent: 65,
		PromptMinimized:              true,
	}, "done", false, nil)
	if event.ProjectRoot != "/tmp/demo" {
		t.Fatalf("project root = %q, want /tmp/demo", event.ProjectRoot)
	}
	if !event.PromptMinimized {
		t.Fatalf("expected prompt minimized flag")
	}
	if event.ContextTokenReductionPercent != 65 {
		t.Fatalf("token reduction percent = %d, want 65", event.ContextTokenReductionPercent)
	}
	if event.ContextSource != "ctx" {
		t.Fatalf("context source = %q, want ctx", event.ContextSource)
	}
	if event.ProviderModel != "gpt-5.4" {
		t.Fatalf("provider model = %q, want gpt-5.4", event.ProviderModel)
	}
	if event.ProviderSessionID != "session-ctx-1" {
		t.Fatalf("provider session id = %q, want session-ctx-1", event.ProviderSessionID)
	}
}
