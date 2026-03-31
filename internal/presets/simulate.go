package presets

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

type PresetSimulationReport struct {
	PresetID        string                 `json:"preset_id"`
	Status          string                 `json:"status"`
	DraftValid      bool                   `json:"draft_valid"`
	Contradictions  []string               `json:"contradictions,omitempty"`
	ScenarioResults []PresetScenarioResult `json:"scenario_results,omitempty"`
	Paths           PresetSimulationPaths  `json:"paths,omitempty"`
	UpdatedAt       string                 `json:"updated_at"`
}

type PresetScenarioResult struct {
	ID     string   `json:"id"`
	Kind   string   `json:"kind"`
	Title  string   `json:"title"`
	Status string   `json:"status"`
	Notes  []string `json:"notes,omitempty"`
}

type PresetSimulationPaths struct {
	Root         string `json:"root,omitempty"`
	JSONPath     string `json:"json_path,omitempty"`
	MarkdownPath string `json:"markdown_path,omitempty"`
}

type PresetDraftPromotionResult struct {
	PresetID       string                 `json:"preset_id"`
	PreviousStatus string                 `json:"previous_status"`
	NewStatus      string                 `json:"new_status"`
	Simulation     PresetSimulationReport `json:"simulation"`
	Paths          PresetDraftPaths       `json:"paths,omitempty"`
}

func SimulateDraft(workspaceRoot string, presetID string) (PresetSimulationReport, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetSimulationReport{}, err
	}
	draft, err := LoadDraft(workspaceRoot, presetID)
	if err != nil {
		return PresetSimulationReport{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	report := PresetSimulationReport{
		PresetID:       draft.Profile.ID,
		DraftValid:     ValidateDraft(draft) == nil,
		Contradictions: detectInterviewContradictions(draft),
		UpdatedAt:      now,
	}
	report.ScenarioResults = simulateScenarios(draft, report.DraftValid, report.Contradictions)
	report.Status = "passed"
	for _, scenario := range report.ScenarioResults {
		if scenario.Status != "passed" {
			report.Status = "blocked"
			break
		}
	}
	report.Paths = simulationPaths(workspaceRoot, presetID)
	if err := project.WriteJSON(report.Paths.JSONPath, report); err != nil {
		return PresetSimulationReport{}, err
	}
	if err := project.WriteString(report.Paths.MarkdownPath, renderSimulationMarkdown(report)); err != nil {
		return PresetSimulationReport{}, err
	}
	return report, nil
}

func LoadSimulationReport(workspaceRoot string, presetID string) (PresetSimulationReport, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetSimulationReport{}, err
	}
	paths := simulationPaths(workspaceRoot, presetID)
	var report PresetSimulationReport
	if err := project.ReadJSON(paths.JSONPath, &report); err != nil {
		return PresetSimulationReport{}, err
	}
	report.Paths = paths
	return report, nil
}

func MarkDraftTested(workspaceRoot string, presetID string) (PresetDraftPromotionResult, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetDraftPromotionResult{}, err
	}
	draft, err := LoadDraft(workspaceRoot, presetID)
	if err != nil {
		return PresetDraftPromotionResult{}, err
	}
	report, err := LoadSimulationReport(workspaceRoot, presetID)
	if err != nil {
		return PresetDraftPromotionResult{}, fmt.Errorf("simulation report is required before marking preset tested: %w", err)
	}
	if report.Status != "passed" || !report.DraftValid || len(report.Contradictions) > 0 {
		return PresetDraftPromotionResult{}, fmt.Errorf("simulation must pass without contradictions before marking preset tested")
	}
	previous := draft.Profile.Status
	now := time.Now().UTC().Format(time.RFC3339)
	draft.Profile.Status = "tested"
	draft.Profile.UpdatedAt = now
	draft.Brief = buildDraftBrief(draft.Profile, now)
	draft.Manifest = buildDraftManifest(draft.Profile)
	draft.EvaluationPack = buildDraftEvaluationPack(draft.Profile, now)
	draft.EvaluationPack.Status = "tested"
	saved, err := SaveDraft(workspaceRoot, draft)
	if err != nil {
		return PresetDraftPromotionResult{}, err
	}
	return PresetDraftPromotionResult{
		PresetID:       saved.Profile.ID,
		PreviousStatus: previous,
		NewStatus:      saved.Profile.Status,
		Simulation:     report,
		Paths:          saved.Paths,
	}, nil
}

func simulateScenarios(draft PresetDraft, draftValid bool, contradictions []string) []PresetScenarioResult {
	results := make([]PresetScenarioResult, 0, len(draft.EvaluationPack.RepresentativeScenarios))
	for _, scenario := range draft.EvaluationPack.RepresentativeScenarios {
		result := PresetScenarioResult{
			ID:    scenario.ID,
			Kind:  scenario.Kind,
			Title: scenario.Title,
		}
		switch scenario.Kind {
		case "typical":
			result.Status = "passed"
			if !draftValid {
				result.Status = "blocked"
				result.Notes = append(result.Notes, "draft bundle is invalid")
			}
			if len(contradictions) > 0 {
				result.Status = "blocked"
				result.Notes = append(result.Notes, contradictions...)
			}
		case "edge":
			result.Status = "passed"
			if len(draft.Profile.Workflow) == 0 || len(draft.Profile.Outputs) == 0 || len(draft.Profile.QualityGates) == 0 {
				result.Status = "warning"
				result.Notes = append(result.Notes, "draft is underspecified for workflow, outputs, or quality gates")
			}
		case "risky":
			result.Status = "passed"
			if (draft.Profile.Behavior.AutonomyLevel == "medium" || draft.Profile.Behavior.AutonomyLevel == "high") && len(draft.Profile.ApprovalPolicy.Rules) == 0 {
				result.Status = "blocked"
				result.Notes = append(result.Notes, "approval policy is missing for elevated autonomy")
			}
			if len(contradictions) > 0 {
				result.Status = "blocked"
				result.Notes = append(result.Notes, "risk scenario is blocked by profile contradictions")
			}
		default:
			result.Status = "warning"
			result.Notes = append(result.Notes, "unknown scenario kind")
		}
		if len(result.Notes) == 0 {
			result.Notes = append(result.Notes, "scenario checks passed")
		}
		results = append(results, result)
	}
	return results
}

func renderSimulationMarkdown(report PresetSimulationReport) string {
	lines := []string{
		"# Preset Simulation Report",
		"",
		fmt.Sprintf("- preset: `%s`", report.PresetID),
		fmt.Sprintf("- status: `%s`", report.Status),
		fmt.Sprintf("- draft valid: `%t`", report.DraftValid),
		"",
	}
	if len(report.Contradictions) > 0 {
		lines = append(lines, "## Contradictions", "")
		for _, item := range report.Contradictions {
			lines = append(lines, "- "+item)
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Scenario Results", "")
	for _, scenario := range report.ScenarioResults {
		lines = append(lines, "### "+scenario.Title, "")
		lines = append(lines, fmt.Sprintf("- kind: `%s`", scenario.Kind))
		lines = append(lines, fmt.Sprintf("- status: `%s`", scenario.Status))
		for _, note := range scenario.Notes {
			lines = append(lines, "- "+note)
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n"
}

func simulationPaths(workspaceRoot string, presetID string) PresetSimulationPaths {
	root := filepath.Join(draftsRoot(workspaceRoot), presetID)
	return PresetSimulationPaths{
		Root:         root,
		JSONPath:     filepath.Join(root, "simulation.json"),
		MarkdownPath: filepath.Join(root, "simulation.md"),
	}
}
