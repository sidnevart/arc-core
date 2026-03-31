package chat

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-os/internal/project"
	"agent-os/internal/provider"
)

type stubAdapter struct{}

func (stubAdapter) Name() string { return "stub" }
func (stubAdapter) CheckInstalled(context.Context) provider.Status {
	return provider.Status{Name: "stub", Installed: true}
}
func (stubAdapter) GetCapabilities(context.Context) []string { return nil }
func (stubAdapter) ApplyProjectScaffold(string) error        { return nil }
func (stubAdapter) EstimateRisk(string) string               { return "low" }
func (stubAdapter) CollectTranscript(string) (string, error) { return "", nil }
func (stubAdapter) RunTask(_ context.Context, req provider.TaskRequest) (provider.TaskResult, error) {
	_ = os.WriteFile(req.LastMessageOut, []byte("assistant reply\n"), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":1}\n"), 0o644)
	return provider.TaskResult{SessionID: "stub-session", StdoutPath: req.TranscriptOut}, nil
}
func (stubAdapter) ResumeSession(_ context.Context, sessionID string, req provider.TaskRequest) (provider.TaskResult, error) {
	_ = os.WriteFile(req.LastMessageOut, []byte("assistant follow-up\n"), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":2}\n"), 0o644)
	return provider.TaskResult{SessionID: sessionID, StdoutPath: req.TranscriptOut}, nil
}

type slowStubAdapter struct {
	delay time.Duration
}

func (a slowStubAdapter) Name() string { return "slow-stub" }
func (a slowStubAdapter) CheckInstalled(context.Context) provider.Status {
	return provider.Status{Name: "slow-stub", Installed: true}
}
func (a slowStubAdapter) GetCapabilities(context.Context) []string { return nil }
func (a slowStubAdapter) ApplyProjectScaffold(string) error        { return nil }
func (a slowStubAdapter) EstimateRisk(string) string               { return "low" }
func (a slowStubAdapter) CollectTranscript(string) (string, error) { return "", nil }
func (a slowStubAdapter) RunTask(_ context.Context, req provider.TaskRequest) (provider.TaskResult, error) {
	time.Sleep(a.delay)
	_ = os.WriteFile(req.LastMessageOut, []byte("async assistant reply\n"), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":1,\"mode\":\"async\"}\n"), 0o644)
	return provider.TaskResult{SessionID: "slow-session", StdoutPath: req.TranscriptOut}, nil
}
func (a slowStubAdapter) ResumeSession(_ context.Context, sessionID string, req provider.TaskRequest) (provider.TaskResult, error) {
	time.Sleep(a.delay)
	_ = os.WriteFile(req.LastMessageOut, []byte("async assistant follow-up\n"), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":2,\"mode\":\"async\"}\n"), 0o644)
	return provider.TaskResult{SessionID: sessionID, StdoutPath: req.TranscriptOut}, nil
}

type failingStubAdapter struct{}

func (failingStubAdapter) Name() string { return "failing-stub" }
func (failingStubAdapter) CheckInstalled(context.Context) provider.Status {
	return provider.Status{Name: "failing-stub", Installed: true}
}
func (failingStubAdapter) GetCapabilities(context.Context) []string { return nil }
func (failingStubAdapter) ApplyProjectScaffold(string) error        { return nil }
func (failingStubAdapter) EstimateRisk(string) string               { return "low" }
func (failingStubAdapter) CollectTranscript(string) (string, error) { return "", nil }
func (failingStubAdapter) RunTask(_ context.Context, req provider.TaskRequest) (provider.TaskResult, error) {
	stderrPath := strings.TrimSuffix(req.TranscriptOut, filepath.Ext(req.TranscriptOut)) + ".stderr.log"
	_ = os.WriteFile(stderrPath, []byte("env: node: No such file or directory\n"), 0o644)
	return provider.TaskResult{StderrPath: stderrPath}, errors.New("exit status 127")
}
func (failingStubAdapter) ResumeSession(_ context.Context, sessionID string, req provider.TaskRequest) (provider.TaskResult, error) {
	return failingStubAdapter{}.RunTask(context.Background(), req)
}

type richOutputStubAdapter struct{}

