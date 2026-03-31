package presets

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

type PresetInterviewSession struct {
	ID              string                    `json:"id"`
	DraftID         string                    `json:"draft_id"`
	Mode            string                    `json:"mode"`
	Status          string                    `json:"status"`
	CurrentQuestion *PresetInterviewQuestion  `json:"current_question,omitempty"`
	Questions       []PresetInterviewQuestion `json:"questions,omitempty"`
	Answers         []PresetInterviewAnswer   `json:"answers,omitempty"`
	AnsweredCount   int                       `json:"answered_count"`
	QuestionCount   int                       `json:"question_count"`
	Confidence      int                       `json:"confidence"`
	Contradictions  []string                  `json:"contradictions,omitempty"`
	SuggestedFixes  []PresetInterviewFix      `json:"suggested_fixes,omitempty"`
	Readiness       PresetInterviewReadiness  `json:"readiness"`
	CreatedAt       string                    `json:"created_at"`
	UpdatedAt       string                    `json:"updated_at"`
}

type PresetInterviewReadiness struct {
	DraftValid      bool     `json:"draft_valid"`
	SimulationReady bool     `json:"simulation_ready"`
	SaveReady       bool     `json:"save_ready"`
	Blockers        []string `json:"blockers,omitempty"`
}

type PresetInterviewQuestion struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Prompt      string `json:"prompt"`
	Help        string `json:"help,omitempty"`
	Expected    string `json:"expected,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

type PresetInterviewAnswer struct {
	Key        string   `json:"key"`
	Value      string   `json:"value"`
	RecordedAt string   `json:"recorded_at"`
	AppliedTo  []string `json:"applied_to,omitempty"`
}

type PresetInterviewFix struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Updates        []string `json:"updates,omitempty"`
	AutoApplicable bool     `json:"auto_applicable"`
}

type PresetInterviewRemediationResult struct {
	Session      PresetInterviewSession `json:"session"`
	AppliedFixes []PresetInterviewFix   `json:"applied_fixes,omitempty"`
}

type StartInterviewOptions struct {
	WorkspaceRoot string
	DraftID       string
	Mode          string
}

type AnswerInterviewOptions struct {
	WorkspaceRoot string
	SessionID     string
	Answer        string
}

func StartInterview(opts StartInterviewOptions) (PresetInterviewSession, error) {
	if err := project.RequireProject(opts.WorkspaceRoot); err != nil {
		return PresetInterviewSession{}, err
	}
	draft, err := LoadDraft(opts.WorkspaceRoot, strings.TrimSpace(opts.DraftID))
	if err != nil {
		return PresetInterviewSession{}, err
	}
	mode := normalizeInterviewMode(opts.Mode)
	now := time.Now().UTC().Format(time.RFC3339)
	session := PresetInterviewSession{
		ID:            fmt.Sprintf("%s-%s", draft.Profile.ID, time.Now().UTC().Format("20060102T150405Z")),
		DraftID:       draft.Profile.ID,
		Mode:          mode,
		Status:        "active",
		Questions:     interviewQuestions(mode, draft),
		Answers:       []PresetInterviewAnswer{},
		QuestionCount: len(interviewQuestions(mode, draft)),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := session.refreshState(opts.WorkspaceRoot); err != nil {
		return PresetInterviewSession{}, err
	}
	if err := saveInterviewSession(opts.WorkspaceRoot, session); err != nil {
		return PresetInterviewSession{}, err
	}
	return session, nil
}

func AnswerInterview(opts AnswerInterviewOptions) (PresetInterviewSession, error) {
	if err := project.RequireProject(opts.WorkspaceRoot); err != nil {
		return PresetInterviewSession{}, err
	}
	session, err := LoadInterview(opts.WorkspaceRoot, strings.TrimSpace(opts.SessionID))
	if err != nil {
		return PresetInterviewSession{}, err
	}
	if session.Status == "completed" {
		return PresetInterviewSession{}, fmt.Errorf("interview %s is already completed", session.ID)
	}
	answer := strings.TrimSpace(opts.Answer)
	if answer == "" {
		return PresetInterviewSession{}, fmt.Errorf("answer is required")
	}
	if session.CurrentQuestion == nil {
		return PresetInterviewSession{}, fmt.Errorf("interview %s has no current question", session.ID)
	}
	updateOpts := DraftUpdateOptions{
		WorkspaceRoot: opts.WorkspaceRoot,
		ID:            session.DraftID,
	}
	appliedTo, err := applyInterviewAnswer(&updateOpts, session.CurrentQuestion.Key, answer)
	if err != nil {
		return PresetInterviewSession{}, err
	}
	if _, err := UpdateDraft(updateOpts); err != nil {
		return PresetInterviewSession{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	session.Answers = append(session.Answers, PresetInterviewAnswer{
		Key:        session.CurrentQuestion.Key,
		Value:      answer,
		RecordedAt: now,
		AppliedTo:  appliedTo,
	})
	session.UpdatedAt = now
	if err := session.refreshState(opts.WorkspaceRoot); err != nil {
		return PresetInterviewSession{}, err
	}
	if err := saveInterviewSession(opts.WorkspaceRoot, session); err != nil {
		return PresetInterviewSession{}, err
	}
	return session, nil
}

func LoadInterview(workspaceRoot string, sessionID string) (PresetInterviewSession, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetInterviewSession{}, err
	}
	var session PresetInterviewSession
	if err := project.ReadJSON(interviewSessionPath(workspaceRoot, sessionID), &session); err != nil {
		return PresetInterviewSession{}, err
	}
	if err := session.refreshState(workspaceRoot); err != nil {
		return PresetInterviewSession{}, err
	}
	return session, nil
}

func RemediateInterview(workspaceRoot string, sessionID string) (PresetInterviewRemediationResult, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetInterviewRemediationResult{}, err
	}
	session, err := LoadInterview(workspaceRoot, strings.TrimSpace(sessionID))
	if err != nil {
		return PresetInterviewRemediationResult{}, err
	}
	draft, err := LoadDraft(workspaceRoot, session.DraftID)
	if err != nil {
		return PresetInterviewRemediationResult{}, err
	}
	fixes := suggestedInterviewFixes(draft)
	applied := make([]PresetInterviewFix, 0, len(fixes))
	updateOpts := DraftUpdateOptions{
		WorkspaceRoot: workspaceRoot,
		ID:            session.DraftID,
	}
	changed := false
	for _, fix := range fixes {
		if !fix.AutoApplicable {
			continue
		}
		if applyInterviewFix(&updateOpts, draft, fix) {
			applied = append(applied, fix)
			changed = true
		}
	}
	if changed {
		if _, err := UpdateDraft(updateOpts); err != nil {
			return PresetInterviewRemediationResult{}, err
		}
	}
	refreshed, err := LoadInterview(workspaceRoot, session.ID)
	if err != nil {
		return PresetInterviewRemediationResult{}, err
	}
	return PresetInterviewRemediationResult{
		Session:      refreshed,
		AppliedFixes: applied,
	}, nil
}

func (s *PresetInterviewSession) refreshState(workspaceRoot string) error {
	draft, err := LoadDraft(workspaceRoot, s.DraftID)
	if err != nil {
		return err
	}
	s.refreshProgress()
	s.Contradictions = detectInterviewContradictions(draft)
	s.SuggestedFixes = suggestedInterviewFixes(draft)
	s.Readiness = buildInterviewReadiness(draft, s.CurrentQuestion == nil, s.Contradictions)
	return nil
}

func (s *PresetInterviewSession) refreshProgress() {
	answered := map[string]PresetInterviewAnswer{}
	for _, answer := range s.Answers {
		answered[strings.TrimSpace(answer.Key)] = answer
	}
	s.QuestionCount = len(s.Questions)
	s.AnsweredCount = len(answered)
	s.CurrentQuestion = nil
	for i := range s.Questions {
		key := strings.TrimSpace(s.Questions[i].Key)
		if _, ok := answered[key]; ok {
			continue
		}
		question := s.Questions[i]
		s.CurrentQuestion = &question
		break
	}
	if s.QuestionCount == 0 {
		s.Confidence = 100
		s.Status = "completed"
		return
	}
	s.Confidence = (s.AnsweredCount * 100) / s.QuestionCount
	if s.CurrentQuestion == nil {
		s.Status = "completed"
		s.Confidence = 100
	} else if strings.TrimSpace(s.Status) == "" {
		s.Status = "active"
	}
}

func normalizeInterviewMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "", "quick":
		return "quick"
	case "deep":
		return "deep"
	case "import-refine", "import_refine":
		return "import_refine"
	default:
		return "quick"
	}
}

func detectInterviewContradictions(draft PresetDraft) []string {
	var contradictions []string
	workflow := toSet(draft.Profile.Workflow)
	qualityGates := toSet(draft.Profile.QualityGates)
	outputs := toSet(draft.Profile.Outputs)
	if outputs["evaluation_pack"] && !workflow["simulate"] {
		contradictions = append(contradictions, "evaluation_pack output is declared but workflow does not include simulate")
	}
	if workflow["publish"] && !qualityGates["validation_ready"] {
		contradictions = append(contradictions, "publish workflow is present but quality_gates do not include validation_ready")
	}
	if draft.Profile.Behavior.AutonomyLevel == "high" && (draft.Profile.BudgetProfile == "ultra_safe" || draft.Profile.BudgetProfile == "emergency_low_limit") {
		contradictions = append(contradictions, "high autonomy conflicts with an aggressively constrained budget profile")
	}
	return contradictions
}

func suggestedInterviewFixes(draft PresetDraft) []PresetInterviewFix {
	fixes := []PresetInterviewFix{}
	workflow := toSet(draft.Profile.Workflow)
	qualityGates := toSet(draft.Profile.QualityGates)
	outputs := toSet(draft.Profile.Outputs)
	if outputs["evaluation_pack"] && !workflow["simulate"] {
		fixes = append(fixes, PresetInterviewFix{
			ID:             "add-simulate-to-workflow",
			Title:          "Add simulate to workflow",
			Summary:        "The draft declares `evaluation_pack`, so the workflow should include `simulate` before save/publish.",
			Updates:        []string{"profile.workflow"},
			AutoApplicable: true,
		})
	}
	if workflow["publish"] && !qualityGates["validation_ready"] {
		fixes = append(fixes, PresetInterviewFix{
			ID:             "add-validation-ready-gate",
			Title:          "Add validation_ready quality gate",
			Summary:        "Publish is present in the workflow, so `validation_ready` should be part of the quality gates.",
			Updates:        []string{"profile.quality_gates"},
			AutoApplicable: true,
		})
	}
	if draft.Profile.Behavior.AutonomyLevel == "high" && (draft.Profile.BudgetProfile == "ultra_safe" || draft.Profile.BudgetProfile == "emergency_low_limit") {
		fixes = append(fixes, PresetInterviewFix{
			ID:             "resolve-autonomy-budget-mismatch",
			Title:          "Resolve autonomy/budget mismatch",
			Summary:        "High autonomy conflicts with a strongly constrained budget profile. Decide whether autonomy should drop or the budget profile should loosen.",
			Updates:        []string{"profile.behavior.autonomy_level", "profile.budget_profile"},
			AutoApplicable: false,
		})
	}
	return fixes
}

func buildInterviewReadiness(draft PresetDraft, interviewComplete bool, contradictions []string) PresetInterviewReadiness {
	readiness := PresetInterviewReadiness{}
	if err := ValidateDraft(draft); err == nil {
		readiness.DraftValid = true
	} else {
		readiness.Blockers = append(readiness.Blockers, err.Error())
	}
	readiness.Blockers = append(readiness.Blockers, contradictions...)
	readiness.SimulationReady = readiness.DraftValid && len(contradictions) == 0
	readiness.SaveReady = readiness.SimulationReady && interviewComplete
	if !interviewComplete {
		readiness.Blockers = append(readiness.Blockers, "interview still has unanswered questions")
	}
	return readiness
}

func toSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		out[key] = true
	}
	return out
}

func interviewQuestions(mode string, draft PresetDraft) []PresetInterviewQuestion {
	common := []PresetInterviewQuestion{
		{
			Key:         "goal",
			Title:       "Real Outcome",
			Prompt:      "What should this preset reliably help the user achieve?",
			Help:        "Use one concrete outcome sentence. Avoid vague 'be helpful' language.",
			Expected:    "single outcome sentence",
			Placeholder: draft.Profile.Goal,
		},
		{
			Key:         "outputs",
			Title:       "Artifacts",
			Prompt:      "Which artifacts or outputs should this preset produce? Use comma-separated values.",
			Help:        "Examples: preset_brief, preset_manifest, evaluation_pack, lesson_plan.",
			Expected:    "comma-separated list",
			Placeholder: strings.Join(draft.Profile.Outputs, ", "),
		},
		{
			Key:         "workflow",
			Title:       "Workflow",
			Prompt:      "What is the ideal step-by-step workflow? Use comma-separated stages in order.",
			Help:        "Examples: interview, normalize, simulate, refine, save.",
			Expected:    "ordered comma-separated list",
			Placeholder: strings.Join(draft.Profile.Workflow, ", "),
		},
	}
	switch mode {
	case "deep":
		return append(common,
			PresetInterviewQuestion{
				Key:         "summary",
				Title:       "Operator Summary",
				Prompt:      "How would you describe this preset to an operator in 1-2 sentences?",
				Help:        "This becomes the short operational summary shown in draft artifacts.",
				Expected:    "1-2 concise sentences",
				Placeholder: draft.Profile.Summary,
			},
			PresetInterviewQuestion{
				Key:         "non_goals",
				Title:       "Non-Goals",
				Prompt:      "What should this preset explicitly avoid doing? Use comma-separated items.",
				Help:        "Use crisp boundaries, not general fears.",
				Expected:    "comma-separated list",
				Placeholder: strings.Join(draft.Profile.NonGoals, ", "),
			},
			PresetInterviewQuestion{
				Key:         "inputs",
				Title:       "Inputs",
				Prompt:      "What inputs or user context does the preset need? Use comma-separated values.",
				Help:        "Examples: repo_path, user_goal, constraints, target_audience.",
				Expected:    "comma-separated list",
				Placeholder: strings.Join(draft.Profile.Inputs, ", "),
			},
			PresetInterviewQuestion{
				Key:         "quality_gates",
				Title:       "Quality Gates",
				Prompt:      "Which quality gates must pass before this preset is considered ready? Use comma-separated values.",
				Help:        "Examples: profile_complete, simulation_ready, validation_ready.",
				Expected:    "comma-separated list",
				Placeholder: strings.Join(draft.Profile.QualityGates, ", "),
			},
			PresetInterviewQuestion{
				Key:         "autonomy",
				Title:       "Autonomy",
				Prompt:      "What autonomy level should this preset have? Use low, medium, or high.",
				Help:        "Higher autonomy requires stronger approval policy coverage.",
				Expected:    "low|medium|high",
				Placeholder: draft.Profile.Behavior.AutonomyLevel,
			},
			PresetInterviewQuestion{
				Key:         "budget_profile",
				Title:       "Budget Bias",
				Prompt:      "Which budget profile best matches this preset? Use balanced, deep_work, ultra_safe, or emergency_low_limit.",
				Help:        "Choose the default cost/risk posture, not a one-off override.",
				Expected:    "budget profile id",
				Placeholder: draft.Profile.BudgetProfile,
			},
		)
	case "import_refine":
		return []PresetInterviewQuestion{
			{
				Key:         "summary",
				Title:       "Imported Summary",
				Prompt:      "Refine the imported preset summary into a cleaner operator-facing description.",
				Help:        "Keep it sharp and practical.",
				Expected:    "1-2 concise sentences",
				Placeholder: draft.Profile.Summary,
			},
			{
				Key:         "non_goals",
				Title:       "Safety Boundaries",
				Prompt:      "Which non-goals should be made explicit before this preset is saved? Use comma-separated items.",
				Help:        "Call out what this preset should not do by default.",
				Expected:    "comma-separated list",
				Placeholder: strings.Join(draft.Profile.NonGoals, ", "),
			},
			{
				Key:         "quality_gates",
				Title:       "Readiness Gates",
				Prompt:      "Which gates must pass before this imported preset is treated as ready? Use comma-separated values.",
				Help:        "These gates should be concrete and inspectable.",
				Expected:    "comma-separated list",
				Placeholder: strings.Join(draft.Profile.QualityGates, ", "),
			},
		}
	default:
		return append(common, PresetInterviewQuestion{
			Key:         "quality_gates",
			Title:       "Readiness Gates",
			Prompt:      "Which quality gates should ARC enforce before this preset is considered ready? Use comma-separated values.",
			Help:        "Keep the list short and operator-readable.",
			Expected:    "comma-separated list",
			Placeholder: strings.Join(draft.Profile.QualityGates, ", "),
		})
	}
}

func applyInterviewAnswer(opts *DraftUpdateOptions, key string, answer string) ([]string, error) {
	value := strings.TrimSpace(answer)
	switch strings.TrimSpace(key) {
	case "summary":
		opts.Summary = value
		return []string{"profile.summary", "brief.summary", "manifest.tagline"}, nil
	case "goal":
		opts.Goal = value
		return []string{"profile.goal", "brief.what_it_does", "evaluation.scenarios"}, nil
	case "non_goals":
		opts.NonGoals = []string{value}
		return []string{"profile.non_goals", "brief.what_it_does_not"}, nil
	case "inputs":
		opts.Inputs = []string{value}
		return []string{"profile.inputs"}, nil
	case "outputs":
		opts.Outputs = []string{value}
		return []string{"profile.outputs", "brief.what_it_does", "evaluation.scenarios"}, nil
	case "workflow":
		opts.Workflow = []string{value}
		return []string{"profile.workflow", "brief.how_to_use"}, nil
	case "quality_gates":
		opts.QualityGates = []string{value}
		return []string{"profile.quality_gates", "brief.what_it_does", "evaluation.acceptance_checklist"}, nil
	case "autonomy":
		opts.AutonomyLevel = value
		return []string{"profile.behavior.autonomy_level"}, nil
	case "budget_profile":
		opts.BudgetProfile = value
		return []string{"profile.budget_profile", "manifest.budget_profile"}, nil
	default:
		return nil, fmt.Errorf("unsupported interview question key %q", key)
	}
}

func applyInterviewFix(opts *DraftUpdateOptions, draft PresetDraft, fix PresetInterviewFix) bool {
	changed := false
	switch fix.ID {
	case "add-simulate-to-workflow":
		workflow := append([]string{}, draft.Profile.Workflow...)
		if !toSet(workflow)["simulate"] {
			workflow = append(workflow, "simulate")
			opts.Workflow = workflow
			changed = true
		}
	case "add-validation-ready-gate":
		gates := append([]string{}, draft.Profile.QualityGates...)
		if !toSet(gates)["validation_ready"] {
			gates = append(gates, "validation_ready")
			opts.QualityGates = gates
			changed = true
		}
	}
	return changed
}

func saveInterviewSession(workspaceRoot string, session PresetInterviewSession) error {
	return project.WriteJSON(interviewSessionPath(workspaceRoot, session.ID), session)
}

func interviewSessionPath(workspaceRoot string, sessionID string) string {
	return filepath.Join(project.ProjectFile(workspaceRoot, "presets", "interviews"), sessionID+".json")
}
