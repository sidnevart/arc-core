package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"agent-os/internal/budget"
	"agent-os/internal/contextpack"
	"agent-os/internal/contexttool"
	"agent-os/internal/governance"
	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/mode"
	"agent-os/internal/presets"
	"agent-os/internal/project"
	"agent-os/internal/provider"
)

type State string

const (
	StateInitialized       State = "initialized"
	StateContextCollecting State = "context_collecting"
	StatePlanning          State = "planning"
	StateNeedsHumanContext State = "needs_human_context"
	StateImplementing      State = "implementing"
	StateVerifying         State = "verifying"
	StateReviewing         State = "reviewing"
	StateDocumenting       State = "documenting"
	StateDone              State = "done"
	StateFailed            State = "failed"
	StateBlocked           State = "blocked"
)

type Transition struct {
	State     State  `json:"state"`
	Timestamp string `json:"timestamp"`
	Notes     string `json:"notes"`
}

type Run struct {
	ID                string               `json:"id"`
	Command           string               `json:"command"`
	Task              string               `json:"task"`
	Mode              string               `json:"mode"`
	Provider          string               `json:"provider"`
	Status            string               `json:"status"`
	CurrentState      State                `json:"current_state"`
	StartedAt         string               `json:"started_at"`
	UpdatedAt         string               `json:"updated_at"`
	DryRun            bool                 `json:"dry_run"`
	ProviderSessionID string               `json:"provider_session_id,omitempty"`
	Artifacts         map[string]string    `json:"artifacts"`
	Metadata          map[string]string    `json:"metadata,omitempty"`
	Transitions       []Transition         `json:"transitions"`
	ProviderResult    *provider.TaskResult `json:"provider_result,omitempty"`
	Verification      map[string]string    `json:"verification,omitempty"`
	Review            map[string]string    `json:"review,omitempty"`
	Docs              map[string]string    `json:"docs,omitempty"`
}

type TraceEvent struct {
	Timestamp string         `json:"timestamp"`
	Type      string         `json:"type"`
	Message   string         `json:"message"`
	Data      map[string]any `json:"data,omitempty"`
}

type TaskOptions struct {
	Root            string
	Task            string
	Mode            string
	Provider        string
	BudgetMode      string
	BudgetOverride  string
	Model           string
	DryRun          bool
	RunChecks       bool
	UseProvider     bool
	ApproveRisky    bool
	ProviderTimeout time.Duration
}

type VerifyOptions struct {
	Root      string
	RunID     string
	RunChecks bool
}

type ContextSelection struct {
	SelectedSource              string           `json:"selected_source"`
	SelectionReason             string           `json:"selection_reason"`
	Selected                    contextpack.Pack `json:"selected"`
	ArcPack                     contextpack.Pack `json:"arc_pack"`
	CtxPack                     contextpack.Pack `json:"ctx_pack"`
	ArcTokens                   int              `json:"arc_tokens"`
	CtxTokens                   int              `json:"ctx_tokens"`
	TokenReduction              int              `json:"token_reduction"`
	ArcQuality                  int              `json:"arc_quality"`
	CtxQuality                  int              `json:"ctx_quality"`
	CtxMemoryMatches            int              `json:"ctx_memory_matches"`
	CtxMemoryBoost              int              `json:"ctx_memory_boost"`
	CtxMemoryTrustBonus         int              `json:"ctx_memory_trust_bonus"`
	CtxMemoryRecencyBonus       int              `json:"ctx_memory_recency_bonus"`
	CtxSourceDiversity          int              `json:"ctx_source_diversity"`
	CtxDiversityBonus           int              `json:"ctx_diversity_bonus"`
	CtxDocFamilyDiversity       int              `json:"ctx_doc_family_diversity"`
	CtxCodeFamilyDiversity      int              `json:"ctx_code_family_diversity"`
	CtxDocClusterDiversity      int              `json:"ctx_doc_cluster_diversity"`
	CtxCodeClusterDiversity     int              `json:"ctx_code_cluster_diversity"`
	CtxDocDominantClusterShare  int              `json:"ctx_doc_dominant_cluster_share"`
	CtxCodeDominantClusterShare int              `json:"ctx_code_dominant_cluster_share"`
}

func Plan(root string, opts TaskOptions) (Run, error) {
	run := newRun("task plan", opts)
	if err := saveRun(root, &run); err != nil {
		return Run{}, err
	}
	_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_created", Message: "planning run created"})

	if err := transition(root, &run, StateContextCollecting, "collecting project context"); err != nil {
		return Run{}, err
	}
	resolution, err := resolveRunEnvironment(root)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	if err := writeEnvironmentArtifacts(root, &run, resolution); err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	memoryPolicy, err := writeMemoryPolicyArtifacts(root, &run, resolution)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "before_context_assembly", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}
	idx, items, pack, ctxResult, selection, err := collectContext(root, opts.Task, opts.Mode)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}

	if err := writeContextArtifacts(root, &run, idx, items, pack, ctxResult, selection); err != nil {
		return Run{}, err
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "after_context_assembly", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}
	if err := transition(root, &run, StatePlanning, "writing planning artifacts"); err != nil {
		return Run{}, err
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "before_persist_memory", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}

	if err := writePlanningArtifacts(root, &run, opts.Task, pack, memoryPolicy); err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}

	if err := finishRun(root, &run); err != nil {
		return Run{}, err
	}
	_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_completed", Message: "planning run completed"})
	return run, nil
}