func (richOutputStubAdapter) Name() string { return "rich-output-stub" }
func (richOutputStubAdapter) CheckInstalled(context.Context) provider.Status {
	return provider.Status{Name: "rich-output-stub", Installed: true}
}
func (richOutputStubAdapter) GetCapabilities(context.Context) []string { return nil }
func (richOutputStubAdapter) ApplyProjectScaffold(string) error        { return nil }
func (richOutputStubAdapter) EstimateRisk(string) string               { return "low" }
func (richOutputStubAdapter) CollectTranscript(string) (string, error) { return "", nil }
func (richOutputStubAdapter) RunTask(_ context.Context, req provider.TaskRequest) (provider.TaskResult, error) {
	reply := strings.Join([]string{
		"# Разбор",
		"",
		"Сделал ответ в markdown и приложил структурированные результаты.",
		"",
		"```arc-diagram mermaid",
		"title: Поток энергии",
		"graph LR",
		"  A[Глюкоза] --> B[АТФ]",
		"```",
		"",
		"```arc-document markdown",
		"title: Краткий конспект",
		"## Главное",
		"",
		"- Глюкоза используется как топливо",
		"- АТФ выступает переносчиком энергии",
		"```",
	}, "\n")
	_ = os.WriteFile(req.LastMessageOut, []byte(reply), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":1,\"rich\":true}\n"), 0o644)
	return provider.TaskResult{SessionID: "rich-session", StdoutPath: req.TranscriptOut}, nil
}
func (richOutputStubAdapter) ResumeSession(_ context.Context, sessionID string, req provider.TaskRequest) (provider.TaskResult, error) {
	return richOutputStubAdapter{}.RunTask(context.Background(), req)
}

type replyOnlyCaptureAdapter struct {
	lastRequest provider.TaskRequest
}

func (a *replyOnlyCaptureAdapter) Name() string { return "reply-only-capture" }
func (a *replyOnlyCaptureAdapter) CheckInstalled(context.Context) provider.Status {
	return provider.Status{Name: "reply-only-capture", Installed: true}
}
func (a *replyOnlyCaptureAdapter) GetCapabilities(context.Context) []string { return nil }
func (a *replyOnlyCaptureAdapter) ApplyProjectScaffold(string) error        { return nil }
func (a *replyOnlyCaptureAdapter) EstimateRisk(string) string               { return "low" }
func (a *replyOnlyCaptureAdapter) CollectTranscript(string) (string, error) { return "", nil }
func (a *replyOnlyCaptureAdapter) RunTask(_ context.Context, req provider.TaskRequest) (provider.TaskResult, error) {
	a.lastRequest = req
	_ = os.WriteFile(req.LastMessageOut, []byte("assistant reply\n"), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":1}\n"), 0o644)
	return provider.TaskResult{SessionID: "reply-only-session", StdoutPath: req.TranscriptOut}, nil
}
func (a *replyOnlyCaptureAdapter) ResumeSession(_ context.Context, sessionID string, req provider.TaskRequest) (provider.TaskResult, error) {
	a.lastRequest = req
	_ = os.WriteFile(req.LastMessageOut, []byte("assistant follow-up\n"), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":2}\n"), 0o644)
	return provider.TaskResult{SessionID: sessionID, StdoutPath: req.TranscriptOut}, nil
}

type flakyCodexRetryAdapter struct {
	callCount int
	timeouts  []time.Duration
}

func (a *flakyCodexRetryAdapter) Name() string { return "codex" }
func (a *flakyCodexRetryAdapter) CheckInstalled(context.Context) provider.Status {
	return provider.Status{Name: "codex", Installed: true}
}
func (a *flakyCodexRetryAdapter) GetCapabilities(context.Context) []string { return nil }
func (a *flakyCodexRetryAdapter) ApplyProjectScaffold(string) error        { return nil }
func (a *flakyCodexRetryAdapter) EstimateRisk(string) string               { return "low" }
func (a *flakyCodexRetryAdapter) CollectTranscript(string) (string, error) { return "", nil }
func (a *flakyCodexRetryAdapter) RunTask(_ context.Context, req provider.TaskRequest) (provider.TaskResult, error) {
	a.callCount++
	a.timeouts = append(a.timeouts, req.Timeout)
	if a.callCount == 1 {
		stderrPath := strings.TrimSuffix(req.TranscriptOut, filepath.Ext(req.TranscriptOut)) + ".stderr.log"
		_ = os.WriteFile(stderrPath, []byte("ERROR codex_core::models_manager::manager: failed to refresh available models: timeout waiting for child process to exit\n"), 0o644)
		return provider.TaskResult{StderrPath: stderrPath}, errors.New("codex timed out after 5m0s: signal: killed")
	}
	_ = os.WriteFile(req.LastMessageOut, []byte("assistant recovered reply\n"), 0o644)
	_ = os.WriteFile(req.TranscriptOut, []byte("{\"turn\":1,\"retry\":true}\n"), 0o644)
	return provider.TaskResult{SessionID: "retry-session", StdoutPath: req.TranscriptOut}, nil
}
func (a *flakyCodexRetryAdapter) ResumeSession(_ context.Context, sessionID string, req provider.TaskRequest) (provider.TaskResult, error) {
	return a.RunTask(context.Background(), req)
}

