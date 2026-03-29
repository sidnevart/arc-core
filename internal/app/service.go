package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"agent-os/internal/chat"
	"agent-os/internal/indexer"
	"agent-os/internal/liveapp"
	"agent-os/internal/memory"
	"agent-os/internal/mode"
	"agent-os/internal/orchestrator"
	"agent-os/internal/presets"
	"agent-os/internal/project"
	"agent-os/internal/provider"
)

type Service struct{}

func NewService() Service {
	return Service{}
}

var localhostURLPattern = regexp.MustCompile(`https?://(?:localhost|127\.0\.0\.1)(?::\d+)?[^\s<>"')\]]*`)
var sessionMaterialPriority = []string{
	"task_map.md",
	"system_flow.md",
	"solution_options.md",
	"validation_checklist.md",
	"implementation_log.md",
	"review_report.md",
	"verification_report.md",
	"docs_delta.md",
	"changed_files.md",
	"workspace_diff.patch",
	"anti_hallucination_report.md",
	"provider_transcript.stderr.log",
	"provider_transcript.jsonl",
}

func builtInAgents() []AgentCard {
	return []AgentCard{
		{
			ID:          "study",
			Name:        "Study",
			Tagline:     "Объясняет, учит и помогает понять тему.",
			Description: "Подходит для обучения, диаграмм, туториалов, вопросов на понимание и спокойного пошагового объяснения.",
			Mode:        "study",
			BuiltIn:     true,
		},
		{
			ID:          "work",
			Name:        "Work",
			Tagline:     "Делает работу вместе с человеком.",
			Description: "Подходит для совместной инженерной работы, планирования, безопасных запусков и прозрачного обсуждения решений.",
			Mode:        "work",
			BuiltIn:     true,
		},
		{
			ID:          "hero",
			Name:        "Hero",
			Tagline:     "Берёт bounded-задачу и делает её автономнее.",
			Description: "Подходит для случаев, когда агент должен сам провести больше шагов и принести уже готовый результат.",
			Mode:        "hero",
			BuiltIn:     true,
		},
	}
}

func agentNameForMode(mode string) string {
	for _, agent := range builtInAgents() {
		if agent.Mode == mode {
			return agent.Name
		}
	}
	if strings.TrimSpace(mode) == "" {
		return "Agent"
	}
	return strings.Title(mode)
}

func (Service) Agents() []AgentCard {
	return builtInAgents()
}

func (s Service) ProjectState(root string) (ProjectState, error) {
	clean := strings.TrimSpace(root)
	if clean == "" {
		return ProjectState{
			Path:    "",
			Name:    "",
			State:   "no_project_selected",
			Message: "Сначала выбери проект.",
		}, nil
	}
	abs, err := filepath.Abs(rootIfEmpty(clean))
	if err != nil {
		return ProjectState{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectState{
				Path:    abs,
				Name:    filepath.Base(abs),
				State:   "missing",
				Message: "Такой папки не существует.",
			}, nil
		}
		return ProjectState{}, err
	}
	if !info.IsDir() {
		return ProjectState{
			Path:    abs,
			Name:    info.Name(),
			State:   "missing",
			Message: "Нужно выбрать именно папку проекта.",
		}, nil
	}
	summary, err := s.WorkspaceSummary(abs)
	if err == nil {
		return ProjectState{
			Path:      abs,
			Name:      summary.Name,
			State:     "ready",
			Message:   "Проект готов к работе.",
			Workspace: &summary,
		}, nil
	}
	return ProjectState{
		Path:    abs,
		Name:    filepath.Base(abs),
		State:   "selected_not_initialized",
		Message: "ARC ещё не настроен в этой папке.",
	}, nil
}

func (Service) ProviderHealth(ctx context.Context) []ProviderHealth {
	registry := provider.Registry()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]ProviderHealth, 0, len(names))
	for _, name := range names {
		status := registry[name].CheckInstalled(ctx)
		out = append(out, ProviderHealth{
			Name:         status.Name,
			Installed:    status.Installed,
			BinaryPath:   status.BinaryPath,
			Capabilities: append([]string{}, status.Capabilities...),
			Notes:        append([]string{}, status.Notes...),
		})
	}
	return out
}

func (s Service) WorkspaceSummary(root string) (WorkspaceSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		if root == "." || root == "" {
			resolved, err = filepath.Abs(rootIfEmpty(root))
			if err != nil {
				return WorkspaceSummary{}, err
			}
		} else {
			return WorkspaceSummary{}, err
		}
	}

	_, _ = project.Repair(resolved)

	proj, err := project.Load(resolved)
	if err != nil {
		return WorkspaceSummary{}, err
	}

	items, err := memory.Load(resolved)
	if err != nil {
		items = nil
	}
	memStatus := memory.Status(items)
	questions := memory.UnknownQuestions(items)
	questionItems := make([]QuestionEntry, 0, len(questions))
	for i, item := range questions {
		if i >= 5 {
			break
		}
		questionItems = append(questionItems, QuestionEntry{
			ID:      item.ID,
			Summary: item.Summary,
			Status:  item.Status,
		})
	}

	var lastRun *RunSummary
	runs, err := orchestrator.ListRuns(resolved)
	if err == nil && len(runs) > 0 {
		summary := runSummaryFromRun(runs[0])
		lastRun = &summary
	}
	installedRecords, _ := presets.ListInstalled(resolved)

	return WorkspaceSummary{
		Root:             proj.Root,
		Name:             proj.Config.Name,
		ArcDir:           proj.ArcDir,
		DefaultProvider:  proj.Config.DefaultProvider,
		EnabledProviders: append([]string{}, proj.Config.EnabledProviders...),
		Mode:             proj.Mode.Mode,
		Autonomy:         proj.Mode.Autonomy,
		Index:            loadIndexStatus(resolved),
		Memory:           memStatus,
		Docs:             loadDocsStatus(resolved),
		LastRun:          lastRun,
		Questions:        questionItems,
		InstalledPresets: installedSummaries(installedRecords),
	}, nil
}

func (s Service) HomeSnapshot(root string) (HomeSnapshot, error) {
	var workspace *WorkspaceSummary
	if summary, err := s.WorkspaceSummary(root); err == nil {
		workspace = &summary
	}

	var runs []RunSummary
	if workspace != nil {
		list, err := s.ListRuns(workspace.Root, 6)
		if err == nil {
			runs = list
		}
	}

	presetCards, _ := s.ListPresetCards(defaultPresetsRoot())

	return HomeSnapshot{
		Workspace: workspace,
		Providers: s.ProviderHealth(context.Background()),
		Runs:      runs,
		Presets:   presetCards,
		Installed: installedFromWorkspace(workspace),
		Chats:     chatsFromWorkspace(workspace),
	}, nil
}

func (s Service) InitWorkspace(root string, providerName string, modeName string) (WorkspaceSummary, error) {
	resolved, err := filepath.Abs(rootIfEmpty(root))
	if err != nil {
		return WorkspaceSummary{}, err
	}
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		providerName = "codex"
	}
	modeName = strings.TrimSpace(modeName)
	if modeName == "" {
		modeName = "work"
	}
	if _, err := project.Init(resolved, project.InitOptions{
		Provider:         providerName,
		EnabledProviders: []string{providerName},
		Mode:             modeName,
	}); err != nil {
		return WorkspaceSummary{}, err
	}
	return s.WorkspaceSummary(resolved)
}

func (s Service) RepairWorkspace(root string) (WorkspaceSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return WorkspaceSummary{}, err
	}
	if _, err := project.Repair(resolved); err != nil {
		return WorkspaceSummary{}, err
	}
	return s.WorkspaceSummary(resolved)
}

func (s Service) SetWorkspaceMode(root string, modeName string) (WorkspaceSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return WorkspaceSummary{}, err
	}
	modeName = strings.TrimSpace(modeName)
	switch modeName {
	case "study", "work", "hero":
	default:
		return WorkspaceSummary{}, fmt.Errorf("unsupported mode %q", modeName)
	}
	def := mode.ByName(modeName)
	if err := project.WriteMode(resolved, modeName, def.Autonomy); err != nil {
		return WorkspaceSummary{}, err
	}
	_ = project.AppendEvent(resolved, project.Event{
		Timestamp: time.Now().UTC(),
		Command:   "desktop set mode",
		Status:    "ok",
		Details: map[string]string{
			"mode": modeName,
			"path": resolved,
		},
	})
	return s.WorkspaceSummary(resolved)
}

func (Service) ListRuns(root string, limit int) ([]RunSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return nil, err
	}
	runs, err := orchestrator.ListRuns(resolved)
	if err != nil {
		return nil, err
	}
	out := make([]RunSummary, 0, len(runs))
	for i, run := range runs {
		if limit > 0 && i >= limit {
			break
		}
		out = append(out, runSummaryFromRun(run))
	}
	return out, nil
}

