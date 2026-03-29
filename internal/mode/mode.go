package mode

import (
	"strconv"
	"strings"
)

type Definition struct {
	Name        string
	Goal        string
	Autonomy    string
	Policies    []string
	Sequence    []string
	Artifacts   []string
	Roles       []string
	Forbidden   []string
	HelpLadder  []string
	Description string
}

func ByName(name string) Definition {
	switch name {
	case "study":
		return Definition{
			Name:        "study",
			Goal:        "Improve understanding without replacing the learner's thinking.",
			Autonomy:    "low",
			Description: "Teaching-first mode with a strict help ladder.",
			Policies: []string{
				"Ask clarifying questions before giving a solution.",
				"Challenge weak reasoning and request evidence.",
				"Record knowledge gaps and keep the user engaged in the reasoning.",
			},
			HelpLadder: []string{
				"clarifying question",
				"hint",
				"single-step explanation",
				"analogous example",
				"partial solution",
				"full solution only after explicit unlock",
			},
			Artifacts: []string{"learning_goal.md", "current_understanding.md", "knowledge_gaps.md", "challenge_log.md", "practice_tasks.md"},
			Roles:     []string{"tutor", "challenger", "evaluator", "resource-scout", "visualizer"},
			Forbidden: []string{"solving the whole task before the user demonstrates understanding"},
		}
	case "hero":
		return Definition{
			Name:        "hero",
			Goal:        "Close a ticket-sized task autonomously with guardrails and reproducible artifacts.",
			Autonomy:    "high",
			Description: "Autonomous delivery mode with a deterministic pipeline.",
			Policies: []string{
				"Do not invent unknowns; use UNKNOWN when evidence is missing.",
				"Search for evidence before asking for human input.",
				"All risky actions require explicit approval outside the provider runtime.",
			},
			Sequence: []string{
				"intake",
				"planner builds business and tech spec",
				"spec reviewer finds holes",
				"context requester gathers missing context",
				"implementer changes code and tests",
				"verifier runs checks",
				"reviewer critiques the result",
				"docs agent updates docs",
			},
			Artifacts: []string{"ticket_spec.md", "business_spec.md", "tech_spec.md", "unknowns.md", "question_bundle.md", "implementation_log.md", "verification_report.md", "review_report.md", "docs_delta.md"},
			Roles:     []string{"planner", "spec-reviewer", "context-requester", "implementer", "verifier", "reviewer", "help-requester", "docs-agent", "orchestrator"},
		}
	default:
		return Definition{
			Name:        "work",
			Goal:        "Improve engineering judgment while still delivering useful project progress.",
			Autonomy:    "medium",
			Description: "Engineering support mode with critique before implementation.",
			Policies: []string{
				"Map the system before coding.",
				"State unknowns before implementation.",
				"Challenge the proposed approach and surface weak spots.",
			},
			Sequence: []string{
				"understand task",
				"find impacted system parts",
				"build flow or sequence",
				"state unknowns",
				"critique the approach",
				"implement routine parts or approved work",
			},
			Artifacts: []string{"task_map.md", "system_flow.md", "solution_options.md", "unknowns.md", "validation_checklist.md"},
			Roles:     []string{"mapper", "mentor", "critic", "implementer-lite", "verifier", "docs-agent"},
		}
	}
}

func Markdown(def Definition) string {
	var b strings.Builder
	b.WriteString("# Mode Policy\n\n")
	b.WriteString("Name: " + def.Name + "\n")
	b.WriteString("Goal: " + def.Goal + "\n")
	b.WriteString("Autonomy: " + def.Autonomy + "\n\n")
	if len(def.Policies) > 0 {
		b.WriteString("Policies:\n")
		for _, item := range def.Policies {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}
	if len(def.Sequence) > 0 {
		b.WriteString("Default sequence:\n")
		for i, item := range def.Sequence {
			b.WriteString("- " + strconv.Itoa(i+1) + ". " + item + "\n")
		}
		b.WriteString("\n")
	}
	if len(def.HelpLadder) > 0 {
		b.WriteString("Help ladder:\n")
		for _, item := range def.HelpLadder {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}
	if len(def.Artifacts) > 0 {
		b.WriteString("Artifacts:\n")
		for _, item := range def.Artifacts {
			b.WriteString("- " + item + "\n")
		}
		b.WriteString("\n")
	}
	if len(def.Roles) > 0 {
		b.WriteString("Roles:\n")
		for _, item := range def.Roles {
			b.WriteString("- " + item + "\n")
		}
	}
	return b.String()
}