type alwaysFailingCodexRetryAdapter struct {
	callCount int
}

func (a *alwaysFailingCodexRetryAdapter) Name() string { return "codex" }
func (a *alwaysFailingCodexRetryAdapter) CheckInstalled(context.Context) provider.Status {
	return provider.Status{Name: "codex", Installed: true}
}
func (a *alwaysFailingCodexRetryAdapter) GetCapabilities(context.Context) []string { return nil }
func (a *alwaysFailingCodexRetryAdapter) ApplyProjectScaffold(string) error        { return nil }
func (a *alwaysFailingCodexRetryAdapter) EstimateRisk(string) string               { return "low" }
func (a *alwaysFailingCodexRetryAdapter) CollectTranscript(string) (string, error) { return "", nil }
func (a *alwaysFailingCodexRetryAdapter) RunTask(_ context.Context, req provider.TaskRequest) (provider.TaskResult, error) {
	a.callCount++
	stderrPath := strings.TrimSuffix(req.TranscriptOut, filepath.Ext(req.TranscriptOut)) + ".stderr.log"
	_ = os.WriteFile(stderrPath, []byte("ERROR codex_core::models_manager::manager: failed to refresh available models: timeout waiting for child process to exit\n"), 0o644)
	return provider.TaskResult{StderrPath: stderrPath}, errors.New("codex timed out after 5m0s: signal: killed")
}
func (a *alwaysFailingCodexRetryAdapter) ResumeSession(_ context.Context, sessionID string, req provider.TaskRequest) (provider.TaskResult, error) {
	return a.RunTask(context.Background(), req)
}

