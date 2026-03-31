package presets

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

type PresetExportResult struct {
	PresetID   string               `json:"preset_id"`
	Status     string               `json:"status"`
	ExportedAt string               `json:"exported_at"`
	Bundles    []PresetExportBundle `json:"bundles"`
}

type PresetExportBundle struct {
	Provider string            `json:"provider"`
	Manifest Manifest          `json:"manifest"`
	Paths    PresetExportPaths `json:"paths"`
}

type PresetExportPaths struct {
	Root          string `json:"root,omitempty"`
	ManifestPath  string `json:"manifest_path,omitempty"`
	ReadmePath    string `json:"readme_path,omitempty"`
	OverviewPath  string `json:"overview_path,omitempty"`
	ProviderPath  string `json:"provider_path,omitempty"`
	BriefJSONPath string `json:"brief_json_path,omitempty"`
	EvalJSONPath  string `json:"evaluation_json_path,omitempty"`
	SimJSONPath   string `json:"simulation_json_path,omitempty"`
	ProfileJSON   string `json:"profile_json_path,omitempty"`
}

func ExportDraft(workspaceRoot string, presetID string) (PresetExportResult, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetExportResult{}, err
	}
	draft, err := LoadDraft(workspaceRoot, presetID)
	if err != nil {
		return PresetExportResult{}, err
	}
	switch strings.TrimSpace(draft.Profile.Status) {
	case "tested", "published":
	default:
		return PresetExportResult{}, fmt.Errorf("preset draft must be tested or published before export")
	}
	report, err := LoadSimulationReport(workspaceRoot, presetID)
	if err != nil {
		return PresetExportResult{}, fmt.Errorf("simulation report is required before export: %w", err)
	}
	if report.Status != "passed" || !report.DraftValid || len(report.Contradictions) > 0 {
		return PresetExportResult{}, fmt.Errorf("simulation must pass without contradictions before export")
	}
	providers := exportProvidersForDraft(draft)
	if len(providers) == 0 {
		return PresetExportResult{}, fmt.Errorf("draft does not declare an exportable provider")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result := PresetExportResult{
		PresetID:   draft.Profile.ID,
		Status:     "exported",
		ExportedAt: now,
		Bundles:    make([]PresetExportBundle, 0, len(providers)),
	}
	for _, provider := range providers {
		bundle, err := exportDraftBundle(workspaceRoot, draft, report, provider)
		if err != nil {
			return PresetExportResult{}, err
		}
		result.Bundles = append(result.Bundles, bundle)
	}
	return result, nil
}

func exportDraftBundle(workspaceRoot string, draft PresetDraft, report PresetSimulationReport, provider string) (PresetExportBundle, error) {
	manifest := buildExportManifest(draft.Profile, provider)
	if err := validateManifest(manifest); err != nil {
		return PresetExportBundle{}, fmt.Errorf("generated export manifest is invalid: %w", err)
	}
	paths := exportPaths(workspaceRoot, manifest.ID, provider)
	readme := renderExportReadme(draft.Profile, provider)
	overview := renderExportOverview(draft, provider)
	providerDoc := renderProviderGuide(draft.Profile, provider)
	if err := project.WriteJSON(paths.ManifestPath, manifest); err != nil {
		return PresetExportBundle{}, err
	}
	if err := project.WriteString(paths.ReadmePath, readme); err != nil {
		return PresetExportBundle{}, err
	}
	if err := project.WriteString(paths.OverviewPath, overview); err != nil {
		return PresetExportBundle{}, err
	}
	if err := project.WriteString(paths.ProviderPath, providerDoc); err != nil {
		return PresetExportBundle{}, err
	}
	if err := project.WriteJSON(paths.ProfileJSON, draft.Profile); err != nil {
		return PresetExportBundle{}, err
	}
	if err := project.WriteJSON(paths.BriefJSONPath, draft.Brief); err != nil {
		return PresetExportBundle{}, err
	}
	if err := project.WriteJSON(paths.EvalJSONPath, draft.EvaluationPack); err != nil {
		return PresetExportBundle{}, err
	}
	if err := project.WriteJSON(paths.SimJSONPath, report); err != nil {
		return PresetExportBundle{}, err
	}
	return PresetExportBundle{
		Provider: provider,
		Manifest: manifest,
		Paths:    paths,
	}, nil
}

