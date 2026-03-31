package presets

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-os/internal/project"
)

type PresetDraft struct {
	Profile        PresetProfile        `json:"profile"`
	Brief          PresetBrief          `json:"brief"`
	Manifest       Manifest             `json:"manifest"`
	EvaluationPack PresetEvaluationPack `json:"evaluation_pack"`
	Paths          PresetDraftPaths     `json:"paths,omitempty"`
}

type PresetDraftPaths struct {
	Root           string `json:"root,omitempty"`
	ProfilePath    string `json:"profile_path,omitempty"`
	BriefJSONPath  string `json:"brief_json_path,omitempty"`
	BriefMDPath    string `json:"brief_markdown_path,omitempty"`
	ManifestPath   string `json:"manifest_path,omitempty"`
	EvaluationPath string `json:"evaluation_path,omitempty"`
	EvaluationMD   string `json:"evaluation_markdown_path,omitempty"`
}

type PresetDraftSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Summary     string   `json:"summary"`
	PresetType  string   `json:"preset_type"`
	TargetAgent string   `json:"target_agent"`
	Status      string   `json:"status"`
	Version     string   `json:"version"`
	Providers   []string `json:"providers,omitempty"`
	UpdatedAt   string   `json:"updated_at"`
}

type PresetProfile struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Summary          string              `json:"summary"`
	PresetType       string              `json:"preset_type"`
	TargetAgent      string              `json:"target_agent"`
	Identity         PresetIdentity      `json:"identity"`
	Goal             string              `json:"goal"`
	NonGoals         []string            `json:"non_goals,omitempty"`
	Behavior         PresetBehavior      `json:"behavior"`
	Environment      PresetEnvironment   `json:"environment"`
	Inputs           []string            `json:"inputs,omitempty"`
	Outputs          []string            `json:"outputs,omitempty"`
	Workflow         []string            `json:"workflow,omitempty"`
	QualityGates     []string            `json:"quality_gates,omitempty"`
	MemoryPolicy     PresetMemoryPolicy  `json:"memory_policy"`
	RetrievalPolicy  PresetRuleBlock     `json:"retrieval_policy"`
	ToolPolicy       PresetRuleBlock     `json:"tool_policy"`
	RuntimePolicy    PresetRuntimePolicy `json:"runtime_policy"`
	ApprovalPolicy   PresetRuleBlock     `json:"approval_policy"`
	FailurePolicy    PresetRuleBlock     `json:"failure_policy"`
	EvaluationPolicy PresetRuleBlock     `json:"evaluation_policy"`
	Artifacts        []string            `json:"artifacts,omitempty"`
	Compatibility    PresetCompatibility `json:"compatibility"`
	BudgetProfile    string              `json:"budget_profile,omitempty"`
	Version          string              `json:"version"`
	Status           string              `json:"status"`
	CreatedAt        string              `json:"created_at"`
	UpdatedAt        string              `json:"updated_at"`
}

type PresetIdentity struct {
	Persona  string `json:"persona,omitempty"`
	Audience string `json:"audience,omitempty"`
}

type PresetBehavior struct {
	AutonomyLevel string   `json:"autonomy_level,omitempty"`
	Tone          string   `json:"tone,omitempty"`
	WorkingStyle  []string `json:"working_style,omitempty"`
	ResponseStyle []string `json:"response_style,omitempty"`
}

type PresetEnvironment struct {
	WorkspaceScope       string   `json:"workspace_scope,omitempty"`
	CompatibleProviders  []string `json:"compatible_providers,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
}

type PresetRuleBlock struct {
	Summary string   `json:"summary,omitempty"`
	Rules   []string `json:"rules,omitempty"`
}

type PresetMemoryPolicy struct {
	Summary       string   `json:"summary,omitempty"`
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
	Rules         []string `json:"rules,omitempty"`
}

type PresetRuntimePolicy struct {
	ExecutionMode  string   `json:"execution_mode,omitempty"`
	AutoStopPolicy string   `json:"auto_stop_policy,omitempty"`
	Rules          []string `json:"rules,omitempty"`
}

type PresetCompatibility struct {
	Providers []string `json:"providers,omitempty"`
	Notes     []string `json:"notes,omitempty"`
}

type PresetBrief struct {
	PresetID      string   `json:"preset_id"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Audience      string   `json:"audience,omitempty"`
	WhatItDoes    []string `json:"what_it_does,omitempty"`
	WhatItDoesNot []string `json:"what_it_does_not,omitempty"`
	HowToUse      []string `json:"how_to_use,omitempty"`
	Markdown      string   `json:"markdown"`
	UpdatedAt     string   `json:"updated_at"`
}

