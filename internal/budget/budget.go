package budget

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
	"agent-os/internal/provider"
)

type Mode string

const (
	ModeUltraSafe         Mode = "ultra_safe"
	ModeBalanced          Mode = "balanced"
	ModeDeepWork          Mode = "deep_work"
	ModeEmergencyLowLimit Mode = "emergency_low_limit"
)

type Classification string

const (
	ClassNoProvider      Classification = "no_provider"
	ClassLocalFirst      Classification = "local_first"
	ClassCheapProviderOK Classification = "cheap_provider_ok"
	ClassPremiumRequired Classification = "premium_required"
	ClassPremiumHighRisk Classification = "premium_high_risk"
)

type LowLimitState string

const (
	LowLimitNormal      LowLimitState = "normal"
	LowLimitWarning     LowLimitState = "warning"
	LowLimitConstrained LowLimitState = "constrained"
	LowLimitEmergency   LowLimitState = "emergency"
)

type Policy struct {
	Mode                              Mode          `json:"mode"`
	LowLimitState                     LowLimitState `json:"low_limit_state"`
	PreferLocal                       bool          `json:"prefer_local"`
	BlockPremiumHighRisk              bool          `json:"block_premium_high_risk"`
	BlockPremiumRequired              bool          `json:"block_premium_required"`
	RequireApprovalForPremiumHighRisk bool          `json:"require_approval_for_premium_high_risk"`
	RequireApprovalForPremiumRequired bool          `json:"require_approval_for_premium_required"`
	Notes                             []string      `json:"notes,omitempty"`
}

type PolicyOverride struct {
	Mode                              string   `json:"mode,omitempty"`
	LowLimitState                     string   `json:"low_limit_state,omitempty"`
	PreferLocal                       *bool    `json:"prefer_local,omitempty"`
	BlockPremiumHighRisk              *bool    `json:"block_premium_high_risk,omitempty"`
	BlockPremiumRequired              *bool    `json:"block_premium_required,omitempty"`
	RequireApprovalForPremiumHighRisk *bool    `json:"require_approval_for_premium_high_risk,omitempty"`
	RequireApprovalForPremiumRequired *bool    `json:"require_approval_for_premium_required,omitempty"`
	Notes                             []string `json:"notes,omitempty"`
}

type PolicyResolution struct {
	RequestedBudgetMode      string         `json:"requested_budget_mode,omitempty"`
	EnvironmentBudgetProfile string         `json:"environment_budget_profile,omitempty"`
	ProjectOverridePath      string         `json:"project_override_path,omitempty"`
	ProjectOverridePresent   bool           `json:"project_override_present"`
	SessionOverridePath      string         `json:"session_override_path,omitempty"`
	SessionOverridePresent   bool           `json:"session_override_present"`
	EffectiveMode            Mode           `json:"effective_mode"`
	EffectiveModeSource      string         `json:"effective_mode_source"`
	AppliedOverrideSources   []string       `json:"applied_override_sources,omitempty"`
	ProjectOverride          PolicyOverride `json:"project_override,omitempty"`
	SessionOverride          PolicyOverride `json:"session_override,omitempty"`
	EffectivePolicy          Policy         `json:"effective_policy"`
}

type Assessment struct {
	Mode             Mode           `json:"mode"`
	LowLimitState    LowLimitState  `json:"low_limit_state"`
	Classification   Classification `json:"classification"`
	Reasoning        []string       `json:"reasoning,omitempty"`
	MatchedSignals   []string       `json:"matched_signals,omitempty"`
	SignalBreakdown  map[string]int `json:"signal_breakdown,omitempty"`
	Confidence       int            `json:"confidence"`
	ConfidenceTier   string         `json:"confidence_tier,omitempty"`
	RequiresApproval bool           `json:"requires_approval"`
	ShouldBlock      bool           `json:"should_block"`
	RouteLocally     bool           `json:"route_locally"`
	RoutingReason    string         `json:"routing_reason,omitempty"`
	RoutingTrigger   string         `json:"routing_trigger,omitempty"`
}