func (Service) RunDetail(root string, runID string) (RunDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return RunDetail{}, err
	}
	run, err := orchestrator.LoadRun(resolved, runID)
	if err != nil {
		return RunDetail{}, err
	}
	artifacts := make([]ArtifactSummary, 0, len(run.Artifacts))
	names := make([]string, 0, len(run.Artifacts))
	for name := range run.Artifacts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		artifacts = append(artifacts, ArtifactSummary{Name: name, Path: run.Artifacts[name]})
	}
	changedFiles := loadChangedFiles(run.Artifacts["changed_files.json"])
	return RunDetail{
		Run:       runSummaryFromRun(run),
		Artifacts: artifacts,
		Changed:   changedFiles,
		Previews:  runArtifactPreviews(run.Artifacts),
		Metadata:  run.Metadata,
		Docs:      run.Docs,
		Review:    run.Review,
		Verify:    run.Verification,
	}, nil
}

func (Service) StartTaskPlanAsync(root string, task string, modeName string, providerName string, sessionID string) (RunSummary, error) {
	opts, err := resolveTaskOptions(root, task, modeName, providerName, true, false)
	if err != nil {
		return RunSummary{}, err
	}
	sessionID, _ = ensureTaskSession(opts.Root, opts.Provider, opts.Mode, task, sessionID)
	summary, err := launchRunAsync(opts.Root, func() (orchestrator.Run, error) {
		return orchestrator.Plan(opts.Root, opts)
	})
	if err != nil {
		return RunSummary{}, err
	}
	if strings.TrimSpace(sessionID) != "" {
		_ = chat.AttachRun(opts.Root, sessionID, summary.ID)
	}
	return summary, nil
}

func (Service) StartTaskRunAsync(root string, task string, modeName string, providerName string, dryRun bool, runChecks bool, allowAutonomy bool, sessionID string) (RunSummary, error) {
	opts, err := resolveTaskOptions(root, task, modeName, providerName, dryRun, runChecks)
	if err != nil {
		return RunSummary{}, err
	}
	if err := validateTaskRunPolicy(opts.Mode, dryRun, allowAutonomy); err != nil {
		return RunSummary{}, err
	}
	sessionID, _ = ensureTaskSession(opts.Root, opts.Provider, opts.Mode, task, sessionID)
	summary, err := launchRunAsync(opts.Root, func() (orchestrator.Run, error) {
		return orchestrator.RunTask(opts.Root, opts)
	})
	if err != nil {
		return RunSummary{}, err
	}
	if strings.TrimSpace(sessionID) != "" {
		_ = chat.AttachRun(opts.Root, sessionID, summary.ID)
	}
	return summary, nil
}

func (s Service) ReviewRun(root string, runID string) (RunDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return RunDetail{}, err
	}
	if _, err := orchestrator.Review(resolved, runID); err != nil {
		return RunDetail{}, err
	}
	return s.RunDetail(resolved, runID)
}

func (Service) MemoryStatus(root string) (memory.Summary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return memory.Summary{}, err
	}
	items, err := memory.Load(resolved)
	if err != nil {
		return memory.Summary{}, err
	}
	return memory.Status(items), nil
}

func (Service) WorkspaceExplorer(root string, limit int) (WorkspaceExplorer, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return WorkspaceExplorer{}, err
	}
	idx, err := refreshIndex(resolved)
	if err != nil {
		return WorkspaceExplorer{}, err
	}
	ordered := orderedWorkspaceFiles(idx.Files)
	files := make([]WorkspaceFileEntry, 0, len(ordered))
	for i, file := range ordered {
		if limit > 0 && i >= limit {
			break
		}
		files = append(files, WorkspaceFileEntry{
			Path:    file.Path,
			Kind:    file.Kind,
			Size:    file.Size,
			ModTime: file.ModTime,
		})
	}
	changes := make([]WorkspaceChange, 0, minInt(len(idx.Recent), 8))
	for i, change := range idx.Recent {
		if i >= 8 {
			break
		}
		changes = append(changes, WorkspaceChange{
			Hash:    change.Hash,
			Date:    change.Date,
			Author:  change.Author,
			Subject: change.Subject,
		})
	}
	dirtyFiles := gitDirtyFiles(resolved, 20)
	return WorkspaceExplorer{
		Files:         files,
		RecentChanges: changes,
		DirtyFiles:    dirtyFiles,
	}, nil
}

func (Service) WorkspaceFileDetail(root string, relPath string) (WorkspaceFileDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	cleanRel, absPath, err := sanitizeWorkspacePath(resolved, relPath)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	idx, err := refreshIndex(resolved)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	detail := WorkspaceFileDetail{
		Path:    cleanRel,
		Kind:    fileKindForPath(idx, cleanRel),
		Size:    info.Size(),
		ModTime: info.ModTime().UTC().Format(time.RFC3339),
	}
	content, truncated := readWorkspaceFileContent(absPath, detail.Kind, 32000)
	detail.Content = content
	detail.Truncated = truncated
	detail.Editable = isPreviewableKind(detail.Kind, absPath) && !truncated
	detail.Symbols = symbolsForFile(idx, cleanRel)
	if docTitle, headings := docInfoForFile(idx, cleanRel); docTitle != "" || len(headings) > 0 {
		detail.DocTitle = docTitle
		detail.DocHeadings = headings
	}
	if change, ok := mostRecentChange(idx); ok {
		detail.RecentChange = &WorkspaceChange{
			Hash:    change.Hash,
			Date:    change.Date,
			Author:  change.Author,
			Subject: change.Subject,
		}
	}
	if gitChange := gitChangeForFile(resolved, cleanRel); gitChange != nil {
		detail.GitChange = gitChange
	}
	return detail, nil
}

func (s Service) SaveWorkspaceFile(root string, relPath string, content string) (WorkspaceFileDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	cleanRel, absPath, err := sanitizeWorkspacePath(resolved, relPath)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	if info.IsDir() {
		return WorkspaceFileDetail{}, fmt.Errorf("cannot save a directory")
	}
	current, err := s.WorkspaceFileDetail(resolved, cleanRel)
	if err != nil {
		return WorkspaceFileDetail{}, err
	}
	if !current.Editable {
		return WorkspaceFileDetail{}, fmt.Errorf("file is not editable in desktop preview")
	}
	if err := os.WriteFile(absPath, []byte(content), info.Mode().Perm()); err != nil {
		return WorkspaceFileDetail{}, err
	}
	return s.WorkspaceFileDetail(resolved, cleanRel)
}

func (Service) ListPresetCards(root string) ([]PresetCard, error) {
	manifests, err := presets.List(root)
	if err != nil {
		return nil, err
	}
	out := make([]PresetCard, 0, len(manifests))
	for _, manifest := range manifests {
		out = append(out, PresetCard{
			ID:         manifest.ID,
			Name:       manifest.Name,
			Tagline:    manifest.Tagline,
			Goal:       manifest.Goal,
			Adapter:    manifest.Adapter,
			Category:   manifest.Category,
			Version:    manifest.Version,
			AuthorName: manifest.Author.Name,
			Trust:      trustForManifest(manifest),
			Files:      append([]string{}, manifest.Files...),
			Path:       manifest.Path,
		})
	}
	return out, nil
}

func (s Service) StartChat(root string, providerName string, mode string, model string, prompt string, dryRun bool, action string, allowAutonomy bool, attachSessionIDs []string) (ChatDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return ChatDetail{}, err
	}
	rawPrompt, augmented, timeout, err := s.prepareChatPrompt(resolved, mode, action, allowAutonomy, prompt, attachSessionIDs)
	if err != nil {
		return ChatDetail{}, err
	}
	session, err := chat.Start(chat.StartOptions{
		Root:       resolved,
		Provider:   providerName,
		Mode:       mode,
		Model:      model,
		Prompt:     augmented,
		UserPrompt: rawPrompt,
		ReplyOnly:  true,
		DryRun:     dryRun,
		Timeout:    timeout,
	})
	if err != nil {
		return ChatDetail{}, err
	}
	return chatDetailFromSession(session), nil
}

func (s Service) StartChatAsync(root string, providerName string, mode string, model string, prompt string, dryRun bool, action string, allowAutonomy bool, attachSessionIDs []string) (ChatDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return ChatDetail{}, err
	}
	rawPrompt, augmented, timeout, err := s.prepareChatPrompt(resolved, mode, action, allowAutonomy, prompt, attachSessionIDs)
	if err != nil {
		return ChatDetail{}, err
	}
	session, err := chat.StartAsync(chat.StartOptions{
		Root:       resolved,
		Provider:   providerName,
		Mode:       mode,
		Model:      model,
		Prompt:     augmented,
		UserPrompt: rawPrompt,
		ReplyOnly:  true,
		DryRun:     dryRun,
		Timeout:    timeout,
	})
	if err != nil {
		return ChatDetail{}, err
	}
	return chatDetailFromSession(session), nil
}

func (s Service) SendChat(root string, sessionID string, model string, prompt string, dryRun bool, action string, allowAutonomy bool, attachSessionIDs []string) (ChatDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return ChatDetail{}, err
	}
	session, err := chat.Load(resolved, sessionID)
	if err != nil {
		return ChatDetail{}, err
	}
	rawPrompt, augmented, timeout, err := s.prepareChatPrompt(resolved, session.Mode, action, allowAutonomy, prompt, attachSessionIDs)
	if err != nil {
		return ChatDetail{}, err
	}
	session, err = chat.Send(chat.SendOptions{
		Root:       resolved,
		SessionID:  sessionID,
		Model:      model,
		Prompt:     augmented,
		UserPrompt: rawPrompt,
		ReplyOnly:  true,
		DryRun:     dryRun,
		Timeout:    timeout,
	})
	if err != nil {
		return ChatDetail{}, err
	}
	return chatDetailFromSession(session), nil
}

