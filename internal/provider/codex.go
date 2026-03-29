package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

type Codex struct{}

func (Codex) Name() string { return "codex" }

func (Codex) CheckInstalled(_ context.Context) Status {
	path, notes, err := resolveBinary("codex")
	if err != nil {
		return Status{
			Name:      "codex",
			Installed: false,
			Notes:     notes,
		}
	}

	return Status{
		Name:         "codex",
		Installed:    true,
		BinaryPath:   path,
		Capabilities: Codex{}.GetCapabilities(context.Background()),
		Notes:        notes,
	}
}

func (Codex) GetCapabilities(_ context.Context) []string {
	return []string{
		"exec",
		"review",
		"resume",
		"json_output",
		"last_message_output",
		"workspace_write_sandbox",
	}
}

func (Codex) RunTask(_ context.Context, req TaskRequest) (TaskResult, error) {
	if req.DryRun {
		return TaskResult{
			ExitCode: 0,
			Notes:    []string{"dry-run enabled; provider execution skipped"},
		}, nil
	}

	args := []string{
		"-a", approvalPolicyForArc(),
		"-C", req.Root,
		"-s", sandboxForRequest(req),
		"exec",
		"--json",
		"-o", req.LastMessageOut,
		"--skip-git-repo-check",
	}
	if req.Model != "" {
		args = append(args, "-m", req.Model)
	}
	args = append(args, req.Prompt)

	return runCodexCommand(req.Root, req.Timeout, args, req.TranscriptOut)
}

func (Codex) ResumeSession(_ context.Context, sessionID string, req TaskRequest) (TaskResult, error) {
	if req.DryRun {
		return TaskResult{
			ExitCode: 0,
			Notes:    []string{"dry-run enabled; provider resume skipped"},
		}, nil
	}

	args := []string{
		"-a", approvalPolicyForArc(),
		"-C", req.Root,
		"-s", sandboxForRequest(req),
		"exec", "resume",
		"--json",
		"-o", req.LastMessageOut,
		"--skip-git-repo-check",
	}
	if sessionID == "" {
		args = append(args, "--last")
	} else {
		args = append(args, sessionID)
	}
	if req.Model != "" {
		args = append(args, "-m", req.Model)
	}
	args = append(args, req.Prompt)

	return runCodexCommand(req.Root, req.Timeout, args, req.TranscriptOut)
}

func (Codex) ApplyProjectScaffold(root string) error {
	return project.WriteString(filepath.Join(root, ".arc", "provider", "AGENTS.md"), `# AGENTS

## ARC Codex Guidance

- Follow the active ARC built-in agent policy for this project.
- Prefer plans, docs, diagrams, demos, and explicit verification over guesses.
- Keep changes bounded and evidence-backed.
- If context is missing, ask for it instead of inventing it.
`)
}

func (Codex) EstimateRisk(task string) string {
	lowRiskKeywords := []string{"docs", "readme", "map", "index", "test"}
	for _, keyword := range lowRiskKeywords {
		if strings.Contains(strings.ToLower(task), keyword) {
			return "low"
		}
	}
	return "medium"
}

func (Codex) CollectTranscript(runDir string) (string, error) {
	path := filepath.Join(runDir, "provider_transcript.jsonl")
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func runCodexCommand(root string, timeout time.Duration, args []string, transcriptPath string) (TaskResult, error) {
	stderrPath := strings.TrimSuffix(transcriptPath, filepath.Ext(transcriptPath)) + ".stderr.log"
	stdout, stderr, err, timedOut := runProviderCommand(
		"codex",
		args,
		root,
		timeout,
		transcriptPath,
		stderrPath,
		"OTEL_SDK_DISABLED=true",
		"OTEL_TRACES_EXPORTER=none",
		"OTEL_METRICS_EXPORTER=none",
		"OTEL_LOGS_EXPORTER=none",
		"OTEL_EXPORTER_OTLP_ENDPOINT=",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=",
	)

	result := TaskResult{
		Command:    append([]string{"codex"}, args...),
		ExitCode:   exitCodeFromErr(err),
		SessionID:  sessionIDFromTranscript(stdout),
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

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func sessionIDFromTranscript(data []byte) string {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] != '{' {
			continue
		}

		var payload map[string]any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		if value := nestedString(payload, "session_id"); value != "" {
			return value
		}
		if value := nestedString(payload, "sessionId"); value != "" {
			return value
		}
	}
	return ""
}

func nestedString(value any, target string) string {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == target {
				if text, ok := child.(string); ok {
					return text
				}
			}
			if found := nestedString(child, target); found != "" {
				return found
			}
		}
	case []any:
		for _, item := range typed {
			if found := nestedString(item, target); found != "" {
				return found
			}
		}
	}
	return ""
}

func sandboxForRequest(req TaskRequest) string {
	if req.ReplyOnly {
		return "read-only"
	}
	return sandboxForMode(req.Mode)
}

func sandboxForMode(mode string) string {
	switch mode {
	case "study":
		return "read-only"
	default:
		return "workspace-write"
	}
}

func approvalPolicyForArc() string {
	return "never"
}
