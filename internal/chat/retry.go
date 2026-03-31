package chat

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
	"agent-os/internal/provider"
)

type retryDecision struct {
	Allow        bool
	Reason       string
	RetryTimeout time.Duration
}

type retryArtifact struct {
	AttemptCount       int           `json:"attempt_count"`
	Status             string        `json:"status"`
	Reason             string        `json:"reason,omitempty"`
	InitialTimeout     time.Duration `json:"initial_timeout"`
	RetryTimeout       time.Duration `json:"retry_timeout,omitempty"`
	InitialError       string        `json:"initial_error,omitempty"`
	RetriedAt          string        `json:"retried_at,omitempty"`
	RecoveredAt        string        `json:"recovered_at,omitempty"`
	FailedAfterRetryAt string        `json:"failed_after_retry_at,omitempty"`
}

func shouldRetryChatTurn(adapter provider.Adapter, req provider.TaskRequest, err error, result provider.TaskResult) retryDecision {
	if err == nil || adapter == nil {
		return retryDecision{}
	}
	if req.DryRun || !req.ReplyOnly {
		return retryDecision{}
	}
	if !strings.EqualFold(strings.TrimSpace(adapter.Name()), "codex") {
		return retryDecision{}
	}
	if !promptRequestsVisualOutput(req.Prompt) {
		return retryDecision{}
	}

	combined := strings.ToLower(strings.TrimSpace(enrichErrorWithStderr(err, result.StderrPath)))
	if !strings.Contains(combined, "failed to refresh available models") {
		return retryDecision{}
	}
	if !strings.Contains(combined, "timed out after") && !strings.Contains(combined, "timeout waiting for child process to exit") {
		return retryDecision{}
	}

	return retryDecision{
		Allow:        true,
		Reason:       "codex_model_refresh_timeout",
		RetryTimeout: retryTimeoutForChatTurn(req.Timeout),
	}
}

func retryTimeoutForChatTurn(initial time.Duration) time.Duration {
	base := initial + (2 * time.Minute)
	if base < 6*time.Minute {
		base = 6 * time.Minute
	}
	if base > 10*time.Minute {
		base = 10 * time.Minute
	}
	return base
}

func promptRequestsVisualOutput(prompt string) bool {
	text := strings.ToLower(prompt)
	for _, marker := range []string{
		"мини-симуля", "симуляц", "simulation", "miniapp", "mini app", "мини приложение", "мини апп",
		"демо", "demo", "мини-сайт", "mini-site", "interactive",
		"схем", "диаграм", "diagram", "mermaid", "flowchart",
		"html", "<svg",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func writeRetryArtifact(runDir string, turnPrefix string, artifact retryArtifact) (string, string, error) {
	jsonPath := filepathForRetryJSON(runDir, turnPrefix)
	mdPath := filepathForRetryMD(runDir, turnPrefix)
	if err := project.WriteJSON(jsonPath, artifact); err != nil {
		return "", "", err
	}
	markdown := []string{
		"# Chat Retry",
		"",
		fmt.Sprintf("- status: `%s`", strings.TrimSpace(artifact.Status)),
		fmt.Sprintf("- attempts: `%d`", artifact.AttemptCount),
		fmt.Sprintf("- initial timeout: `%s`", artifact.InitialTimeout),
	}
	if strings.TrimSpace(artifact.Reason) != "" {
		markdown = append(markdown, fmt.Sprintf("- reason: `%s`", strings.TrimSpace(artifact.Reason)))
	}
	if artifact.RetryTimeout > 0 {
		markdown = append(markdown, fmt.Sprintf("- retry timeout: `%s`", artifact.RetryTimeout))
	}
	if strings.TrimSpace(artifact.InitialError) != "" {
		markdown = append(markdown, "", "## Initial Error", "", artifact.InitialError)
	}
	if strings.TrimSpace(artifact.RetriedAt) != "" {
		markdown = append(markdown, "", fmt.Sprintf("- retried at: `%s`", strings.TrimSpace(artifact.RetriedAt)))
	}
	if strings.TrimSpace(artifact.RecoveredAt) != "" {
		markdown = append(markdown, fmt.Sprintf("- recovered at: `%s`", strings.TrimSpace(artifact.RecoveredAt)))
	}
	if strings.TrimSpace(artifact.FailedAfterRetryAt) != "" {
		markdown = append(markdown, fmt.Sprintf("- failed after retry at: `%s`", strings.TrimSpace(artifact.FailedAfterRetryAt)))
	}
	if err := project.WriteString(mdPath, strings.Join(markdown, "\n")+"\n"); err != nil {
		return "", "", err
	}
	return jsonPath, mdPath, nil
}

func filepathForRetryJSON(runDir string, turnPrefix string) string {
	return filepath.Join(strings.TrimSpace(runDir), strings.TrimSpace(turnPrefix)+".retry.json")
}

func filepathForRetryMD(runDir string, turnPrefix string) string {
	return filepath.Join(strings.TrimSpace(runDir), strings.TrimSpace(turnPrefix)+".retry.md")
}