func RunTask(root string, opts TaskOptions) (Run, error) {
	run := newRun("task run", opts)
	providerFailed := false
	var budgetAssessment *budget.Assessment
	var effectiveBudgetMode budget.Mode
	var providerUsed bool
	effectiveUseProvider := opts.UseProvider
	budgetNotes := []string{}
	budgetModeSource := "default"
	if err := saveRun(root, &run); err != nil {
		return Run{}, err
	}
	defer func() {
		if budgetAssessment == nil {
			return
		}
		event := budget.NewUsageEvent(run.ID, budget.Request{
			Task:        run.Task,
			Provider:    run.Provider,
			UseProvider: effectiveUseProvider,
			DryRun:      opts.DryRun,
			BudgetMode:  string(effectiveBudgetMode),
		}, *budgetAssessment, run.Status, providerUsed, run.ProviderResult, budgetNotes...)
		_ = budget.AppendUsageEvent(root, event)
		eventPath := filepath.Join(runDir(root, run.ID), "budget_usage_event.json")
		_ = project.WriteJSON(eventPath, event)
		run.Artifacts["budget_usage_event.json"] = eventPath
		_ = saveRun(root, &run)
	}()
	_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_created", Message: "task run created"})

	if err := transition(root, &run, StateContextCollecting, "collecting project context"); err != nil {
		return Run{}, err
	}
	resolution, err := resolveRunEnvironment(root)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	if err := writeEnvironmentArtifacts(root, &run, resolution); err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	memoryPolicy, err := writeMemoryPolicyArtifacts(root, &run, resolution)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "before_context_assembly", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}
	idx, items, pack, ctxResult, selection, err := collectContext(root, opts.Task, opts.Mode)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	if err := writeContextArtifacts(root, &run, idx, items, pack, ctxResult, selection); err != nil {
		return Run{}, err
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "after_context_assembly", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}

	if err := transition(root, &run, StatePlanning, "building specs and question bundle"); err != nil {
		return Run{}, err
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "before_persist_memory", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}
	if err := writePlanningArtifacts(root, &run, opts.Task, pack, memoryPolicy); err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}

	reqBudget := budget.Request{
		Task:        opts.Task,
		Provider:    opts.Provider,
		UseProvider: opts.UseProvider,
		DryRun:      opts.DryRun,
	}
	policyResolution, err := budget.ResolvePolicy(root, opts.BudgetMode, resolution.EffectiveBudgetProfile, opts.BudgetOverride)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	effectiveBudgetMode = policyResolution.EffectiveMode
	budgetModeSource = policyResolution.EffectiveModeSource
	reqBudget.BudgetMode = string(effectiveBudgetMode)
	policy := policyResolution.EffectivePolicy
	assessedBudget := budget.AssessWithPolicy(reqBudget, policy)
	budgetAssessment = &assessedBudget
	budgetPath := filepath.Join(runDir(root, run.ID), "budget_assessment.json")
	if err := project.WriteJSON(budgetPath, map[string]any{
		"policy":                     policy,
		"policy_resolution":          policyResolution,
		"assessment":                 assessedBudget,
		"provider":                   opts.Provider,
		"task":                       opts.Task,
		"requested_budget_mode":      opts.BudgetMode,
		"environment_budget_profile": resolution.EffectiveBudgetProfile,
		"budget_mode_source":         budgetModeSource,
		"budget_mode":                assessedBudget.Mode,
		"budget_override_path":       opts.BudgetOverride,
		"project_override_present":   policyResolution.ProjectOverridePresent,
		"session_override_present":   policyResolution.SessionOverridePresent,
		"applied_override_sources":   policyResolution.AppliedOverrideSources,
		"project_override_path":      policyResolution.ProjectOverridePath,
		"session_override_path":      policyResolution.SessionOverridePath,
	}); err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	run.Artifacts["budget_assessment.json"] = budgetPath
	policyResolutionPath := filepath.Join(runDir(root, run.ID), "budget_policy_resolution.json")
	if err := project.WriteJSON(policyResolutionPath, policyResolution); err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	run.Artifacts["budget_policy_resolution.json"] = policyResolutionPath
	run.Metadata["budget_requested_mode"] = opts.BudgetMode
	run.Metadata["budget_override_path"] = opts.BudgetOverride
	run.Metadata["budget_mode"] = string(assessedBudget.Mode)
	run.Metadata["budget_mode_source"] = budgetModeSource
	run.Metadata["budget_project_override_present"] = fmt.Sprintf("%t", policyResolution.ProjectOverridePresent)
	run.Metadata["budget_session_override_present"] = fmt.Sprintf("%t", policyResolution.SessionOverridePresent)
	run.Metadata["budget_override_sources"] = strings.Join(policyResolution.AppliedOverrideSources, ",")
	run.Metadata["budget_low_limit_state"] = string(assessedBudget.LowLimitState)
	run.Metadata["budget_classification"] = string(assessedBudget.Classification)
	run.Metadata["budget_confidence"] = fmt.Sprintf("%d", assessedBudget.Confidence)
	run.Metadata["budget_confidence_tier"] = assessedBudget.ConfidenceTier
	run.Metadata["budget_matched_signals"] = strings.Join(assessedBudget.MatchedSignals, ",")
	run.Metadata["budget_signal_breakdown"] = formatBudgetBreakdown(assessedBudget.SignalBreakdown)
	run.Metadata["budget_requires_approval"] = fmt.Sprintf("%t", assessedBudget.RequiresApproval)
	run.Metadata["budget_should_block"] = fmt.Sprintf("%t", assessedBudget.ShouldBlock)
	run.Metadata["budget_route_locally"] = fmt.Sprintf("%t", assessedBudget.RouteLocally)
	run.Metadata["budget_routing_reason"] = assessedBudget.RoutingReason
	run.Metadata["budget_routing_trigger"] = assessedBudget.RoutingTrigger
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "budget_assessment",
		Message:   "budget assessment completed",
		Data: map[string]any{
			"budget_mode":        assessedBudget.Mode,
			"budget_mode_source": budgetModeSource,
			"project_override":   policyResolution.ProjectOverridePresent,
			"session_override":   policyResolution.SessionOverridePresent,
			"low_limit_state":    assessedBudget.LowLimitState,
			"classification":     assessedBudget.Classification,
			"confidence":         assessedBudget.Confidence,
			"confidence_tier":    assessedBudget.ConfidenceTier,
			"matched_signals":    assessedBudget.MatchedSignals,
			"signal_breakdown":   assessedBudget.SignalBreakdown,
			"requires_approval":  assessedBudget.RequiresApproval,
			"should_block":       assessedBudget.ShouldBlock,
			"route_locally":      assessedBudget.RouteLocally,
			"prefer_local":       policy.PreferLocal,
			"routing_reason":     assessedBudget.RoutingReason,
			"routing_trigger":    assessedBudget.RoutingTrigger,
			"reasoning":          assessedBudget.Reasoning,
		},
	})
	_ = saveRun(root, &run)
	if assessedBudget.ShouldBlock && !opts.ApproveRisky && opts.UseProvider && !opts.DryRun {
		_ = transition(root, &run, StateBlocked, "budget policy blocked provider execution")
		run.Status = "blocked"
		run.Metadata["blocked_reason"] = "budget_policy"
		_ = saveRun(root, &run)
		_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_blocked", Message: "task blocked by budget policy"})
		return run, nil
	}
	if assessedBudget.RequiresApproval && !opts.ApproveRisky && opts.UseProvider && !opts.DryRun {
		_ = transition(root, &run, StateBlocked, "budget policy requires approval")
		run.Status = "blocked"
		run.Metadata["blocked_reason"] = "budget_approval_required"
		_ = saveRun(root, &run)
		_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_blocked", Message: "task blocked until budget approval is granted"})
		return run, nil
	}

	if assessedBudget.RouteLocally && opts.UseProvider {
		effectiveUseProvider = false
		budgetNotes = append(budgetNotes, assessedBudget.RoutingReason)
		run.Metadata["provider_execution_mode"] = "local_routed"
		_ = saveRun(root, &run)
		_ = appendTrace(root, run.ID, TraceEvent{
			Timestamp: nowUTC(),
			Type:      "budget_routing",
			Message:   "provider execution rerouted to local-only path",
			Data: map[string]any{
				"classification": assessedBudget.Classification,
				"reason":         assessedBudget.RoutingReason,
				"budget_mode":    assessedBudget.Mode,
			},
		})
	} else if opts.UseProvider {
		run.Metadata["provider_execution_mode"] = "provider"
	} else {
		effectiveUseProvider = false
		run.Metadata["provider_execution_mode"] = "local_only"
	}

	assessment := governance.Assess(opts.Task)
	if err := writeApprovalReport(root, &run, assessment, opts.ApproveRisky); err != nil {
		return Run{}, err
	}
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "approval_assessment",
		Message:   "approval assessment completed",
		Data: map[string]any{
			"risk_level":        assessment.RiskLevel,
			"requires_approval": assessment.RequiresApproval,
			"approved_for_run":  opts.ApproveRisky,
			"gates":             assessment.Gates,
		},
	})
	run.Metadata["risk_level"] = assessment.RiskLevel
	run.Metadata["requires_approval"] = fmt.Sprintf("%t", assessment.RequiresApproval)
	if assessment.RequiresApproval && !opts.ApproveRisky && effectiveUseProvider && !opts.DryRun {
		_ = transition(root, &run, StateBlocked, "approval gate triggered")
		run.Status = "blocked"
		run.Metadata["blocked_reason"] = "approval_required"
		_ = saveRun(root, &run)
		_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_blocked", Message: "task blocked by approval gate"})
		return run, nil
	}

	if err := transition(root, &run, StateImplementing, "executing implementation stage"); err != nil {
		return Run{}, err
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "before_run", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}
	if err := writeImplementationLog(root, &run, "Implementation stage started."); err != nil {
		return Run{}, err
	}

	if effectiveUseProvider && !opts.DryRun {
		adapter, err := provider.Get(opts.Provider)
		if err != nil {
			_ = failRun(root, &run, err)
			return Run{}, err
		}
		status := adapter.CheckInstalled(context.Background())
		if !status.Installed {
			_ = transition(root, &run, StateBlocked, "selected provider is not installed")
			run.Status = "blocked"
			run.Metadata["blocked_reason"] = "provider_unavailable"
			_ = saveRun(root, &run)
			_ = writeQuestionBundle(root, &run, []string{"Selected provider is unavailable. Install it or rerun with --dry-run."}, memoryPolicy)
			return run, nil
		}

		req := provider.TaskRequest{
			Root:           root,
			Prompt:         providerPrompt(opts.Task, pack, run.Mode),
			Mode:           run.Mode,
			Model:          opts.Model,
			RunDir:         runDir(root, run.ID),
			LastMessageOut: filepath.Join(runDir(root, run.ID), "provider_last_message.md"),
			TranscriptOut:  filepath.Join(runDir(root, run.ID), "provider_transcript.jsonl"),
			DryRun:         opts.DryRun,
			Timeout:        opts.ProviderTimeout,
		}
		result, err := adapter.RunTask(context.Background(), req)
		providerUsed = true
		run.ProviderResult = &result
		if result.SessionID != "" {
			run.ProviderSessionID = result.SessionID
		}
		_ = appendTrace(root, run.ID, TraceEvent{
			Timestamp: nowUTC(),
			Type:      "provider_execution",
			Message:   "provider execution finished",
			Data: map[string]any{
				"provider":    run.Provider,
				"exit_code":   result.ExitCode,
				"session_id":  result.SessionID,
				"stdout_path": result.StdoutPath,
				"stderr_path": result.StderrPath,
			},
		})
		if result.StdoutPath != "" {
			run.Artifacts["provider_transcript.jsonl"] = result.StdoutPath
		}
		if result.StderrPath != "" {
			run.Artifacts["provider_stderr.log"] = result.StderrPath
		}
		if _, statErr := os.Stat(req.LastMessageOut); statErr == nil {
			run.Artifacts["provider_last_message.md"] = req.LastMessageOut
		}
		_ = saveRun(root, &run)
		if err != nil {
			providerFailed = true
			run.Metadata["provider_error"] = err.Error()
			_ = saveRun(root, &run)
		}
	}
	if err := runHookLifecycle(root, &run, resolution, memoryPolicy, "after_run", opts); err != nil {
		return Run{}, err
	}
	if run.Status == "blocked" {
		return run, nil
	}

	if err := transition(root, &run, StateVerifying, "producing verification report"); err != nil {
		return Run{}, err
	}
	verification, err := Verify(root, VerifyOptions{Root: root, RunID: run.ID, RunChecks: opts.RunChecks})
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	run.Verification = verification
	if path := verification["report"]; path != "" {
		run.Artifacts["verification_report.md"] = path
	}
	_ = saveRun(root, &run)

	if err := transition(root, &run, StateReviewing, "producing review report"); err != nil {
		return Run{}, err
	}
	review, err := Review(root, run.ID)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	run.Review = review
	if path := review["report"]; path != "" {
		run.Artifacts["review_report.md"] = path
	}
	if path := review["anti_hallucination_report"]; path != "" {
		run.Artifacts["anti_hallucination_report.md"] = path
	}
	if warning := review["evidence_warning"]; warning != "" {
		run.Metadata["evidence_warning"] = warning
	}
	_ = saveRun(root, &run)

	if err := transition(root, &run, StateDocumenting, "producing docs delta"); err != nil {
		return Run{}, err
	}
	docs, err := GenerateDocs(root, run.ID)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	run.Docs = docs
	if path := docs["report"]; path != "" {
		run.Artifacts["docs_delta.md"] = path
	}
	_ = saveRun(root, &run)

	if providerFailed {
		run.CurrentState = StateFailed
		run.Status = "failed"
		run.Transitions = append(run.Transitions, Transition{
			State:     StateFailed,
			Timestamp: nowUTC(),
			Notes:     "provider execution failed; post-run reports were still generated",
		})
		_ = saveRun(root, &run)
		_ = writePostRunArtifacts(root, &run)
		updateMetrics(root, run)
		_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_failed", Message: "task run completed with provider failure"})
		return run, nil
	}

	if err := finishRun(root, &run); err != nil {
		return Run{}, err
	}
	_ = writePostRunArtifacts(root, &run)
	updateMetrics(root, run)
	_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_completed", Message: "task run completed"})
	return run, nil
}