type PresetEvaluationPack struct {
	PresetID                string           `json:"preset_id"`
	Status                  string           `json:"status"`
	RepresentativeScenarios []PresetScenario `json:"representative_scenarios,omitempty"`
	AcceptanceChecklist     []string         `json:"acceptance_checklist,omitempty"`
	AntiFailureRules        []string         `json:"anti_failure_rules,omitempty"`
	UpdatedAt               string           `json:"updated_at"`
}

type PresetScenario struct {
	ID                string   `json:"id"`
	Kind              string   `json:"kind"`
	Title             string   `json:"title"`
	InputSummary      string   `json:"input_summary"`
	ExpectedBehavior  []string `json:"expected_behavior,omitempty"`
	ExpectedArtifacts []string `json:"expected_artifacts,omitempty"`
	ApprovalExpected  bool     `json:"approval_expected,omitempty"`
	RiskNotes         []string `json:"risk_notes,omitempty"`
}

type DraftInitOptions struct {
	WorkspaceRoot string
	ID            string
	Name          string
	Summary       string
	Goal          string
	PresetType    string
	TargetAgent   string
	Providers     []string
	Version       string
	BudgetProfile string
	AutonomyLevel string
}

type DraftUpdateOptions struct {
	WorkspaceRoot string
	ID            string
	Name          string
	Summary       string
	Goal          string
	PresetType    string
	TargetAgent   string
	Providers     []string
	Version       string
	BudgetProfile string
	AutonomyLevel string
	NonGoals      []string
	Inputs        []string
	Outputs       []string
	Workflow      []string
	QualityGates  []string
	Status        string
}

func InitDraft(opts DraftInitOptions) (PresetDraft, error) {
	if err := project.RequireProject(opts.WorkspaceRoot); err != nil {
		return PresetDraft{}, err
	}
	draft, err := NewPresetDraft(opts)
	if err != nil {
		return PresetDraft{}, err
	}
	return SaveDraft(opts.WorkspaceRoot, draft)
}

func UpdateDraft(opts DraftUpdateOptions) (PresetDraft, error) {
	if err := project.RequireProject(opts.WorkspaceRoot); err != nil {
		return PresetDraft{}, err
	}
	id := strings.TrimSpace(opts.ID)
	if id == "" {
		return PresetDraft{}, fmt.Errorf("draft id is required")
	}
	draft, err := LoadDraft(opts.WorkspaceRoot, id)
	if err != nil {
		return PresetDraft{}, err
	}
	previousEvaluationStatus := strings.TrimSpace(draft.EvaluationPack.Status)
	profile := draft.Profile
	if value := strings.TrimSpace(opts.Name); value != "" {
		profile.Name = value
	}
	if value := strings.TrimSpace(opts.Summary); value != "" {
		profile.Summary = value
	}
	if value := strings.TrimSpace(opts.Goal); value != "" {
		profile.Goal = value
	}
	if value := strings.TrimSpace(opts.PresetType); value != "" {
		profile.PresetType = value
	}
	if value := strings.TrimSpace(opts.TargetAgent); value != "" {
		profile.TargetAgent = value
	}
	if providers := normalizeProviderList(opts.Providers); len(providers) > 0 {
		profile.Environment.CompatibleProviders = providers
		profile.Compatibility.Providers = append([]string{}, providers...)
	}
	if value := strings.TrimSpace(opts.Version); value != "" {
		profile.Version = value
	}
	if value := strings.TrimSpace(opts.BudgetProfile); value != "" {
		profile.BudgetProfile = value
	}
	if value := strings.TrimSpace(opts.AutonomyLevel); value != "" {
		profile.Behavior.AutonomyLevel = value
	}
	if nonGoals := normalizeTextList(opts.NonGoals); len(nonGoals) > 0 {
		profile.NonGoals = nonGoals
	}
	if inputs := normalizeIdentifierList(opts.Inputs); len(inputs) > 0 {
		profile.Inputs = inputs
	}
	if outputs := normalizeOutputList(opts.Outputs); len(outputs) > 0 {
		profile.Outputs = outputs
	}
	if workflow := normalizeWorkflowList(opts.Workflow); len(workflow) > 0 {
		profile.Workflow = workflow
	}
	if qualityGates := normalizeQualityGateList(opts.QualityGates); len(qualityGates) > 0 {
		profile.QualityGates = qualityGates
	}
	if value := strings.TrimSpace(opts.Status); value != "" {
		profile.Status = value
	}
	now := time.Now().UTC().Format(time.RFC3339)
	profile.UpdatedAt = now
	draft.Profile = profile
	draft.Brief = buildDraftBrief(profile, now)
	draft.Manifest = buildDraftManifest(profile)
	draft.EvaluationPack = buildDraftEvaluationPack(profile, now)
	if previousEvaluationStatus != "" {
		draft.EvaluationPack.Status = previousEvaluationStatus
	}
	return SaveDraft(opts.WorkspaceRoot, draft)
}