type UsageEvent struct {
	Timestamp                    string         `json:"timestamp"`
	RunID                        string         `json:"run_id"`
	ProjectRoot                  string         `json:"project_root,omitempty"`
	Task                         string         `json:"task"`
	Provider                     string         `json:"provider"`
	ProviderModel                string         `json:"provider_model,omitempty"`
	ProviderSessionID            string         `json:"provider_session_id,omitempty"`
	BudgetMode                   Mode           `json:"budget_mode"`
	BudgetModeSource             string         `json:"budget_mode_source,omitempty"`
	EnvironmentBudgetProfile     string         `json:"environment_budget_profile,omitempty"`
	LowLimitState                LowLimitState  `json:"low_limit_state"`
	Classification               Classification `json:"classification"`
	Confidence                   int            `json:"confidence"`
	ConfidenceTier               string         `json:"confidence_tier,omitempty"`
	Status                       string         `json:"status"`
	UsedProvider                 bool           `json:"used_provider"`
	RouteLocally                 bool           `json:"route_locally,omitempty"`
	DryRun                       bool           `json:"dry_run"`
	ExitCode                     int            `json:"exit_code,omitempty"`
	ContextSource                string         `json:"context_source,omitempty"`
	ContextSelectionReason       string         `json:"context_selection_reason,omitempty"`
	ContextArcTokens             int            `json:"context_arc_tokens,omitempty"`
	ContextCtxTokens             int            `json:"context_ctx_tokens,omitempty"`
	ContextSelectedTokens        int            `json:"context_selected_tokens,omitempty"`
	ContextTokenReduction        int            `json:"context_token_reduction,omitempty"`
	ContextTokenReductionPercent int            `json:"context_token_reduction_percent,omitempty"`
	PromptMinimized              bool           `json:"prompt_minimized,omitempty"`
	Reasoning                    []string       `json:"reasoning,omitempty"`
	MatchedSignals               []string       `json:"matched_signals,omitempty"`
	SignalBreakdown              map[string]int `json:"signal_breakdown,omitempty"`
	RoutingTrigger               string         `json:"routing_trigger,omitempty"`
	Notes                        []string       `json:"notes,omitempty"`
}

type UsageContext struct {
	ProjectRoot                  string
	ProviderModel                string
	ProviderSessionID            string
	BudgetModeSource             string
	EnvironmentBudgetProfile     string
	ContextSource                string
	ContextSelectionReason       string
	ContextArcTokens             int
	ContextCtxTokens             int
	ContextSelectedTokens        int
	ContextTokenReduction        int
	ContextTokenReductionPercent int
	PromptMinimized              bool
}

type Request struct {
	Task        string
	Provider    string
	UseProvider bool
	DryRun      bool
	BudgetMode  string
}

type matchedSignal struct {
	signal string
	weight int
}

func ParseMode(raw string) Mode {
	switch Mode(strings.TrimSpace(strings.ToLower(raw))) {
	case ModeUltraSafe:
		return ModeUltraSafe
	case ModeDeepWork:
		return ModeDeepWork
	case ModeEmergencyLowLimit:
		return ModeEmergencyLowLimit
	default:
		return ModeBalanced
	}
}

func ResolveEffectiveMode(requested string, presetProfile string) (Mode, string) {
	if strings.TrimSpace(requested) != "" {
		return ParseMode(requested), "requested"
	}
	if strings.TrimSpace(presetProfile) != "" {
		return ParseMode(presetProfile), "preset_profile"
	}
	return ModeBalanced, "default"
}

func ResolveEffectiveModeWithOverrides(requested string, sessionOverride PolicyOverride, projectOverride PolicyOverride, presetProfile string) (Mode, string) {
	if strings.TrimSpace(requested) != "" {
		return ParseMode(requested), "requested"
	}
	if strings.TrimSpace(sessionOverride.Mode) != "" {
		return ParseMode(sessionOverride.Mode), "session_override"
	}
	if strings.TrimSpace(projectOverride.Mode) != "" {
		return ParseMode(projectOverride.Mode), "project_override"
	}
	if strings.TrimSpace(presetProfile) != "" {
		return ParseMode(presetProfile), "preset_profile"
	}
	return ModeBalanced, "default"
}