func Verify(root string, opts VerifyOptions) (map[string]string, error) {
	run, err := LoadRun(root, opts.RunID)
	if err != nil {
		return nil, err
	}

	reportPath := filepath.Join(runDir(root, run.ID), "verification_report.md")
	results := []string{}
	if opts.RunChecks {
		results = append(results, runDefaultChecks(root)...)
	} else {
		results = append(results, "Checks were not executed. Re-run with --run-checks to execute detected checks.")
	}

	content := "# Verification Report\n\n"
	content += fmt.Sprintf("- run_id: %s\n- mode: %s\n- provider: %s\n\n", run.ID, run.Mode, run.Provider)
	for _, line := range results {
		content += "- " + line + "\n"
	}
	if err := project.WriteString(reportPath, content); err != nil {
		return nil, err
	}
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "verification",
		Message:   "verification report written",
		Data: map[string]any{
			"report": reportPath,
			"checks": results,
		},
	})
	run.Artifacts["verification_report.md"] = reportPath
	run.Verification = map[string]string{"report": reportPath}
	_ = saveRun(root, &run)

	return map[string]string{"report": reportPath}, nil
}

func Review(root string, runID string) (map[string]string, error) {
	run, err := LoadRun(root, runID)
	if err != nil {
		return nil, err
	}

	evidencePath, evidenceWarning, err := writeEvidenceReport(root, &run)
	if err != nil {
		return nil, err
	}

	reportPath := filepath.Join(runDir(root, run.ID), "review_report.md")
	lines := []string{
		"# Review Report",
		"",
		"- Focus: bugs, regressions, unsupported assumptions, missing tests.",
		"- Anti-hallucination audit: " + evidencePath,
	}
	if run.ProviderResult == nil {
		lines = append(lines, "- No provider result captured; review is limited to generated artifacts.")
	}
	if evidenceWarning {
		lines = append(lines, "- Warning: evidence labels are missing or too weak in the current run artifacts.")
	}
	if run.Verification == nil {
		lines = append(lines, "- Verification stage has not been recorded yet.")
	}
	if run.Status == "blocked" {
		lines = append(lines, "- Run is blocked; implementation did not fully execute.")
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := project.WriteString(reportPath, content); err != nil {
		return nil, err
	}
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "review",
		Message:   "review report written",
		Data: map[string]any{
			"report": reportPath,
		},
	})
	run.Artifacts["review_report.md"] = reportPath
	_ = saveRun(root, &run)
	return map[string]string{
		"report":                    reportPath,
		"anti_hallucination_report": evidencePath,
		"evidence_warning":          fmt.Sprintf("%t", evidenceWarning),
	}, nil
}

func GenerateDocs(root string, runID string) (map[string]string, error) {
	return GenerateDocsWithApply(root, runID, false)
}

func GenerateDocsWithApply(root string, runID string, apply bool) (map[string]string, error) {
	run, err := LoadRun(root, runID)
	if err != nil {
		return nil, err
	}

	reportPath := filepath.Join(runDir(root, run.ID), "docs_delta.md")
	content := "# Docs Delta\n\n"
	content += "- Update user-facing docs if command behavior changed.\n"
	content += "- Update repo maps if new important code surfaces were introduced.\n"
	content += "- Update memory and decisions if architecture understanding changed.\n"
	content += fmt.Sprintf("- apply: %t\n", apply)
	content += "\nArtifacts:\n"
	for name, path := range run.Artifacts {
		content += fmt.Sprintf("- %s: %s\n", name, path)
	}
	if err := project.WriteString(reportPath, content); err != nil {
		return nil, err
	}
	appliedPaths := []string{}
	if apply {
		applied, err := applyDocsUpdates(root)
		if err != nil {
			return nil, err
		}
		appliedPaths = applied
	}
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "docs",
		Message:   "docs delta written",
		Data: map[string]any{
			"report":        reportPath,
			"apply":         apply,
			"applied_paths": appliedPaths,
		},
	})
	run.Artifacts["docs_delta.md"] = reportPath
	_ = saveRun(root, &run)
	result := map[string]string{"report": reportPath}
	if len(appliedPaths) > 0 {
		result["applied"] = strings.Join(appliedPaths, ",")
	}
	return result, nil
}

func LoadRun(root string, runID string) (Run, error) {
	if runID == "" {
		latest, err := project.LatestRunDir(root)
		if err != nil {
			return Run{}, err
		}
		runID = filepath.Base(latest)
	}
	var run Run
	if err := project.ReadJSON(filepath.Join(runDir(root, runID), "run.json"), &run); err != nil {
		return Run{}, err
	}
	return run, nil
}

func ListRuns(root string) ([]Run, error) {
	runsDir := project.ProjectFile(root, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return nil, err
	}
	runs := []Run{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		run, err := LoadRun(root, entry.Name())
		if err != nil {
			continue
		}
		runs = append(runs, run)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].ID > runs[j].ID })
	return runs, nil
}

func Resume(root string, runID string, prompt string, model string, dryRun bool) (Run, error) {
	run, err := LoadRun(root, runID)
	if err != nil {
		return Run{}, err
	}
	adapter, err := provider.Get(run.Provider)
	if err != nil {
		return Run{}, err
	}
	req := provider.TaskRequest{
		Root:           root,
		Prompt:         prompt,
		Mode:           run.Mode,
		Model:          model,
		RunDir:         runDir(root, run.ID),
		LastMessageOut: filepath.Join(runDir(root, run.ID), "provider_last_message.md"),
		TranscriptOut:  filepath.Join(runDir(root, run.ID), "provider_transcript.jsonl"),
		DryRun:         dryRun,
		Timeout:        2 * time.Minute,
	}
	result, err := adapter.ResumeSession(context.Background(), run.ProviderSessionID, req)
	run.ProviderResult = &result
	run.UpdatedAt = nowUTC()
	if result.SessionID != "" {
		run.ProviderSessionID = result.SessionID
	}
	if err != nil {
		run.Status = "failed"
		run.CurrentState = StateFailed
		run.Metadata["resume_error"] = err.Error()
	} else {
		run.Status = "done"
		run.CurrentState = StateDone
	}
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "resume",
		Message:   "provider resume executed",
		Data: map[string]any{
			"session_id": result.SessionID,
			"exit_code":  result.ExitCode,
			"dry_run":    dryRun,
		},
	})
	if result.StdoutPath != "" {
		run.Artifacts["provider_transcript.jsonl"] = result.StdoutPath
	}
	if result.StderrPath != "" {
		run.Artifacts["provider_stderr.log"] = result.StderrPath
	}
	if req.LastMessageOut != "" {
		if _, statErr := os.Stat(req.LastMessageOut); statErr == nil {
			run.Artifacts["provider_last_message.md"] = req.LastMessageOut
		}
	}
	_ = saveRun(root, &run)
	return run, err
}