func NewPresetDraft(opts DraftInitOptions) (PresetDraft, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	id := strings.TrimSpace(opts.ID)
	name := strings.TrimSpace(opts.Name)
	summary := strings.TrimSpace(opts.Summary)
	goal := strings.TrimSpace(opts.Goal)
	presetType := strings.TrimSpace(opts.PresetType)
	if presetType == "" {
		presetType = "domain"
	}
	targetAgent := strings.TrimSpace(opts.TargetAgent)
	if targetAgent == "" {
		targetAgent = "codex"
	}
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "0.1.0"
	}
	budgetProfile := strings.TrimSpace(opts.BudgetProfile)
	if budgetProfile == "" {
		budgetProfile = "balanced"
	}
	autonomy := strings.TrimSpace(opts.AutonomyLevel)
	if autonomy == "" {
		autonomy = "low"
	}
	providers := normalizeProviderList(opts.Providers)
	if len(providers) == 0 {
		providers = []string{targetAgent}
	}

	profile := PresetProfile{
		ID:          id,
		Name:        name,
		Summary:     summary,
		PresetType:  presetType,
		TargetAgent: targetAgent,
		Identity: PresetIdentity{
			Persona:  "preset-builder",
			Audience: "preset authors and project operators",
		},
		Goal:     goal,
		NonGoals: []string{"Do not publish or install without simulation and validation."},
		Behavior: PresetBehavior{
			AutonomyLevel: autonomy,
			Tone:          "clear and operational",
			WorkingStyle:  []string{"guided_interview", "deterministic_outputs"},
			ResponseStyle: []string{"structured_summary", "reviewable_artifacts"},
		},
		Environment: PresetEnvironment{
			WorkspaceScope:       "project",
			CompatibleProviders:  append([]string{}, providers...),
			RequiredCapabilities: []string{"structured_outputs", "policy_binding"},
		},
		Inputs:       []string{"goal", "environment_constraints", "workflow_preferences"},
		Outputs:      []string{"preset_brief", "preset_manifest", "evaluation_pack"},
		Workflow:     []string{"interview", "simulate", "refine", "save"},
		QualityGates: []string{"profile_complete", "simulation_ready", "validation_ready"},
		MemoryPolicy: PresetMemoryPolicy{
			Summary:       "Keep durable preset state namespaced and reviewable.",
			AllowedScopes: []string{"project", "session", fmt.Sprintf("presets/%s", id)},
			Rules:         []string{"Write durable preset state only into namespaced preset scopes."},
		},
		RetrievalPolicy: PresetRuleBlock{
			Summary: "Prefer local project docs and ARC memory before broad context expansion.",
			Rules:   []string{"Prefer local project evidence before broad provider-side assumptions."},
		},
		ToolPolicy: PresetRuleBlock{
			Summary: "Do not assume destructive tools or silent runtime powers before bindings exist.",
			Rules:   []string{"Keep tool assumptions explicit and reviewable."},
		},
		RuntimePolicy: PresetRuntimePolicy{
			ExecutionMode:  "preview_only",
			AutoStopPolicy: "manual_or_idle",
			Rules:          []string{"Default to preview-safe runtime behavior until binding generation refines it."},
		},
		ApprovalPolicy: PresetRuleBlock{
			Summary: "Risky actions require approval before execution.",
			Rules:   []string{"Require approval before risky or destructive execution."},
		},
		FailurePolicy: PresetRuleBlock{
			Summary: "Fail explicitly and keep generated artifacts reviewable.",
			Rules:   []string{"Do not silently skip validation or simulation failures."},
		},
		EvaluationPolicy: PresetRuleBlock{
			Summary: "Every preset needs typical, edge, and risky evaluation scenarios.",
			Rules:   []string{"Generate representative, edge-case, and risky-case checks before publish."},
		},
		Artifacts:     []string{"brief.md", "manifest.json", "evaluation.json"},
		Compatibility: PresetCompatibility{Providers: append([]string{}, providers...), Notes: []string{"Draft compatibility only until runtime bindings are generated."}},
		BudgetProfile: budgetProfile,
		Version:       version,
		Status:        "draft",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := ValidateProfile(profile); err != nil {
		return PresetDraft{}, err
	}

	brief := buildDraftBrief(profile, now)
	evaluation := buildDraftEvaluationPack(profile, now)
	manifest := buildDraftManifest(profile)
	if err := validateManifest(manifest); err != nil {
		return PresetDraft{}, fmt.Errorf("generated manifest draft is invalid: %w", err)
	}
	return PresetDraft{
		Profile:        profile,
		Brief:          brief,
		Manifest:       manifest,
		EvaluationPack: evaluation,
	}, nil
}

