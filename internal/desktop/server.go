package desktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	bridge     Bridge
	uiRoot     string
	sharedRoot string
}

func NewServer(uiRoot string, presetsRoot string) Server {
	return Server{
		bridge:     NewBridge(presetsRoot),
		uiRoot:     uiRoot,
		sharedRoot: filepath.Join(filepath.Dir(uiRoot), "wailsapp", "frontend", "shared"),
	}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/home", s.handleHome)
	mux.HandleFunc("/api/project-state", s.handleProjectState)
	mux.HandleFunc("/api/workspace", s.handleWorkspace)
	mux.HandleFunc("/api/init-workspace", s.handleInitWorkspace)
	mux.HandleFunc("/api/workspace-mode", s.handleWorkspaceMode)
	mux.HandleFunc("/api/workspace-files", s.handleWorkspaceFiles)
	mux.HandleFunc("/api/workspace-file", s.handleWorkspaceFile)
	mux.HandleFunc("/api/workspace-file-save", s.handleWorkspaceFileSave)
	mux.HandleFunc("/api/project-materials", s.handleProjectMaterials)
	mux.HandleFunc("/api/project-materials-upload", s.handleProjectMaterialsUpload)
	mux.HandleFunc("/api/project-materials-delete", s.handleProjectMaterialsDelete)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/allowed-actions", s.handleAllowedActions)
	mux.HandleFunc("/api/developer-access", s.handleDeveloperAccess)
	mux.HandleFunc("/api/developer-role", s.handleDeveloperRole)
	mux.HandleFunc("/api/providers", s.handleProviders)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/task-plan", s.handleTaskPlan)
	mux.HandleFunc("/api/task-run", s.handleTaskRun)
	mux.HandleFunc("/api/task-review", s.handleTaskReview)
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/live-apps", s.handleLiveApps)
	mux.HandleFunc("/api/live-app-start", s.handleLiveAppStart)
	mux.HandleFunc("/api/live-app-ensure", s.handleLiveAppEnsure)
	mux.HandleFunc("/api/live-app-stop", s.handleLiveAppStop)
	mux.HandleFunc("/api/testing-scenarios", s.handleTestingScenarios)
	mux.HandleFunc("/api/testing-start", s.handleTestingStart)
	mux.HandleFunc("/api/testing-status", s.handleTestingStatus)
	mux.HandleFunc("/api/testing-control", s.handleTestingControl)
	mux.HandleFunc("/api/verifiers", s.handleVerifiers)
	mux.HandleFunc("/api/verification-profiles", s.handleVerificationProfiles)
	mux.HandleFunc("/api/verification-start", s.handleVerificationStart)
	mux.HandleFunc("/api/verification-status", s.handleVerificationStatus)
	mux.HandleFunc("/api/chats", s.handleChats)
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/chat-start", s.handleChatStart)
	mux.HandleFunc("/api/chat-send", s.handleChatSend)
	mux.HandleFunc("/api/memory", s.handleMemory)
	mux.HandleFunc("/api/presets", s.handlePresets)
	mux.HandleFunc("/api/preset", s.handlePreset)
	mux.HandleFunc("/api/preset-preview", s.handlePresetPreview)
	mux.HandleFunc("/api/preset-install", s.handlePresetInstall)
	mux.HandleFunc("/api/preset-rollback", s.handlePresetRollback)
	mux.Handle("/", s.staticHandler())
	return mux
}

func (s Server) staticHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, filepath.Join(s.uiRoot, "index.html"))
			return
		}
		trimmed := r.URL.Path
		for len(trimmed) > 0 && trimmed[0] == '/' {
			trimmed = trimmed[1:]
		}
		if strings.HasPrefix(trimmed, "shared/") {
			if path, ok := safeStaticAssetPath(s.sharedRoot, strings.TrimPrefix(trimmed, "shared/")); ok {
				if _, err := os.Stat(path); err == nil {
					http.ServeFile(w, r, path)
					return
				}
			}
			http.NotFound(w, r)
			return
		}
		path, ok := safeStaticAssetPath(s.uiRoot, trimmed)
		if !ok {
			http.NotFound(w, r)
			return
		}
		if _, err := os.Stat(path); err == nil {
			http.ServeFile(w, r, path)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.uiRoot, "index.html"))
	})
}

func safeStaticAssetPath(root string, relative string) (string, bool) {
	cleaned := filepath.Clean(relative)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", false
	}
	return filepath.Join(root, cleaned), true
}

func (s Server) handleHome(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.bridge.Home(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, snapshot)
}

func (s Server) handleProjectState(w http.ResponseWriter, r *http.Request) {
	state, err := s.bridge.ProjectState(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, state)
}

func (s Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	summary, err := s.bridge.Workspace(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, summary)
}

func (s Server) handleInitWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req WorkspaceInitRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	summary, err := s.bridge.InitWorkspace(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, summary)
}

func (s Server) handleWorkspaceMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req WorkspaceModeRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	summary, err := s.bridge.SetWorkspaceMode(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, summary)
}

func (s Server) handleWorkspaceFiles(w http.ResponseWriter, r *http.Request) {
	explorer, err := s.bridge.WorkspaceFiles(queryPath(r), 400)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, explorer)
}

func (s Server) handleWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("file")
	if relPath == "" {
		writeError(w, errors.New("file is required"), http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.WorkspaceFile(queryPath(r), relPath)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleWorkspaceFileSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req WorkspaceFileSaveRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.SaveWorkspaceFile(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleProjectMaterials(w http.ResponseWriter, r *http.Request) {
	items, err := s.bridge.ProjectMaterials(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, items)
}

func (s Server) handleProjectMaterialsUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req ProjectMaterialsUploadRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	items, err := s.bridge.UploadProjectMaterials(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, items)
}

func (s Server) handleProjectMaterialsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req ProjectMaterialDeleteRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	items, err := s.bridge.DeleteProjectMaterial(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, items)
}

func (s Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.bridge.Agents())
}