func DefaultPolicy(mode Mode) Policy {
	switch mode {
	case ModeUltraSafe:
		return Policy{
			Mode:                              mode,
			LowLimitState:                     LowLimitConstrained,
			PreferLocal:                       true,
			BlockPremiumHighRisk:              true,
			RequireApprovalForPremiumRequired: true,
			RequireApprovalForPremiumHighRisk: true,
			Notes:                             []string{"ultra_safe mode prefers local work and blocks premium high-risk requests by default"},
		}
	case ModeDeepWork:
		return Policy{
			Mode:          mode,
			LowLimitState: LowLimitWarning,
			PreferLocal:   false,
			Notes:         []string{"deep_work mode allows premium provider work without extra budget gates"},
		}
	case ModeEmergencyLowLimit:
		return Policy{
			Mode:                              mode,
			LowLimitState:                     LowLimitEmergency,
			PreferLocal:                       true,
			BlockPremiumHighRisk:              true,
			BlockPremiumRequired:              true,
			RequireApprovalForPremiumRequired: true,
			RequireApprovalForPremiumHighRisk: true,
			Notes:                             []string{"emergency_low_limit mode aggressively protects provider budget"},
		}
	default:
		return Policy{
			Mode:          ModeBalanced,
			LowLimitState: LowLimitNormal,
			PreferLocal:   true,
			Notes:         []string{"balanced mode classifies requests and records usage without blocking ordinary provider work"},
		}
	}
}

func Assess(req Request) Assessment {
	mode := ParseMode(req.BudgetMode)
	policy := DefaultPolicy(mode)
	return AssessWithPolicy(req, policy)
}

func AssessWithPolicy(req Request, policy Policy) Assessment {
	mode := policy.Mode
	classification, reasoning, matchedSignals, confidence, signalBreakdown := classify(req)
	assessment := Assessment{
		Mode:            mode,
		LowLimitState:   policy.LowLimitState,
		Classification:  classification,
		Reasoning:       append([]string{}, reasoning...),
		MatchedSignals:  append([]string{}, matchedSignals...),
		SignalBreakdown: cloneBreakdown(signalBreakdown),
		Confidence:      confidence,
		ConfidenceTier:  confidenceTier(confidence),
	}
	if classification == ClassPremiumHighRisk {
		if policy.RequireApprovalForPremiumHighRisk {
			assessment.RequiresApproval = true
		}
		if policy.BlockPremiumHighRisk {
			assessment.ShouldBlock = true
		}
	}
	if classification == ClassPremiumRequired && policy.RequireApprovalForPremiumRequired {
		assessment.RequiresApproval = true
	}
	if classification == ClassPremiumRequired && policy.BlockPremiumRequired {
		assessment.ShouldBlock = true
	}
	assessment.RouteLocally, assessment.RoutingReason, assessment.RoutingTrigger = shouldRouteLocally(req, policy, assessment)
	return assessment
}

func shouldRouteLocally(req Request, policy Policy, assessment Assessment) (bool, string, string) {
	if !req.UseProvider {
		return true, "provider execution was already disabled for this run", "provider_disabled"
	}
	if req.DryRun {
		return true, "dry-run execution stays local by design", "dry_run"
	}
	if !policy.PreferLocal {
		return false, "", ""
	}
	switch assessment.Classification {
	case ClassNoProvider:
		return true, "request classification does not require provider execution", "no_provider"
	case ClassLocalFirst:
		return true, "budget policy prefers local execution for local-first tasks", "local_first"
	case ClassCheapProviderOK:
		if assessment.ConfidenceTier == "low" && len(assessment.MatchedSignals) == 0 {
			if policy.LowLimitState == LowLimitEmergency {
				return true, "emergency low-limit state prefers local execution for cheap-provider tasks with low provider confidence", "emergency_low_confidence"
			}
			if policy.LowLimitState == LowLimitConstrained {
				return true, "constrained low-limit state prefers local execution for weak cheap-provider tasks", "constrained_low_confidence"
			}
		}
		return false, "", ""
	default:
		return false, "", ""
	}
}

