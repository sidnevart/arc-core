package app

import (
	"fmt"
	"strings"

	"agent-os/internal/chat"
	"agent-os/internal/project"
)

type agentActionPolicy struct {
	AllowedActions AllowedSessionActions
}

func allowedActionsForMode(mode string) AllowedSessionActions {
	switch normalizeAgentMode(mode) {
	case "study":
		return AllowedSessionActions{
			AgentID:               "study",
			AgentName:             "Study",
			CanExplain:            true,
			CanPlan:               true,
			CanSafeRun:            false,
			CanDo:                 false,
			StudyFallbackAction:   "plan",
			UnavailableReasonSafe: "Study объясняет и учит. Для него автономные запуски отключены.",
			UnavailableReasonDo:   "Study не делает задачу за пользователя целиком. Он объясняет, разбивает задачу и помогает учиться.",
			Notes: []string{
				"Сначала объясняет и даёт план, а не забирает задачу себе.",
				"Показывает схемы, туториалы и учебные материалы вместо полного автономного выполнения.",
			},
		}
	case "hero":
		return AllowedSessionActions{
			AgentID:    "hero",
			AgentName:  "Hero",
			CanExplain: true,
			CanPlan:    true,
			CanSafeRun: true,
			CanDo:      true,
			Notes: []string{
				"Hero может брать больше ответственности и сам доводить bounded-задачу до результата.",
			},
		}
	default:
		return AllowedSessionActions{
			AgentID:             "work",
			AgentName:           "Work",
			CanExplain:          true,
			CanPlan:             true,
			CanSafeRun:          true,
			CanDo:               true,
			DoRequiresUnlock:    true,
			UnavailableReasonDo: "Work по умолчанию работает вместе с человеком. Для более автономного шага нужен явный unlock.",
			Notes: []string{
				"Сначала планирует и помогает безопасно, а не молча делает работу вместо человека.",
				"Для автономного выполнения нужен явный unlock только на этот запуск.",
			},
		}
	}
}

func normalizeAgentMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "study", "hero":
		return strings.TrimSpace(strings.ToLower(mode))
	default:
		return "work"
	}
}

func (Service) AllowedActions(root string, mode string, sessionID string) (AllowedSessionActions, error) {
	resolvedMode := normalizeAgentMode(mode)
	if strings.TrimSpace(sessionID) != "" {
		resolved, err := discoverProjectMaybe(root)
		if err != nil {
			return AllowedSessionActions{}, err
		}
		session, err := loadSessionMaybe(resolved, sessionID)
		if err != nil {
			return AllowedSessionActions{}, err
		}
		resolvedMode = normalizeAgentMode(session.Mode)
	}
	return allowedActionsForMode(resolvedMode), nil
}

func validateTaskRunPolicy(mode string, dryRun bool, allowAutonomy bool) error {
	policy := allowedActionsForMode(mode)
	if dryRun {
		if policy.CanSafeRun {
			return nil
		}
		return fmt.Errorf(policy.UnavailableReasonSafe)
	}
	if !policy.CanDo {
		return fmt.Errorf(policy.UnavailableReasonDo)
	}
	if policy.DoRequiresUnlock && !allowAutonomy {
		return fmt.Errorf("Work требует явный unlock перед более автономным выполнением.")
	}
	return nil
}

func enforceChatAction(mode string, action string, allowAutonomy bool, prompt string) (string, error) {
	mode = normalizeAgentMode(mode)
	action = normalizeActionID(action)
	trimmedPrompt := strings.TrimSpace(prompt)
	switch mode {
	case "study":
		if action == "do" || action == "safe" {
			preface := "Study policy: do not complete the task autonomously. Explain the approach, break it into small steps, use visual aids or practice prompts when useful, and keep the user in the learning loop."
			return preface + "\n\n" + arcResponseContract(mode) + "\n\nUser request:\n" + trimmedPrompt, nil
		}
	case "work":
		if action == "do" && !allowAutonomy {
			return "", fmt.Errorf("Work сначала помогает совместно. Для автономного выполнения нужен явный unlock.")
		}
	}
	if trimmedPrompt == "" {
		return prompt, nil
	}
	return arcResponseContract(mode) + "\n\nUser request:\n" + trimmedPrompt, nil
}

func normalizeActionID(action string) string {
	switch strings.TrimSpace(strings.ToLower(action)) {
	case "plan", "safe", "do", "review":
		return strings.TrimSpace(strings.ToLower(action))
	default:
		return "explain"
	}
}

func discoverProjectMaybe(root string) (string, error) {
	return project.DiscoverRoot(rootIfEmpty(root))
}

func loadSessionMaybe(root string, sessionID string) (chat.Session, error) {
	return chat.Load(root, sessionID)
}

func arcResponseContract(mode string) string {
	visualRule := "Only emit HTML demo/simulation blocks when the user explicitly asks for a demo, simulation, mini-app, or site."
	switch normalizeAgentMode(mode) {
	case "study", "hero":
		visualRule = "If a diagram, demo, or simulation would materially improve understanding or delivery, you may emit it even without an explicit visual request."
	}
	return strings.Join([]string{
		"ARC response contract:",
		"- This is a reply-only ARC chat surface. Do not edit project files, do not apply patches, and do not attempt repository modifications or long-running workspace tasks.",
		"- Write the normal answer in clear Markdown outside of special blocks.",
		"- When you produce a visual or interactive result, use one of these fenced blocks exactly: ```arc-diagram mermaid```, ```arc-diagram svg```, ```arc-document markdown```, ```arc-demo html```, ```arc-simulation html```.",
		"- If you use a fenced output block, start it with an optional `title: ...` line when a clear title helps.",
		"- Use `arc-diagram mermaid` for flow/process diagrams, `arc-document markdown` for polished long-form notes, `arc-demo html` for launchable demos, and `arc-simulation html` for interactive teaching miniapps.",
		"- If the user asks for a demo, simulation, mini-app, site, or interactive explanation, return the full HTML inside the fenced block instead of saying how you would build it.",
		"- Keep the prose concise and let ARC render the structured output separately.",
		"- " + visualRule,
	}, "\n")
}
