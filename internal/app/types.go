package app

import (
	"agent-os/internal/memory"
	"agent-os/internal/presets"
)

type MemorySummary = memory.Summary
type PresetInstallPreview = presets.InstallPreview
type PresetInstallResult = presets.InstallResult

type ProjectState struct {
	Path      string            `json:"path"`
	Name      string            `json:"name"`
	State     string            `json:"state"`
	Message   string            `json:"message,omitempty"`
	Workspace *WorkspaceSummary `json:"workspace,omitempty"`
}

type AgentCard struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	BuiltIn     bool   `json:"built_in"`
}

type AllowedSessionActions struct {
	AgentID               string   `json:"agent_id"`
	AgentName             string   `json:"agent_name"`
	CanExplain            bool     `json:"can_explain"`
	CanPlan               bool     `json:"can_plan"`
	CanSafeRun            bool     `json:"can_safe_run"`
	CanDo                 bool     `json:"can_do"`
	DoRequiresUnlock      bool     `json:"do_requires_unlock,omitempty"`
	StudyFallbackAction   string   `json:"study_fallback_action,omitempty"`
	UnavailableReasonSafe string   `json:"unavailable_reason_safe,omitempty"`
	UnavailableReasonDo   string   `json:"unavailable_reason_do,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

type DeveloperAccessState struct {
	Role              string   `json:"role"`
	CanUseTesting     bool     `json:"can_use_testing"`
	CanManageRole     bool     `json:"can_manage_role,omitempty"`
	AllowedRoles      []string `json:"allowed_roles,omitempty"`
	VisibleScreens    []string `json:"visible_screens,omitempty"`
	AvailableFeatures []string `json:"available_features,omitempty"`
	Source            string   `json:"source,omitempty"`
}

type VerifierDefinition struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Domain   string `json:"domain"`
	Blocking bool   `json:"blocking"`
}

type VerificationProfile struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	VerifierIDs []string `json:"verifier_ids"`
}

type VerificationFinding struct {
	Severity       string   `json:"severity"`
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	Evidence       []string `json:"evidence,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
}

type VerificationResult struct {
	VerifierID       string                `json:"verifier_id"`
	Title            string                `json:"title"`
	Domain           string                `json:"domain"`
	Blocking         bool                  `json:"blocking"`
	Verdict          string                `json:"verdict"`
	Severity         string                `json:"severity"`
	Summary          string                `json:"summary"`
	Findings         []VerificationFinding `json:"findings,omitempty"`
	ReportPath       string                `json:"report_path,omitempty"`
	BlockedGoals     []string              `json:"blocked_goals,omitempty"`
	RecommendedNext  string                `json:"recommended_next,omitempty"`
}

type VerificationRun struct {
	ID               string               `json:"id"`
	ProfileID        string               `json:"profile_id,omitempty"`
	VerifierID       string               `json:"verifier_id,omitempty"`
	Title            string               `json:"title"`
	Summary          string               `json:"summary,omitempty"`
	Status           string               `json:"status"`
	OverallVerdict   string               `json:"overall_verdict"`
	CreatedAt        string               `json:"created_at"`
	UpdatedAt        string               `json:"updated_at"`
	LastError        string               `json:"last_error,omitempty"`
	BlockingFailures int                  `json:"blocking_failures"`
	WarningCount     int                  `json:"warning_count"`
	Results          []VerificationResult `json:"results,omitempty"`
	SummaryPath      string               `json:"summary_path,omitempty"`
}

type SessionSummary struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Summary              string   `json:"summary"`
	AgentID              string   `json:"agent_id"`
	AgentName            string   `json:"agent_name"`
	Mode                 string   `json:"mode"`
	Status               string   `json:"status"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
	LastUserMessage      string   `json:"last_user_message,omitempty"`
	LastAssistantMessage string   `json:"last_assistant_message,omitempty"`
	RelatedRunIDs        []string `json:"related_run_ids,omitempty"`
	MaterialCount        int      `json:"material_count"`
}

type ProjectMaterialSummary struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Path             string `json:"path"`
	Size             int64  `json:"size"`
	UploadedAt       string `json:"uploaded_at"`
	Source           string `json:"source"`
	RelatedSessionID string `json:"related_session_id,omitempty"`
	MimeType         string `json:"mime_type,omitempty"`
}

type SessionMaterialCard struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Source     string   `json:"source"`
	Preview    string   `json:"preview,omitempty"`
	Path       string   `json:"path,omitempty"`
	URL        string   `json:"url,omitempty"`
	Files      []string `json:"files,omitempty"`
	OpenLabel  string   `json:"open_label,omitempty"`
	Launchable bool     `json:"launchable,omitempty"`
	LiveAppID  string   `json:"live_app_id,omitempty"`
	Status     string   `json:"status,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type SessionDetail struct {
	Session        SessionSummary        `json:"session"`
	Messages       []ChatMessage         `json:"messages"`
	Runs           []RunSummary          `json:"runs,omitempty"`
	Materials      []SessionMaterialCard `json:"materials,omitempty"`
	LiveApps       []LiveAppSummary      `json:"live_apps,omitempty"`
	Metadata       map[string]string     `json:"metadata,omitempty"`
	Live           *ChatLiveState        `json:"live,omitempty"`
	NextAction     string                `json:"next_action,omitempty"`
	ProjectRoot    string                `json:"project_root,omitempty"`
	AllowedActions AllowedSessionActions `json:"allowed_actions"`
}

type TestingScenarioSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	AgentID string `json:"agent_id"`
	Steps   int    `json:"steps"`
}

type TestingStep struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Status     string `json:"status"`
	Details    string `json:"details,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	LiveAppID  string `json:"live_app_id,omitempty"`
	PreviewURL string `json:"preview_url,omitempty"`
}

type TestingRun struct {
	ID          string        `json:"id"`
	ScenarioID  string        `json:"scenario_id"`
	Title       string        `json:"title"`
	Summary     string        `json:"summary,omitempty"`
	AgentID     string        `json:"agent_id"`
	Status      string        `json:"status"`
	StepMode    bool          `json:"step_mode"`
	CurrentStep int           `json:"current_step"`
	CreatedAt   string        `json:"created_at"`
	UpdatedAt   string        `json:"updated_at"`
	SessionID   string        `json:"session_id,omitempty"`
	LastError   string        `json:"last_error,omitempty"`
	Steps       []TestingStep `json:"steps"`
}

type LiveAppSummary struct {
	ID             string `json:"id"`
	SessionID      string `json:"session_id,omitempty"`
	Title          string `json:"title"`
	Origin         string `json:"origin,omitempty"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	Port           int    `json:"port,omitempty"`
	PreviewURL     string `json:"preview_url,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	AutoStopPolicy string `json:"auto_stop_policy,omitempty"`
	StopReason     string `json:"stop_reason,omitempty"`
}

type LiveAppDetail struct {
	LiveAppSummary
	SourcePath string   `json:"source_path,omitempty"`
	StdoutPath string   `json:"stdout_path,omitempty"`
	StderrPath string   `json:"stderr_path,omitempty"`
	Command    []string `json:"command,omitempty"`
}

type WorkspaceSummary struct {
	Root             string                   `json:"root"`
	Name             string                   `json:"name"`
	ArcDir           string                   `json:"arc_dir"`
	DefaultProvider  string                   `json:"default_provider"`
	EnabledProviders []string                 `json:"enabled_providers"`
	Mode             string                   `json:"mode"`
	Autonomy         string                   `json:"autonomy"`
	Index            IndexStatus              `json:"index"`
	Memory           memory.Summary           `json:"memory"`
	Docs             DocsStatus               `json:"docs"`
	LastRun          *RunSummary              `json:"last_run,omitempty"`
	Questions        []QuestionEntry          `json:"questions,omitempty"`
	InstalledPresets []InstalledPresetSummary `json:"installed_presets,omitempty"`
}

type WorkspaceExplorer struct {
	Files         []WorkspaceFileEntry `json:"files"`
	RecentChanges []WorkspaceChange    `json:"recent_changes,omitempty"`
	DirtyFiles    []WorkspaceGitChange `json:"dirty_files,omitempty"`
}

type WorkspaceFileEntry struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

type WorkspaceFileDetail struct {
	Path         string              `json:"path"`
	Kind         string              `json:"kind"`
	Size         int64               `json:"size"`
	ModTime      string              `json:"mod_time"`
	Content      string              `json:"content,omitempty"`
	Truncated    bool                `json:"truncated"`
	Editable     bool                `json:"editable"`
	Symbols      []WorkspaceSymbol   `json:"symbols,omitempty"`
	DocTitle     string              `json:"doc_title,omitempty"`
	DocHeadings  []string            `json:"doc_headings,omitempty"`
	RecentChange *WorkspaceChange    `json:"recent_change,omitempty"`
	GitChange    *WorkspaceGitChange `json:"git_change,omitempty"`
}

type WorkspaceSymbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Language string `json:"language"`
	Line     int    `json:"line"`
}

type WorkspaceChange struct {
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Author  string `json:"author"`
	Subject string `json:"subject"`
}

type WorkspaceGitChange struct {
	Path        string `json:"path"`
	Status      string `json:"status"`
	DiffPreview string `json:"diff_preview,omitempty"`
}

type IndexStatus struct {
	Ready        bool `json:"ready"`
	Files        int  `json:"files"`
	Symbols      int  `json:"symbols"`
	Dependencies int  `json:"dependencies"`
	Docs         int  `json:"docs"`
}

type DocsStatus struct {
	Ready         bool     `json:"ready"`
	GeneratedMaps []string `json:"generated_maps"`
}

type ProviderHealth struct {
	Name         string   `json:"name"`
	Installed    bool     `json:"installed"`
	BinaryPath   string   `json:"binary_path,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Notes        []string `json:"notes,omitempty"`
}