func classify(req Request) (Classification, []string, []string, int, map[string]int) {
	if !req.UseProvider || req.DryRun {
		return ClassNoProvider, []string{"provider execution is disabled for this run"}, []string{"provider_disabled"}, 100, map[string]int{"provider_disabled": 1}
	}
	task := strings.ToLower(req.Task)
	localSignals := []matchedSignal{
		{signal: "grep", weight: 3},
		{signal: "search", weight: 3},
		{signal: "find", weight: 2},
		{signal: "list files", weight: 3},
		{signal: "show files", weight: 3},
		{signal: "status", weight: 2},
		{signal: "refresh docs", weight: 3},
		{signal: "index", weight: 2},
		{signal: "summarize repo", weight: 3},
		{signal: "summarise repo", weight: 3},
		{signal: "read codebase", weight: 3},
		{signal: "inspect", weight: 1},
	}
	highRiskSignals := []matchedSignal{
		{signal: "deploy", weight: 5},
		{signal: "production", weight: 5},
		{signal: "database migration", weight: 5},
		{signal: "infra", weight: 4},
		{signal: "billing", weight: 5},
		{signal: "payments", weight: 5},
		{signal: "security incident", weight: 5},
		{signal: "incident", weight: 4},
		{signal: "large refactor", weight: 4},
		{signal: "rewrite", weight: 4},
		{signal: "migrate", weight: 4},
	}
	premiumSignals := []matchedSignal{
		{signal: "architecture", weight: 3},
		{signal: "rfc", weight: 3},
		{signal: "design", weight: 3},
		{signal: "plan", weight: 2},
		{signal: "investigate", weight: 2},
		{signal: "implement", weight: 4},
		{signal: "refactor", weight: 4},
		{signal: "build", weight: 3},
	}
	localMatches, localScore := collectMatches(task, localSignals)
	highRiskMatches, highRiskScore := collectMatches(task, highRiskSignals)
	premiumMatches, premiumScore := collectMatches(task, premiumSignals)
	breakdown := map[string]int{
		"local":     localScore,
		"premium":   premiumScore,
		"high_risk": highRiskScore,
	}
	if highRiskScore > 0 {
		return ClassPremiumHighRisk, summarizeMatches("premium high-risk", highRiskMatches), signalNames(highRiskMatches), confidenceFromScore(highRiskScore, 40), breakdown
	}
	if premiumScore > 0 && premiumScore >= localScore {
		return ClassPremiumRequired, summarizeMatches("premium-required", premiumMatches), signalNames(premiumMatches), confidenceFromScore(premiumScore, 35), breakdown
	}
	if localScore > 0 {
		return ClassLocalFirst, summarizeMatches("local-first", localMatches), signalNames(localMatches), confidenceFromScore(localScore, 30), breakdown
	}
	return ClassCheapProviderOK, []string{"task does not match local-first or premium-risk heuristics"}, nil, 25, breakdown
}

func collectMatches(task string, signals []matchedSignal) ([]matchedSignal, int) {
	out := []matchedSignal{}
	score := 0
	for _, signal := range signals {
		if strings.Contains(task, signal.signal) {
			out = append(out, signal)
			score += signal.weight
		}
	}
	return out, score
}

func summarizeMatches(label string, matches []matchedSignal) []string {
	reasons := make([]string, 0, len(matches)+1)
	total := 0
	for _, match := range matches {
		reasons = append(reasons, fmt.Sprintf("task contains %s signal %q (weight=%d)", label, match.signal, match.weight))
		total += match.weight
	}
	reasons = append(reasons, fmt.Sprintf("%s score=%d", label, total))
	return reasons
}

func signalNames(matches []matchedSignal) []string {
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.signal)
	}
	return out
}