func (s Service) SendChatAsync(root string, sessionID string, model string, prompt string, dryRun bool, action string, allowAutonomy bool, attachSessionIDs []string) (ChatDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return ChatDetail{}, err
	}
	session, err := chat.Load(resolved, sessionID)
	if err != nil {
		return ChatDetail{}, err
	}
	rawPrompt, augmented, timeout, err := s.prepareChatPrompt(resolved, session.Mode, action, allowAutonomy, prompt, attachSessionIDs)
	if err != nil {
		return ChatDetail{}, err
	}
	session, err = chat.SendAsync(chat.SendOptions{
		Root:       resolved,
		SessionID:  sessionID,
		Model:      model,
		Prompt:     augmented,
		UserPrompt: rawPrompt,
		ReplyOnly:  true,
		DryRun:     dryRun,
		Timeout:    timeout,
	})
	if err != nil {
		return ChatDetail{}, err
	}
	return chatDetailFromSession(session), nil
}
func (Service) ListChats(root string, limit int) ([]ChatSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return nil, err
	}
	sessions, err := chat.List(resolved)
	if err != nil {
		return nil, err
	}
	out := make([]ChatSummary, 0, len(sessions))
	for i, session := range sessions {
		if limit > 0 && i >= limit {
			break
		}
		out = append(out, chatSummaryFromSession(session))
	}
	return out, nil
}

func (Service) ChatDetail(root string, sessionID string) (ChatDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return ChatDetail{}, err
	}
	session, err := chat.Load(resolved, sessionID)
	if err != nil {
		return ChatDetail{}, err
	}
	return chatDetailFromSession(session), nil
}

func (Service) ListSessions(root string, limit int, query string, mode string, status string) ([]SessionSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return nil, err
	}
	sessions, err := chat.List(resolved)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	mode = strings.TrimSpace(mode)
	status = strings.TrimSpace(status)
	out := make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		summary := sessionSummaryFromSession(session)
		if mode != "" && summary.Mode != mode {
			continue
		}
		if status != "" && summary.Status != status {
			continue
		}
		if query != "" && !sessionMatchesQuery(summary, query) {
			continue
		}
		out = append(out, summary)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (Service) SessionDetail(root string, sessionID string) (SessionDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return SessionDetail{}, err
	}
	session, err := chat.Load(resolved, sessionID)
	if err != nil {
		return SessionDetail{}, err
	}
	return Service{}.sessionDetailWithLiveApps(resolved, session), nil
}

func (Service) PresetPreview(root string, id string) (PresetPreview, error) {
	manifest, err := presets.LoadByID(root, id)
	if err != nil {
		return PresetPreview{}, err
	}
	readme, _ := os.ReadFile(filepath.Join(filepath.Dir(manifest.Path), "README.md"))
	return PresetPreview{
		Manifest: PresetCard{
			ID:         manifest.ID,
			Name:       manifest.Name,
			Tagline:    manifest.Tagline,
			Goal:       manifest.Goal,
			Adapter:    manifest.Adapter,
			Category:   manifest.Category,
			Version:    manifest.Version,
			AuthorName: manifest.Author.Name,
			Trust:      trustForManifest(manifest),
			Files:      append([]string{}, manifest.Files...),
			Path:       manifest.Path,
		},
		Files:  append([]string{}, manifest.Files...),
		Readme: string(readme),
	}, nil
}

func (Service) PreviewPresetInstall(workspaceRoot string, catalogRoot string, id string) (presets.InstallPreview, error) {
	resolved, err := project.DiscoverRoot(workspaceRoot)
	if err != nil {
		return presets.InstallPreview{}, err
	}
	return presets.PreviewInstall(presets.PreviewOptions{
		WorkspaceRoot: resolved,
		CatalogRoot:   catalogRoot,
		PresetID:      id,
	})
}

func (Service) InstallPreset(workspaceRoot string, catalogRoot string, id string, allowOverwrite bool) (presets.InstallResult, error) {
	resolved, err := project.DiscoverRoot(workspaceRoot)
	if err != nil {
		return presets.InstallResult{}, err
	}
	return presets.Install(presets.InstallOptions{
		WorkspaceRoot:  resolved,
		CatalogRoot:    catalogRoot,
		PresetID:       id,
		AllowOverwrite: allowOverwrite,
	})
}

func (Service) RollbackPreset(workspaceRoot string, installID string) (InstalledPresetSummary, error) {
	resolved, err := project.DiscoverRoot(workspaceRoot)
	if err != nil {
		return InstalledPresetSummary{}, err
	}
	record, err := presets.Rollback(presets.RollbackOptions{
		WorkspaceRoot: resolved,
		InstallID:     installID,
	})
	if err != nil {
		return InstalledPresetSummary{}, err
	}
	return InstalledPresetSummary{
		InstallID:   record.InstallID,
		PresetID:    record.PresetID,
		Version:     record.Version,
		Status:      record.Status,
		InstalledAt: record.InstalledAt,
		ReportPath:  record.ReportPath,
	}, nil
}

func runSummaryFromRun(run orchestrator.Run) RunSummary {
	return RunSummary{
		ID:            run.ID,
		Task:          run.Task,
		Status:        run.Status,
		State:         string(run.CurrentState),
		Mode:          run.Mode,
		Provider:      run.Provider,
		StartedAt:     run.StartedAt,
		UpdatedAt:     run.UpdatedAt,
		DryRun:        run.DryRun,
		ArtifactCount: len(run.Artifacts),
	}
}

func loadIndexStatus(root string) IndexStatus {
	if bundle, err := refreshIndex(root); err == nil {
		return IndexStatus{
			Ready:        true,
			Files:        len(bundle.Files),
			Symbols:      len(bundle.Symbols),
			Dependencies: len(bundle.Dependencies),
			Docs:         len(bundle.Docs),
		}
	}
	return IndexStatus{}
}

func refreshIndex(root string) (indexer.Result, error) {
	bundle, err := indexer.Build(root)
	if err != nil {
		return indexer.Result{}, err
	}
	_ = indexer.Save(root, bundle)
	_ = indexer.WriteIndividual(root, bundle)
	return bundle, nil
}

func loadDocsStatus(root string) DocsStatus {
	candidates := []string{
		project.ProjectFile(root, "maps", "REPO_MAP.md"),
		project.ProjectFile(root, "maps", "DOCS_MAP.md"),
		project.ProjectFile(root, "maps", "CLI_MAP.md"),
		project.ProjectFile(root, "maps", "ARTIFACTS_MAP.md"),
		project.ProjectFile(root, "maps", "RUNTIME_STATUS.md"),
	}
	found := []string{}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			found = append(found, filepath.Base(candidate))
		}
	}
	return DocsStatus{
		Ready:         len(found) > 0,
		GeneratedMaps: found,
	}
}

func trustForManifest(manifest presets.Manifest) string {
	if strings.Contains(manifest.Path, string(filepath.Separator)+"official"+string(filepath.Separator)) {
		return "first_party"
	}
	return "community"
}

func rootIfEmpty(root string) string {
	if strings.TrimSpace(root) == "" {
		return "."
	}
	return root
}

func defaultPresetsRoot() string {
	candidates := []string{"presets/official", "../presets/official", "../../presets/official"}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join(rootIfEmpty(""), "presets", "official")
}

func chatSummaryFromSession(session chat.Session) ChatSummary {
	summary := ChatSummary{
		ID:                session.ID,
		Provider:          session.Provider,
		Mode:              session.Mode,
		Status:            session.Status,
		CreatedAt:         session.CreatedAt,
		UpdatedAt:         session.UpdatedAt,
		ProviderSessionID: session.ProviderSessionID,
		MessageCount:      len(session.Messages),
	}
	for i := len(session.Messages) - 1; i >= 0; i-- {
		content := visibleSessionMessageContent(session.Messages[i])
		switch session.Messages[i].Role {
		case "user":
			if summary.LastUserMessage == "" {
				summary.LastUserMessage = content
			}
		case "assistant":
			if summary.LastAssistantMessage == "" {
				summary.LastAssistantMessage = content
			}
		}
	}
	return summary
}

func sessionSummaryFromSession(session chat.Session) SessionSummary {
	chatSummary := chatSummaryFromSession(session)
	title := strings.TrimSpace(firstSessionTitleMessage(session))
	if title == "" {
		title = strings.TrimSpace(chatSummary.LastAssistantMessage)
	}
	if title == "" {
		title = "Новая сессия"
	}
	summary := strings.TrimSpace(chatSummary.LastAssistantMessage)
	if summary == "" {
		summary = strings.TrimSpace(chatSummary.LastUserMessage)
	}
	materials := sessionMaterialCards(session.Root, session)
	return SessionSummary{
		ID:                   session.ID,
		Title:                excerpt(title, 68),
		Summary:              excerpt(summary, 140),
		AgentID:              session.Mode,
		AgentName:            agentNameForMode(session.Mode),
		Mode:                 session.Mode,
		Status:               session.Status,
		CreatedAt:            session.CreatedAt,
		UpdatedAt:            session.UpdatedAt,
		LastUserMessage:      chatSummary.LastUserMessage,
		LastAssistantMessage: chatSummary.LastAssistantMessage,
		RelatedRunIDs:        chat.RelatedRunIDs(session),
		MaterialCount:        len(materials),
	}
}

