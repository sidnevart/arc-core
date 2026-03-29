package wailsapp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"agent-os/internal/app"
	"agent-os/internal/desktop"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	bridge  desktop.Bridge
	logMu   sync.Mutex
	logFile *os.File
	logger  *log.Logger
	logPath string

	settingsMu         sync.Mutex
	uiSettings         desktopUISettings
	chatScaleMenuItems map[int]*menu.MenuItem
}

func New() *App {
	presetsRoot := resolvePresetsRoot()
	return &App{
		bridge:             desktop.NewBridge(presetsRoot),
		logPath:            "",
		uiSettings:         loadDesktopUISettings(),
		chatScaleMenuItems: map[int]*menu.MenuItem{},
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.initLogger()
	a.logf("info", "desktop startup completed")
	a.logf("info", "desktop presets root: "+a.bridge.PresetsRoot())
}

func (a *App) BeforeClose(ctx context.Context) bool {
	a.logf("info", "desktop shutdown requested")
	a.closeLogger()
	return false
}

func desktopLogPath() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, "Library", "Logs", "ARC Desktop", "desktop.log")
	}
	return filepath.Join(os.TempDir(), "arc-desktop.log")
}

func desktopLogCandidates() []string {
	primary := desktopLogPath()
	fallback := filepath.Join(os.TempDir(), "arc-desktop.log")
	if primary == fallback {
		return []string{primary}
	}
	return []string{primary, fallback}
}

func resolvePresetsRoot() string {
	candidates := []string{
		"presets/official",
		"../presets/official",
		"../../presets/official",
	}
	for _, candidate := range candidates {
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		return "presets/official"
	}
	dir := filepath.Dir(execPath)
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "presets", "official")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "presets/official"
}

func (a *App) initLogger() {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	if a.logger != nil {
		return
	}
	for _, candidate := range desktopLogCandidates() {
		if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
			continue
		}
		file, err := os.OpenFile(candidate, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			continue
		}
		a.logPath = candidate
		a.logFile = file
		a.logger = log.New(file, "", log.LstdFlags|log.Lmicroseconds)
		return
	}
	a.logPath = filepath.Join(os.TempDir(), "arc-desktop.log")
}

func (a *App) closeLogger() {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	if a.logFile != nil {
		_ = a.logFile.Close()
	}
	a.logFile = nil
	a.logger = nil
}

func (a *App) logf(level string, message string) {
	a.logMu.Lock()
	defer a.logMu.Unlock()
	if a.logger == nil {
		for _, candidate := range desktopLogCandidates() {
			if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
				continue
			}
			file, err := os.OpenFile(candidate, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				continue
			}
			a.logPath = candidate
			a.logFile = file
			a.logger = log.New(file, "", log.LstdFlags|log.Lmicroseconds)
			break
		}
	}
	if a.logger == nil {
		log.Printf("[desktop][%s] %s", level, message)
		return
	}
	a.logger.Printf("[%s] %s", level, message)
}

func normalizeDirectoryDialogStart(path string) string {
	candidate := path
	if candidate == "" {
		candidate = "."
	}
	abs, err := filepath.Abs(candidate)
	if err == nil {
		candidate = abs
	}
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	return "."
}

func (a *App) ChooseWorkspaceDirectory(startPath string) (string, error) {
	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                      "Выберите проект",
		DefaultDirectory:           normalizeDirectoryDialogStart(startPath),
		CanCreateDirectories:       true,
		ShowHiddenFiles:            true,
		TreatPackagesAsDirectories: true,
		ResolvesAliases:            true,
	})
	if err != nil {
		a.logf("error", "ChooseWorkspaceDirectory failed: "+err.Error())
		return "", err
	}
	if selected == "" {
		a.logf("info", "ChooseWorkspaceDirectory cancelled")
	} else {
		a.logf("info", "ChooseWorkspaceDirectory selected: "+selected)
	}
	return selected, nil
}

func (a *App) DesktopLogPath() string {
	if a.logPath == "" {
		a.logPath = desktopLogCandidates()[0]
	}
	return a.logPath
}