func updateMetrics(root string, run Run) {
	path := project.ProjectFile(root, "evals", "metrics.json")
	metrics := map[string]map[string]int{}
	if err := project.ReadJSON(path, &metrics); err != nil || metrics == nil {
		metrics = map[string]map[string]int{}
	}
	if _, ok := metrics[run.Mode]; !ok {
		metrics[run.Mode] = defaultMetrics()
	}
	for key, value := range defaultMetrics() {
		if _, ok := metrics[run.Mode][key]; !ok {
			metrics[run.Mode][key] = value
		}
	}
	metrics[run.Mode]["runs"]++
	switch run.Status {
	case "done":
		metrics[run.Mode]["completed"]++
	case "blocked":
		metrics[run.Mode]["blocked"]++
	case "failed":
		metrics[run.Mode]["failed"]++
	}
	if run.Artifacts["docs_delta.md"] != "" {
		metrics[run.Mode]["docs_reports"]++
	}
	if run.Metadata["evidence_warning"] == "true" {
		metrics[run.Mode]["evidence_warnings"]++
	}
	_ = project.WriteJSON(path, metrics)
}

func newRun(command string, opts TaskOptions) Run {
	now := time.Now().UTC()
	id := fmt.Sprintf("%s-%09d", now.Format("20060102T150405Z"), now.Nanosecond())
	return Run{
		ID:           id,
		Command:      command,
		Task:         opts.Task,
		Mode:         opts.Mode,
		Provider:     opts.Provider,
		Status:       "running",
		CurrentState: StateInitialized,
		StartedAt:    nowUTC(),
		UpdatedAt:    nowUTC(),
		DryRun:       opts.DryRun,
		Artifacts:    map[string]string{},
		Metadata:     map[string]string{},
		Transitions:  []Transition{{State: StateInitialized, Timestamp: nowUTC(), Notes: "run created"}},
	}
}

func saveRun(root string, run *Run) error {
	dir, err := project.EnsureRunDir(root, run.ID)
	if err != nil {
		return err
	}
	run.UpdatedAt = nowUTC()
	return project.WriteJSON(filepath.Join(dir, "run.json"), run)
}

func transition(root string, run *Run, state State, notes string) error {
	run.CurrentState = state
	run.Transitions = append(run.Transitions, Transition{State: state, Timestamp: nowUTC(), Notes: notes})
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "transition",
		Message:   notes,
		Data: map[string]any{
			"state": string(state),
		},
	})
	return saveRun(root, run)
}

func finishRun(root string, run *Run) error {
	run.CurrentState = StateDone
	run.Status = "done"
	run.Transitions = append(run.Transitions, Transition{State: StateDone, Timestamp: nowUTC(), Notes: "run completed"})
	return saveRun(root, run)
}

func failRun(root string, run *Run, err error) error {
	run.CurrentState = StateFailed
	run.Status = "failed"
	run.Metadata["error"] = err.Error()
	run.Transitions = append(run.Transitions, Transition{State: StateFailed, Timestamp: nowUTC(), Notes: err.Error()})
	_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_failed", Message: err.Error()})
	return saveRun(root, run)
}

func formatBudgetBreakdown(breakdown map[string]int) string {
	if len(breakdown) == 0 {
		return ""
	}
	keys := make([]string, 0, len(breakdown))
	for key := range breakdown {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, breakdown[key]))
	}
	return strings.Join(parts, ",")
}

func resolveRunEnvironment(root string) (presets.EnvironmentResolution, error) {
	return presets.ResolveEnvironment(root, "", nil)
}

func writeEnvironmentArtifacts(root string, run *Run, resolution presets.EnvironmentResolution) error {
	dir := runDir(root, run.ID)
	jsonPath := filepath.Join(dir, "environment_resolution.json")
	mdPath := filepath.Join(dir, "environment_resolution.md")
	if err := project.WriteJSON(jsonPath, resolution); err != nil {
		return err
	}
	if err := project.WriteString(mdPath, presets.RenderEnvironmentResolutionMarkdown(resolution)); err != nil {
		return err
	}
	run.Artifacts["environment_resolution.json"] = jsonPath
	run.Artifacts["environment_resolution.md"] = mdPath
	run.Metadata["environment_runtime_ceiling"] = resolution.EffectiveRuntimeCeiling
	run.Metadata["environment_budget_profile"] = resolution.EffectiveBudgetProfile
	run.Metadata["environment_layer_count"] = fmt.Sprintf("%d", len(resolution.Layers))
	run.Metadata["environment_hook_count"] = fmt.Sprintf("%d", len(resolution.HookRegistry))
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "environment_resolution",
		Message:   "environment resolution written",
		Data: map[string]any{
			"layers":                len(resolution.Layers),
			"hook_count":            len(resolution.HookRegistry),
			"runtime_ceiling":       resolution.EffectiveRuntimeCeiling,
			"budget_profile":        resolution.EffectiveBudgetProfile,
			"environment_conflicts": resolution.EnvironmentConflicts,
		},
	})
	return saveRun(root, run)
}

func writeMemoryPolicyArtifacts(root string, run *Run, resolution presets.EnvironmentResolution) (presets.MemoryPolicy, error) {
	policy := presets.BuildMemoryPolicy(resolution, run.ID)
	dir := runDir(root, run.ID)
	jsonPath := filepath.Join(dir, "memory_policy.json")
	mdPath := filepath.Join(dir, "memory_policy.md")
	if err := project.WriteJSON(jsonPath, policy); err != nil {
		return presets.MemoryPolicy{}, err
	}
	if err := project.WriteString(mdPath, presets.RenderMemoryPolicyMarkdown(policy)); err != nil {
		return presets.MemoryPolicy{}, err
	}
	run.Artifacts["memory_policy.json"] = jsonPath
	run.Artifacts["memory_policy.md"] = mdPath
	run.Metadata["memory_allowed_scope_count"] = fmt.Sprintf("%d", len(policy.AllowedScopes))
	run.Metadata["memory_system_writable"] = fmt.Sprintf("%t", policy.SystemWritable)
	return policy, saveRun(root, run)
}

func runHookLifecycle(root string, run *Run, resolution presets.EnvironmentResolution, memoryPolicy presets.MemoryPolicy, lifecycle string, opts TaskOptions) error {
	summary, err := presets.ExecuteHooks(resolution, presets.HookRunOptions{
		RunID:               run.ID,
		RunDir:              runDir(root, run.ID),
		Lifecycle:           lifecycle,
		ApproveRisky:        opts.ApproveRisky,
		DryRun:              opts.DryRun,
		WorkspaceRoot:       root,
		AllowedMemoryScopes: append([]string{}, memoryPolicy.AllowedScopes...),
	})
	run.Artifacts["hook_execution.json"] = filepath.Join(runDir(root, run.ID), "hook_execution.json")
	run.Artifacts["hook_execution.md"] = filepath.Join(runDir(root, run.ID), "hook_execution.md")
	run.Artifacts["hook_memory_events.jsonl"] = filepath.Join(runDir(root, run.ID), "hook_memory_events.jsonl")
	if len(summary.Executions) > 0 {
		run.Metadata["hook_last_lifecycle"] = lifecycle
		run.Metadata["hook_execution_count"] = fmt.Sprintf("%d", len(summary.Executions))
		_ = appendTrace(root, run.ID, TraceEvent{
			Timestamp: nowUTC(),
			Type:      "hook_execution",
			Message:   "hook lifecycle processed",
			Data: map[string]any{
				"lifecycle": lifecycle,
				"count":     len(summary.Executions),
			},
		})
		_ = saveRun(root, run)
	}
	if err == nil {
		return nil
	}
	if hookErr, ok := err.(presets.HookExecutionError); ok && hookErr.RequiresApproval {
		_ = transition(root, run, StateBlocked, hookErr.Message)
		run.Status = "blocked"
		run.Metadata["blocked_reason"] = "hook_approval_required"
		_ = saveRun(root, run)
		return nil
	}
	_ = failRun(root, run, err)
	return err
}

func collectContext(root string, task string, modeName string) (indexer.Result, []memory.Item, contextpack.Pack, contexttool.AssembleResult, ContextSelection, error) {
	idx, err := contexttool.BuildIndex(root)
	if err != nil {
		return indexer.Result{}, nil, contextpack.Pack{}, contexttool.AssembleResult{}, ContextSelection{}, err
	}
	_ = indexer.WriteIndividual(root, idx)
	_ = indexer.Save(root, idx)
	items, err := memory.Load(root)
	if err != nil {
		return indexer.Result{}, nil, contextpack.Pack{}, contexttool.AssembleResult{}, ContextSelection{}, err
	}
	modeDef := mode.ByName(modeName)
	arcPack := contextpack.Build(root, task, modeDef, idx, items)
	ctxResult, err := contexttool.Assemble(root, task)
	if err != nil {
		return indexer.Result{}, nil, contextpack.Pack{}, contexttool.AssembleResult{}, ContextSelection{}, err
	}
	selection := chooseContextPack(arcPack, ctxResult)
	return idx, items, selection.Selected, ctxResult, selection, nil
}