func firstSessionTitleMessage(session chat.Session) string {
	for _, message := range session.Messages {
		if message.Role != "user" {
			continue
		}
		content := strings.TrimSpace(visibleSessionMessageContent(message))
		if content != "" {
			return content
		}
	}
	return ""
}

func (s Service) sessionDetailWithLiveApps(root string, session chat.Session) SessionDetail {
	detail := chatDetailFromSession(session)
	liveApps, _ := s.ListLiveApps(root, session.ID)
	return SessionDetail{
		Session:        sessionSummaryFromSession(session),
		Messages:       detail.Messages,
		Runs:           sessionRuns(root, session),
		Materials:      sessionMaterialCards(root, session),
		LiveApps:       liveApps,
		Metadata:       detail.Metadata,
		Live:           detail.Live,
		NextAction:     sessionNextAction(session),
		ProjectRoot:    root,
		AllowedActions: allowedActionsForMode(session.Mode),
	}
}

func chatDetailFromSession(session chat.Session) ChatDetail {
	messages := make([]ChatMessage, 0, len(session.Messages))
	for _, message := range session.Messages {
		messages = append(messages, ChatMessage{
			Turn:             message.Turn,
			Role:             message.Role,
			Content:          visibleSessionMessageContent(message),
			CreatedAt:        message.CreatedAt,
			Artifacts:        message.Artifacts,
			ArtifactPreviews: artifactPreviews(message.Artifacts),
			Outputs:          messageOutputs(session.Root, message),
			Failure:          message.Failure,
		})
	}
	return ChatDetail{
		Session:  chatSummaryFromSession(session),
		Messages: messages,
		Metadata: copyStringMap(session.Metadata),
		Live:     liveChatState(session),
	}
}

func sessionMatchesQuery(summary SessionSummary, query string) bool {
	haystacks := []string{
		strings.ToLower(summary.Title),
		strings.ToLower(summary.Summary),
		strings.ToLower(summary.LastUserMessage),
		strings.ToLower(summary.LastAssistantMessage),
	}
	for _, haystack := range haystacks {
		if strings.Contains(haystack, query) {
			return true
		}
	}
	return false
}

func sessionRuns(root string, session chat.Session) []RunSummary {
	runIDs := chat.RelatedRunIDs(session)
	if len(runIDs) == 0 {
		return nil
	}
	out := make([]RunSummary, 0, len(runIDs))
	for _, runID := range runIDs {
		run, err := orchestrator.LoadRun(root, runID)
		if err != nil {
			continue
		}
		out = append(out, runSummaryFromRun(run))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func sessionMaterialCards(root string, session chat.Session) []SessionMaterialCard {
	out := []SessionMaterialCard{}
	if len(session.Messages) > 0 {
		last := session.Messages[len(session.Messages)-1]
		if strings.TrimSpace(last.Content) != "" {
			out = append(out, SessionMaterialCard{
				ID:        session.ID + ":answer",
				Type:      "answer",
				Title:     "Ответ агента",
				Summary:   "Последний ответ в этой сессии.",
				Source:    "conversation",
				Preview:   excerpt(last.Content, 800),
				OpenLabel: "Открыть ответ",
			})
		}
		for _, url := range extractLocalURLs(last.Content) {
			out = append(out, SessionMaterialCard{
				ID:        session.ID + ":demo:" + url,
				Type:      "demo",
				Title:     "Демо",
				Summary:   "Агент подготовил локальное демо, которое можно открыть прямо из сессии.",
				Source:    "conversation",
				URL:       url,
				Preview:   url,
				OpenLabel: "Открыть демо отдельно",
			})
		}
	}
	out = append(out, lessonMaterialCards(root, session)...)
	out = append(out, messageOutputMaterials(root, session)...)
	out = append(out, messageArtifactMaterials(root, session)...)
	for _, run := range sessionRuns(root, session) {
		runDetail, err := Service{}.RunDetail(root, run.ID)
		if err != nil {
			continue
		}
		out = append(out, runMaterials(runDetail)...)
	}
	return dedupeMaterials(out)
}

func messageOutputs(root string, message chat.Message) []MessageOutputRef {
	if len(message.Outputs) == 0 {
		return nil
	}
	out := make([]MessageOutputRef, 0, len(message.Outputs))
	for _, output := range message.Outputs {
		ref := MessageOutputRef{
			ID:         output.ID,
			Kind:       output.Kind,
			Title:      output.Title,
			Preview:    outputPreview(output),
			Path:       relativeArtifactPath(root, output.Path),
			URL:        strings.TrimSpace(output.PreviewURL),
			Launchable: output.Launchable,
			Inline:     output.Inline,
			LiveAppID:  output.LiveAppID,
			Status:     output.Status,
			Error:      output.Error,
		}
		out = append(out, ref)
	}
	return out
}

func runMaterials(detail RunDetail) []SessionMaterialCard {
	out := []SessionMaterialCard{}
	for _, name := range sessionMaterialPriority {
		preview, ok := detail.Previews[name]
		if !ok || strings.TrimSpace(preview) == "" {
			continue
		}
		cardType, title, summary := classifyMaterial(name)
		out = append(out, SessionMaterialCard{
			ID:         detail.Run.ID + ":" + name,
			Type:       cardType,
			Title:      title,
			Summary:    summary,
			Source:     detail.Run.ID,
			Preview:    preview,
			Path:       detailPreviewPath(detail, name),
			Files:      append([]string{}, detail.Changed...),
			OpenLabel:  "Открыть материал",
			Launchable: materialLaunchable(detailPreviewPath(detail, name), detail.Changed),
		})
		for _, url := range extractLocalURLs(preview) {
			out = append(out, SessionMaterialCard{
				ID:        detail.Run.ID + ":demo:" + url,
				Type:      "demo",
				Title:     "Демо",
				Summary:   "Во время выполнения появился локальный preview.",
				Source:    detail.Run.ID,
				URL:       url,
				Preview:   url,
				Files:     append([]string{}, detail.Changed...),
				OpenLabel: "Открыть демо отдельно",
			})
		}
	}
	return out
}

func classifyMaterial(name string) (string, string, string) {
	switch name {
	case "task_map.md", "solution_options.md", "validation_checklist.md":
		return "plan", "План", "Агент собрал план и следующие шаги."
	case "implementation_log.md", "changed_files.md", "workspace_diff.patch":
		return "changes", "Изменения", "Здесь видно, что агент менял и почему."
	case "review_report.md", "verification_report.md", "anti_hallucination_report.md":
		return "review", "Проверка", "Проверка результата и найденные риски."
	case "docs_delta.md":
		return "document", "Документ", "Обновление документации или поясняющих материалов."
	default:
		return "document", "Материал", "Дополнительный материал этой сессии."
	}
}

func classifyMessageArtifact(key string, path string) (string, string, string) {
	filename := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case ext == ".svg":
		return "diagram", "Диаграмма", "Агент подготовил визуальную схему, которую можно открыть прямо в материалах."
	case ext == ".html" || ext == ".htm":
		if strings.Contains(filename, "sim") || strings.Contains(filename, "lesson") {
			return "simulation", "Симуляция", "Интерактивный материал для объяснения темы прямо внутри ARC."
		}
		return "demo", "Демо", "Локальное демо, которое можно открыть в ARC или отдельно."
	case strings.Contains(filename, "plan"):
		return "plan", "План", "Пошаговый план, который агент подготовил в этой сессии."
	case strings.Contains(filename, "diagram"):
		return "diagram", "Диаграмма", "Визуальная схема для объяснения идеи или процесса."
	case strings.Contains(filename, "change") || strings.Contains(filename, "diff"):
		return "changes", "Изменения", "Сводка изменений и того, что поменялось."
	case strings.Contains(filename, "review") || strings.Contains(filename, "check"):
		return "review", "Проверка", "Проверка результата и короткий разбор рисков."
	default:
		if key == "notes" || key == "doc" || ext == ".md" || ext == ".txt" {
			return "document", "Документ", "Полезный текстовый материал для работы или обучения."
		}
		return "document", "Материал", "Дополнительный материал, подготовленный агентом."
	}
}

func messageOutputMaterials(root string, session chat.Session) []SessionMaterialCard {
	out := []SessionMaterialCard{}
	for i := len(session.Messages) - 1; i >= 0; i-- {
		message := session.Messages[i]
		if message.Role != "assistant" || len(message.Outputs) == 0 {
			continue
		}
		for _, output := range message.Outputs {
			out = append(out, SessionMaterialCard{
				ID:         output.ID,
				Type:       output.Kind,
				Title:      output.Title,
				Summary:    messageOutputSummary(output),
				Source:     "assistant",
				Preview:    outputPreview(output),
				Path:       relativeArtifactPath(root, output.Path),
				URL:        strings.TrimSpace(output.PreviewURL),
				Files:      nonEmptyStrings(relativeArtifactPath(root, output.Path)),
				OpenLabel:  messageOutputOpenLabel(output),
				Launchable: output.Launchable,
				LiveAppID:  output.LiveAppID,
				Status:     output.Status,
				Error:      output.Error,
			})
		}
	}
	return out
}

func outputPreview(output chat.Output) string {
	if strings.TrimSpace(output.Preview) != "" {
		return output.Preview
	}
	switch output.Kind {
	case "diagram":
		return readArtifactPreview(output.Path, 12000)
	case "document":
		return readArtifactPreview(output.Path, 2400)
	default:
		return readArtifactPreview(output.Path, 800)
	}
}

func messageOutputSummary(output chat.Output) string {
	switch output.Kind {
	case "diagram":
		return "Визуальная схема встроена прямо в разговор."
	case "simulation":
		if strings.TrimSpace(output.PreviewURL) != "" {
			return "Интерактивная мини-симуляция уже запущена в ARC."
		}
		return "Интерактивная мини-симуляция готова к запуску."
	case "demo":
		if strings.TrimSpace(output.PreviewURL) != "" {
			return "Демо уже поднято и доступно прямо в разговоре."
		}
		return "Локальное демо готово к запуску."
	default:
		return "Полезный материал, который агент подготовил в ответе."
	}
}

func messageOutputOpenLabel(output chat.Output) string {
	switch output.Kind {
	case "diagram":
		return "Открыть схему"
	case "simulation", "demo":
		return "Открыть миниприложение"
	default:
		return "Открыть материал"
	}
}

func detailPreviewPath(detail RunDetail, name string) string {
	for _, artifact := range detail.Artifacts {
		if artifact.Name == name {
			return artifact.Path
		}
	}
	return ""
}

func dedupeMaterials(items []SessionMaterialCard) []SessionMaterialCard {
	if len(items) == 0 {
		return nil
	}
	out := make([]SessionMaterialCard, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out
}

func sessionNextAction(session chat.Session) string {
	switch session.Status {
	case "running":
		return "Подожди завершения или открой live-материалы."
	case "failed":
		return "Открой последний ответ и попроси агента объяснить, что пошло не так."
	default:
		if len(chat.RelatedRunIDs(session)) > 0 {
			return "Открой материалы сессии, а затем продолжи разговор или подтяни эту сессию в новый контекст."
		}
		return "Продолжи разговор или попроси агента сделать план."
	}
}

func (Service) ListLiveApps(root string, sessionID string) ([]LiveAppSummary, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return nil, err
	}
	apps, err := liveapp.List(resolved)
	if err != nil {
		return nil, err
	}
	out := make([]LiveAppSummary, 0, len(apps))
	for _, item := range apps {
		if strings.TrimSpace(sessionID) != "" && item.SessionID != sessionID {
			continue
		}
		out = append(out, liveAppSummaryFromApp(item))
	}
	return out, nil
}

func (Service) StartMaterialLiveApp(root string, sessionID string, materialID string) (LiveAppDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return LiveAppDetail{}, err
	}
	detail, err := Service{}.SessionDetail(resolved, sessionID)
	if err != nil {
		return LiveAppDetail{}, err
	}
	var selected *SessionMaterialCard
	for i := range detail.Materials {
		if detail.Materials[i].ID == materialID {
			selected = &detail.Materials[i]
			break
		}
	}
	if selected == nil {
		return LiveAppDetail{}, fmt.Errorf("material %q not found", materialID)
	}
	sourcePath := materialSourcePath(resolved, *selected)
	if sourcePath == "" {
		return LiveAppDetail{}, fmt.Errorf("этот материал пока нельзя открыть как встроенное демо")
	}
	if existing, ok, err := existingLiveAppForOrigin(resolved, sessionID, selected.ID); err == nil && ok {
		app, err := ensureLiveAppFromSource(resolved, existing, existing.SourcePath)
		if err != nil {
			return LiveAppDetail{}, err
		}
		return liveAppDetailFromApp(app), nil
	}
	app, err := startLiveAppFromSource(resolved, sessionID, selected.Title, selected.ID, selected.Type, sourcePath, "idle_20m")
	if err != nil {
		return LiveAppDetail{}, err
	}
	return liveAppDetailFromApp(app), nil
}