func (s Server) handleAllowedActions(w http.ResponseWriter, r *http.Request) {
	actions, err := s.bridge.AllowedActions(queryPath(r), r.URL.Query().Get("mode"), r.URL.Query().Get("session_id"))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, actions)
}

func (s Server) handleDeveloperAccess(w http.ResponseWriter, r *http.Request) {
	state, err := s.bridge.DeveloperAccess(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, state)
}

func (s Server) handleDeveloperRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req DeveloperRoleRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	state, err := s.bridge.SetDeveloperRole(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, state)
}

func (s Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.bridge.Providers())
}

func (s Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.bridge.Runs(queryPath(r), 20)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, runs)
}

func (s Server) handleRun(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		writeError(w, errors.New("run_id is required"), http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.Run(queryPath(r), runID)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleTaskPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req TaskPlanRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.bridge.TaskPlan(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, run)
}

func (s Server) handleTaskRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req TaskRunRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.bridge.TaskRun(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, run)
}

func (s Server) handleTaskReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req TaskReviewRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.TaskReview(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	mode := r.URL.Query().Get("mode")
	status := r.URL.Query().Get("status")
	sessions, err := s.bridge.Sessions(queryPath(r), 100, query, mode, status)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, sessions)
}

func (s Server) handleSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		writeError(w, errors.New("session_id is required"), http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.Session(queryPath(r), sessionID)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleLiveApps(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	apps, err := s.bridge.LiveApps(queryPath(r), sessionID)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, apps)
}

func (s Server) handleLiveAppStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req LiveAppStartRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	var (
		detail any
		err    error
	)
	if req.MaterialID != "" {
		detail, err = s.bridge.StartMaterialLiveApp(req)
	} else {
		detail, err = s.bridge.StartLessonDemo(req)
	}
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleLiveAppStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req LiveAppStopRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.StopLiveApp(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleLiveAppEnsure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req LiveAppEnsureRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.EnsureLiveApp(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleTestingScenarios(w http.ResponseWriter, r *http.Request) {
	scenarios, err := s.bridge.TestingScenarios(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, scenarios)
}

func (s Server) handleTestingStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req TestingStartRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.bridge.StartTesting(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, run)
}

func (s Server) handleTestingStatus(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		writeError(w, errors.New("run_id is required"), http.StatusBadRequest)
		return
	}
	run, err := s.bridge.TestingStatus(queryPath(r), runID)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, run)
}

func (s Server) handleTestingControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req TestingControlRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.bridge.TestingControl(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, run)
}

func (s Server) handleVerifiers(w http.ResponseWriter, r *http.Request) {
	items, err := s.bridge.Verifiers(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, items)
}

func (s Server) handleVerificationProfiles(w http.ResponseWriter, r *http.Request) {
	items, err := s.bridge.VerificationProfiles(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, items)
}

func (s Server) handleVerificationStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req VerificationStartRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	run, err := s.bridge.StartVerification(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, run)
}

func (s Server) handleVerificationStatus(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		writeError(w, errors.New("run_id is required"), http.StatusBadRequest)
		return
	}
	run, err := s.bridge.VerificationStatus(queryPath(r), runID)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, run)
}

func (s Server) handleChats(w http.ResponseWriter, r *http.Request) {
	chats, err := s.bridge.Chats(queryPath(r), 20)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, chats)
}

func (s Server) handleChat(w http.ResponseWriter, r *http.Request) {
	detail, err := s.bridge.Chat(queryPath(r), r.URL.Query().Get("session_id"))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleChatStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req ChatStartRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.ChatStart(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleChatSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req ChatSendRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	detail, err := s.bridge.ChatSend(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, detail)
}

func (s Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	status, err := s.bridge.Memory(queryPath(r))
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, status)
}

func (s Server) handlePresets(w http.ResponseWriter, r *http.Request) {
	cards, err := s.bridge.Presets()
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, cards)
}

func (s Server) handlePreset(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, errors.New("id is required"), http.StatusBadRequest)
		return
	}
	preview, err := s.bridge.Preset(id)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, preview)
}

func (s Server) handlePresetPreview(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, errors.New("id is required"), http.StatusBadRequest)
		return
	}
	preview, err := s.bridge.PresetInstallPreview(queryPath(r), id)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, preview)
}

func (s Server) handlePresetInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req PresetInstallRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		writeError(w, errors.New("id is required"), http.StatusBadRequest)
		return
	}
	result, err := s.bridge.PresetInstall(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}

func (s Server) handlePresetRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, errors.New("POST required"), http.StatusMethodNotAllowed)
		return
	}
	var req PresetRollbackRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	if req.InstallID == "" {
		writeError(w, errors.New("install_id is required"), http.StatusBadRequest)
		return
	}
	record, err := s.bridge.PresetRollback(req)
	if err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}
	writeJSON(w, record)
}

func queryPath(r *http.Request) string {
	path := r.URL.Query().Get("path")
	if path == "" {
		return "."
	}
	return path
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func writeError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func decodeJSON(body io.ReadCloser, value any) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(value)
}

func ValidatePaths(uiRoot string, presetsRoot string) error {
	for _, item := range []struct {
		name string
		path string
	}{
		{name: "ui root", path: uiRoot},
		{name: "presets root", path: presetsRoot},
	} {
		if _, err := os.Stat(item.path); err != nil {
			return fmt.Errorf("%s %s is unavailable: %w", item.name, item.path, err)
		}
	}
	return nil
}