func writeContextArtifacts(root string, run *Run, idx indexer.Result, items []memory.Item, pack contextpack.Pack, ctxResult contexttool.AssembleResult, selection ContextSelection) error {
	dir := runDir(root, run.ID)
	if err := project.WriteJSON(filepath.Join(dir, "index_snapshot.json"), idx); err != nil {
		return err
	}
	if err := project.WriteJSON(filepath.Join(dir, "memory_snapshot.json"), items); err != nil {
		return err
	}
	if err := project.WriteJSON(filepath.Join(dir, "context_pack.json"), pack); err != nil {
		return err
	}
	if err := project.WriteString(filepath.Join(dir, "context_pack.md"), contextpack.Markdown(pack)); err != nil {
		return err
	}
	if err := project.WriteJSON(filepath.Join(dir, "arc_context_pack.json"), selection.ArcPack); err != nil {
		return err
	}
	if err := project.WriteString(filepath.Join(dir, "arc_context_pack.md"), contextpack.Markdown(selection.ArcPack)); err != nil {
		return err
	}
	if err := project.WriteJSON(filepath.Join(dir, "ctx_context_pack.json"), ctxResult.Pack); err != nil {
		return err
	}
	if err := project.WriteString(filepath.Join(dir, "ctx_context_pack.md"), contextpack.Markdown(ctxResult.Pack)); err != nil {
		return err
	}
	if err := project.WriteJSON(filepath.Join(dir, "ctx_context_metadata.json"), map[string]any{
		"output_dir":                  ctxResult.OutputDir,
		"built_index":                 ctxResult.BuiltIndex,
		"matched_terms":               ctxResult.MatchedTerms,
		"quality_score":               ctxResult.QualityScore,
		"term_coverage":               ctxResult.TermCoverage,
		"matched_sections":            ctxResult.MatchedSections,
		"memory_match_count":          ctxResult.MemoryMatchCount,
		"matched_memory_ids":          ctxResult.MatchedMemoryIDs,
		"memory_boost":                ctxResult.MemoryBoost,
		"memory_trust_bonus":          ctxResult.MemoryTrustBonus,
		"memory_recency_bonus":        ctxResult.MemoryRecencyBonus,
		"source_kinds":                ctxResult.SourceKinds,
		"source_diversity":            ctxResult.SourceDiversity,
		"diversity_bonus":             ctxResult.DiversityBonus,
		"doc_family_diversity":        ctxResult.DocFamilyDiversity,
		"code_family_diversity":       ctxResult.CodeFamilyDiversity,
		"doc_cluster_diversity":       ctxResult.DocClusterDiversity,
		"code_cluster_diversity":      ctxResult.CodeClusterDiversity,
		"doc_dominant_cluster_share":  ctxResult.DocDominantClusterShare,
		"code_dominant_cluster_share": ctxResult.CodeDominantClusterShare,
		"section_provenance":          ctxResult.SectionProvenance,
		"accounting":                  ctxResult.Accounting,
		"reuse":                       ctxResult.Reuse,
		"pack_json":                   ctxResult.PackJSONPath,
		"pack_md":                     ctxResult.PackMDPath,
		"metadata":                    ctxResult.MetadataPath,
	}); err != nil {
		return err
	}
	if err := project.WriteJSON(filepath.Join(dir, "context_selection.json"), selection); err != nil {
		return err
	}
	run.Artifacts["context_pack.md"] = filepath.Join(dir, "context_pack.md")
	run.Artifacts["context_pack.json"] = filepath.Join(dir, "context_pack.json")
	run.Artifacts["arc_context_pack.md"] = filepath.Join(dir, "arc_context_pack.md")
	run.Artifacts["arc_context_pack.json"] = filepath.Join(dir, "arc_context_pack.json")
	run.Artifacts["ctx_context_pack.md"] = filepath.Join(dir, "ctx_context_pack.md")
	run.Artifacts["ctx_context_pack.json"] = filepath.Join(dir, "ctx_context_pack.json")
	run.Artifacts["ctx_context_metadata.json"] = filepath.Join(dir, "ctx_context_metadata.json")
	run.Artifacts["context_selection.json"] = filepath.Join(dir, "context_selection.json")
	run.Metadata["context_source"] = selection.SelectedSource
	run.Metadata["context_arc_tokens"] = fmt.Sprintf("%d", selection.ArcTokens)
	run.Metadata["context_ctx_tokens"] = fmt.Sprintf("%d", selection.CtxTokens)
	run.Metadata["context_token_reduction"] = fmt.Sprintf("%d", selection.TokenReduction)
	run.Metadata["context_arc_quality"] = fmt.Sprintf("%d", selection.ArcQuality)
	run.Metadata["context_ctx_quality"] = fmt.Sprintf("%d", selection.CtxQuality)
	run.Metadata["context_ctx_memory_matches"] = fmt.Sprintf("%d", selection.CtxMemoryMatches)
	run.Metadata["context_ctx_memory_boost"] = fmt.Sprintf("%d", selection.CtxMemoryBoost)
	run.Metadata["context_ctx_memory_trust_bonus"] = fmt.Sprintf("%d", selection.CtxMemoryTrustBonus)
	run.Metadata["context_ctx_memory_recency_bonus"] = fmt.Sprintf("%d", selection.CtxMemoryRecencyBonus)
	run.Metadata["context_ctx_source_diversity"] = fmt.Sprintf("%d", selection.CtxSourceDiversity)
	run.Metadata["context_ctx_diversity_bonus"] = fmt.Sprintf("%d", selection.CtxDiversityBonus)
	run.Metadata["context_ctx_doc_family_diversity"] = fmt.Sprintf("%d", selection.CtxDocFamilyDiversity)
	run.Metadata["context_ctx_code_family_diversity"] = fmt.Sprintf("%d", selection.CtxCodeFamilyDiversity)
	run.Metadata["context_ctx_doc_cluster_diversity"] = fmt.Sprintf("%d", selection.CtxDocClusterDiversity)
	run.Metadata["context_ctx_code_cluster_diversity"] = fmt.Sprintf("%d", selection.CtxCodeClusterDiversity)
	run.Metadata["context_ctx_doc_dominant_cluster_share"] = fmt.Sprintf("%d", selection.CtxDocDominantClusterShare)
	run.Metadata["context_ctx_code_dominant_cluster_share"] = fmt.Sprintf("%d", selection.CtxCodeDominantClusterShare)
	run.Metadata["context_ctx_candidate_total"] = fmt.Sprintf("%d", ctxResult.Accounting.CandidateTotal)
	run.Metadata["context_ctx_selected_total"] = fmt.Sprintf("%d", ctxResult.Accounting.SelectedTotal)
	run.Metadata["context_ctx_index_source"] = ctxResult.Reuse.IndexSource
	run.Metadata["context_ctx_reused_artifact_count"] = fmt.Sprintf("%d", ctxResult.Reuse.ReusedArtifactCount)
	run.Metadata["context_selection_reason"] = selection.SelectionReason
	return saveRun(root, run)
}