func (Service) EnsureLiveApp(root string, appID string) (LiveAppDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return LiveAppDetail{}, err
	}
	items, err := liveapp.List(resolved)
	if err != nil {
		return LiveAppDetail{}, err
	}
	var item liveapp.App
	found := false
	for _, candidate := range items {
		if candidate.ID == strings.TrimSpace(appID) {
			item = candidate
			found = true
			break
		}
	}
	if !found {
		return LiveAppDetail{}, fmt.Errorf("live app %q not found", appID)
	}
	app, err := ensureLiveAppFromSource(resolved, item, item.SourcePath)
	if err != nil {
		return LiveAppDetail{}, err
	}
	return liveAppDetailFromApp(app), nil
}

func (Service) StopLiveApp(root string, appID string) (LiveAppDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return LiveAppDetail{}, err
	}
	app, err := liveapp.Stop(resolved, appID, "stopped_by_user")
	if err != nil {
		return LiveAppDetail{}, err
	}
	return liveAppDetailFromApp(app), nil
}

func (Service) LaunchLessonDemo(root string, sessionID string, lessonID string) (LiveAppDetail, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return LiveAppDetail{}, err
	}
	sourcePath, title, err := ensureLessonDemo(resolved, lessonID)
	if err != nil {
		return LiveAppDetail{}, err
	}
	origin := "lesson:" + lessonID
	if existing, ok, err := existingLiveAppForOrigin(resolved, sessionID, origin); err == nil && ok {
		app, err := ensureLiveAppFromSource(resolved, existing, sourcePath)
		if err != nil {
			return LiveAppDetail{}, err
		}
		return liveAppDetailFromApp(app), nil
	}
	if strings.TrimSpace(sessionID) != "" {
		_, _ = chat.MergeMetadata(resolved, sessionID, map[string]string{"lesson_id": lessonID})
	}
	app, err := startLiveAppFromSource(resolved, sessionID, title, origin, "simulation", sourcePath, "idle_20m")
	if err != nil {
		return LiveAppDetail{}, err
	}
	return liveAppDetailFromApp(app), nil
}

func startLiveAppFromSource(root string, sessionID string, title string, origin string, kind string, sourcePath string, autoStopPolicy string) (liveapp.App, error) {
	return liveapp.StartStaticPreview(liveapp.StartOptions{
		Root:           root,
		SessionID:      sessionID,
		Title:          title,
		Origin:         origin,
		Type:           kind,
		SourcePath:     sourcePath,
		AutoStopAfter:  20 * time.Minute,
		AutoStopPolicy: autoStopPolicy,
	})
}

func existingLiveAppForOrigin(root string, sessionID string, origin string) (liveapp.App, bool, error) {
	items, err := liveapp.List(root)
	if err != nil {
		return liveapp.App{}, false, err
	}
	trimmedSession := strings.TrimSpace(sessionID)
	trimmedOrigin := strings.TrimSpace(origin)
	for _, item := range items {
		if trimmedSession != "" && item.SessionID != trimmedSession {
			continue
		}
		if strings.TrimSpace(item.Origin) != trimmedOrigin {
			continue
		}
		return item, true, nil
	}
	return liveapp.App{}, false, nil
}

func ensureLiveAppFromSource(root string, item liveapp.App, sourcePath string) (liveapp.App, error) {
	if item.Status == "ready" || item.Status == "starting" {
		return item, nil
	}
	candidate := strings.TrimSpace(sourcePath)
	if candidate == "" {
		candidate = strings.TrimSpace(item.SourcePath)
	}
	if !launchablePath(candidate) {
		return liveapp.App{}, fmt.Errorf("источник демо недоступен")
	}
	return startLiveAppFromSource(root, item.SessionID, item.Title, item.Origin, item.Type, candidate, defaultLivePolicy(item.AutoStopPolicy))
}

func defaultLivePolicy(policy string) string {
	if strings.TrimSpace(policy) == "" {
		return "idle_20m"
	}
	return policy
}

