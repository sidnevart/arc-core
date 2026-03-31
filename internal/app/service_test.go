package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-os/internal/chat"
	"agent-os/internal/indexer"
	"agent-os/internal/liveapp"
	"agent-os/internal/memory"
	"agent-os/internal/project"
)

func TestWorkspaceSummary(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.23.6\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := indexer.Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := indexer.Save(root, idx); err != nil {
		t.Fatal(err)
	}
	if err := memory.Save(root, []memory.Item{
		{
			ID:             "q-1",
			Scope:          "project",
			Kind:           "question",
			Source:         "test",
			Confidence:     "medium",
			CreatedAt:      "2026-03-27T00:00:00Z",
			LastVerifiedAt: "2026-03-27T00:00:00Z",
			Status:         "active",
			Summary:        "Need a clearer provider strategy.",
		},
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := NewService().WorkspaceSummary(root)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Name == "" || summary.Root != root {
		t.Fatalf("unexpected workspace summary: %#v", summary)
	}
	if !summary.Index.Ready || summary.Index.Files == 0 {
		t.Fatalf("expected index to be ready, got %#v", summary.Index)
	}
	if len(summary.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(summary.Questions))
	}
}

func TestHomeSnapshotWithoutProjectDoesNotFail(t *testing.T) {
	root := t.TempDir()

	snapshot, err := NewService().HomeSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Workspace != nil {
		t.Fatalf("expected no workspace for plain directory, got %#v", snapshot.Workspace)
	}
	if len(snapshot.Providers) == 0 {
		t.Fatal("expected provider health even without initialized project")
	}
}

func TestInitWorkspaceCreatesArcProject(t *testing.T) {
	root := t.TempDir()

	summary, err := NewService().InitWorkspace(root, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Root != root {
		t.Fatalf("expected summary root %q, got %#v", root, summary)
	}
	if summary.DefaultProvider != "codex" {
		t.Fatalf("expected default provider codex, got %#v", summary)
	}
	if summary.Mode != "work" {
		t.Fatalf("expected default mode work, got %#v", summary)
	}
	if _, err := os.Stat(filepath.Join(root, ".arc", "project.yaml")); err != nil {
		t.Fatalf("expected project scaffold, got %v", err)
	}
}

func TestWorkspaceExplorerAndFileDetail(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	source := "package sample\n\nfunc Example() {}\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Sample\n\n## Usage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".arc/index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := initGitRepo(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source+"\n// dirty change\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewService()
	explorer, err := svc.WorkspaceExplorer(root, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(explorer.Files) == 0 {
		t.Fatal("expected explorer files")
	}
	if len(explorer.DirtyFiles) == 0 {
		t.Fatalf("expected dirty files, got %#v", explorer.DirtyFiles)
	}

	detail, err := svc.WorkspaceFileDetail(root, "sample.go")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Path != "sample.go" {
		t.Fatalf("unexpected detail path: %#v", detail)
	}
	if !strings.Contains(detail.Content, "func Example") {
		t.Fatalf("expected file content, got %q", detail.Content)
	}
	if len(detail.Symbols) == 0 {
		t.Fatalf("expected symbols, got %#v", detail.Symbols)
	}
	if detail.GitChange == nil || detail.GitChange.Status == "" {
		t.Fatalf("expected git change, got %#v", detail.GitChange)
	}
}

func TestSaveWorkspaceFile(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	source := "package sample\n\nfunc Example() {}\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := initGitRepo(root); err != nil {
		t.Fatal(err)
	}

	svc := NewService()
	detail, err := svc.SaveWorkspaceFile(root, "sample.go", "package sample\n\nfunc Example() {\n\tprintln(\"edited\")\n}\n")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Editable {
		t.Fatalf("expected editable detail, got %#v", detail)
	}
	if !strings.Contains(detail.Content, "edited") {
		t.Fatalf("expected updated content, got %q", detail.Content)
	}
	data, err := os.ReadFile(filepath.Join(root, "sample.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "edited") {
		t.Fatalf("expected persisted file contents, got %q", string(data))
	}
}

func TestStartTaskPlanAsyncCreatesRun(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, err := NewService().StartTaskPlanAsync(root, "Plan a small change in sample.go", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ID == "" {
		t.Fatalf("expected run id, got %#v", summary)
	}
	detail, err := NewService().RunDetail(root, summary.ID)
	for i := 0; i < 20; i++ {
		if err != nil {
			t.Fatal(err)
		}
		if detail.Run.Status == "done" || detail.Run.Status == "failed" || detail.Run.Status == "blocked" {
			break
		}
		time.Sleep(100 * time.Millisecond)
		detail, err = NewService().RunDetail(root, summary.ID)
	}
	if err != nil {
		t.Fatal(err)
	}
	if detail.Run.Task == "" {
		t.Fatalf("expected run task, got %#v", detail.Run)
	}
}

func TestLoadChangedFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "changed_files.json")
	if err := os.WriteFile(path, []byte("[\"a.go\",\"docs/readme.md\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := loadChangedFiles(path)
	if len(files) != 2 || files[0] != "a.go" || files[1] != "docs/readme.md" {
		t.Fatalf("unexpected changed files: %#v", files)
	}
}

func TestProjectStateForUninitializedFolder(t *testing.T) {
	root := t.TempDir()
	state, err := NewService().ProjectState(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.State != "selected_not_initialized" {
		t.Fatalf("expected selected_not_initialized, got %#v", state)
	}
}

func TestEnsureLiveAppRestartsStoppedPreviewFromSource(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	sourceDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	app, err := liveapp.StartStaticPreview(liveapp.StartOptions{
		Root:           root,
		SessionID:      "session-1",
		Title:          "Demo",
		Origin:         "output-1",
		Type:           "simulation",
		SourcePath:     sourceDir,
		AutoStopAfter:  time.Minute,
		AutoStopPolicy: "idle_1m",
	})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("sandbox does not allow opening localhost listeners")
		}
		t.Fatal(err)
	}
	stopped, err := liveapp.Stop(root, app.ID, "test_stop")
	if err != nil {
		t.Fatal(err)
	}
	if stopped.Status != "stopped" {
		t.Fatalf("expected stopped app, got %#v", stopped)
	}

	detail, err := NewService().EnsureLiveApp(root, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Status != "ready" {
		t.Fatalf("expected ready ensured app, got %#v", detail)
	}
	if detail.SourcePath == "" || !strings.Contains(detail.SourcePath, "demo") {
		t.Fatalf("expected durable source path, got %#v", detail)
	}
	if detail.ID == app.ID {
		t.Fatalf("expected restarted app with a fresh runtime id, got same id %q", detail.ID)
	}
}

func TestSyncSessionMaterialWithLiveAppsPrefersSharedRuntimeTruth(t *testing.T) {
	item := SessionMaterialCard{
		ID:         "output-1",
		Type:       "simulation",
		Title:      "Miniapp",
		Launchable: true,
		Status:     "stopped",
	}
	live := []LiveAppSummary{{
		ID:         "live-1",
		Origin:     "output-1",
		Type:       "simulation",
		Status:     "ready",
		PreviewURL: "http://127.0.0.1:4567/index.html",
	}}

	got := syncSessionMaterialWithLiveApps(item, live)
	if got.LiveAppID != "live-1" || got.Status != "ready" || got.URL == "" {
		t.Fatalf("expected live runtime to win, got %#v", got)
	}
}

func TestSyncMessageOutputWithLiveAppsFallsBackToStoppedWhenRuntimeMissing(t *testing.T) {
	item := MessageOutputRef{
		ID:         "output-2",
		Kind:       "demo",
		Launchable: true,
		LiveAppID:  "old-live",
		Status:     "ready",
		URL:        "http://127.0.0.1:9999/index.html",
	}

	got := syncMessageOutputWithLiveApps(item, nil)
	if got.Status != "stopped" {
		t.Fatalf("expected missing runtime to degrade to stopped, got %#v", got)
	}
	if got.LiveAppID != "" || got.URL != "" {
		t.Fatalf("expected stale runtime pointers to be cleared, got %#v", got)
	}
}

func TestSelectLiveAppForOriginPrefersActiveOriginOverStoppedPinnedID(t *testing.T) {
	liveApps := []LiveAppSummary{
		{
			ID:         "old-live",
			Origin:     "output-3",
			Status:     "stopped",
			PreviewURL: "",
			UpdatedAt:  "2026-03-29T10:00:00Z",
			StopReason: "stopped_by_user",
		},
		{
			ID:         "new-live",
			Origin:     "output-3",
			Status:     "ready",
			PreviewURL: "http://127.0.0.1:4567/index.html",
			UpdatedAt:  "2026-03-29T10:05:00Z",
		},
	}

	got, ok := selectLiveAppForOrigin(liveApps, "old-live", "output-3")
	if !ok {
		t.Fatal("expected a matching live app")
	}
	if got.ID != "new-live" || got.Status != "ready" {
		t.Fatalf("expected active origin match to win, got %#v", got)
	}
}

func TestSessionSummaryUsesFirstUserMessageForTitle(t *testing.T) {
	session := chat.Session{
		ID:        "chat-1",
		Root:      t.TempDir(),
		Provider:  "codex",
		Mode:      "work",
		Status:    "ready",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []chat.Message{
			{Turn: 1, Role: "user", Content: "Первая тема про миниапп", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
			{Turn: 1, Role: "assistant", Content: "Ответ", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
			{Turn: 2, Role: "user", Content: "Второе сообщение уже про детали", CreatedAt: time.Now().UTC().Format(time.RFC3339)},
		},
	}

	summary := sessionSummaryFromSession(session)
	if !strings.Contains(summary.Title, "Первая тема") {
		t.Fatalf("expected title to come from first user message, got %#v", summary)
	}
	if !strings.Contains(summary.LastUserMessage, "Второе сообщение") {
		t.Fatalf("expected last user message to remain the latest one, got %#v", summary)
	}
	if summary.AgentTagline == "" || len(summary.AgentShortDescription) != 3 {
		t.Fatalf("expected agent guidance metadata in session summary, got %#v", summary)
	}
}

func TestListPresetCardsIncludesShortDescription(t *testing.T) {
	root := t.TempDir()
	presetDir := filepath.Join(root, "sample")
	if err := os.MkdirAll(presetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "id": "sample-preset",
  "name": "Sample Preset",
  "tagline": "Short tagline.",
  "short_description": [
    "First sentence.",
    "Second sentence.",
    "Third sentence."
  ],
  "goal": "Sample goal.",
  "adapter": "arc",
  "category": "test",
  "preset_type": "domain",
  "version": "1.0.0",
  "files": [],
  "author": {"name": "ARC Team", "handle": "arc"}
}`
	if err := os.WriteFile(filepath.Join(presetDir, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	cards, err := NewService().ListPresetCards(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || len(cards[0].ShortDescription) != 3 {
		t.Fatalf("expected preset card short description, got %#v", cards)
	}
}

func TestAugmentPromptWithSessionsUsesSanitizedChatExcerpts(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	session := chat.Session{
		ID:        "attached-chat",
		Root:      root,
		Provider:  "codex",
		Mode:      "study",
		Status:    "ready",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Messages: []chat.Message{
			{
				Turn: 1,
				Role: "user",
				Content: strings.Join([]string{
					"ARC response contract:",
					"This is a reply-only ARC chat surface.",
					"",
					"Current request:",
					"собери симуляцию кровообращения",
				}, "\n"),
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
			{
				Turn:      1,
				Role:      "assistant",
				Content:   "Готовлю схему и миниприложение.",
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	sessionDir := filepath.Join(root, ".arc", "chats", session.ID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := project.WriteJSON(filepath.Join(sessionDir, "session.json"), session); err != nil {
		t.Fatal(err)
	}

	augmented := NewService().augmentPromptWithSessions(root, "новый запрос", []string{session.ID})
	if strings.Contains(augmented, "ARC response contract:") {
		t.Fatalf("expected sanitized excerpts, got %q", augmented)
	}
	if !strings.Contains(augmented, "собери симуляцию кровообращения") {
		t.Fatalf("expected cleaned user excerpt in attached session context, got %q", augmented)
	}
}

func TestPrepareChatPromptKeepsVisibleUserTextClean(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "study",
	}); err != nil {
		t.Fatal(err)
	}

	rawPrompt, providerPrompt, timeout, err := NewService().prepareChatPrompt(
		root,
		"study",
		"do",
		false,
		"Сделай мини-симуляцию про фотосинтез",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if rawPrompt != "Сделай мини-симуляцию про фотосинтез" {
		t.Fatalf("expected visible prompt to stay raw, got %q", rawPrompt)
	}
	if !strings.Contains(providerPrompt, "This is a reply-only ARC chat surface.") {
		t.Fatalf("expected provider prompt to include ARC response contract, got %q", providerPrompt)
	}
	if strings.Contains(rawPrompt, "This is a reply-only ARC chat surface.") {
		t.Fatalf("raw prompt leaked internal provider contract: %q", rawPrompt)
	}
	if timeout <= 0 {
		t.Fatalf("expected positive timeout, got %v", timeout)
	}
}

func TestChatProviderTimeoutUsesLongerBudgetForVisualRequests(t *testing.T) {
	if got := chatProviderTimeout("Сделай мини-симуляцию про клетку"); got != 8*time.Minute {
		t.Fatalf("expected visual chat timeout of 8m, got %s", got)
	}
	if got := chatProviderTimeout("Коротко объясни, что такое ATP"); got != 3*time.Minute {
		t.Fatalf("expected default chat timeout of 3m, got %s", got)
	}
}

func TestSessionNextActionExplainsModelRefreshTimeoutForMiniapps(t *testing.T) {
	session := chat.Session{
		Status: "failed",
		Metadata: map[string]string{
			"last_error": "codex timed out after 5m0s: signal: killed - ERROR codex_core::models_manager::manager: failed to refresh available models: timeout waiting for child process to exit",
		},
		Messages: []chat.Message{
			{Role: "user", Content: "объясни мне плиз на мини аппке кого убил раскольников"},
		},
	}

	got := sessionNextAction(session)
	if !strings.Contains(got, "Повтори запрос ещё раз") {
		t.Fatalf("expected retry guidance, got %q", got)
	}
	if !strings.Contains(got, "миниапп") {
		t.Fatalf("expected miniapp guidance, got %q", got)
	}
}

func TestSessionNextActionMentionsExhaustedAutoRetryForMiniapps(t *testing.T) {
	session := chat.Session{
		Status: "failed",
		Metadata: map[string]string{
			"last_error":         "codex timed out after 5m0s: signal: killed - ERROR codex_core::models_manager::manager: failed to refresh available models: timeout waiting for child process to exit",
			"chat_retry_status":  "exhausted",
			"chat_retry_reason":  "codex_model_refresh_timeout",
			"chat_retry_count":   "1",
		},
		Messages: []chat.Message{
			{Role: "user", Content: "объясни мне плиз на мини аппке кого убил раскольников"},
		},
	}

	got := sessionNextAction(session)
	if !strings.Contains(got, "ARC уже сам попробовал повторить") {
		t.Fatalf("expected exhausted retry guidance, got %q", got)
	}
	if !strings.Contains(got, "миниапп") {
		t.Fatalf("expected miniapp guidance, got %q", got)
	}
}

func TestAllowedActionsByMode(t *testing.T) {
	service := NewService()
	study, err := service.AllowedActions(".", "study", "")
	if err != nil {
		t.Fatal(err)
	}
	if study.CanDo || study.CanSafeRun {
		t.Fatalf("expected study to block autonomous actions, got %#v", study)
	}

	work, err := service.AllowedActions(".", "work", "")
	if err != nil {
		t.Fatal(err)
	}
	if !work.CanDo || !work.DoRequiresUnlock {
		t.Fatalf("expected work to require unlock for do, got %#v", work)
	}

	hero, err := service.AllowedActions(".", "hero", "")
	if err != nil {
		t.Fatal(err)
	}
	if !hero.CanDo || !hero.CanSafeRun {
		t.Fatalf("expected hero to allow autonomous actions, got %#v", hero)
	}
}

func TestStartTaskRunAsyncHonorsAgentPolicy(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewService().StartTaskRunAsync(root, "Implement sample.go", "study", "codex", true, true, false, ""); err == nil {
		t.Fatal("expected study dry run to be blocked")
	}

	if _, err := NewService().StartTaskRunAsync(root, "Implement sample.go", "work", "codex", false, true, false, ""); err == nil {
		t.Fatal("expected work live run without unlock to be blocked")
	}
}

func TestTestingScenarioStepCreatesSessionAndMaterials(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	if _, err := service.SetDeveloperRole(root, "developer"); err != nil {
		t.Fatal(err)
	}
	run, err := service.StartTestingScenario(root, "study-biology-tour", "study", true)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "paused" {
		t.Fatalf("expected paused testing run, got %#v", run)
	}

	run, err = service.TestingControl(root, run.ID, "next")
	if err != nil {
		t.Fatal(err)
	}
	if run.CurrentStep != 0 || run.Steps[0].Status != "done" {
		t.Fatalf("expected first testing step to complete, got %#v", run)
	}
	if run.SessionID == "" {
		t.Fatalf("expected testing step to create session, got %#v", run)
	}

	detail, err := service.SessionDetail(root, run.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.AllowedActions.AgentID != "study" {
		t.Fatalf("expected study allowed actions, got %#v", detail.AllowedActions)
	}
	if len(detail.Materials) == 0 {
		t.Fatalf("expected testing scenario to create materials, got %#v", detail)
	}
	foundDiagram := false
	for _, material := range detail.Materials {
		if material.Type == "diagram" {
			foundDiagram = true
			break
		}
	}
	if !foundDiagram {
		t.Fatalf("expected diagram material, got %#v", detail.Materials)
	}
}

func TestStartTaskPlanAsyncCreatesLinkedSession(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runSummary, err := NewService().StartTaskPlanAsync(root, "Plan a change linked to a session", "work", "codex", "")
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := NewService().ListSessions(root, 10, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) == 0 {
		t.Fatalf("expected at least one session after planning")
	}
	found := false
	for _, session := range sessions {
		for _, runID := range session.RelatedRunIDs {
			if runID == runSummary.ID {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected run %s to be linked to a session", runSummary.ID)
	}
}

func initGitRepo(root string) error {
	commands := [][]string{
		{"git", "-C", root, "init"},
		{"git", "-C", root, "config", "user.email", "tests@example.com"},
		{"git", "-C", root, "config", "user.name", "ARC Tests"},
		{"git", "-C", root, "add", "."},
		{"git", "-C", root, "commit", "-m", "init"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func TestChatDetailFromSessionIncludesArtifactPreviews(t *testing.T) {
	root := t.TempDir()
	transcriptPath := filepath.Join(root, "turn-001.transcript.jsonl")
	stderrPath := filepath.Join(root, "turn-001.stderr.log")
	if err := os.WriteFile(transcriptPath, []byte("{\"type\":\"assistant\",\"content\":\"first\"}\n{\"type\":\"assistant\",\"content\":\"second\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stderrPath, []byte("provider warning line 1\nprovider warning line 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	detail := chatDetailFromSession(chat.Session{
		ID:        "chat-1",
		Provider:  "codex",
		Mode:      "work",
		Status:    "failed",
		CreatedAt: "2026-03-27T12:00:00Z",
		UpdatedAt: "2026-03-27T12:01:00Z",
		Metadata: map[string]string{
			"last_error": "provider crashed",
		},
		Messages: []chat.Message{
			{
				Turn:      1,
				Role:      "assistant",
				Content:   "assistant reply",
				CreatedAt: "2026-03-27T12:01:00Z",
				Artifacts: map[string]string{
					"transcript": transcriptPath,
					"stderr":     stderrPath,
				},
			},
		},
	})

	if detail.Metadata["last_error"] != "provider crashed" {
		t.Fatalf("expected metadata to be preserved, got %#v", detail.Metadata)
	}
	if len(detail.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(detail.Messages))
	}
	if detail.Messages[0].ArtifactPreviews["transcript"] == "" {
		t.Fatalf("expected transcript preview, got %#v", detail.Messages[0].ArtifactPreviews)
	}
	if detail.Messages[0].ArtifactPreviews["stderr"] == "" {
		t.Fatalf("expected stderr preview, got %#v", detail.Messages[0].ArtifactPreviews)
	}
}

func TestChatDetailFromRunningSessionIncludesLivePreview(t *testing.T) {
	root := t.TempDir()
	chatDir := filepath.Join(root, ".arc", "chats", "chat-live")
	if err := os.MkdirAll(chatDir, 0o755); err != nil {
		t.Fatal(err)
	}
	transcriptPath := filepath.Join(chatDir, "turn-001.transcript.jsonl")
	stderrPath := filepath.Join(chatDir, "turn-001.transcript.stderr.log")
	if err := os.WriteFile(transcriptPath, []byte("{\"event\":\"token\",\"content\":\"partial output\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stderrPath, []byte("stderr partial line\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	detail := chatDetailFromSession(chat.Session{
		ID:        "chat-live",
		Root:      root,
		Provider:  "codex",
		Mode:      "work",
		Status:    "running",
		CreatedAt: "2026-03-27T12:00:00Z",
		UpdatedAt: "2026-03-27T12:00:10Z",
		Messages: []chat.Message{
			{
				Turn:      1,
				Role:      "user",
				Content:   "stream me",
				CreatedAt: "2026-03-27T12:00:00Z",
			},
		},
	})

	if detail.Live == nil {
		t.Fatal("expected live chat state")
	}
	if detail.Live.Turn != 1 {
		t.Fatalf("expected live turn 1, got %d", detail.Live.Turn)
	}
	if detail.Live.ArtifactPreviews["transcript"] == "" {
		t.Fatalf("expected transcript preview, got %#v", detail.Live.ArtifactPreviews)
	}
	if detail.Live.ArtifactPreviews["stderr"] == "" {
		t.Fatalf("expected stderr preview, got %#v", detail.Live.ArtifactPreviews)
	}
}