type RunSummary struct {
	ID            string `json:"id"`
	Task          string `json:"task"`
	Status        string `json:"status"`
	State         string `json:"state"`
	Mode          string `json:"mode"`
	Provider      string `json:"provider"`
	StartedAt     string `json:"started_at"`
	UpdatedAt     string `json:"updated_at"`
	DryRun        bool   `json:"dry_run"`
	ArtifactCount int    `json:"artifact_count"`
}

type ArtifactSummary struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type RunDetail struct {
	Run       RunSummary        `json:"run"`
	Artifacts []ArtifactSummary `json:"artifacts"`
	Changed   []string          `json:"changed_files,omitempty"`
	Previews  map[string]string `json:"previews,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Docs      map[string]string `json:"docs,omitempty"`
	Review    map[string]string `json:"review,omitempty"`
	Verify    map[string]string `json:"verify,omitempty"`
}

type QuestionEntry struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
}

type HomeSnapshot struct {
	Workspace *WorkspaceSummary        `json:"workspace,omitempty"`
	Providers []ProviderHealth         `json:"providers"`
	Runs      []RunSummary             `json:"runs"`
	Presets   []PresetCard             `json:"presets"`
	Installed []InstalledPresetSummary `json:"installed,omitempty"`
	Chats     []ChatSummary            `json:"chats,omitempty"`
}

type PresetCard struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Tagline    string   `json:"tagline"`
	Goal       string   `json:"goal"`
	Adapter    string   `json:"adapter"`
	Category   string   `json:"category"`
	Version    string   `json:"version"`
	AuthorName string   `json:"author_name"`
	Trust      string   `json:"trust"`
	Files      []string `json:"files,omitempty"`
	Path       string   `json:"path"`
}

type PresetPreview struct {
	Manifest PresetCard `json:"manifest"`
	Files    []string   `json:"files"`
	Readme   string     `json:"readme,omitempty"`
}

type InstalledPresetSummary struct {
	InstallID   string `json:"install_id"`
	PresetID    string `json:"preset_id"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	InstalledAt string `json:"installed_at"`
	ReportPath  string `json:"report_path,omitempty"`
}

type ChatSummary struct {
	ID                   string `json:"id"`
	Provider             string `json:"provider"`
	Mode                 string `json:"mode"`
	Status               string `json:"status"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
	ProviderSessionID    string `json:"provider_session_id,omitempty"`
	LastUserMessage      string `json:"last_user_message,omitempty"`
	LastAssistantMessage string `json:"last_assistant_message,omitempty"`
	MessageCount         int    `json:"message_count"`
}

type ChatDetail struct {
	Session  ChatSummary       `json:"session"`
	Messages []ChatMessage     `json:"messages"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Live     *ChatLiveState    `json:"live,omitempty"`
}

type ChatMessage struct {
	Turn             int               `json:"turn"`
	Role             string            `json:"role"`
	Content          string            `json:"content"`
	CreatedAt        string            `json:"created_at"`
	Artifacts        map[string]string `json:"artifacts,omitempty"`
	ArtifactPreviews map[string]string `json:"artifact_previews,omitempty"`
	Outputs          []MessageOutputRef `json:"outputs,omitempty"`
	Failure          string            `json:"failure,omitempty"`
}

type MessageOutputRef struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Preview    string `json:"preview,omitempty"`
	Path       string `json:"path,omitempty"`
	URL        string `json:"url,omitempty"`
	Launchable bool   `json:"launchable,omitempty"`
	Inline     bool   `json:"inline,omitempty"`
	LiveAppID  string `json:"live_app_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
}

type ChatLiveState struct {
	Turn             int               `json:"turn"`
	Artifacts        map[string]string `json:"artifacts,omitempty"`
	ArtifactPreviews map[string]string `json:"artifact_previews,omitempty"`
}