func chatsFromWorkspace(workspace *WorkspaceSummary) []ChatSummary {
	if workspace == nil {
		return nil
	}
	sessions, err := chat.List(workspace.Root)
	if err != nil {
		return nil
	}
	out := make([]ChatSummary, 0, minInt(len(sessions), 6))
	for i, session := range sessions {
		if i >= 6 {
			break
		}
		out = append(out, chatSummaryFromSession(session))
	}
	return out
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func artifactPreviews(artifacts map[string]string) map[string]string {
	if len(artifacts) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, path := range artifacts {
		preview := readArtifactPreview(path, previewLimitForArtifact(key))
		if preview == "" {
			continue
		}
		out[key] = preview
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runArtifactPreviews(artifacts map[string]string) map[string]string {
	if len(artifacts) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, path := range artifacts {
		if !previewableRunArtifact(key, path) {
			continue
		}
		preview := readArtifactPreview(path, 2400)
		if preview == "" {
			continue
		}
		out[key] = preview
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func previewableRunArtifact(name string, path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".json", ".jsonl", ".log", ".txt", ".patch", ".diff":
		return true
	}
	switch name {
	case "changed_files.md", "changed_files.json", "implementation_log.md", "verification_report.md", "review_report.md", "docs_delta.md", "provider_last_message.md", "workspace_diff.patch":
		return true
	default:
		return false
	}
}

func loadChangedFiles(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	var files []string
	if err := project.ReadJSON(path, &files); err != nil || len(files) == 0 {
		return nil
	}
	return files
}

func sanitizeWorkspacePath(root string, relPath string) (string, string, error) {
	cleanRel := filepath.Clean(strings.TrimSpace(relPath))
	if cleanRel == "." || cleanRel == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(cleanRel) {
		return "", "", fmt.Errorf("absolute paths are not allowed")
	}
	absPath := filepath.Join(root, cleanRel)
	relCheck, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", "", err
	}
	if strings.HasPrefix(relCheck, "..") {
		return "", "", fmt.Errorf("path escapes workspace")
	}
	return filepath.ToSlash(relCheck), absPath, nil
}

func fileKindForPath(idx indexer.Result, relPath string) string {
	for _, file := range idx.Files {
		if file.Path == relPath {
			return file.Kind
		}
	}
	switch strings.ToLower(filepath.Ext(relPath)) {
	case ".go":
		return "go"
	case ".md", ".txt", ".adoc":
		return "doc"
	case ".json", ".yaml", ".yml", ".toml":
		return "config"
	default:
		return "file"
	}
}

func readWorkspaceFileContent(path string, kind string, limit int) (string, bool) {
	if !isPreviewableKind(kind, path) {
		return "[binary or unsupported preview]", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	if len(data) <= limit {
		return string(data), false
	}
	return string(data[:limit]), true
}

func isPreviewableKind(kind string, path string) bool {
	switch kind {
	case "go", "doc", "config", "file":
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".gif", ".pdf", ".zip", ".gz", ".tar", ".db", ".sqlite", ".wasm", ".ico":
			return false
		default:
			return true
		}
	default:
		return false
	}
}

func symbolsForFile(idx indexer.Result, relPath string) []WorkspaceSymbol {
	out := []WorkspaceSymbol{}
	for _, symbol := range idx.Symbols {
		if symbol.Path != relPath {
			continue
		}
		out = append(out, WorkspaceSymbol{
			Name:     symbol.Name,
			Kind:     symbol.Kind,
			Language: symbol.Language,
			Line:     symbol.Line,
		})
	}
	return out
}

func docInfoForFile(idx indexer.Result, relPath string) (string, []string) {
	for _, doc := range idx.Docs {
		if doc.Path == relPath {
			return doc.Title, append([]string{}, doc.Headings...)
		}
	}
	return "", nil
}

func mostRecentChange(idx indexer.Result) (indexer.ChangeEntry, bool) {
	if len(idx.Recent) == 0 {
		return indexer.ChangeEntry{}, false
	}
	return idx.Recent[0], true
}

func orderedWorkspaceFiles(files []indexer.FileEntry) []indexer.FileEntry {
	primary := make([]indexer.FileEntry, 0, len(files))
	internal := make([]indexer.FileEntry, 0, len(files))
	for _, file := range files {
		if strings.HasPrefix(file.Path, ".arc/") {
			internal = append(internal, file)
			continue
		}
		primary = append(primary, file)
	}
	return append(primary, internal...)
}

func gitDirtyFiles(root string, limit int) []WorkspaceGitChange {
	cmd := exec.Command("git", "-C", root, "status", "--short")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	out := make([]WorkspaceGitChange, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		status, path := parseGitStatusLine(line)
		if path == "" {
			continue
		}
		out = append(out, WorkspaceGitChange{
			Path:   path,
			Status: status,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func gitChangeForFile(root string, relPath string) *WorkspaceGitChange {
	status := gitStatusForFile(root, relPath)
	if status == "" {
		return nil
	}
	return &WorkspaceGitChange{
		Path:        relPath,
		Status:      status,
		DiffPreview: gitDiffPreview(root, relPath, 8000),
	}
}

func gitStatusForFile(root string, relPath string) string {
	cmd := exec.Command("git", "-C", root, "status", "--short", "--", relPath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(output))
	if line == "" {
		return ""
	}
	status, _ := parseGitStatusLine(line)
	return status
}

func gitDiffPreview(root string, relPath string, limit int) string {
	cmd := exec.Command("git", "-C", root, "diff", "--", relPath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		cmd = exec.Command("git", "-C", root, "diff", "--cached", "--", relPath)
		output, err = cmd.Output()
		if err != nil {
			return ""
		}
		text = strings.TrimSpace(string(output))
	}
	if text == "" {
		return ""
	}
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func parseGitStatusLine(line string) (string, string) {
	if len(line) < 3 {
		return "", ""
	}
	status := strings.TrimSpace(line[:2])
	path := strings.TrimSpace(line[3:])
	if strings.Contains(path, " -> ") {
		parts := strings.Split(path, " -> ")
		path = parts[len(parts)-1]
	}
	return status, filepath.ToSlash(path)
}

func liveChatState(session chat.Session) *ChatLiveState {
	if session.Status != "running" {
		return nil
	}
	turn := assistantTurnForMessages(session.Messages)
	artifacts := map[string]string{
		"last_message": project.ProjectFile(session.Root, "chats", session.ID, fmt.Sprintf("turn-%03d.assistant.md", turn)),
		"transcript":   project.ProjectFile(session.Root, "chats", session.ID, fmt.Sprintf("turn-%03d.transcript.jsonl", turn)),
		"stderr":       project.ProjectFile(session.Root, "chats", session.ID, fmt.Sprintf("turn-%03d.transcript.stderr.log", turn)),
	}
	previews := artifactPreviews(artifacts)
	return &ChatLiveState{
		Turn:             turn,
		Artifacts:        artifacts,
		ArtifactPreviews: previews,
	}
}

func previewLimitForArtifact(kind string) int {
	switch kind {
	case "transcript", "stderr":
		return 1400
	default:
		return 900
	}
}

func readArtifactPreview(path string, limit int) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return ""
	}
	if len(text) <= limit {
		return text
	}
	if limit < 64 {
		return text[:limit]
	}
	return "..." + text[len(text)-limit+3:]
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func assistantTurnForMessages(messages []chat.Message) int {
	count := 0
	for _, message := range messages {
		if message.Role == "assistant" {
			count++
		}
	}
	return count + 1
}

func visibleSessionMessageContent(message chat.Message) string {
	content := strings.TrimSpace(message.Content)
	if message.Role == "user" {
		if cleaned := chat.SanitizeVisibleChatText(content); strings.TrimSpace(cleaned) != "" {
			return cleaned
		}
	}
	return content
}

func (Service) augmentPromptWithSessions(root string, prompt string, attachSessionIDs []string) string {
	if len(attachSessionIDs) == 0 {
		return prompt
	}
	parts := []string{}
	for _, sessionID := range attachSessionIDs {
		session, err := chat.Load(root, sessionID)
		if err != nil {
			continue
		}
		summary := sessionSummaryFromSession(session)
		parts = append(parts, fmt.Sprintf(
			"Attached session context:\n- title: %s\n- agent: %s\n- summary: %s\n- last user message: %s\n- last assistant message: %s",
			summary.Title,
			summary.AgentName,
			summary.Summary,
			excerpt(summary.LastUserMessage, 220),
			excerpt(summary.LastAssistantMessage, 220),
		))
	}
	if len(parts) == 0 {
		return prompt
	}
	return strings.Join(parts, "\n\n") + "\n\nCurrent request:\n" + prompt
}

func (s Service) prepareChatPrompt(root string, mode string, action string, allowAutonomy bool, prompt string, attachSessionIDs []string) (string, string, time.Duration, error) {
	rawPrompt := prompt
	enforcedPrompt, err := enforceChatAction(mode, action, allowAutonomy, prompt)
	if err != nil {
		return "", "", 0, err
	}
	return rawPrompt, s.augmentPromptWithSessions(root, enforcedPrompt, attachSessionIDs), chatProviderTimeout(enforcedPrompt), nil
}

func ensureTaskSession(root string, providerName string, mode string, task string, sessionID string) (string, error) {
	if strings.TrimSpace(sessionID) != "" {
		if err := chat.AppendUserMessage(root, sessionID, task); err != nil {
			return sessionID, err
		}
		return sessionID, nil
	}
	session, err := chat.Create(chat.CreateOptions{
		Root:     root,
		Provider: providerName,
		Mode:     mode,
		Model:    "",
		Prompt:   task,
	})
	if err != nil {
		return "", err
	}
	return session.ID, nil
}

func extractLocalURLs(text string) []string {
	matches := localhostURLPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		out = append(out, match)
	}
	return out
}

func excerpt(text string, limit int) string {
	value := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if len(value) <= limit || limit <= 0 {
		return value
	}
	if limit < 2 {
		return value[:limit]
	}
	return value[:limit-1] + "…"
}

func resolveTaskOptions(root string, task string, modeName string, providerName string, dryRun bool, runChecks bool) (orchestrator.TaskOptions, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return orchestrator.TaskOptions{}, err
	}
	proj, err := project.Load(resolved)
	if err != nil {
		return orchestrator.TaskOptions{}, err
	}
	if strings.TrimSpace(modeName) == "" {
		modeName = proj.Mode.Mode
	}
	if strings.TrimSpace(providerName) == "" {
		providerName = proj.Config.DefaultProvider
	}
	return orchestrator.TaskOptions{
		Root:            resolved,
		Task:            strings.TrimSpace(task),
		Mode:            modeName,
		Provider:        providerName,
		DryRun:          dryRun,
		RunChecks:       runChecks,
		UseProvider:     true,
		ApproveRisky:    false,
		ProviderTimeout: 2 * time.Minute,
	}, nil
}

func launchRunAsync(root string, fn func() (orchestrator.Run, error)) (RunSummary, error) {
	before, _ := orchestrator.ListRuns(root)
	known := map[string]struct{}{}
	for _, run := range before {
		known[run.ID] = struct{}{}
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := fn()
		errCh <- err
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		runs, err := orchestrator.ListRuns(root)
		if err == nil {
			for _, run := range runs {
				if _, exists := known[run.ID]; exists {
					continue
				}
				return runSummaryFromRun(run), nil
			}
		}
		select {
		case err := <-errCh:
			if err != nil {
				return RunSummary{}, err
			}
		default:
		}
		time.Sleep(60 * time.Millisecond)
	}

	select {
	case err := <-errCh:
		if err != nil {
			return RunSummary{}, err
		}
	default:
	}
	return RunSummary{}, fmt.Errorf("timed out waiting for new run to be created")
}

func installedSummaries(records []presets.InstalledRecord) []InstalledPresetSummary {
	out := make([]InstalledPresetSummary, 0, len(records))
	for _, record := range records {
		out = append(out, InstalledPresetSummary{
			InstallID:   record.InstallID,
			PresetID:    record.PresetID,
			Version:     record.Version,
			Status:      record.Status,
			InstalledAt: record.InstalledAt,
			ReportPath:  record.ReportPath,
		})
	}
	return out
}

func installedFromWorkspace(workspace *WorkspaceSummary) []InstalledPresetSummary {
	if workspace == nil {
		return nil
	}
	return workspace.InstalledPresets
}

func liveAppSummaryFromApp(item liveapp.App) LiveAppSummary {
	return LiveAppSummary{
		ID:             item.ID,
		SessionID:      item.SessionID,
		Title:          item.Title,
		Origin:         item.Origin,
		Type:           item.Type,
		Status:         item.Status,
		Port:           item.Port,
		PreviewURL:     item.PreviewURL,
		StartedAt:      item.StartedAt,
		UpdatedAt:      item.UpdatedAt,
		AutoStopPolicy: item.AutoStopPolicy,
		StopReason:     item.StopReason,
	}
}

func liveAppDetailFromApp(item liveapp.App) LiveAppDetail {
	return LiveAppDetail{
		LiveAppSummary: liveAppSummaryFromApp(item),
		SourcePath:     item.SourcePath,
		StdoutPath:     item.StdoutPath,
		StderrPath:     item.StderrPath,
		Command:        append([]string{}, item.Command...),
	}
}

func materialLaunchable(path string, files []string) bool {
	if launchablePath(path) {
		return true
	}
	for _, file := range files {
		if launchablePath(file) {
			return true
		}
	}
	return false
}

func materialSourcePath(root string, material SessionMaterialCard) string {
	if material.Path != "" {
		candidate := material.Path
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(root, candidate)
		}
		if launchablePath(candidate) {
			return candidate
		}
	}
	for _, file := range material.Files {
		candidate := file
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(root, candidate)
		}
		if launchablePath(candidate) {
			return candidate
		}
	}
	return ""
}

func launchablePath(path string) bool {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return false
	}
	info, err := os.Stat(clean)
	if err == nil && info.IsDir() {
		if _, err := os.Stat(filepath.Join(clean, "index.html")); err == nil {
			return true
		}
	}
	switch strings.ToLower(filepath.Ext(clean)) {
	case ".html", ".htm", ".svg":
		return true
	default:
		return false
	}
}

func ensureLessonDemo(root string, lessonID string) (string, string, error) {
	switch strings.TrimSpace(lessonID) {
	case "", "cellular-respiration":
		dir := project.ProjectFile(root, "lessons", "cellular-respiration")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", "", err
		}
		indexPath := filepath.Join(dir, "index.html")
		if err := os.WriteFile(indexPath, []byte(cellularRespirationDemoHTML()), 0o644); err != nil {
			return "", "", err
		}
		if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(cellularRespirationNotesMarkdown()), 0o644); err != nil {
			return "", "", err
		}
		if err := os.WriteFile(filepath.Join(dir, "self-check.md"), []byte(cellularRespirationSelfCheckMarkdown()), 0o644); err != nil {
			return "", "", err
		}
		if err := os.WriteFile(filepath.Join(dir, "diagram.svg"), []byte(cellularRespirationDiagramSVG()), 0o644); err != nil {
			return "", "", err
		}
		return dir, "Мини-симуляция: клеточное дыхание", nil
	default:
		return "", "", fmt.Errorf("unknown lesson %q", lessonID)
	}
}