func confidenceFromScore(score int, factor int) int {
	confidence := score * factor
	if confidence > 100 {
		return 100
	}
	if confidence < 0 {
		return 0
	}
	return confidence
}

func confidenceTier(confidence int) string {
	switch {
	case confidence >= 75:
		return "high"
	case confidence >= 40:
		return "medium"
	default:
		return "low"
	}
}

func cloneBreakdown(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func ApplyOverride(policy Policy, override PolicyOverride, source string) Policy {
	if trimmed := strings.TrimSpace(override.LowLimitState); trimmed != "" {
		policy.LowLimitState = ParseLowLimitState(trimmed)
	}
	if override.PreferLocal != nil {
		policy.PreferLocal = *override.PreferLocal
	}
	if override.BlockPremiumHighRisk != nil {
		policy.BlockPremiumHighRisk = *override.BlockPremiumHighRisk
	}
	if override.BlockPremiumRequired != nil {
		policy.BlockPremiumRequired = *override.BlockPremiumRequired
	}
	if override.RequireApprovalForPremiumHighRisk != nil {
		policy.RequireApprovalForPremiumHighRisk = *override.RequireApprovalForPremiumHighRisk
	}
	if override.RequireApprovalForPremiumRequired != nil {
		policy.RequireApprovalForPremiumRequired = *override.RequireApprovalForPremiumRequired
	}
	if len(override.Notes) > 0 {
		policy.Notes = append(policy.Notes, fmt.Sprintf("%s override applied", source))
		policy.Notes = append(policy.Notes, override.Notes...)
	}
	return policy
}

func ParseLowLimitState(raw string) LowLimitState {
	switch LowLimitState(strings.TrimSpace(strings.ToLower(raw))) {
	case LowLimitWarning:
		return LowLimitWarning
	case LowLimitConstrained:
		return LowLimitConstrained
	case LowLimitEmergency:
		return LowLimitEmergency
	default:
		return LowLimitNormal
	}
}

func LoadOverride(path string) (PolicyOverride, bool, error) {
	if strings.TrimSpace(path) == "" {
		return PolicyOverride{}, false, nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return PolicyOverride{}, false, nil
	} else if err != nil {
		return PolicyOverride{}, false, err
	}
	var override PolicyOverride
	if err := project.ReadJSON(path, &override); err != nil {
		return PolicyOverride{}, false, err
	}
	return override, true, nil
}

func WriteOverride(path string, override PolicyOverride) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("override path is required")
	}
	return project.WriteJSON(path, override)
}