func SaveDraft(workspaceRoot string, draft PresetDraft) (PresetDraft, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetDraft{}, err
	}
	if err := ValidateDraft(draft); err != nil {
		return PresetDraft{}, err
	}
	paths := draftPaths(workspaceRoot, draft.Profile.ID)
	draft.Paths = paths
	if err := project.WriteJSON(paths.ProfilePath, draft.Profile); err != nil {
		return PresetDraft{}, err
	}
	if err := project.WriteJSON(paths.BriefJSONPath, draft.Brief); err != nil {
		return PresetDraft{}, err
	}
	if err := project.WriteString(paths.BriefMDPath, strings.TrimSpace(draft.Brief.Markdown)+"\n"); err != nil {
		return PresetDraft{}, err
	}
	if err := project.WriteJSON(paths.ManifestPath, draft.Manifest); err != nil {
		return PresetDraft{}, err
	}
	if err := project.WriteJSON(paths.EvaluationPath, draft.EvaluationPack); err != nil {
		return PresetDraft{}, err
	}
	if err := project.WriteString(paths.EvaluationMD, renderEvaluationMarkdown(draft.EvaluationPack)); err != nil {
		return PresetDraft{}, err
	}
	return draft, nil
}

func LoadDraft(workspaceRoot string, presetID string) (PresetDraft, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetDraft{}, err
	}
	paths := draftPaths(workspaceRoot, presetID)
	var profile PresetProfile
	if err := project.ReadJSON(paths.ProfilePath, &profile); err != nil {
		return PresetDraft{}, err
	}
	var brief PresetBrief
	if err := project.ReadJSON(paths.BriefJSONPath, &brief); err != nil {
		return PresetDraft{}, err
	}
	var manifest Manifest
	if err := project.ReadJSON(paths.ManifestPath, &manifest); err != nil {
		return PresetDraft{}, err
	}
	var evaluation PresetEvaluationPack
	if err := project.ReadJSON(paths.EvaluationPath, &evaluation); err != nil {
		return PresetDraft{}, err
	}
	draft := PresetDraft{
		Profile:        profile,
		Brief:          brief,
		Manifest:       manifest,
		EvaluationPack: evaluation,
		Paths:          paths,
	}
	if err := ValidateDraft(draft); err != nil {
		return PresetDraft{}, err
	}
	return draft, nil
}