func chooseContextPack(arcPack contextpack.Pack, ctxResult contexttool.AssembleResult) ContextSelection {
	ctxPack := ctxResult.Pack
	selected := arcPack
	source := "arc"
	reason := "arc_default"
	arcQuality := estimatePackQuality(arcPack)
	ctxQuality := ctxResult.QualityScore
	if ctxQuality == 0 {
		ctxQuality = estimatePackQuality(ctxPack)
	}
	if ctxPack.ApproxTokens > 0 && arcPack.ApproxTokens == 0 {
		selected = ctxPack
		source = "ctx"
		reason = "ctx_only_non_zero"
	} else if ctxPack.ApproxTokens > 0 && ctxPack.ApproxTokens <= arcPack.ApproxTokens && ctxQuality >= arcQuality {
		selected = ctxPack
		source = "ctx"
		reason = "ctx_smaller_or_equal_and_quality_not_worse"
	} else if ctxPack.ApproxTokens > 0 && withinPercent(ctxPack.ApproxTokens, arcPack.ApproxTokens, 15) && ctxQuality > arcQuality {
		selected = ctxPack
		source = "ctx"
		reason = "ctx_higher_quality_within_token_window"
	} else if ctxPack.ApproxTokens > 0 && ctxResult.MemoryMatchCount > 0 && withinPercent(ctxPack.ApproxTokens, arcPack.ApproxTokens, 25) && ctxQuality+ctxResult.MemoryBoost >= arcQuality {
		selected = ctxPack
		source = "ctx"
		reason = "ctx_memory_match_within_extended_token_window"
	} else if ctxPack.ApproxTokens > 0 &&
		ctxResult.DocClusterDiversity >= 2 &&
		ctxResult.CodeClusterDiversity >= 2 &&
		ctxResult.DocDominantClusterShare <= 50 &&
		ctxResult.CodeDominantClusterShare <= 50 &&
		withinPercent(ctxPack.ApproxTokens, arcPack.ApproxTokens, 20) &&
		ctxQuality >= arcQuality {
		selected = ctxPack
		source = "ctx"
		reason = "ctx_cluster_balanced_within_extended_token_window"
	} else if ctxPack.ApproxTokens > 0 && ctxResult.SourceDiversity >= 3 && withinPercent(ctxPack.ApproxTokens, arcPack.ApproxTokens, 20) && ctxQuality+ctxResult.DiversityBonus >= arcQuality {
		selected = ctxPack
		source = "ctx"
		reason = "ctx_diverse_sources_within_extended_token_window"
	} else if arcPack.ApproxTokens > 0 {
		reason = "arc_kept_for_size_or_quality"
	}
	return ContextSelection{
		SelectedSource:              source,
		SelectionReason:             reason,
		Selected:                    selected,
		ArcPack:                     arcPack,
		CtxPack:                     ctxPack,
		ArcTokens:                   arcPack.ApproxTokens,
		CtxTokens:                   ctxPack.ApproxTokens,
		TokenReduction:              arcPack.ApproxTokens - ctxPack.ApproxTokens,
		ArcQuality:                  arcQuality,
		CtxQuality:                  ctxQuality,
		CtxMemoryMatches:            ctxResult.MemoryMatchCount,
		CtxMemoryBoost:              ctxResult.MemoryBoost,
		CtxMemoryTrustBonus:         ctxResult.MemoryTrustBonus,
		CtxMemoryRecencyBonus:       ctxResult.MemoryRecencyBonus,
		CtxSourceDiversity:          ctxResult.SourceDiversity,
		CtxDiversityBonus:           ctxResult.DiversityBonus,
		CtxDocFamilyDiversity:       ctxResult.DocFamilyDiversity,
		CtxCodeFamilyDiversity:      ctxResult.CodeFamilyDiversity,
		CtxDocClusterDiversity:      ctxResult.DocClusterDiversity,
		CtxCodeClusterDiversity:     ctxResult.CodeClusterDiversity,
		CtxDocDominantClusterShare:  ctxResult.DocDominantClusterShare,
		CtxCodeDominantClusterShare: ctxResult.CodeDominantClusterShare,
	}
}

func estimatePackQuality(pack contextpack.Pack) int {
	terms := selectionTerms(pack.Task)
	if len(terms) == 0 {
		return len(pack.Sections) * 10
	}
	covered := 0
	sectionMatches := 0
	seenSections := map[string]bool{}
	for _, term := range terms {
		termLower := strings.ToLower(term)
		for _, section := range pack.Sections {
			blob := strings.ToLower(section.Title + "\n" + section.Source + "\n" + section.Content)
			if strings.Contains(blob, termLower) {
				if !seenSections[section.Title] {
					seenSections[section.Title] = true
					sectionMatches++
				}
				covered++
				break
			}
		}
	}
	return covered*100 + sectionMatches*12 + qualitySectionBonus(pack)
}

func qualitySectionBonus(pack contextpack.Pack) int {
	bonus := 0
	for _, section := range pack.Sections {
		switch section.Title {
		case "Relevant Docs":
			bonus += 15
		case "Relevant Code Surfaces":
			bonus += 20
		case "Relevant Memory":
			bonus += 10
		case "Query Signals":
			bonus += 10
		}
	}
	return bonus
}