func (a *App) OpenExternalURL(rawURL string) error {
	url := strings.TrimSpace(rawURL)
	if url == "" {
		return fmt.Errorf("url is required")
	}
	runtime.BrowserOpenURL(a.ctx, url)
	a.logf("info", "OpenExternalURL: "+url)
	return nil
}

func (a *App) LogFrontend(level string, message string) {
	if level == "" {
		level = "info"
	}
	a.logf(level, fmt.Sprintf("[frontend] %s", message))
}

func (a *App) Home(path string) (app.HomeSnapshot, error) {
	return a.bridge.Home(path)
}

func (a *App) ProjectState(path string) (app.ProjectState, error) {
	return a.bridge.ProjectState(path)
}

func (a *App) Workspace(path string) (app.WorkspaceSummary, error) {
	return a.bridge.Workspace(path)
}

func (a *App) Agents() []app.AgentCard {
	return a.bridge.Agents()
}

func (a *App) AllowedActions(path string, mode string, sessionID string) (app.AllowedSessionActions, error) {
	return a.bridge.AllowedActions(path, mode, sessionID)
}

func (a *App) DeveloperAccess(path string) (app.DeveloperAccessState, error) {
	return a.bridge.DeveloperAccess(path)
}

func (a *App) SetDeveloperRole(req desktop.DeveloperRoleRequest) (app.DeveloperAccessState, error) {
	return a.bridge.SetDeveloperRole(req)
}

func (a *App) TestingScenarios(path string) ([]app.TestingScenarioSummary, error) {
	return a.bridge.TestingScenarios(path)
}

func (a *App) StartTesting(req desktop.TestingStartRequest) (app.TestingRun, error) {
	return a.bridge.StartTesting(req)
}

func (a *App) TestingStatus(path string, runID string) (app.TestingRun, error) {
	return a.bridge.TestingStatus(path, runID)
}

func (a *App) TestingControl(req desktop.TestingControlRequest) (app.TestingRun, error) {
	return a.bridge.TestingControl(req)
}

func (a *App) Verifiers(path string) ([]app.VerifierDefinition, error) {
	return a.bridge.Verifiers(path)
}

func (a *App) VerificationProfiles(path string) ([]app.VerificationProfile, error) {
	return a.bridge.VerificationProfiles(path)
}

func (a *App) StartVerification(req desktop.VerificationStartRequest) (app.VerificationRun, error) {
	return a.bridge.StartVerification(req)
}

func (a *App) VerificationStatus(path string, runID string) (app.VerificationRun, error) {
	return a.bridge.VerificationStatus(path, runID)
}

func (a *App) WorkspaceFiles(path string, limit int) (app.WorkspaceExplorer, error) {
	return a.bridge.WorkspaceFiles(path, limit)
}

func (a *App) ProjectMaterials(path string) ([]app.ProjectMaterialSummary, error) {
	return a.bridge.ProjectMaterials(path)
}

func (a *App) UploadProjectMaterials(req desktop.ProjectMaterialsUploadRequest) ([]app.ProjectMaterialSummary, error) {
	return a.bridge.UploadProjectMaterials(req)
}

func (a *App) DeleteProjectMaterial(req desktop.ProjectMaterialDeleteRequest) ([]app.ProjectMaterialSummary, error) {
	return a.bridge.DeleteProjectMaterial(req)
}

func (a *App) WorkspaceFile(path string, relPath string) (app.WorkspaceFileDetail, error) {
	return a.bridge.WorkspaceFile(path, relPath)
}

func (a *App) SaveWorkspaceFile(req desktop.WorkspaceFileSaveRequest) (app.WorkspaceFileDetail, error) {
	return a.bridge.SaveWorkspaceFile(req)
}

func (a *App) InitWorkspace(req desktop.WorkspaceInitRequest) (app.WorkspaceSummary, error) {
	summary, err := a.bridge.InitWorkspace(req)
	if err != nil {
		a.logf("error", "InitWorkspace failed for "+req.Path+": "+err.Error())
		return app.WorkspaceSummary{}, err
	}
	a.logf("info", "InitWorkspace completed for "+req.Path)
	return summary, nil
}