func buildExportManifest(profile PresetProfile, provider string) Manifest {
	return Manifest{
		ID:                  exportBundleID(profile.ID, provider),
		Name:                exportBundleName(profile.Name, provider),
		Tagline:             profile.Summary,
		ShortDescription:    exportShortDescription(profile),
		Goal:                profile.Goal,
		Adapter:             provider,
		Category:            "draft-export",
		Persona:             fallbackString(profile.Identity.Persona, "preset-export"),
		Version:             profile.Version,
		Files:               []string{providerFileName(provider), "docs/overview.md"},
		SafetyNotes:         []string{"Generated from a tested ARC preset draft.", "Review the provider guide before installing into a project."},
		PresetType:          profile.PresetType,
		CompatibleProviders: []string{provider},
		Permissions:         Permissions{Runtime: profile.RuntimePolicy.ExecutionMode},
		MemoryScopes:        exportMemoryScopes(profile.MemoryPolicy.AllowedScopes, exportBundleID(profile.ID, provider)),
		RuntimePolicy:       RuntimePolicy{AutoStopPolicy: profile.RuntimePolicy.AutoStopPolicy},
		QualityGates:        append([]string{}, profile.QualityGates...),
		BudgetProfile:       profile.BudgetProfile,
		Author:              Author{Name: "ARC Preset Studio", Handle: "arc"},
	}
}

func exportShortDescription(profile PresetProfile) []string {
	lines := []string{
		fmt.Sprintf("Best for %s work that targets %s and stays within explicit ARC policy blocks.", profile.PresetType, profile.TargetAgent),
		"Use it when you want a tested preset bundle with readable docs, provider instructions, and evaluation evidence.",
		"Generated from a tested draft, so it is ready for local install and further catalog review.",
	}
	if len(lines) > 3 {
		lines = lines[:3]
	}
	return lines
}