func selectionTerms(task string) []string {
	stop := map[string]struct{}{
		"what": {}, "with": {}, "that": {}, "this": {}, "from": {}, "into": {}, "and": {},
		"для": {}, "как": {}, "что": {}, "это": {}, "или": {}, "над": {}, "под": {}, "при": {},
	}
	seen := map[string]struct{}{}
	out := []string{}
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		token := strings.ToLower(string(current))
		current = current[:0]
		if len([]rune(token)) < 3 {
			return
		}
		if _, ok := stop[token]; ok {
			return
		}
		if _, ok := seen[token]; ok {
			return
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	for _, r := range task {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			current = append(current, r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func withinPercent(a int, b int, pct int) bool {
	if a <= 0 || b <= 0 {
		return false
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff*100 <= b*pct
}

func writePlanningArtifacts(root string, run *Run, task string, pack contextpack.Pack, memoryPolicy presets.MemoryPolicy) error {
	modeDef := mode.ByName(run.Mode)
	switch run.Mode {
	case "study":
		writeArtifact(root, run, "learning_goal.md", "# Learning Goal\n\n"+task+"\n")
		writeArtifact(root, run, "current_understanding.md", "# Current Understanding\n\n[HUMAN_PROVIDED] Describe what you already understand before asking for the full solution.\n")
		writeArtifact(root, run, "knowledge_gaps.md", "# Knowledge Gaps\n\n- `UNKNOWN` Missing evidence or unclear concepts should be listed here.\n")
		writeArtifact(root, run, "challenge_log.md", "# Challenge Log\n\n- Clarifying questions and challenges will be recorded here.\n")
		writeArtifact(root, run, "practice_tasks.md", "# Practice Tasks\n\n- Derive 2-3 short exercises from the current topic.\n")
	case "hero":
		writeArtifact(root, run, "ticket_spec.md", heroSpec("Ticket Spec", task, pack))
		writeArtifact(root, run, "business_spec.md", heroSpec("Business Spec", task, pack))
		writeArtifact(root, run, "tech_spec.md", heroSpec("Tech Spec", task, pack))
		writeArtifact(root, run, "unknowns.md", "# Unknowns\n\n- `UNKNOWN` Fill with missing evidence discovered during planning.\n")
		writeQuestionBundle(root, run, []string{"What repo-specific rules or architectural constraints are still missing from AGENTS/docs/maps?"}, memoryPolicy)
	default:
		writeArtifact(root, run, "task_map.md", "# Task Map\n\n"+task+"\n")
		writeArtifact(root, run, "system_flow.md", "# System Flow\n\n- Summarize impacted system surfaces from the context pack.\n")
		writeArtifact(root, run, "solution_options.md", "# Solution Options\n\n- Option 1: minimal implementation slice\n- Option 2: deeper refactor\n- Option 3: defer if blocked by context\n")
		writeArtifact(root, run, "unknowns.md", "# Unknowns\n\n- List evidence gaps and assumptions.\n")
		writeArtifact(root, run, "validation_checklist.md", "# Validation Checklist\n\n- Build\n- Tests\n- Docs update\n")
	}

	_ = memory.AddAllowed(root, memory.Item{
		ID:             "run-" + run.ID,
		Scope:          "runs/" + run.ID,
		Kind:           "fact",
		Source:         "task pipeline",
		Confidence:     "high",
		CreatedAt:      nowUTC(),
		LastVerifiedAt: nowUTC(),
		Status:         "active",
		Tags:           []string{run.Mode, "run"},
		Summary:        fmt.Sprintf("Run %s planned task: %s", run.ID, task),
	}, memoryPolicy.AllowedScopes)

	run.Metadata["active_roles"] = strings.Join(modeDef.Roles, ",")
	run.Metadata["active_skills"] = strings.Join(skillsForMode(run.Mode), ",")
	_ = writeArtifact(root, run, "active_roles.md", markdownList("Active Roles", modeDef.Roles))
	_ = writeArtifact(root, run, "active_skills.md", markdownList("Active Skills", skillsForMode(run.Mode)))
	_ = saveRun(root, run)
	_ = project.WriteString(filepath.Join(runDir(root, run.ID), "mode_policy.md"), mode.Markdown(modeDef))
	run.Artifacts["mode_policy.md"] = filepath.Join(runDir(root, run.ID), "mode_policy.md")
	return saveRun(root, run)
}

func writeApprovalReport(root string, run *Run, assessment governance.Assessment, approved bool) error {
	var b strings.Builder
	b.WriteString("# Approval Report\n\n")
	b.WriteString(fmt.Sprintf("- risk_level: %s\n", assessment.RiskLevel))
	b.WriteString(fmt.Sprintf("- requires_approval: %t\n", assessment.RequiresApproval))
	b.WriteString(fmt.Sprintf("- approved_for_run: %t\n\n", approved))
	if len(assessment.Triggers) > 0 {
		b.WriteString("Triggers:\n")
		for _, trigger := range assessment.Triggers {
			b.WriteString("- " + trigger + "\n")
		}
		b.WriteString("\n")
	}
	if len(assessment.Gates) > 0 {
		b.WriteString("Gates:\n")
		for _, gate := range assessment.Gates {
			b.WriteString("- " + gate + "\n")
		}
	} else {
		b.WriteString("No approval gate was triggered.\n")
	}
	if err := writeArtifact(root, run, "approval_report.md", b.String()); err != nil {
		return err
	}
	return saveRun(root, run)
}

func writeImplementationLog(root string, run *Run, message string) error {
	path := filepath.Join(runDir(root, run.ID), "implementation_log.md")
	content := "# Implementation Log\n\n"
	content += "- started_at: " + run.StartedAt + "\n"
	content += "- provider: " + run.Provider + "\n"
	content += "- dry_run: " + fmt.Sprintf("%t", run.DryRun) + "\n\n"
	content += message + "\n"
	if err := project.WriteString(path, content); err != nil {
		return err
	}
	run.Artifacts["implementation_log.md"] = path
	_ = appendTrace(root, run.ID, TraceEvent{
		Timestamp: nowUTC(),
		Type:      "implementation_log",
		Message:   "implementation log updated",
		Data: map[string]any{
			"path": path,
		},
	})
	return saveRun(root, run)
}

func writeArtifact(root string, run *Run, name string, content string) error {
	path := filepath.Join(runDir(root, run.ID), name)
	if err := project.WriteString(path, content); err != nil {
		return err
	}
	run.Artifacts[name] = path
	return nil
}

func writeQuestionBundle(root string, run *Run, questions []string, memoryPolicy presets.MemoryPolicy) error {
	var b strings.Builder
	b.WriteString("# Question Bundle\n\n")
	for _, question := range questions {
		b.WriteString("- " + question + "\n")
	}
	if err := writeArtifact(root, run, "question_bundle.md", b.String()); err != nil {
		return err
	}
	return memory.AddAllowed(root, memory.Item{
		ID:             "question-" + run.ID,
		Scope:          "runs/" + run.ID,
		Kind:           "question",
		Source:         "task pipeline",
		Confidence:     "medium",
		CreatedAt:      nowUTC(),
		LastVerifiedAt: nowUTC(),
		Status:         "active",
		Tags:           []string{run.Mode, "question"},
		Summary:        strings.Join(questions, " "),
	}, memoryPolicy.AllowedScopes)
}

func heroSpec(title string, task string, pack contextpack.Pack) string {
	content := "# " + title + "\n\n"
	content += "Task: " + task + "\n\n"
	content += "Context summary:\n"
	for i, section := range pack.Sections {
		if i >= 6 {
			break
		}
		content += fmt.Sprintf("- %s (%s)\n", section.Title, section.Source)
	}
	return content
}

func skillsForMode(modeName string) []string {
	switch modeName {
	case "study":
		return []string{"teach-challenge", "build-visualizer", "compact-memory"}
	case "hero":
		return []string{"plan-task", "review-spec", "request-context", "implement-feature", "verify-changes", "review-code", "write-docs", "compact-memory"}
	default:
		return []string{"plan-task", "review-spec", "request-context", "verify-changes", "write-docs"}
	}
}

func markdownList(title string, items []string) string {
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	for _, item := range items {
		b.WriteString("- " + item + "\n")
	}
	return b.String()
}

func providerPrompt(task string, pack contextpack.Pack, modeName string) string {
	var b strings.Builder
	b.WriteString("You are running inside Agent Runtime CLI.\n")
	b.WriteString("Mode: " + modeName + "\n")
	b.WriteString("Task: " + task + "\n\n")
	b.WriteString("Follow the mode policy and anti-hallucination protocol.\n")
	b.WriteString("Use evidence labels when uncertainty matters.\n")
	b.WriteString("Do not invent missing repo facts.\n\n")
	b.WriteString("Context pack:\n\n")
	b.WriteString(contextpack.Markdown(pack))
	return b.String()
}

func writeEvidenceReport(root string, run *Run) (string, bool, error) {
	labels := []string{"CODE_VERIFIED", "DOC_VERIFIED", "COMMAND_VERIFIED", "HUMAN_PROVIDED", "INFERRED", "UNKNOWN"}
	counts := map[string]int{}
	scanned := []string{}

	entries, err := os.ReadDir(runDir(root, run.ID))
	if err != nil {
		return "", false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(runDir(root, run.ID), entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		scanned = append(scanned, entry.Name())
		text := string(content)
		for _, label := range labels {
			counts[label] += strings.Count(text, label)
		}
	}

	verified := counts["CODE_VERIFIED"] + counts["DOC_VERIFIED"] + counts["COMMAND_VERIFIED"] + counts["HUMAN_PROVIDED"]
	warning := verified == 0 || (counts["INFERRED"] > 0 && verified == 0)

	var b strings.Builder
	b.WriteString("# Anti-Hallucination Report\n\n")
	b.WriteString("Scanned markdown artifacts:\n")
	for _, name := range scanned {
		b.WriteString("- " + name + "\n")
	}
	if len(scanned) == 0 {
		b.WriteString("- none\n")
	}
	b.WriteString("\nEvidence labels:\n")
	for _, label := range labels {
		b.WriteString(fmt.Sprintf("- %s: %d\n", label, counts[label]))
	}
	b.WriteString("\n")
	if warning {
		b.WriteString("Status: warning\n")
		b.WriteString("- Evidence labels are currently weak or absent; treat unsupported statements as `UNKNOWN` or `INFERRED` until verified.\n")
	} else {
		b.WriteString("Status: ok\n")
	}

	if err := writeArtifact(root, run, "anti_hallucination_report.md", b.String()); err != nil {
		return "", false, err
	}
	run.Metadata["evidence_warning"] = fmt.Sprintf("%t", warning)
	run.Metadata["evidence_verified_count"] = fmt.Sprintf("%d", verified)
	_ = saveRun(root, run)
	return filepath.Join(runDir(root, run.ID), "anti_hallucination_report.md"), warning, nil
}

func runDefaultChecks(root string) []string {
	results := []string{}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		cmd := exec.Command("go", "test", "./...")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(root, ".arc", "cache", "go-build"))
		if output, err := cmd.CombinedOutput(); err != nil {
			results = append(results, "go test ./... failed: "+shorten(string(output), 400))
		} else {
			results = append(results, "go test ./... passed")
		}
	} else {
		if _, err := os.Stat(filepath.Join(root, "package.json")); err == nil {
			if _, err := os.Stat(filepath.Join(root, "node_modules")); err == nil {
				cmd := exec.Command("npm", "test", "--", "--runInBand")
				cmd.Dir = root
				if output, err := cmd.CombinedOutput(); err != nil {
					results = append(results, "npm test failed: "+shorten(string(output), 400))
				} else {
					results = append(results, "npm test passed")
				}
			} else {
				results = append(results, "package.json detected but node_modules is missing; skipped npm test")
			}
		}
		if _, err := os.Stat(filepath.Join(root, "pyproject.toml")); err == nil {
			if _, err := exec.LookPath("pytest"); err == nil {
				cmd := exec.Command("pytest", "-q")
				cmd.Dir = root
				if output, err := cmd.CombinedOutput(); err != nil {
					results = append(results, "pytest failed: "+shorten(string(output), 400))
				} else {
					results = append(results, "pytest passed")
				}
			} else {
				results = append(results, "pyproject.toml detected but pytest is unavailable; skipped pytest")
			}
		}
		if _, err := os.Stat(filepath.Join(root, "Makefile")); err == nil {
			cmd := exec.Command("make", "-n", "test")
			cmd.Dir = root
			if output, err := cmd.CombinedOutput(); err == nil {
				results = append(results, "make test target detected: "+shorten(string(output), 240))
			}
		}
		if len(results) == 0 {
			results = append(results, "No built-in check detected for this repository type.")
		}
	}
	return results
}

func runDir(root string, runID string) string {
	return project.ProjectFile(root, "runs", runID)
}

func shorten(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func defaultMetrics() map[string]int {
	return map[string]int{
		"runs":              0,
		"completed":         0,
		"blocked":           0,
		"failed":            0,
		"docs_reports":      0,
		"evidence_warnings": 0,
	}
}

func appendTrace(root string, runID string, event TraceEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	tracePath := filepath.Join(runDir(root, runID), "trace.jsonl")
	file, err := os.OpenFile(tracePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(data, '\n'))
	return err
}

func writePostRunArtifacts(root string, run *Run) error {
	beforePath := filepath.Join(runDir(root, run.ID), "index_snapshot.json")
	var before indexer.Result
	if err := project.ReadJSON(beforePath, &before); err != nil {
		return err
	}
	after, err := indexer.Build(root)
	if err != nil {
		return err
	}
	afterPath := filepath.Join(runDir(root, run.ID), "index_post_run.json")
	if err := project.WriteJSON(afterPath, after); err != nil {
		return err
	}
	changed := diffChangedFiles(before, after)
	changedPath := filepath.Join(runDir(root, run.ID), "changed_files.json")
	if err := project.WriteJSON(changedPath, changed); err != nil {
		return err
	}
	markdownPath := filepath.Join(runDir(root, run.ID), "changed_files.md")
	var b strings.Builder
	b.WriteString("# Changed Files\n\n")
	if len(changed) == 0 {
		b.WriteString("No file changes detected between pre-run and post-run snapshots.\n")
	} else {
		for _, path := range changed {
			b.WriteString("- " + path + "\n")
		}
	}
	if err := project.WriteString(markdownPath, b.String()); err != nil {
		return err
	}
	run.Artifacts["index_post_run.json"] = afterPath
	run.Artifacts["changed_files.json"] = changedPath
	run.Artifacts["changed_files.md"] = markdownPath
	if patchPath, err := writeWorkspaceDiffArtifact(root, run, changed); err == nil && patchPath != "" {
		run.Artifacts["workspace_diff.patch"] = patchPath
	}
	return saveRun(root, run)
}

func diffChangedFiles(before indexer.Result, after indexer.Result) []string {
	beforeMap := map[string]string{}
	for _, item := range before.Files {
		beforeMap[item.Path] = fmt.Sprintf("%d:%s", item.Size, item.ModTime)
	}
	afterMap := map[string]string{}
	for _, item := range after.Files {
		afterMap[item.Path] = fmt.Sprintf("%d:%s", item.Size, item.ModTime)
	}
	seen := map[string]bool{}
	changed := []string{}
	for path, beforeSig := range beforeMap {
		afterSig, ok := afterMap[path]
		if !ok || afterSig != beforeSig {
			changed = append(changed, path)
		}
		seen[path] = true
	}
	for path := range afterMap {
		if !seen[path] {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}

func writeWorkspaceDiffArtifact(root string, run *Run, changed []string) (string, error) {
	if len(changed) == 0 {
		return "", nil
	}
	args := []string{"-C", root, "diff", "--no-ext-diff", "--binary", "--"}
	args = append(args, changed...)
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "", nil
	}
	patchPath := filepath.Join(runDir(root, run.ID), "workspace_diff.patch")
	if err := project.WriteString(patchPath, string(output)); err != nil {
		return "", err
	}
	return patchPath, nil
}

func applyDocsUpdates(root string) ([]string, error) {
	idx, err := indexer.Build(root)
	if err != nil {
		return nil, err
	}

	repoMap := "# REPO MAP\n\n"
	repoMap += fmt.Sprintf("- Files indexed: %d\n", len(idx.Files))
	repoMap += fmt.Sprintf("- Symbols indexed: %d\n", len(idx.Symbols))
	repoMap += fmt.Sprintf("- Dependencies indexed: %d\n\n", len(idx.Dependencies))
	repoMap += "Important code surfaces:\n"
	for i, symbol := range idx.Symbols {
		if i >= 20 {
			break
		}
		repoMap += fmt.Sprintf("- %s (%s) in %s:%d\n", symbol.Name, symbol.Kind, symbol.Path, symbol.Line)
	}

	docsMap := "# DOCS MAP\n\n"
	docsMap += fmt.Sprintf("- Docs indexed: %d\n\n", len(idx.Docs))
	docsMap += "Relevant docs:\n"
	for i, doc := range idx.Docs {
		if i >= 20 {
			break
		}
		docsMap += fmt.Sprintf("- %s: %s\n", doc.Path, doc.Title)
	}

	repoMapPath := project.ProjectFile(root, "maps", "REPO_MAP.md")
	docsMapPath := project.ProjectFile(root, "maps", "DOCS_MAP.md")
	cliMapPath := project.ProjectFile(root, "maps", "CLI_MAP.md")
	artifactsMapPath := project.ProjectFile(root, "maps", "ARTIFACTS_MAP.md")
	runtimeStatusPath := project.ProjectFile(root, "maps", "RUNTIME_STATUS.md")
	if err := project.WriteString(repoMapPath, repoMap); err != nil {
		return nil, err
	}
	if err := project.WriteString(docsMapPath, docsMap); err != nil {
		return nil, err
	}
	if err := project.WriteString(cliMapPath, renderCLIMap()); err != nil {
		return nil, err
	}
	if err := project.WriteString(artifactsMapPath, renderArtifactsMap()); err != nil {
		return nil, err
	}
	runtimeStatus, err := renderRuntimeStatus(root)
	if err != nil {
		return nil, err
	}
	if err := project.WriteString(runtimeStatusPath, runtimeStatus); err != nil {
		return nil, err
	}
	return []string{repoMapPath, docsMapPath, cliMapPath, artifactsMapPath, runtimeStatusPath}, nil
}

func renderCLIMap() string {
	commands := []string{
		"arc doctor",
		"arc init [--path DIR] [--provider codex,claude] [--mode study|work|hero]",
		"arc mode set <study|work|hero> [--path DIR]",
		"arc mode show [--path DIR]",
		"arc index build [--path DIR]",
		"arc index refresh [--path DIR]",
		"arc task plan [--path DIR] [--mode MODE] [--provider NAME] <task>",
		"arc task run [--path DIR] [--mode MODE] [--provider NAME] [--dry-run] [--run-checks] [--approve-risky] [--provider-timeout DURATION] <task>",
		"arc task verify [--path DIR] [--run-id ID] [--run-checks]",
		"arc task review [--path DIR] [--run-id ID]",
		"arc docs generate [--path DIR] [--run-id ID] [--apply]",
		"arc memory status [--path DIR]",
		"arc memory compact [--path DIR]",
		"arc questions show [--path DIR]",
		"arc run status [--path DIR] [--run-id ID]",
		"arc run resume [--path DIR] [--run-id ID] [--model MODEL] [--dry-run] [prompt]",
		"arc learn <topic>",
		"arc learn quiz [--path DIR]",
		"arc learn prove <claim>",
	}

	var b strings.Builder
	b.WriteString("# CLI MAP\n\n")
	b.WriteString("Supported command surface:\n")
	for _, command := range commands {
		b.WriteString("- `" + command + "`\n")
	}
	return b.String()
}

func renderArtifactsMap() string {
	modeNames := []string{"study", "work", "hero"}
	common := []string{
		"context_pack.md",
		"context_pack.json",
		"mode_policy.md",
		"active_roles.md",
		"active_skills.md",
		"implementation_log.md",
		"verification_report.md",
		"review_report.md",
		"anti_hallucination_report.md",
		"docs_delta.md",
		"trace.jsonl",
		"index_snapshot.json",
		"index_post_run.json",
		"changed_files.json",
		"changed_files.md",
	}
	providerArtifacts := []string{
		"provider_transcript.jsonl",
		"provider_stderr.log",
		"provider_last_message.md",
	}

	var b strings.Builder
	b.WriteString("# ARTIFACTS MAP\n\n")
	b.WriteString("Common run artifacts:\n")
	for _, item := range common {
		b.WriteString("- " + item + "\n")
	}
	b.WriteString("\nProvider artifacts:\n")
	for _, item := range providerArtifacts {
		b.WriteString("- " + item + "\n")
	}
	b.WriteString("\nMode-specific artifacts:\n")
	for _, modeName := range modeNames {
		def := mode.ByName(modeName)
		b.WriteString("\n## " + modeName + "\n")
		for _, item := range def.Artifacts {
			b.WriteString("- " + item + "\n")
		}
	}
	return b.String()
}

func renderRuntimeStatus(root string) (string, error) {
	metrics := map[string]map[string]int{}
	_ = project.ReadJSON(project.ProjectFile(root, "evals", "metrics.json"), &metrics)

	runSummary := "No runs found."
	if latest, err := LoadRun(root, ""); err == nil {
		runSummary = fmt.Sprintf("Latest run: %s (%s/%s)", latest.ID, latest.Status, latest.CurrentState)
	}

	names := []string{}
	for name := range provider.Registry() {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("# RUNTIME STATUS\n\n")
	b.WriteString(runSummary + "\n\n")
	b.WriteString("Provider status:\n")
	for _, name := range names {
		adapter, _ := provider.Get(name)
		status := adapter.CheckInstalled(context.Background())
		b.WriteString(fmt.Sprintf("- %s: installed=%t", status.Name, status.Installed))
		if status.BinaryPath != "" {
			b.WriteString(" path=" + status.BinaryPath)
		}
		b.WriteString("\n")
	}
	b.WriteString("\nMetrics:\n")
	for _, modeName := range []string{"study", "work", "hero"} {
		values := defaultMetrics()
		if existing, ok := metrics[modeName]; ok {
			for key := range values {
				if value, exists := existing[key]; exists {
					values[key] = value
				}
			}
		}
		b.WriteString(fmt.Sprintf("- %s: runs=%d completed=%d failed=%d blocked=%d docs_reports=%d evidence_warnings=%d\n",
			modeName,
			values["runs"],
			values["completed"],
			values["failed"],
			values["blocked"],
			values["docs_reports"],
			values["evidence_warnings"],
		))
	}
	return b.String(), nil
}