func TestRunTurnAndList(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	session := Session{
		ID:        "test-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "work",
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{Turn: 1, Role: "user", Content: "hello", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	got, err := runTurn(root, session, stubAdapter{}, "hello", "", false, time.Minute, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderSessionID != "stub-session" {
		t.Fatalf("expected session id, got %q", got.ProviderSessionID)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got.Messages))
	}
	if got.Messages[1].Content != "assistant reply" {
		t.Fatalf("unexpected assistant content: %q", got.Messages[1].Content)
	}
	if _, err := os.Stat(filepath.Join(project.ProjectFile(root, "chats"), "test-chat", "session.json")); err != nil {
		t.Fatal(err)
	}
}

func TestSendRejectsRunningSession(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	session := Session{
		ID:        "running-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "work",
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{Turn: 1, Role: "user", Content: "hello", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}
	if err := saveSession(root, session); err != nil {
		t.Fatal(err)
	}
	_, _, err := prepareSend(SendOptions{
		Root:      root,
		SessionID: session.ID,
		Prompt:    "another prompt",
	})
	if err == nil {
		t.Fatal("expected running session guard")
	}
}

func TestLoadSanitizesHistoricalAugmentedUserPrompt(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	session := Session{
		ID:        "dirty-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "work",
		Status:    "ready",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{
				Turn: 1,
				Role: "user",
				Content: strings.Join([]string{
					"ARC response contract:",
					"This is a reply-only ARC chat surface.",
					"",
					"User request:",
					"построй симуляцию фотосинтеза",
				}, "\n"),
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if err := saveSession(root, session); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Messages[0].Content; got != "построй симуляцию фотосинтеза" {
		t.Fatalf("expected sanitized user prompt, got %q", got)
	}

	reloaded, err := Load(root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Messages[0].Content; got != "построй симуляцию фотосинтеза" {
		t.Fatalf("expected persisted sanitized prompt, got %q", got)
	}
}

func TestStartAsyncReturnsRunningThenCompletes(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	session, adapter, err := prepareStart(StartOptions{
		Root:     root,
		Provider: "codex",
		Mode:     "work",
		Prompt:   "start async",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = runTurn(root, session, slowStubAdapter{delay: 120 * time.Millisecond}, "start async", "", false, time.Second, true)
	}()

	initial, err := Load(root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if initial.Status != "running" {
		t.Fatalf("expected running status immediately, got %q", initial.Status)
	}
	if len(initial.Messages) != 1 {
		t.Fatalf("expected only user message before completion, got %d", len(initial.Messages))
	}
	_ = adapter

	var final Session
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		final, err = Load(root, session.ID)
		if err != nil {
			t.Fatal(err)
		}
		if final.Status == "ready" && len(final.Messages) == 2 {
			break
		}
		time.Sleep(40 * time.Millisecond)
	}
	if final.Status != "ready" {
		t.Fatalf("expected ready status, got %q", final.Status)
	}
	if len(final.Messages) != 2 {
		t.Fatalf("expected assistant message after async completion, got %d", len(final.Messages))
	}
	if final.Messages[1].Content != "async assistant reply" {
		t.Fatalf("unexpected assistant content: %q", final.Messages[1].Content)
	}
}

func TestRunTurnAddsStderrContextToLastError(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	session := Session{
		ID:        "failed-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "hero",
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{Turn: 1, Role: "user", Content: "run it", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	got, err := runTurn(root, session, failingStubAdapter{}, "run it", "", false, time.Minute, true)
	if err == nil {
		t.Fatal("expected provider failure")
	}
	if got.Status != "failed" {
		t.Fatalf("expected failed session, got %q", got.Status)
	}
	if !strings.Contains(got.Metadata["last_error"], "env: node: No such file or directory") {
		t.Fatalf("expected stderr context in last_error, got %q", got.Metadata["last_error"])
	}
}

func TestRunTurnMarksChatRequestsReplyOnly(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	session := Session{
		ID:        "reply-only-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "work",
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{Turn: 1, Role: "user", Content: "сделай мини-симуляцию", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	adapter := &replyOnlyCaptureAdapter{}
	if _, err := runTurn(root, session, adapter, "сделай мини-симуляцию", "", false, time.Minute, true); err != nil {
		t.Fatal(err)
	}
	if !adapter.lastRequest.ReplyOnly {
		t.Fatal("expected normal chat run to mark provider request as reply-only")
	}
}

func TestRunTurnRetriesTransientCodexModelRefreshTimeoutForVisualPrompt(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "study",
	}); err != nil {
		t.Fatal(err)
	}

	session := Session{
		ID:        "retry-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "study",
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{Turn: 1, Role: "user", Content: "объясни мне плиз на мини аппке кого убил раскольников", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	adapter := &flakyCodexRetryAdapter{}
	got, err := runTurn(root, session, adapter, "объясни мне плиз на мини аппке кого убил раскольников", "", false, 5*time.Minute, true)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.callCount != 2 {
		t.Fatalf("expected 2 provider attempts, got %d", adapter.callCount)
	}
	if len(adapter.timeouts) != 2 || adapter.timeouts[1] <= adapter.timeouts[0] {
		t.Fatalf("expected retry to use a larger timeout, got %#v", adapter.timeouts)
	}
	if got.Status != "ready" {
		t.Fatalf("expected recovered session to be ready, got %q", got.Status)
	}
	if got.Metadata["chat_retry_status"] != "recovered" {
		t.Fatalf("expected recovered retry status, got %#v", got.Metadata)
	}
	if got.Metadata["last_error"] != "" {
		t.Fatalf("expected last_error to be cleared after recovery, got %#v", got.Metadata)
	}
	if got.Messages[1].Artifacts["retry"] == "" {
		t.Fatalf("expected retry artifact on assistant message, got %#v", got.Messages[1].Artifacts)
	}
}

func TestRunTurnMarksRetryExhaustedWhenTransientCodexFailureRepeats(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "study",
	}); err != nil {
		t.Fatal(err)
	}

	session := Session{
		ID:        "retry-failed-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "study",
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{Turn: 1, Role: "user", Content: "сделай мини-симуляцию про раскольникова", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	adapter := &alwaysFailingCodexRetryAdapter{}
	got, err := runTurn(root, session, adapter, "сделай мини-симуляцию про раскольникова", "", false, 5*time.Minute, true)
	if err == nil {
		t.Fatal("expected final provider failure")
	}
	if adapter.callCount != 2 {
		t.Fatalf("expected 2 provider attempts, got %d", adapter.callCount)
	}
	if got.Metadata["chat_retry_status"] != "exhausted" {
		t.Fatalf("expected exhausted retry status, got %#v", got.Metadata)
	}
	if got.Metadata["chat_retry_reason"] != "codex_model_refresh_timeout" {
		t.Fatalf("expected retry reason metadata, got %#v", got.Metadata)
	}
	if got.Messages[1].Artifacts["retry"] == "" {
		t.Fatalf("expected retry artifact on failed assistant message, got %#v", got.Messages[1].Artifacts)
	}
}

func TestParseFallbackOutputBlocksExtractsSimulationHTML(t *testing.T) {
	prompt := "Сделай мини-симуляцию по фотосинтезу"
	content := strings.Join([]string{
		"Ниже мини-приложение.",
		"",
		"```html",
		"<html><body><h1>Photosynthesis</h1></body></html>",
		"```",
	}, "\n")

	cleaned, blocks := parseFallbackOutputBlocks(prompt, content)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 fallback block, got %d", len(blocks))
	}
	if blocks[0].Kind != "simulation" {
		t.Fatalf("expected simulation block, got %#v", blocks[0])
	}
	if blocks[0].Format != "html" {
		t.Fatalf("expected html format, got %#v", blocks[0])
	}
	if strings.Contains(cleaned, "```html") {
		t.Fatalf("expected html fence to be removed from cleaned content, got %q", cleaned)
	}
}

func TestParseFallbackOutputBlocksExtractsMermaidDiagram(t *testing.T) {
	prompt := "Объясни через схему"
	content := strings.Join([]string{
		"Показываю схему.",
		"",
		"```mermaid",
		"graph LR",
		"  A[Свет] --> B[Энергия]",
		"```",
	}, "\n")

	cleaned, blocks := parseFallbackOutputBlocks(prompt, content)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 fallback block, got %d", len(blocks))
	}
	if blocks[0].Kind != "diagram" || blocks[0].Format != "mermaid" {
		t.Fatalf("unexpected fallback block: %#v", blocks[0])
	}
	if strings.Contains(cleaned, "```mermaid") {
		t.Fatalf("expected mermaid fence to be removed from cleaned content, got %q", cleaned)
	}
}

func TestRunTurnMaterializesRichOutputs(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "study",
	}); err != nil {
		t.Fatal(err)
	}

	session := Session{
		ID:        "rich-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "study",
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []Message{
			{Turn: 1, Role: "user", Content: "объясни через схему и конспект", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	got, err := runTurn(root, session, richOutputStubAdapter{}, "объясни через схему и конспект", "", false, time.Minute, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got.Messages))
	}
	reply := got.Messages[1]
	if !strings.Contains(reply.Content, "Сделал ответ в markdown") {
		t.Fatalf("expected markdown body to remain, got %q", reply.Content)
	}
	if len(reply.Outputs) != 2 {
		t.Fatalf("expected 2 structured outputs, got %#v", reply.Outputs)
	}
	if reply.Outputs[0].Kind != "diagram" || reply.Outputs[0].Format != "svg" || !strings.Contains(reply.Outputs[0].Preview, "<svg") {
		t.Fatalf("expected rendered svg diagram output, got %#v", reply.Outputs[0])
	}
	if reply.Outputs[1].Kind != "document" || reply.Outputs[1].Format != "markdown" {
		t.Fatalf("expected markdown document output, got %#v", reply.Outputs[1])
	}
	if _, err := os.Stat(reply.Outputs[0].Path); err != nil {
		t.Fatalf("expected diagram artifact on disk: %v", err)
	}
	if _, ok := reply.Artifacts["diagram_01"]; !ok {
		t.Fatalf("expected diagram artifact key in message artifacts, got %#v", reply.Artifacts)
	}
	if _, ok := reply.Artifacts["document_02"]; !ok {
		t.Fatalf("expected document artifact key in message artifacts, got %#v", reply.Artifacts)
	}
}

func TestShouldAutoLaunchOutputPolicy(t *testing.T) {
	if !shouldAutoLaunchOutput("сделай мини-симуляцию клеточного дыхания", "work", "simulation") {
		t.Fatal("expected explicit simulation request to auto-launch in work mode")
	}
	if shouldAutoLaunchOutput("объясни тему", "work", "simulation") {
		t.Fatal("did not expect implicit simulation launch in work mode")
	}
	if !shouldAutoLaunchOutput("объясни тему", "study", "simulation") {
		t.Fatal("expected study mode to allow agent-judgment launch for simulations")
	}
	if shouldAutoLaunchOutput("объясни тему", "work", "diagram") {
		t.Fatal("diagrams should not use live-app auto-launch path")
	}
}

func TestPrepareStartStoresVisibleUserPromptInsteadOfAugmentedProviderPrompt(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	session, _, err := prepareStart(StartOptions{
		Root:       root,
		Provider:   "codex",
		Mode:       "study",
		Prompt:     "ARC response contract...\n\nUser request:\nСделай симуляцию",
		UserPrompt: "Сделай симуляцию",
		Timeout:    time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := session.Messages[0].Content; got != "Сделай симуляцию" {
		t.Fatalf("expected stored user message to stay clean, got %q", got)
	}
}