func lessonMaterialCards(root string, session chat.Session) []SessionMaterialCard {
	lessonID := strings.TrimSpace(session.Metadata["lesson_id"])
	if lessonID == "" {
		return nil
	}
	dir, title, err := ensureLessonDemo(root, lessonID)
	if err != nil {
		return nil
	}
	return []SessionMaterialCard{
		{
			ID:         session.ID + ":lesson:notes",
			Type:       "document",
			Title:      title + " · Конспект",
			Summary:    "Короткое объяснение темы простыми словами.",
			Source:     "lesson",
			Path:       filepath.Join(dir, "notes.md"),
			Preview:    readArtifactPreview(filepath.Join(dir, "notes.md"), 2400),
			OpenLabel:  "Открыть конспект",
			Launchable: false,
		},
		{
			ID:        session.ID + ":lesson:diagram",
			Type:      "diagram",
			Title:     title + " · Диаграмма",
			Summary:   "Визуальная схема пути глюкозы к ATP.",
			Source:    "lesson",
			Path:      filepath.Join(dir, "diagram.svg"),
			Preview:   readArtifactPreview(filepath.Join(dir, "diagram.svg"), 12000),
			OpenLabel: "Открыть диаграмму",
		},
		{
			ID:         session.ID + ":lesson:selfcheck",
			Type:       "document",
			Title:      title + " · Самопроверка",
			Summary:    "Вопросы на понимание, чтобы закрепить материал.",
			Source:     "lesson",
			Path:       filepath.Join(dir, "self-check.md"),
			Preview:    readArtifactPreview(filepath.Join(dir, "self-check.md"), 2400),
			OpenLabel:  "Открыть вопросы",
			Launchable: false,
		},
		{
			ID:         session.ID + ":lesson:demo",
			Type:       "simulation",
			Title:      title,
			Summary:    "Интерактивная мини-симуляция доступна прямо в приложении.",
			Source:     "lesson",
			Path:       filepath.Join(dir, "index.html"),
			Preview:    "Локальная мини-симуляция готова к запуску.",
			OpenLabel:  "Открыть демо",
			Launchable: true,
		},
	}
}

func messageArtifactMaterials(root string, session chat.Session) []SessionMaterialCard {
	out := []SessionMaterialCard{}
	for i := len(session.Messages) - 1; i >= 0; i-- {
		message := session.Messages[i]
		if message.Role != "assistant" {
			continue
		}
		outputPaths := map[string]struct{}{}
		for _, output := range message.Outputs {
			if strings.TrimSpace(output.Path) != "" {
				outputPaths[output.Path] = struct{}{}
			}
		}
		for key, path := range message.Artifacts {
			if strings.TrimSpace(path) == "" {
				continue
			}
			switch key {
			case "last_message", "transcript", "stderr":
				continue
			}
			if _, ok := outputPaths[path]; ok {
				continue
			}
			cardType, title, summary := classifyMessageArtifact(key, path)
			previewLimit := 2400
			if cardType == "diagram" {
				previewLimit = 12000
			}
			out = append(out, SessionMaterialCard{
				ID:         fmt.Sprintf("%s:%s:%s", session.ID, key, filepath.Base(path)),
				Type:       cardType,
				Title:      title,
				Summary:    summary,
				Source:     "assistant",
				Preview:    readArtifactPreview(path, previewLimit),
				Path:       relativeArtifactPath(root, path),
				Files:      []string{relativeArtifactPath(root, path)},
				OpenLabel:  "Открыть материал",
				Launchable: launchablePath(path),
			})
		}
	}
	return out
}

func relativeArtifactPath(root string, path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.ToSlash(rel)
}

