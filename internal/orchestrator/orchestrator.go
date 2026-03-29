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

	"agent-os/internal/contextpack"
	"agent-os/internal/governance"
	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/mode"
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

func Plan(root string, opts TaskOptions) (Run, error) {
	run := newRun("task plan", opts)
	if err := saveRun(root, &run); err != nil {
		return Run{}, err
	}
	_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_created", Message: "planning run created"})

	if err := transition(root, &run, StateContextCollecting, "collecting project context"); err != nil {
		return Run{}, err
	}
	idx, items, pack, err := collectContext(root, opts.Task, opts.Mode)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}

	if err := writeContextArtifacts(root, &run, idx, items, pack); err != nil {
		return Run{}, err
	}
	if err := transition(root, &run, StatePlanning, "writing planning artifacts"); err != nil {
		return Run{}, err
	}

	if err := writePlanningArtifacts(root, &run, opts.Task, pack); err != nil {
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
	if err := saveRun(root, &run); err != nil {
		return Run{}, err
	}
	_ = appendTrace(root, run.ID, TraceEvent{Timestamp: nowUTC(), Type: "run_created", Message: "task run created"})

	if err := transition(root, &run, StateContextCollecting, "collecting project context"); err != nil {
		return Run{}, err
	}
	idx, items, pack, err := collectContext(root, opts.Task, opts.Mode)
	if err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
	}
	if err := writeContextArtifacts(root, &run, idx, items, pack); err != nil {
		return Run{}, err
	}

	if err := transition(root, &run, StatePlanning, "building specs and question bundle"); err != nil {
		return Run{}, err
	}
	if err := writePlanningArtifacts(root, &run, opts.Task, pack); err != nil {
		_ = failRun(root, &run, err)
		return Run{}, err
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
	if assessment.RequiresApproval && !opts.ApproveRisky && opts.UseProvider && !opts.DryRun {
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
	if err := writeImplementationLog(root, &run, "Implementation stage started."); err != nil {
		return Run{}, err
	}

	if opts.UseProvider && !opts.DryRun {
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
			_ = writeQuestionBundle(root, &run, []string{"Selected provider is unavailable. Install it or rerun with --dry-run."})
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

func collectContext(root string, task string, modeName string) (indexer.Result, []memory.Item, contextpack.Pack, error) {
	idx, err := indexer.Build(root)
	if err != nil {
		return indexer.Result{}, nil, contextpack.Pack{}, err
	}
	_ = indexer.WriteIndividual(root, idx)
	_ = indexer.Save(root, idx)

	items, err := memory.Load(root)
	if err != nil {
		return indexer.Result{}, nil, contextpack.Pack{}, err
	}
	modeDef := mode.ByName(modeName)
	pack := contextpack.Build(root, task, modeDef, idx, items)
	return idx, items, pack, nil
}

func writeContextArtifacts(root string, run *Run, idx indexer.Result, items []memory.Item, pack contextpack.Pack) error {
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
	run.Artifacts["context_pack.md"] = filepath.Join(dir, "context_pack.md")
	run.Artifacts["context_pack.json"] = filepath.Join(dir, "context_pack.json")
	return saveRun(root, run)
}

func writePlanningArtifacts(root string, run *Run, task string, pack contextpack.Pack) error {
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
		writeQuestionBundle(root, run, []string{"What repo-specific rules or architectural constraints are still missing from AGENTS/docs/maps?"})
	default:
		writeArtifact(root, run, "task_map.md", "# Task Map\n\n"+task+"\n")
		writeArtifact(root, run, "system_flow.md", "# System Flow\n\n- Summarize impacted system surfaces from the context pack.\n")
		writeArtifact(root, run, "solution_options.md", "# Solution Options\n\n- Option 1: minimal implementation slice\n- Option 2: deeper refactor\n- Option 3: defer if blocked by context\n")
		writeArtifact(root, run, "unknowns.md", "# Unknowns\n\n- List evidence gaps and assumptions.\n")
		writeArtifact(root, run, "validation_checklist.md", "# Validation Checklist\n\n- Build\n- Tests\n- Docs update\n")
	}

	_ = memory.Add(root, memory.Item{
		ID:             "run-" + run.ID,
		Scope:          "run",
		Kind:           "fact",
		Source:         "task pipeline",
		Confidence:     "high",
		CreatedAt:      nowUTC(),
		LastVerifiedAt: nowUTC(),
		Status:         "active",
		Tags:           []string{run.Mode, "run"},
		Summary:        fmt.Sprintf("Run %s planned task: %s", run.ID, task),
	})

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

func writeQuestionBundle(root string, run *Run, questions []string) error {
	var b strings.Builder
	b.WriteString("# Question Bundle\n\n")
	for _, question := range questions {
		b.WriteString("- " + question + "\n")
	}
	if err := writeArtifact(root, run, "question_bundle.md", b.String()); err != nil {
		return err
	}
	return memory.Add(root, memory.Item{
		ID:             "question-" + run.ID,
		Scope:          "run",
		Kind:           "question",
		Source:         "task pipeline",
		Confidence:     "medium",
		CreatedAt:      nowUTC(),
		LastVerifiedAt: nowUTC(),
		Status:         "active",
		Tags:           []string{run.Mode, "question"},
		Summary:        strings.Join(questions, " "),
	})
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
