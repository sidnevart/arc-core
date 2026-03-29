package app

import (
	"strings"
	"time"
)

func chatProviderTimeout(prompt string) time.Duration {
	if promptRequestsVisualOutput(prompt) {
		return 5 * time.Minute
	}
	return 3 * time.Minute
}

func promptRequestsVisualOutput(prompt string) bool {
	text := strings.ToLower(prompt)
	for _, marker := range []string{
		"мини-симуля", "симуляц", "simulation", "miniapp", "mini app", "мини приложение",
		"демо", "demo", "мини-сайт", "mini-site", "site", "interactive",
		"схем", "диаграм", "diagram", "mermaid", "flowchart",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