func exportMemoryScopes(scopes []string, bundleID string) []string {
	out := make([]string, 0, len(scopes)+1)
	seen := map[string]struct{}{}
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" || scope == "session" || scope == "project" {
			if scope == "" {
				continue
			}
		}
		if strings.HasPrefix(scope, "presets/") {
			scope = "presets/" + bundleID
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func renderExportReadme(profile PresetProfile, provider string) string {
	lines := []string{
		"# " + exportBundleName(profile.Name, provider),
		"",
		profile.Summary,
		"",
		"## Goal",
		"",
		profile.Goal,
		"",
		"## What This Preset Is For",
		"",
		fmt.Sprintf("- target agent: `%s`", profile.TargetAgent),
		fmt.Sprintf("- compatible provider: `%s`", provider),
		fmt.Sprintf("- autonomy: `%s`", profile.Behavior.AutonomyLevel),
		fmt.Sprintf("- budget profile: `%s`", profile.BudgetProfile),
		"",
		"Install this preset into a project when you want the ARC draft semantics in a reusable provider-specific bundle.",
		"",
	}
	return strings.Join(lines, "\n")
}

func renderExportOverview(draft PresetDraft, provider string) string {
	lines := []string{
		"# Overview",
		"",
		fmt.Sprintf("- preset id: `%s`", exportBundleID(draft.Profile.ID, provider)),
		fmt.Sprintf("- source draft: `%s`", draft.Profile.ID),
		fmt.Sprintf("- provider: `%s`", provider),
		fmt.Sprintf("- status: `%s`", draft.Profile.Status),
		"",
		"## Goal",
		"",
		draft.Profile.Goal,
		"",
		"## Workflow",
		"",
		"- " + strings.Join(draft.Profile.Workflow, "\n- "),
		"",
		"## Quality Gates",
		"",
		"- " + strings.Join(draft.Profile.QualityGates, "\n- "),
		"",
		"## What It Does",
		"",
	}
	for _, item := range draft.Brief.WhatItDoes {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "", "## What It Does Not Do", "")
	for _, item := range draft.Brief.WhatItDoesNot {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "", "## Evaluation Evidence", "")
	for _, item := range draft.EvaluationPack.AcceptanceChecklist {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func renderProviderGuide(profile PresetProfile, provider string) string {
	title := exportBundleName(profile.Name, provider)
	lines := []string{
		"# " + title,
		"",
		fmt.Sprintf("Act as %s.", strings.TrimSpace(profile.Summary)),
		"",
		"Rules:",
		"",
		fmt.Sprintf("- Keep the main goal in view: %s", profile.Goal),
	}
	for _, item := range profile.Behavior.WorkingStyle {
		lines = append(lines, "- Working style: "+item)
	}
	for _, item := range profile.Behavior.ResponseStyle {
		lines = append(lines, "- Response style: "+item)
	}
	for _, item := range profile.QualityGates {
		lines = append(lines, "- Quality gate: "+item)
	}
	for _, item := range profile.FailurePolicy.Rules {
		lines = append(lines, "- Failure policy: "+item)
	}
	if len(profile.ApprovalPolicy.Rules) > 0 {
		for _, item := range profile.ApprovalPolicy.Rules {
			lines = append(lines, "- Approval policy: "+item)
		}
	}
	lines = append(lines, fmt.Sprintf("- Stay within `%s` runtime permissions and `%s` budget mode defaults.", profile.RuntimePolicy.ExecutionMode, profile.BudgetProfile))
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func exportProvidersForDraft(draft PresetDraft) []string {
	providers := normalizeCSVList(draft.Profile.Environment.CompatibleProviders)
	out := make([]string, 0, len(providers))
	for _, provider := range providers {
		switch provider {
		case "codex", "claude":
			out = append(out, provider)
		}
	}
	if len(out) == 0 {
		switch strings.TrimSpace(draft.Profile.TargetAgent) {
		case "codex", "claude":
			out = append(out, strings.TrimSpace(draft.Profile.TargetAgent))
		}
	}
	return out
}

func providerFileName(provider string) string {
	switch provider {
	case "claude":
		return "CLAUDE.md"
	default:
		return "CODEX.md"
	}
}

func exportBundleID(baseID string, provider string) string {
	baseID = strings.TrimSpace(baseID)
	suffix := "-" + strings.TrimSpace(provider)
	if strings.HasSuffix(baseID, suffix) {
		return baseID
	}
	return baseID + suffix
}

func exportBundleName(baseName string, provider string) string {
	label := strings.Title(strings.TrimSpace(provider))
	if strings.Contains(strings.ToLower(baseName), strings.ToLower(label)) {
		return baseName
	}
	return strings.TrimSpace(baseName) + " " + label
}

func exportsRoot(workspaceRoot string) string {
	return project.ProjectFile(workspaceRoot, "presets", "exports")
}

func exportPaths(workspaceRoot string, bundleID string, provider string) PresetExportPaths {
	root := filepath.Join(exportsRoot(workspaceRoot), bundleID)
	return PresetExportPaths{
		Root:          root,
		ManifestPath:  filepath.Join(root, "manifest.yaml"),
		ReadmePath:    filepath.Join(root, "README.md"),
		OverviewPath:  filepath.Join(root, "payload", "docs", "overview.md"),
		ProviderPath:  filepath.Join(root, "payload", providerFileName(provider)),
		BriefJSONPath: filepath.Join(root, "brief.json"),
		EvalJSONPath:  filepath.Join(root, "evaluation.json"),
		SimJSONPath:   filepath.Join(root, "simulation.json"),
		ProfileJSON:   filepath.Join(root, "profile.json"),
	}
}