func ValidateDraft(draft PresetDraft) error {
	if err := ValidateProfile(draft.Profile); err != nil {
		return err
	}
	if strings.TrimSpace(draft.Brief.PresetID) != draft.Profile.ID {
		return fmt.Errorf("brief preset_id must match profile id")
	}
	if strings.TrimSpace(draft.Brief.Title) == "" || strings.TrimSpace(draft.Brief.Markdown) == "" {
		return fmt.Errorf("brief requires title and markdown")
	}
	if strings.TrimSpace(draft.EvaluationPack.PresetID) != draft.Profile.ID {
		return fmt.Errorf("evaluation_pack preset_id must match profile id")
	}
	if len(draft.EvaluationPack.RepresentativeScenarios) < 3 {
		return fmt.Errorf("evaluation_pack requires at least 3 representative scenarios")
	}
	seenScenarioKinds := map[string]struct{}{}
	for _, scenario := range draft.EvaluationPack.RepresentativeScenarios {
		if strings.TrimSpace(scenario.Kind) == "" || strings.TrimSpace(scenario.Title) == "" {
			return fmt.Errorf("evaluation scenarios require kind and title")
		}
		seenScenarioKinds[strings.TrimSpace(scenario.Kind)] = struct{}{}
	}
	for _, required := range []string{"typical", "edge", "risky"} {
		if _, ok := seenScenarioKinds[required]; !ok {
			return fmt.Errorf("evaluation_pack must include %s scenario", required)
		}
	}
	if len(draft.EvaluationPack.AcceptanceChecklist) == 0 {
		return fmt.Errorf("evaluation_pack requires acceptance_checklist")
	}
	if len(draft.EvaluationPack.AntiFailureRules) == 0 {
		return fmt.Errorf("evaluation_pack requires anti_failure_rules")
	}
	if strings.TrimSpace(draft.Manifest.ID) != draft.Profile.ID {
		return fmt.Errorf("manifest id must match profile id")
	}
	if err := validateManifest(draft.Manifest); err != nil {
		return fmt.Errorf("manifest is invalid: %w", err)
	}
	return nil
}

func ValidateProfile(profile PresetProfile) error {
	if strings.TrimSpace(profile.ID) == "" || strings.TrimSpace(profile.Name) == "" || strings.TrimSpace(profile.Summary) == "" || strings.TrimSpace(profile.Goal) == "" {
		return fmt.Errorf("profile requires id, name, summary, and goal")
	}
	if !allowedPresetType(profile.PresetType) {
		return fmt.Errorf("unsupported preset_type %q", profile.PresetType)
	}
	if !allowedBudgetProfile(profile.BudgetProfile) {
		return fmt.Errorf("unsupported budget_profile %q", profile.BudgetProfile)
	}
	switch strings.TrimSpace(profile.TargetAgent) {
	case "codex", "claude", "arc":
	default:
		return fmt.Errorf("unsupported target_agent %q", profile.TargetAgent)
	}
	switch strings.TrimSpace(profile.Status) {
	case "draft", "tested", "published", "installed", "deprecated", "archived":
	default:
		return fmt.Errorf("unsupported status %q", profile.Status)
	}
	switch strings.TrimSpace(profile.Behavior.AutonomyLevel) {
	case "", "low", "medium", "high":
	default:
		return fmt.Errorf("unsupported autonomy_level %q", profile.Behavior.AutonomyLevel)
	}
	if strings.TrimSpace(profile.Version) == "" {
		return fmt.Errorf("profile requires version")
	}
	if err := validateStringList(profile.Environment.CompatibleProviders, "environment.compatible_providers"); err != nil {
		return err
	}
	if err := validateStringList(profile.Inputs, "inputs"); err != nil {
		return err
	}
	if err := validateStringList(profile.Outputs, "outputs"); err != nil {
		return err
	}
	if err := validateStringList(profile.Workflow, "workflow"); err != nil {
		return err
	}
	if err := validateStringList(profile.QualityGates, "quality_gates"); err != nil {
		return err
	}
	if err := validateStringList(profile.Artifacts, "artifacts"); err != nil {
		return err
	}
	if err := validateStringList(profile.MemoryPolicy.AllowedScopes, "memory_policy.allowed_scopes"); err != nil {
		return err
	}
	for _, scope := range profile.MemoryPolicy.AllowedScopes {
		if !allowedMemoryScope(strings.TrimSpace(scope)) {
			return fmt.Errorf("unsupported memory scope %q", scope)
		}
	}
	if strings.TrimSpace(profile.RuntimePolicy.ExecutionMode) == "" {
		profile.RuntimePolicy.ExecutionMode = "preview_only"
	}
	if !allowedRuntimePermission(profile.RuntimePolicy.ExecutionMode) {
		return fmt.Errorf("unsupported runtime_policy.execution_mode %q", profile.RuntimePolicy.ExecutionMode)
	}
	if len(profile.Outputs) == 0 || len(profile.Workflow) == 0 || len(profile.QualityGates) == 0 {
		return fmt.Errorf("profile requires outputs, workflow, and quality_gates")
	}
	if strings.TrimSpace(profile.Behavior.AutonomyLevel) == "medium" || strings.TrimSpace(profile.Behavior.AutonomyLevel) == "high" {
		if len(profile.ApprovalPolicy.Rules) == 0 {
			return fmt.Errorf("approval_policy rules are required when autonomy_level is %q", profile.Behavior.AutonomyLevel)
		}
	}
	return nil
}

