package presets

import (
	"testing"

	"agent-os/internal/project"
)

func TestInterviewStartAnswerAndShowMutateDraft(t *testing.T) {
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
		ID:            "preset-author",
		Name:          "Preset Author",
		Summary:       "Initial summary",
		Goal:          "Initial goal",
		TargetAgent:   "codex",
	}); err != nil {
		t.Fatal(err)
	}

	session, err := StartInterview(StartInterviewOptions{
		WorkspaceRoot: root,
		DraftID:       "preset-author",
		Mode:          "quick",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.CurrentQuestion == nil || session.CurrentQuestion.Key != "goal" {
		t.Fatalf("expected goal question first, got %#v", session.CurrentQuestion)
	}

	session, err = AnswerInterview(AnswerInterviewOptions{
		WorkspaceRoot: root,
		SessionID:     session.ID,
		Answer:        "Help authors turn vague agent ideas into reviewable preset drafts.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.AnsweredCount != 1 {
		t.Fatalf("expected one answered question, got %#v", session)
	}

	draft, err := LoadDraft(root, "preset-author")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Profile.Goal != "Help authors turn vague agent ideas into reviewable preset drafts." {
		t.Fatalf("expected updated goal, got %#v", draft.Profile)
	}

	loaded, err := LoadInterview(root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentQuestion == nil || loaded.CurrentQuestion.Key != "outputs" {
		t.Fatalf("expected outputs as next question, got %#v", loaded.CurrentQuestion)
	}
	if !loaded.Readiness.SimulationReady {
		t.Fatalf("expected simulation readiness while draft stays valid, got %#v", loaded.Readiness)
	}
	if loaded.Readiness.SaveReady {
		t.Fatalf("expected save to stay blocked while interview is incomplete, got %#v", loaded.Readiness)
	}
}

func TestInterviewDetectsWorkflowArtifactContradiction(t *testing.T) {
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
		ID:            "contradiction-draft",
		Name:          "Contradiction Draft",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateDraft(DraftUpdateOptions{
		WorkspaceRoot: root,
		ID:            "contradiction-draft",
		Outputs:       []string{"evaluation_pack"},
		Workflow:      []string{"interview", "refine", "save"},
	}); err != nil {
		t.Fatal(err)
	}
	session, err := StartInterview(StartInterviewOptions{
		WorkspaceRoot: root,
		DraftID:       "contradiction-draft",
		Mode:          "quick",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(session.Contradictions) == 0 {
		t.Fatalf("expected contradiction, got %#v", session)
	}
	if session.Readiness.SimulationReady {
		t.Fatalf("expected simulation readiness to be blocked, got %#v", session.Readiness)
	}
	if len(session.SuggestedFixes) == 0 || session.SuggestedFixes[0].ID != "add-simulate-to-workflow" {
		t.Fatalf("expected suggested fix for simulate workflow, got %#v", session.SuggestedFixes)
	}
}

func TestInterviewRemediateAppliesDeterministicFixes(t *testing.T) {
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
		ID:            "remediate-draft",
		Name:          "Remediate Draft",
		Summary:       "summary",
		Goal:          "goal",
		TargetAgent:   "codex",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdateDraft(DraftUpdateOptions{
		WorkspaceRoot: root,
		ID:            "remediate-draft",
		Outputs:       []string{"evaluation_pack"},
		Workflow:      []string{"interview", "refine", "publish"},
		QualityGates:  []string{"profile_complete"},
	}); err != nil {
		t.Fatal(err)
	}
	session, err := StartInterview(StartInterviewOptions{
		WorkspaceRoot: root,
		DraftID:       "remediate-draft",
		Mode:          "quick",
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := RemediateInterview(root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AppliedFixes) != 2 {
		t.Fatalf("expected 2 applied fixes, got %#v", result.AppliedFixes)
	}
	draft, err := LoadDraft(root, "remediate-draft")
	if err != nil {
		t.Fatal(err)
	}
	if !toSet(draft.Profile.Workflow)["simulate"] {
		t.Fatalf("expected simulate to be added, got %#v", draft.Profile.Workflow)
	}
	if !toSet(draft.Profile.QualityGates)["validation_ready"] {
		t.Fatalf("expected validation_ready to be added, got %#v", draft.Profile.QualityGates)
	}
	if len(result.Session.Contradictions) != 0 {
		t.Fatalf("expected contradictions to be cleared, got %#v", result.Session.Contradictions)
	}
}