func ClearOverride(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("override path is required")
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func ProjectOverridePath(root string) string {
	return project.ProjectFile(root, "budget", "project_override.json")
}

func LoadProjectOverride(root string) (PolicyOverride, bool, error) {
	return LoadOverride(ProjectOverridePath(root))
}

func WriteProjectOverride(root string, override PolicyOverride) error {
	return WriteOverride(ProjectOverridePath(root), override)
}

func ClearProjectOverride(root string) error {
	return ClearOverride(ProjectOverridePath(root))
}

func ResolvePolicy(root string, requested string, presetProfile string, sessionOverridePath string) (PolicyResolution, error) {
	projectOverridePath := ProjectOverridePath(root)
	projectOverride, hasProjectOverride, err := LoadOverride(projectOverridePath)
	if err != nil {
		return PolicyResolution{}, err
	}
	sessionOverride, hasSessionOverride, err := LoadOverride(sessionOverridePath)
	if err != nil {
		return PolicyResolution{}, err
	}
	if strings.TrimSpace(sessionOverridePath) != "" && !hasSessionOverride {
		return PolicyResolution{}, fmt.Errorf("budget override file not found: %s", sessionOverridePath)
	}
	mode, source := ResolveEffectiveModeWithOverrides(requested, sessionOverride, projectOverride, presetProfile)
	effective := DefaultPolicy(mode)
	appliedSources := []string{}
	if hasProjectOverride {
		effective = ApplyOverride(effective, projectOverride, "project")
		appliedSources = append(appliedSources, "project_override")
	}
	if hasSessionOverride {
		effective = ApplyOverride(effective, sessionOverride, "session")
		appliedSources = append(appliedSources, "session_override")
	}
	if err := project.WriteJSON(project.ProjectFile(root, "budget", "policy.json"), effective); err != nil {
		return PolicyResolution{}, err
	}
	return PolicyResolution{
		RequestedBudgetMode:      requested,
		EnvironmentBudgetProfile: presetProfile,
		ProjectOverridePath:      projectOverridePath,
		ProjectOverridePresent:   hasProjectOverride,
		SessionOverridePath:      strings.TrimSpace(sessionOverridePath),
		SessionOverridePresent:   hasSessionOverride,
		EffectiveMode:            mode,
		EffectiveModeSource:      source,
		AppliedOverrideSources:   appliedSources,
		ProjectOverride:          projectOverride,
		SessionOverride:          sessionOverride,
		EffectivePolicy:          effective,
	}, nil
}

func EnsurePolicy(root string, mode Mode) (Policy, error) {
	path := project.ProjectFile(root, "budget", "policy.json")
	effective := DefaultPolicy(mode)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := project.WriteJSON(path, effective); err != nil {
			return Policy{}, err
		}
		return effective, nil
	} else if err != nil {
		return Policy{}, err
	}
	var stored Policy
	if err := project.ReadJSON(path, &stored); err != nil {
		return Policy{}, err
	}
	if stored.Mode == mode {
		return stored, nil
	}
	if err := project.WriteJSON(path, effective); err != nil {
		return Policy{}, err
	}
	return effective, nil
}

func AppendUsageEvent(root string, event UsageEvent) error {
	path := project.ProjectFile(root, "budget", "usage_events.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(payload, '\n')); err != nil {
		return err
	}
	return nil
}

func NewUsageEvent(runID string, req Request, assessment Assessment, usageCtx UsageContext, runStatus string, providerUsed bool, result *provider.TaskResult, notes ...string) UsageEvent {
	event := UsageEvent{
		Timestamp:                    time.Now().UTC().Format(time.RFC3339),
		RunID:                        runID,
		ProjectRoot:                  usageCtx.ProjectRoot,
		Task:                         req.Task,
		Provider:                     req.Provider,
		ProviderModel:                usageCtx.ProviderModel,
		ProviderSessionID:            usageCtx.ProviderSessionID,
		BudgetMode:                   assessment.Mode,
		BudgetModeSource:             usageCtx.BudgetModeSource,
		EnvironmentBudgetProfile:     usageCtx.EnvironmentBudgetProfile,
		LowLimitState:                assessment.LowLimitState,
		Classification:               assessment.Classification,
		Confidence:                   assessment.Confidence,
		ConfidenceTier:               assessment.ConfidenceTier,
		Status:                       runStatus,
		UsedProvider:                 providerUsed,
		RouteLocally:                 assessment.RouteLocally,
		DryRun:                       req.DryRun,
		ContextSource:                usageCtx.ContextSource,
		ContextSelectionReason:       usageCtx.ContextSelectionReason,
		ContextArcTokens:             usageCtx.ContextArcTokens,
		ContextCtxTokens:             usageCtx.ContextCtxTokens,
		ContextSelectedTokens:        usageCtx.ContextSelectedTokens,
		ContextTokenReduction:        usageCtx.ContextTokenReduction,
		ContextTokenReductionPercent: usageCtx.ContextTokenReductionPercent,
		PromptMinimized:              usageCtx.PromptMinimized,
		Reasoning:                    append([]string{}, assessment.Reasoning...),
		MatchedSignals:               append([]string{}, assessment.MatchedSignals...),
		SignalBreakdown:              cloneBreakdown(assessment.SignalBreakdown),
		RoutingTrigger:               assessment.RoutingTrigger,
		Notes:                        append([]string{}, notes...),
	}
	if result != nil {
		event.ExitCode = result.ExitCode
	}
	return event
}
