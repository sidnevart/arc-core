package governance

import (
	"strings"
)

type Assessment struct {
	RiskLevel        string   `json:"risk_level"`
	RequiresApproval bool     `json:"requires_approval"`
	Triggers         []string `json:"triggers"`
	Gates            []string `json:"gates"`
}

func Assess(task string) Assessment {
	normalized := strings.ToLower(task)
	triggers := []string{}
	gates := []string{}

	type rule struct {
		keywords []string
		gate     string
		trigger  string
	}

	rules := []rule{
		{keywords: []string{"delete", "remove", "rm ", "drop table", "truncate"}, gate: "destructive shell commands", trigger: "destructive_change"},
		{keywords: []string{"secret", "token", "password", "env", ".env", "credential"}, gate: "secret or config changes", trigger: "secrets"},
		{keywords: []string{"migrate", "migration", "database", "schema"}, gate: "database migration", trigger: "database"},
		{keywords: []string{"deploy", "publish", "release", "npm publish", "docker push"}, gate: "external release or publish", trigger: "publish"},
		{keywords: []string{"push", "pull request", "pr ", "merge"}, gate: "git push or PR creation", trigger: "git_push"},
		{keywords: []string{"ci", "workflow", "github actions", "pipeline"}, gate: "CI/CD modification", trigger: "ci_cd"},
		{keywords: []string{"curl ", "http://", "https://", "api ", "webhook", "mcp"}, gate: "network or external integration", trigger: "network"},
	}

	for _, rule := range rules {
		for _, keyword := range rule.keywords {
			if strings.Contains(normalized, keyword) {
				if !contains(triggers, rule.trigger) {
					triggers = append(triggers, rule.trigger)
				}
				if !contains(gates, rule.gate) {
					gates = append(gates, rule.gate)
				}
				break
			}
		}
	}

	level := "low"
	if len(gates) > 0 {
		level = "high"
	} else if strings.Contains(normalized, "refactor") || strings.Contains(normalized, "adapter") || strings.Contains(normalized, "architecture") {
		level = "medium"
	}

	return Assessment{
		RiskLevel:        level,
		RequiresApproval: len(gates) > 0,
		Triggers:         triggers,
		Gates:            gates,
	}
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
