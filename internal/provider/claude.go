package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

type Claude struct{}

func (Claude) Name() string { return "claude" }

func (Claude) CheckInstalled(_ context.Context) Status {
	path, notes, err := resolveBinary("claude")
	if err != nil {
		return Status{
			Name:      "claude",
			Installed: false,
			Notes:     notes,
		}
	}

	return Status{
		Name:         "claude",
		Installed:    true,
		BinaryPath:   path,
		Capabilities: Claude{}.GetCapabilities(context.Background()),
		Notes:        notes,
	}
}

func (Claude) GetCapabilities(_ context.Context) []string {
	return []string{
		"local_runtime_detection",
		"project_guidance_via_CLAUDE_md",
		"print_mode",
		"json_output",
		"resume",
		"permission_mode",
	}
}

func (Claude) RunTask(_ context.Context, req TaskRequest) (TaskResult, error) {
	if req.DryRun {
		return TaskResult{ExitCode: 0, Notes: []string{"dry-run enabled; claude execution skipped"}}, nil
	}

	args := []string{
		"-p", req.Prompt,
		"--output-format", "json",
		"--permission-mode", permissionModeForMode(req.Mode),
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}

	return runClaudeCommand(req.Timeout, args, req.Root, req.LastMessageOut, req.TranscriptOut)
}

func (Claude) ResumeSession(_ context.Context, sessionID string, req TaskRequest) (TaskResult, error) {
	if req.DryRun {
		return TaskResult{ExitCode: 0, Notes: []string{"dry-run enabled; claude resume skipped"}}, nil
	}

	args := []string{
		"-p",
		"--output-format", "json",
	}
	if sessionID != "" {
		args = append(args, "--resume", sessionID)
	} else {
		args = append(args, "--continue")
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	args = append(args, req.Prompt)

	return runClaudeCommand(req.Timeout, args, req.Root, req.LastMessageOut, req.TranscriptOut)
}

func (Claude) ApplyProjectScaffold(root string) error {
	return project.WriteString(filepath.Join(root, ".arc", "provider", "CLAUDE.md"), `# CLAUDE

## ARC Claude Guidance

- Follow the active ARC built-in agent policy for this project.
- Explain decisions clearly and surface uncertainty instead of guessing.
- Prefer structured plans, reviews, and durable artifacts over vague summaries.
- Keep the human in control when the current agent is Study or Work.
`)
}

func (Claude) EstimateRisk(_ string) string { return "medium" }

func (Claude) CollectTranscript(runDir string) (string, error) {
	path := filepath.Join(runDir, "claude_transcript.log")
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func runClaudeCommand(timeout time.Duration, args []string, root string, lastMessagePath string, transcriptPath string) (TaskResult, error) {
	stderrPath := strings.TrimSuffix(transcriptPath, filepath.Ext(transcriptPath)) + ".stderr.log"
	stdout, stderr, err, timedOut := runProviderCommand("claude", args, root, timeout, transcriptPath, stderrPath)

	sessionID := ""
	lastMessage := strings.TrimSpace(string(stdout))
	var payload map[string]any
	if json.Unmarshal(stdout, &payload) == nil {
		if value, ok := payload["session_id"].(string); ok {
			sessionID = value
		}
		if value, ok := payload["result"].(string); ok && value != "" {
			lastMessage = value
		}
	}
	_ = project.WriteString(lastMessagePath, lastMessage+"\n")

	result := TaskResult{
		Command:    append([]string{"claude"}, args...),
		ExitCode:   exitCodeFromErr(err),
		SessionID:  sessionID,
		StdoutPath: transcriptPath,
		StderrPath: stderrPath,
	}
	if len(stderr) > 0 {
		result.Notes = append(result.Notes, "provider emitted stderr output")
	}
	if timedOut {
		result.Notes = append(result.Notes, "provider command timed out")
	}
	return result, err
}

func permissionModeForMode(mode string) string {
	switch mode {
	case "study":
		return "plan"
	case "hero":
		return "acceptEdits"
	default:
		return "default"
	}
}