func (a *App) SetWorkspaceMode(req desktop.WorkspaceModeRequest) (app.WorkspaceSummary, error) {
	summary, err := a.bridge.SetWorkspaceMode(req)
	if err != nil {
		a.logf("error", "SetWorkspaceMode failed for "+req.Path+": "+err.Error())
		return app.WorkspaceSummary{}, err
	}
	a.logf("info", "SetWorkspaceMode completed for "+req.Path+": "+req.Mode)
	return summary, nil
}

func (a *App) Providers() []app.ProviderHealth {
	return a.bridge.Providers()
}

func (a *App) Runs(path string, limit int) ([]app.RunSummary, error) {
	return a.bridge.Runs(path, limit)
}

func (a *App) Run(path string, runID string) (app.RunDetail, error) {
	return a.bridge.Run(path, runID)
}

func (a *App) TaskPlan(req desktop.TaskPlanRequest) (app.RunSummary, error) {
	return a.bridge.TaskPlan(req)
}

func (a *App) TaskRun(req desktop.TaskRunRequest) (app.RunSummary, error) {
	return a.bridge.TaskRun(req)
}

func (a *App) TaskReview(req desktop.TaskReviewRequest) (app.RunDetail, error) {
	return a.bridge.TaskReview(req)
}

func (a *App) Chats(path string, limit int) ([]app.ChatSummary, error) {
	return a.bridge.Chats(path, limit)
}

func (a *App) Sessions(path string, limit int, query string, mode string, status string) ([]app.SessionSummary, error) {
	return a.bridge.Sessions(path, limit, query, mode, status)
}

func (a *App) Session(path string, sessionID string) (app.SessionDetail, error) {
	return a.bridge.Session(path, sessionID)
}

func (a *App) LiveApps(path string, sessionID string) ([]app.LiveAppSummary, error) {
	return a.bridge.LiveApps(path, sessionID)
}

func (a *App) StartMaterialLiveApp(req desktop.LiveAppStartRequest) (app.LiveAppDetail, error) {
	return a.bridge.StartMaterialLiveApp(req)
}

func (a *App) StartLessonDemo(req desktop.LiveAppStartRequest) (app.LiveAppDetail, error) {
	return a.bridge.StartLessonDemo(req)
}

func (a *App) EnsureLiveApp(req desktop.LiveAppEnsureRequest) (app.LiveAppDetail, error) {
	return a.bridge.EnsureLiveApp(req)
}

func (a *App) StopLiveApp(req desktop.LiveAppStopRequest) (app.LiveAppDetail, error) {
	return a.bridge.StopLiveApp(req)
}

func (a *App) Chat(path string, sessionID string) (app.ChatDetail, error) {
	return a.bridge.Chat(path, sessionID)
}

func (a *App) ChatStart(req desktop.ChatStartRequest) (app.ChatDetail, error) {
	return a.bridge.ChatStart(req)
}

func (a *App) ChatSend(req desktop.ChatSendRequest) (app.ChatDetail, error) {
	return a.bridge.ChatSend(req)
}

func (a *App) Memory(path string) (app.MemorySummary, error) {
	return a.bridge.Memory(path)
}

func (a *App) Presets() ([]app.PresetCard, error) {
	return a.bridge.Presets()
}

func (a *App) Preset(id string) (app.PresetPreview, error) {
	return a.bridge.Preset(id)
}

func (a *App) PresetInstallPreview(path string, id string) (app.PresetInstallPreview, error) {
	return a.bridge.PresetInstallPreview(path, id)
}

func (a *App) PresetInstall(req desktop.PresetInstallRequest) (app.PresetInstallResult, error) {
	return a.bridge.PresetInstall(req)
}

func (a *App) PresetRollback(req desktop.PresetRollbackRequest) (app.InstalledPresetSummary, error) {
	return a.bridge.PresetRollback(req)
}