func cellularRespirationDemoHTML() string {
	return `<!doctype html>
<html lang="ru">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <title>Клеточное дыхание</title>
    <style>
      :root { color-scheme: light; --bg:#f6f9fc; --card:#ffffff; --line:#d9e4ef; --text:#17324a; --muted:#59708a; --blue:#2f7ce0; --orange:#f59b23; --green:#1d9f64; }
      * { box-sizing:border-box; }
      body { margin:0; font-family: Avenir Next, Segoe UI, sans-serif; color:var(--text); background:linear-gradient(180deg,#f7fbff 0%,#eef4fb 100%); }
      .wrap { max-width:1100px; margin:0 auto; padding:28px; display:grid; gap:20px; }
      .hero, .card { background:var(--card); border:1px solid var(--line); border-radius:20px; box-shadow:0 18px 40px rgba(18,35,52,.08); padding:20px; }
      h1,h2 { margin:0 0 12px; }
      p { line-height:1.6; color:var(--muted); }
      .diagram { display:grid; grid-template-columns:repeat(7,1fr); gap:12px; align-items:center; }
      .stage { background:#f8fbff; border:1px solid var(--line); border-radius:18px; padding:16px; min-height:146px; grid-column:span 2; }
      .arrow { text-align:center; font-size:28px; color:var(--orange); }
      .controls { display:grid; grid-template-columns:1.3fr .7fr; gap:18px; }
      input[type=range] { width:100%; }
      .meter { display:grid; gap:10px; }
      .pill { display:inline-flex; padding:8px 12px; border-radius:999px; background:#edf5ff; color:var(--blue); font-weight:600; }
      .stats { display:grid; grid-template-columns:repeat(3,1fr); gap:12px; }
      .stat { background:#f8fbff; border:1px solid var(--line); border-radius:16px; padding:14px; }
      .value { font-size:28px; font-weight:700; margin-top:6px; }
      .questions { display:grid; gap:12px; }
      .question { background:#fbfdff; border:1px solid var(--line); border-radius:16px; padding:14px; }
      .note { color:var(--muted); font-size:14px; }
      @media (max-width: 900px) { .diagram, .controls, .stats { grid-template-columns:1fr; } .arrow { display:none; } .stage { grid-column:span 1; } }
    </style>
  </head>
  <body>
    <div class="wrap">
      <section class="hero">
        <span class="pill">Study demo</span>
        <h1>Клеточное дыхание простыми словами</h1>
        <p>Клетка берёт глюкозу и кислород, проходит через несколько этапов и в конце получает ATP — удобную форму энергии. Ниже есть схема процесса и мини-симуляция, которая показывает, что меняется, когда кислорода мало.</p>
      </section>
      <section class="card">
        <h2>Схема пути: глюкоза → ATP</h2>
        <div class="diagram">
          <div class="stage"><strong>1. Гликолиз</strong><p>Глюкоза делится на более маленькие молекулы. Уже здесь клетка получает немного ATP.</p></div>
          <div class="arrow">→</div>
          <div class="stage"><strong>2. Цикл Кребса</strong><p>Извлекается больше электронов и подготавливаются переносчики энергии.</p></div>
          <div class="arrow">→</div>
          <div class="stage"><strong>3. Цепь переноса электронов</strong><p>На внутренней мембране митохондрии создаётся основной объём ATP.</p></div>
        </div>
      </section>
      <section class="controls">
        <div class="card">
          <h2>Мини-симуляция: что будет, если кислорода мало?</h2>
          <div class="meter">
            <label for="oxygen"><strong>Уровень кислорода</strong></label>
            <input id="oxygen" type="range" min="0" max="100" value="100">
            <div class="note">Подвигай слайдер и посмотри, как меняются ATP, скорость процесса и накопление лактата.</div>
          </div>
        </div>
        <div class="card">
          <h2>Состояние клетки</h2>
          <div class="stats">
            <div class="stat"><div class="note">ATP</div><div class="value" id="atp">36</div></div>
            <div class="stat"><div class="note">Скорость</div><div class="value" id="speed">100%</div></div>
            <div class="stat"><div class="note">Лактат</div><div class="value" id="lactate">Низкий</div></div>
          </div>
          <p id="explain">Кислорода достаточно: митохондрия работает полноценно и клетка получает максимальный выход энергии.</p>
        </div>
      </section>
      <section class="card">
        <h2>Проверь себя</h2>
        <div class="questions">
          <div class="question"><strong>1.</strong> Почему без кислорода клетка получает меньше ATP?</div>
          <div class="question"><strong>2.</strong> На каком этапе образуется основная часть ATP?</div>
          <div class="question"><strong>3.</strong> Почему при нехватке кислорода растёт лактат?</div>
        </div>
      </section>
    </div>
    <script>
      const oxygen = document.getElementById("oxygen");
      const atp = document.getElementById("atp");
      const speed = document.getElementById("speed");
      const lactate = document.getElementById("lactate");
      const explain = document.getElementById("explain");
      function render() {
        const value = Number(oxygen.value);
        const atpValue = Math.max(2, Math.round(2 + value * 0.34));
        const speedValue = Math.max(15, value);
        atp.textContent = String(atpValue);
        speed.textContent = speedValue + "%";
        if (value >= 70) {
          lactate.textContent = "Низкий";
          explain.textContent = "Кислорода достаточно: митохондрия работает полноценно и клетка получает максимальный выход энергии.";
        } else if (value >= 35) {
          lactate.textContent = "Средний";
          explain.textContent = "Кислорода меньше нормы: часть процесса ещё идёт эффективно, но общий выход энергии уже падает.";
        } else {
          lactate.textContent = "Высокий";
          explain.textContent = "Кислорода мало: клетка сильнее опирается на менее выгодные пути и получает мало ATP, а лактат растёт.";
        }
      }
      oxygen.addEventListener("input", render);
      render();
    </script>
  </body>
</html>`
}

func cellularRespirationNotesMarkdown() string {
	return `# Клеточное дыхание

Клеточное дыхание — это способ, которым клетка превращает глюкозу в удобную энергию в форме ATP.

## Коротко по шагам

1. **Гликолиз**: глюкоза распадается на более простые молекулы, и клетка получает немного ATP.
2. **Цикл Кребса**: из молекул извлекаются электроны и подготавливаются переносчики энергии.
3. **Цепь переноса электронов**: на внутренней мембране митохондрии получается основная часть ATP.

## Почему важен кислород

Кислород нужен как конечный “приёмник” электронов. Если кислорода мало, клетка получает заметно меньше ATP и сильнее опирается на менее выгодные обходные пути.
`
}

func cellularRespirationSelfCheckMarkdown() string {
	return `# Проверь себя

1. Почему без кислорода клетка получает меньше ATP?
2. На каком этапе образуется основная часть ATP?
3. Почему при нехватке кислорода растёт лактат?
`
}

func cellularRespirationDiagramSVG() string {
	return `<svg xmlns="http://www.w3.org/2000/svg" width="960" height="360" viewBox="0 0 960 360" fill="none">
  <rect width="960" height="360" rx="28" fill="#F7FBFF"/>
  <rect x="36" y="88" width="250" height="176" rx="24" fill="#FFFFFF" stroke="#D8E5F0" stroke-width="2"/>
  <rect x="354" y="88" width="250" height="176" rx="24" fill="#FFFFFF" stroke="#D8E5F0" stroke-width="2"/>
  <rect x="672" y="88" width="250" height="176" rx="24" fill="#FFFFFF" stroke="#D8E5F0" stroke-width="2"/>
  <text x="70" y="132" fill="#18324A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="28" font-weight="700">1. Гликолиз</text>
  <text x="70" y="172" fill="#5C738A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="20">Глюкоза распадается и клетка</text>
  <text x="70" y="202" fill="#5C738A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="20">получает немного ATP.</text>
  <text x="388" y="132" fill="#18324A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="28" font-weight="700">2. Цикл Кребса</text>
  <text x="388" y="172" fill="#5C738A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="20">Извлекаются электроны и</text>
  <text x="388" y="202" fill="#5C738A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="20">готовятся переносчики энергии.</text>
  <text x="706" y="132" fill="#18324A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="28" font-weight="700">3. Цепь переноса</text>
  <text x="706" y="164" fill="#18324A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="28" font-weight="700">электронов</text>
  <text x="706" y="202" fill="#5C738A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="20">Здесь образуется основная</text>
  <text x="706" y="232" fill="#5C738A" font-family="Avenir Next, Segoe UI, sans-serif" font-size="20">часть ATP.</text>
  <path d="M300 176H340" stroke="#F5A43A" stroke-width="10" stroke-linecap="round"/>
  <path d="M620 176H658" stroke="#F5A43A" stroke-width="10" stroke-linecap="round"/>
  <path d="M328 156L352 176L328 196" stroke="#F5A43A" stroke-width="10" stroke-linecap="round" stroke-linejoin="round"/>
  <path d="M646 156L670 176L646 196" stroke="#F5A43A" stroke-width="10" stroke-linecap="round" stroke-linejoin="round"/>
  <rect x="370" y="282" width="220" height="42" rx="21" fill="#EAF5EF"/>
  <text x="422" y="309" fill="#1D7A4E" font-family="Avenir Next, Segoe UI, sans-serif" font-size="22" font-weight="700">Глюкоза → ATP</text>
</svg>`
}
