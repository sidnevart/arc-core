package provider

import (
	"context"
	"fmt"
	"time"
)

type Status struct {
	Name         string   `json:"name"`
	Installed    bool     `json:"installed"`
	BinaryPath   string   `json:"binary_path,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Notes        []string `json:"notes,omitempty"`
}

type TaskRequest struct {
	Root           string
	Prompt         string
	Mode           string
	Model          string
	RunDir         string
	LastMessageOut string
	TranscriptOut  string
	ReplyOnly      bool
	DryRun         bool
	Timeout        time.Duration
}

type TaskResult struct {
	Command    []string `json:"command,omitempty"`
	ExitCode   int      `json:"exit_code"`
	SessionID  string   `json:"session_id,omitempty"`
	Notes      []string `json:"notes,omitempty"`
	StdoutPath string   `json:"stdout_path,omitempty"`
	StderrPath string   `json:"stderr_path,omitempty"`
}

type Adapter interface {
	Name() string
	CheckInstalled(context.Context) Status
	GetCapabilities(context.Context) []string
	RunTask(context.Context, TaskRequest) (TaskResult, error)
	ResumeSession(context.Context, string, TaskRequest) (TaskResult, error)
	ApplyProjectScaffold(string) error
	EstimateRisk(string) string
	CollectTranscript(string) (string, error)
}

func Registry() map[string]Adapter {
	return map[string]Adapter{
		"codex":  Codex{},
		"claude": Claude{},
	}
}

func Get(name string) (Adapter, error) {
	adapter, ok := Registry()[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}
	return adapter, nil
}
