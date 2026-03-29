package desktop

import (
	"context"
	"fmt"
	"strings"

	"agent-os/internal/app"
)

type Bridge struct {
	service     app.Service
	presetsRoot string
}

type TaskPlanRequest struct {
	Path      string `json:"path"`
	Task      string `json:"task"`
	Mode      string `json:"mode"`
	Provider  string `json:"provider"`
	SessionID string `json:"session_id,omitempty"`
}

type TaskRunRequest struct {
	Path          string `json:"path"`
	Task          string `json:"task"`
	Mode          string `json:"mode"`
	Provider      string `json:"provider"`
	DryRun        bool   `json:"dry_run"`
	RunChecks     bool   `json:"run_checks"`
	AllowAutonomy bool   `json:"allow_autonomy,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
}

type TaskReviewRequest struct {
	Path  string `json:"path"`
	RunID string `json:"run_id"`
}

type ChatStartRequest struct {
	Path             string   `json:"path"`
	Provider         string   `json:"provider"`
	Mode             string   `json:"mode"`
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	DryRun           bool     `json:"dry_run"`
	Async            bool     `json:"async"`
	Action           string   `json:"action,omitempty"`
	AllowAutonomy    bool     `json:"allow_autonomy,omitempty"`
	AttachSessionIDs []string `json:"attach_session_ids,omitempty"`
}

type ChatSendRequest struct {
	Path             string   `json:"path"`
	SessionID        string   `json:"session_id"`
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	DryRun           bool     `json:"dry_run"`
	Async            bool     `json:"async"`
	Action           string   `json:"action,omitempty"`
	AllowAutonomy    bool     `json:"allow_autonomy,omitempty"`
	AttachSessionIDs []string `json:"attach_session_ids,omitempty"`
}

type WorkspaceFileSaveRequest struct {
	Path    string `json:"path"`
	File    string `json:"file"`
	Content string `json:"content"`
}

type ProjectMaterialsUploadRequest struct {
	Path      string                    `json:"path"`
	SessionID string                    `json:"session_id,omitempty"`
	Files     []app.UploadedProjectFile `json:"files"`
}

type ProjectMaterialDeleteRequest struct {
	Path       string `json:"path"`
	MaterialID string `json:"material_id"`
}

type WorkspaceInitRequest struct {
	Path     string `json:"path"`
	Provider string `json:"provider"`
	Mode     string `json:"mode"`
}

type WorkspaceModeRequest struct {
	Path string `json:"path"`
	Mode string `json:"mode"`
}

type DeveloperRoleRequest struct {
	Path string `json:"path"`
	Role string `json:"role"`
}

type PresetInstallRequest struct {
	Path           string `json:"path"`
	ID             string `json:"id"`
	AllowOverwrite bool   `json:"allow_overwrite"`
}

type PresetRollbackRequest struct {
	Path      string `json:"path"`
	InstallID string `json:"install_id"`
}

type LiveAppStartRequest struct {
	Path       string `json:"path"`
	SessionID  string `json:"session_id,omitempty"`
	MaterialID string `json:"material_id,omitempty"`
	LessonID   string `json:"lesson_id,omitempty"`
}

type LiveAppStopRequest struct {
	Path  string `json:"path"`
	AppID string `json:"app_id"`
}

type LiveAppEnsureRequest struct {
	Path  string `json:"path"`
	AppID string `json:"app_id"`
}

type TestingStartRequest struct {
	Path     string `json:"path"`
	Scenario string `json:"scenario"`
	AgentID  string `json:"agent_id,omitempty"`
	StepMode bool   `json:"step_mode"`
}

type TestingControlRequest struct {
	Path   string `json:"path"`
	RunID  string `json:"run_id"`
	Action string `json:"action"`
}

type VerificationStartRequest struct {
	Path       string `json:"path"`
	ProfileID  string `json:"profile_id,omitempty"`
	VerifierID string `json:"verifier_id,omitempty"`
}

func NewBridge(presetsRoot string) Bridge {
	return Bridge{
		service:     app.NewService(),
		presetsRoot: presetsRoot,
	}
}

func (b Bridge) PresetsRoot() string {
	return b.presetsRoot
}

func (b Bridge) Home(path string) (app.HomeSnapshot, error) {
	snapshot, err := b.service.HomeSnapshot(path)
	if err != nil {
		return app.HomeSnapshot{}, err
	}
	if b.presetsRoot != "" {
		if cards, err := b.service.ListPresetCards(b.presetsRoot); err == nil {
			snapshot.Presets = cards
		}
	}
	return snapshot, nil
}

func (b Bridge) Workspace(path string) (app.WorkspaceSummary, error) {
	return b.service.WorkspaceSummary(path)
}

func (b Bridge) ProjectState(path string) (app.ProjectState, error) {
	return b.service.ProjectState(path)
}

func (b Bridge) Agents() []app.AgentCard {
	return b.service.Agents()
}

func (b Bridge) AllowedActions(path string, mode string, sessionID string) (app.AllowedSessionActions, error) {
	return b.service.AllowedActions(path, mode, sessionID)
}

func (b Bridge) DeveloperAccess(path string) (app.DeveloperAccessState, error) {
	return b.service.DeveloperAccessState(path)
}

func (b Bridge) SetDeveloperRole(req DeveloperRoleRequest) (app.DeveloperAccessState, error) {
	return b.service.SetDeveloperRole(req.Path, req.Role)
}

func (b Bridge) TestingScenarios(path string) ([]app.TestingScenarioSummary, error) {
	return b.service.TestingScenarios(path)
}

func (b Bridge) StartTesting(req TestingStartRequest) (app.TestingRun, error) {
	return b.service.StartTestingScenario(req.Path, req.Scenario, req.AgentID, req.StepMode)
}

func (b Bridge) TestingStatus(path string, runID string) (app.TestingRun, error) {
	return b.service.TestingStatus(path, runID)
}

func (b Bridge) TestingControl(req TestingControlRequest) (app.TestingRun, error) {
	return b.service.TestingControl(req.Path, req.RunID, req.Action)
}

func (b Bridge) Verifiers(path string) ([]app.VerifierDefinition, error) {
	return b.service.Verifiers(path)
}

func (b Bridge) VerificationProfiles(path string) ([]app.VerificationProfile, error) {
	return b.service.VerificationProfiles(path)
}

func (b Bridge) StartVerification(req VerificationStartRequest) (app.VerificationRun, error) {
	switch {
	case strings.TrimSpace(req.ProfileID) != "":
		return b.service.StartVerificationProfile(req.Path, req.ProfileID)
	case strings.TrimSpace(req.VerifierID) != "":
		return b.service.RunVerifier(req.Path, req.VerifierID)
	default:
		return app.VerificationRun{}, fmt.Errorf("profile_id or verifier_id is required")
	}
}

func (b Bridge) VerificationStatus(path string, runID string) (app.VerificationRun, error) {
	return b.service.VerificationStatus(path, runID)
}

func (b Bridge) WorkspaceFiles(path string, limit int) (app.WorkspaceExplorer, error) {
	return b.service.WorkspaceExplorer(path, limit)
}

func (b Bridge) ProjectMaterials(path string) ([]app.ProjectMaterialSummary, error) {
	return b.service.ProjectMaterials(path)
}

func (b Bridge) UploadProjectMaterials(req ProjectMaterialsUploadRequest) ([]app.ProjectMaterialSummary, error) {
	return b.service.UploadProjectMaterials(req.Path, req.SessionID, req.Files)
}

func (b Bridge) DeleteProjectMaterial(req ProjectMaterialDeleteRequest) ([]app.ProjectMaterialSummary, error) {
	return b.service.DeleteProjectMaterial(req.Path, req.MaterialID)
}

func (b Bridge) WorkspaceFile(path string, relPath string) (app.WorkspaceFileDetail, error) {
	return b.service.WorkspaceFileDetail(path, relPath)
}

func (b Bridge) SaveWorkspaceFile(req WorkspaceFileSaveRequest) (app.WorkspaceFileDetail, error) {
	return b.service.SaveWorkspaceFile(req.Path, req.File, req.Content)
}

func (b Bridge) InitWorkspace(req WorkspaceInitRequest) (app.WorkspaceSummary, error) {
	return b.service.InitWorkspace(req.Path, req.Provider, req.Mode)
}

func (b Bridge) SetWorkspaceMode(req WorkspaceModeRequest) (app.WorkspaceSummary, error) {
	return b.service.SetWorkspaceMode(req.Path, req.Mode)
}

func (b Bridge) Providers() []app.ProviderHealth {
	return b.service.ProviderHealth(context.Background())
}

func (b Bridge) Runs(path string, limit int) ([]app.RunSummary, error) {
	return b.service.ListRuns(path, limit)
}

func (b Bridge) Run(path string, runID string) (app.RunDetail, error) {
	return b.service.RunDetail(path, runID)
}

func (b Bridge) TaskPlan(req TaskPlanRequest) (app.RunSummary, error) {
	return b.service.StartTaskPlanAsync(req.Path, req.Task, req.Mode, req.Provider, req.SessionID)
}

func (b Bridge) TaskRun(req TaskRunRequest) (app.RunSummary, error) {
	return b.service.StartTaskRunAsync(req.Path, req.Task, req.Mode, req.Provider, req.DryRun, req.RunChecks, req.AllowAutonomy, req.SessionID)
}

func (b Bridge) TaskReview(req TaskReviewRequest) (app.RunDetail, error) {
	return b.service.ReviewRun(req.Path, req.RunID)
}

func (b Bridge) Chats(path string, limit int) ([]app.ChatSummary, error) {
	return b.service.ListChats(path, limit)
}

func (b Bridge) Sessions(path string, limit int, query string, mode string, status string) ([]app.SessionSummary, error) {
	return b.service.ListSessions(path, limit, query, mode, status)
}

func (b Bridge) Session(path string, sessionID string) (app.SessionDetail, error) {
	return b.service.SessionDetail(path, sessionID)
}

func (b Bridge) LiveApps(path string, sessionID string) ([]app.LiveAppSummary, error) {
	return b.service.ListLiveApps(path, sessionID)
}

func (b Bridge) StartMaterialLiveApp(req LiveAppStartRequest) (app.LiveAppDetail, error) {
	return b.service.StartMaterialLiveApp(req.Path, req.SessionID, req.MaterialID)
}

func (b Bridge) StartLessonDemo(req LiveAppStartRequest) (app.LiveAppDetail, error) {
	return b.service.LaunchLessonDemo(req.Path, req.SessionID, req.LessonID)
}

func (b Bridge) EnsureLiveApp(req LiveAppEnsureRequest) (app.LiveAppDetail, error) {
	return b.service.EnsureLiveApp(req.Path, req.AppID)
}

func (b Bridge) StopLiveApp(req LiveAppStopRequest) (app.LiveAppDetail, error) {
	return b.service.StopLiveApp(req.Path, req.AppID)
}

func (b Bridge) Chat(path string, sessionID string) (app.ChatDetail, error) {
	return b.service.ChatDetail(path, sessionID)
}

func (b Bridge) ChatStart(req ChatStartRequest) (app.ChatDetail, error) {
	if req.Async {
		return b.service.StartChatAsync(req.Path, req.Provider, req.Mode, req.Model, req.Prompt, req.DryRun, req.Action, req.AllowAutonomy, req.AttachSessionIDs)
	}
	return b.service.StartChat(req.Path, req.Provider, req.Mode, req.Model, req.Prompt, req.DryRun, req.Action, req.AllowAutonomy, req.AttachSessionIDs)
}

func (b Bridge) ChatSend(req ChatSendRequest) (app.ChatDetail, error) {
	if req.Async {
		return b.service.SendChatAsync(req.Path, req.SessionID, req.Model, req.Prompt, req.DryRun, req.Action, req.AllowAutonomy, req.AttachSessionIDs)
	}
	return b.service.SendChat(req.Path, req.SessionID, req.Model, req.Prompt, req.DryRun, req.Action, req.AllowAutonomy, req.AttachSessionIDs)
}

func (b Bridge) Memory(path string) (app.MemorySummary, error) {
	return b.service.MemoryStatus(path)
}

func (b Bridge) Presets() ([]app.PresetCard, error) {
	return b.service.ListPresetCards(b.presetsRoot)
}

func (b Bridge) Preset(id string) (app.PresetPreview, error) {
	return b.service.PresetPreview(b.presetsRoot, id)
}

func (b Bridge) PresetInstallPreview(path string, id string) (app.PresetInstallPreview, error) {
	return b.service.PreviewPresetInstall(path, b.presetsRoot, id)
}

func (b Bridge) PresetInstall(req PresetInstallRequest) (app.PresetInstallResult, error) {
	return b.service.InstallPreset(req.Path, b.presetsRoot, req.ID, req.AllowOverwrite)
}

func (b Bridge) PresetRollback(req PresetRollbackRequest) (app.InstalledPresetSummary, error) {
	return b.service.RollbackPreset(req.Path, req.InstallID)
}