func ListDrafts(workspaceRoot string) ([]PresetDraftSummary, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return nil, err
	}
	root := draftsRoot(workspaceRoot)
	entries, err := filepath.Glob(filepath.Join(root, "*", "profile.json"))
	if err != nil {
		return nil, err
	}
	out := make([]PresetDraftSummary, 0, len(entries))
	for _, path := range entries {
		var profile PresetProfile
		if err := project.ReadJSON(path, &profile); err != nil {
			continue
		}
		out = append(out, PresetDraftSummary{
			ID:          profile.ID,
			Name:        profile.Name,
			Summary:     profile.Summary,
			PresetType:  profile.PresetType,
			TargetAgent: profile.TargetAgent,
			Status:      profile.Status,
			Version:     profile.Version,
			Providers:   append([]string{}, profile.Environment.CompatibleProviders...),
			UpdatedAt:   profile.UpdatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt == out[j].UpdatedAt {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out, nil
}

func buildDraftBrief(profile PresetProfile, now string) PresetBrief {
	howToUse := []string{
		fmt.Sprintf("Use %s when the main target agent is %s and the desired outcome is %s.", profile.Name, profile.TargetAgent, strings.ToLower(profile.Goal)),
		fmt.Sprintf("Start with the guided workflow: %s.", strings.Join(profile.Workflow, " -> ")),
		"Do not publish or install this preset before simulation and validation pass.",
	}
	whatItDoes := []string{
		profile.Goal,
		fmt.Sprintf("Produces %s.", strings.Join(profile.Outputs, ", ")),
		fmt.Sprintf("Keeps quality gates explicit: %s.", strings.Join(profile.QualityGates, ", ")),
	}
	whatItDoesNot := append([]string{}, profile.NonGoals...)
	if len(whatItDoesNot) == 0 {
		whatItDoesNot = []string{"It does not bypass simulation, validation, or approval rules."}
	}
	markdownLines := []string{
		"# " + profile.Name,
		"",
		profile.Summary,
		"",
		"## Goal",
		"",
		profile.Goal,
		"",
		"## What It Does",
		"",
	}
	for _, item := range whatItDoes {
		markdownLines = append(markdownLines, "- "+item)
	}
	markdownLines = append(markdownLines, "", "## What It Does Not Do", "")
	for _, item := range whatItDoesNot {
		markdownLines = append(markdownLines, "- "+item)
	}
	markdownLines = append(markdownLines, "", "## How To Use", "")
	for _, item := range howToUse {
		markdownLines = append(markdownLines, "- "+item)
	}
	return PresetBrief{
		PresetID:      profile.ID,
		Title:         profile.Name,
		Summary:       profile.Summary,
		Audience:      profile.Identity.Audience,
		WhatItDoes:    whatItDoes,
		WhatItDoesNot: whatItDoesNot,
		HowToUse:      howToUse,
		Markdown:      strings.Join(markdownLines, "\n"),
		UpdatedAt:     now,
	}
}

func buildDraftEvaluationPack(profile PresetProfile, now string) PresetEvaluationPack {
	outputs := append([]string{}, profile.Outputs...)
	return PresetEvaluationPack{
		PresetID: profile.ID,
		Status:   "draft",
		RepresentativeScenarios: []PresetScenario{
			{
				ID:                "typical",
				Kind:              "typical",
				Title:             "Typical preset authoring flow",
				InputSummary:      fmt.Sprintf("The user wants %s under normal project conditions.", strings.ToLower(profile.Goal)),
				ExpectedBehavior:  []string{"Collect the minimum missing details.", "Produce a coherent draft profile."},
				ExpectedArtifacts: outputs,
			},
			{
				ID:                "edge",
				Kind:              "edge",
				Title:             "Underspecified constraints",
				InputSummary:      "The user gives a vague goal with missing environment and workflow details.",
				ExpectedBehavior:  []string{"Surface contradictions or missing blocks.", "Keep the preset in draft state."},
				ExpectedArtifacts: []string{"preset_brief", "evaluation_pack"},
			},
			{
				ID:                "risky",
				Kind:              "risky",
				Title:             "Risky autonomy or policy conflict",
				InputSummary:      "The user requests stronger autonomy than the current approval/runtime policy safely allows.",
				ExpectedBehavior:  []string{"Surface the risk explicitly.", "Require stronger approval policy before publish."},
				ExpectedArtifacts: []string{"evaluation_pack"},
				ApprovalExpected:  true,
				RiskNotes:         []string{"Approval policy must stay explicit before publish."},
			},
		},
		AcceptanceChecklist: []string{
			"Profile has goal, workflow, outputs, and quality gates.",
			"Manifest draft stays valid for local storage and later bindings.",
			"Brief explains what the preset will do and not do.",
			"Typical, edge, and risky scenarios exist before publish.",
		},
		AntiFailureRules: []string{
			"Do not publish without simulation and validation.",
			"Do not silently escalate runtime permissions.",
			"Do not let one preset overwrite another preset's namespaced state.",
		},
		UpdatedAt: now,
	}
}

func buildDraftManifest(profile PresetProfile) Manifest {
	short := []string{
		fmt.Sprintf("Best for %s presets that target %s and need explicit artifacts.", profile.PresetType, profile.TargetAgent),
		"Use it when you want a guided preset draft with explicit workflow, quality gates, and policy blocks.",
		"By default it stays in draft mode and expects simulation before install or publish.",
	}
	return Manifest{
		ID:                  profile.ID,
		Name:                profile.Name,
		Tagline:             profile.Summary,
		ShortDescription:    short,
		Goal:                profile.Goal,
		Adapter:             "arc",
		Category:            "draft",
		Persona:             fallbackString(profile.Identity.Persona, "preset-draft"),
		Version:             profile.Version,
		Files:               []string{},
		SafetyNotes:         []string{"Draft manifest only; runtime bindings still need simulation and validation."},
		PresetType:          profile.PresetType,
		CompatibleProviders: append([]string{}, profile.Environment.CompatibleProviders...),
		Permissions:         Permissions{Runtime: profile.RuntimePolicy.ExecutionMode},
		MemoryScopes:        append([]string{}, profile.MemoryPolicy.AllowedScopes...),
		RuntimePolicy:       RuntimePolicy{AutoStopPolicy: profile.RuntimePolicy.AutoStopPolicy},
		QualityGates:        append([]string{}, profile.QualityGates...),
		BudgetProfile:       profile.BudgetProfile,
		Author:              Author{Name: "ARC Preset Studio", Handle: "arc"},
	}
}

func renderEvaluationMarkdown(pack PresetEvaluationPack) string {
	lines := []string{
		"# Evaluation Pack",
		"",
		fmt.Sprintf("- preset: `%s`", pack.PresetID),
		fmt.Sprintf("- status: `%s`", pack.Status),
		"",
		"## Representative Scenarios",
		"",
	}
	for _, scenario := range pack.RepresentativeScenarios {
		lines = append(lines, "### "+scenario.Title, "", fmt.Sprintf("- kind: `%s`", scenario.Kind), fmt.Sprintf("- input: %s", scenario.InputSummary))
		if len(scenario.ExpectedBehavior) > 0 {
			lines = append(lines, "- expected behavior:")
			for _, item := range scenario.ExpectedBehavior {
				lines = append(lines, "  - "+item)
			}
		}
		if len(scenario.ExpectedArtifacts) > 0 {
			lines = append(lines, "- expected artifacts: "+strings.Join(scenario.ExpectedArtifacts, ", "))
		}
		if scenario.ApprovalExpected {
			lines = append(lines, "- approval expected: true")
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Acceptance Checklist", "")
	for _, item := range pack.AcceptanceChecklist {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "", "## Anti-Failure Rules", "")
	for _, item := range pack.AntiFailureRules {
		lines = append(lines, "- "+item)
	}
	return strings.Join(lines, "\n") + "\n"
}

func draftsRoot(workspaceRoot string) string {
	return project.ProjectFile(workspaceRoot, "presets", "platform")
}

func draftPaths(workspaceRoot string, presetID string) PresetDraftPaths {
	root := filepath.Join(draftsRoot(workspaceRoot), presetID)
	return PresetDraftPaths{
		Root:           root,
		ProfilePath:    filepath.Join(root, "profile.json"),
		BriefJSONPath:  filepath.Join(root, "brief.json"),
		BriefMDPath:    filepath.Join(root, "brief.md"),
		ManifestPath:   filepath.Join(root, "manifest.json"),
		EvaluationPath: filepath.Join(root, "evaluation.json"),
		EvaluationMD:   filepath.Join(root, "evaluation.md"),
	}
}

func normalizeCSVList(values []string) []string {
	out := normalizeOrderedCSVList(values, func(value string) string {
		return value
	})
	sort.Strings(out)
	return out
}

func normalizeProviderList(values []string) []string {
	out := normalizeOrderedCSVList(values, func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	})
	sort.Strings(out)
	return out
}

func normalizeTextList(values []string) []string {
	return normalizeOrderedCSVList(values, func(value string) string {
		return strings.TrimSpace(value)
	})
}

func normalizeIdentifierList(values []string) []string {
	return normalizeOrderedCSVList(values, func(value string) string {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.ReplaceAll(value, "-", "_")
		value = strings.ReplaceAll(value, " ", "_")
		return value
	})
}

func normalizeOutputList(values []string) []string {
	aliases := map[string]string{
		"brief":             "preset_brief",
		"preset_brief":      "preset_brief",
		"manifest":          "preset_manifest",
		"preset_manifest":   "preset_manifest",
		"evaluation":        "evaluation_pack",
		"evaluation_pack":   "evaluation_pack",
		"evaluation-report": "evaluation_pack",
	}
	return normalizeOrderedCSVList(values, func(value string) string {
		key := strings.TrimSpace(strings.ToLower(value))
		key = strings.ReplaceAll(key, " ", "_")
		if mapped, ok := aliases[key]; ok {
			return mapped
		}
		key = strings.ReplaceAll(key, "-", "_")
		if mapped, ok := aliases[key]; ok {
			return mapped
		}
		return key
	})
}

func normalizeWorkflowList(values []string) []string {
	aliases := map[string]string{
		"interview":     "interview",
		"normalize":     "normalize",
		"normalise":     "normalize",
		"simulation":    "simulate",
		"simulate":      "simulate",
		"refine":        "refine",
		"save":          "save",
		"publish":       "publish",
		"publish_ready": "publish",
		"install":       "install",
	}
	return normalizeOrderedCSVList(values, func(value string) string {
		key := strings.TrimSpace(strings.ToLower(value))
		key = strings.ReplaceAll(key, "-", "_")
		key = strings.ReplaceAll(key, " ", "_")
		if mapped, ok := aliases[key]; ok {
			return mapped
		}
		return key
	})
}

func normalizeQualityGateList(values []string) []string {
	aliases := map[string]string{
		"profile_complete": "profile_complete",
		"profile-complete": "profile_complete",
		"simulation":       "simulation_ready",
		"simulation_ready": "simulation_ready",
		"simulation-ready": "simulation_ready",
		"validation":       "validation_ready",
		"validated":        "validation_ready",
		"validation_ready": "validation_ready",
		"validation-ready": "validation_ready",
		"brief_reviewed":   "brief_reviewed",
		"brief-reviewed":   "brief_reviewed",
	}
	return normalizeOrderedCSVList(values, func(value string) string {
		key := strings.TrimSpace(strings.ToLower(value))
		key = strings.ReplaceAll(key, " ", "_")
		if mapped, ok := aliases[key]; ok {
			return mapped
		}
		key = strings.ReplaceAll(key, "-", "_")
		if mapped, ok := aliases[key]; ok {
			return mapped
		}
		return key
	})
}

func normalizeOrderedCSVList(values []string, transform func(string) string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			value := strings.TrimSpace(part)
			if value == "" {
				continue
			}
			if transform != nil {
				value = transform(value)
			}
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}

func fallbackString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
