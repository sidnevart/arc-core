package chat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-os/internal/project"
	"agent-os/internal/provider"
)

type Session struct {
	ID                string            `json:"id"`
	Root              string            `json:"root"`
	Provider          string            `json:"provider"`
	Mode              string            `json:"mode"`
	Model             string            `json:"model,omitempty"`
	Status            string            `json:"status"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
	ProviderSessionID string            `json:"provider_session_id,omitempty"`
	Messages          []Message         `json:"messages"`
	Artifacts         map[string]any    `json:"artifacts,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

type Message struct {
	Turn              int               `json:"turn"`
	Role              string            `json:"role"`
	Content           string            `json:"content"`
	CreatedAt         string            `json:"created_at"`
	ProviderSessionID string            `json:"provider_session_id,omitempty"`
	Artifacts         map[string]string `json:"artifacts,omitempty"`
	Outputs           []Output          `json:"outputs,omitempty"`
	Failure           string            `json:"failure,omitempty"`
}

type Output struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Format     string `json:"format,omitempty"`
	Path       string `json:"path,omitempty"`
	Preview    string `json:"preview,omitempty"`
	PreviewURL string `json:"preview_url,omitempty"`
	Launchable bool   `json:"launchable,omitempty"`
	Inline     bool   `json:"inline,omitempty"`
	LiveAppID  string `json:"live_app_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
}

type StartOptions struct {
	Root       string
	Provider   string
	Mode       string
	Model      string
	Prompt     string
	UserPrompt string
	ReplyOnly  bool
	DryRun     bool
	Timeout    time.Duration
}

type SendOptions struct {
	Root       string
	SessionID  string
	Model      string
	Prompt     string
	UserPrompt string
	ReplyOnly  bool
	DryRun     bool
	Timeout    time.Duration
}

type CreateOptions struct {
	Root     string
	Provider string
	Mode     string
	Model    string
	Prompt   string
}

func Create(opts CreateOptions) (Session, error) {
	if err := project.RequireProject(opts.Root); err != nil {
		return Session{}, err
	}
	session := Session{
		ID:        newSessionID(),
		Root:      opts.Root,
		Provider:  opts.Provider,
		Mode:      opts.Mode,
		Model:     opts.Model,
		Status:    "ready",
		CreatedAt: nowUTC(),
		UpdatedAt: nowUTC(),
		Messages: []Message{
			{
				Turn:      1,
				Role:      "user",
				Content:   opts.Prompt,
				CreatedAt: nowUTC(),
			},
		},
		Artifacts: map[string]any{},
		Metadata:  map[string]string{},
	}
	if err := saveSession(opts.Root, session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func Start(opts StartOptions) (Session, error) {
	session, adapter, err := prepareStart(opts)
	if err != nil {
		return Session{}, err
	}
	return runTurn(opts.Root, session, adapter, opts.Prompt, opts.Model, opts.DryRun, opts.Timeout, true)
}

func Send(opts SendOptions) (Session, error) {
	session, adapter, err := prepareSend(opts)
	if err != nil {
		return Session{}, err
	}
	return runTurn(opts.Root, session, adapter, opts.Prompt, opts.Model, opts.DryRun, opts.Timeout, false)
}

func StartAsync(opts StartOptions) (Session, error) {
	session, adapter, err := prepareStart(opts)
	if err != nil {
		return Session{}, err
	}
	go func(session Session, adapter provider.Adapter, opts StartOptions) {
		_, _ = runTurn(opts.Root, session, adapter, opts.Prompt, opts.Model, opts.DryRun, opts.Timeout, true)
	}(session, adapter, opts)
	return session, nil
}

func SendAsync(opts SendOptions) (Session, error) {
	session, adapter, err := prepareSend(opts)
	if err != nil {
		return Session{}, err
	}
	go func(session Session, adapter provider.Adapter, opts SendOptions) {
		_, _ = runTurn(opts.Root, session, adapter, opts.Prompt, opts.Model, opts.DryRun, opts.Timeout, false)
	}(session, adapter, opts)
	return session, nil
}

func prepareStart(opts StartOptions) (Session, provider.Adapter, error) {
	if err := project.RequireProject(opts.Root); err != nil {
		return Session{}, nil, err
	}
	adapter, err := provider.Get(opts.Provider)
	if err != nil {
		return Session{}, nil, err
	}
	session := Session{
		ID:        newSessionID(),
		Root:      opts.Root,
		Provider:  opts.Provider,
		Mode:      opts.Mode,
		Model:     opts.Model,
		Status:    "running",
		CreatedAt: nowUTC(),
		UpdatedAt: nowUTC(),
		Messages: []Message{
			{
				Turn:      1,
				Role:      "user",
				Content:   visibleChatPrompt(opts.UserPrompt, opts.Prompt),
				CreatedAt: nowUTC(),
			},
		},
		Artifacts: map[string]any{},
		Metadata:  map[string]string{},
	}
	if err := saveSession(opts.Root, session); err != nil {
		return Session{}, nil, err
	}
	return session, adapter, nil
}

func prepareSend(opts SendOptions) (Session, provider.Adapter, error) {
	session, err := Load(opts.Root, opts.SessionID)
	if err != nil {
		return Session{}, nil, err
	}
	if session.Status == "running" {
		return Session{}, nil, fmt.Errorf("chat session %s is already running", session.ID)
	}
	adapter, err := provider.Get(session.Provider)
	if err != nil {
		return Session{}, nil, err
	}
	turn := assistantTurn(session)
	session.Messages = append(session.Messages, Message{
		Turn:      turn,
		Role:      "user",
		Content:   visibleChatPrompt(opts.UserPrompt, opts.Prompt),
		CreatedAt: nowUTC(),
	})
	session.UpdatedAt = nowUTC()
	session.Status = "running"
	if opts.Model != "" {
		session.Model = opts.Model
	}
	if err := saveSession(opts.Root, session); err != nil {
		return Session{}, nil, err
	}
	return session, adapter, nil
}

func Load(root string, sessionID string) (Session, error) {
	if sessionID == "" {
		latest, err := latestSessionDir(root)
		if err != nil {
			return Session{}, err
		}
		sessionID = filepath.Base(latest)
	}
	var session Session
	if err := project.ReadJSON(filepath.Join(chatDir(root, sessionID), "session.json"), &session); err != nil {
		return Session{}, err
	}
	if sanitizeSessionVisibleMessages(&session) {
		if err := saveSession(root, session); err != nil {
			return Session{}, err
		}
	}
	return session, nil
}

func List(root string) ([]Session, error) {
	dir := project.ProjectFile(root, "chats")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Session{}, nil
		}
		return nil, err
	}
	out := []Session{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		session, err := Load(root, entry.Name())
		if err != nil {
			continue
		}
		out = append(out, session)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}

func AttachRun(root string, sessionID string, runID string) error {
	if strings.TrimSpace(runID) == "" {
		return nil
	}
	session, err := Load(root, sessionID)
	if err != nil {
		return err
	}
	if session.Metadata == nil {
		session.Metadata = map[string]string{}
	}
	related := RelatedRunIDs(session)
	for _, existing := range related {
		if existing == runID {
			return nil
		}
	}
	related = append(related, runID)
	session.Metadata["related_runs"] = strings.Join(related, ",")
	session.UpdatedAt = nowUTC()
	return saveSession(root, session)
}

func RelatedRunIDs(session Session) []string {
	raw := strings.TrimSpace(session.Metadata["related_runs"])
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func AppendUserMessage(root string, sessionID string, prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return nil
	}
	session, err := Load(root, sessionID)
	if err != nil {
		return err
	}
	turn := assistantTurn(session)
	session.Messages = append(session.Messages, Message{
		Turn:      turn,
		Role:      "user",
		Content:   prompt,
		CreatedAt: nowUTC(),
	})
	session.UpdatedAt = nowUTC()
	return saveSession(root, session)
}

func AppendAssistantMessage(root string, sessionID string, content string, artifacts map[string]string) (Session, error) {
	session, err := Load(root, sessionID)
	if err != nil {
		return Session{}, err
	}
	turn := assistantTurn(session)
	session.Messages = append(session.Messages, Message{
		Turn:      turn,
		Role:      "assistant",
		Content:   content,
		CreatedAt: nowUTC(),
		Artifacts: copyArtifacts(artifacts),
	})
	session.UpdatedAt = nowUTC()
	session.Status = "ready"
	if err := saveSession(root, session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func MergeMetadata(root string, sessionID string, values map[string]string) (Session, error) {
	session, err := Load(root, sessionID)
	if err != nil {
		return Session{}, err
	}
	if session.Metadata == nil {
		session.Metadata = map[string]string{}
	}
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			delete(session.Metadata, key)
			continue
		}
		session.Metadata[key] = value
	}
	session.UpdatedAt = nowUTC()
	if err := saveSession(root, session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func runTurn(root string, session Session, adapter provider.Adapter, prompt string, model string, dryRun bool, timeout time.Duration, first bool) (Session, error) {
	runDir := chatDir(root, session.ID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return Session{}, err
	}
	turn := assistantTurn(session)
	turnPrefix := fmt.Sprintf("turn-%03d", turn)
	lastMessagePath := filepath.Join(runDir, turnPrefix+".assistant.md")
	transcriptPath := filepath.Join(runDir, turnPrefix+".transcript.jsonl")
	req := provider.TaskRequest{
		Root:           root,
		Prompt:         prompt,
		Mode:           session.Mode,
		Model:          chooseModel(model, session.Model),
		RunDir:         runDir,
		LastMessageOut: lastMessagePath,
		TranscriptOut:  transcriptPath,
		ReplyOnly:      replyOnlyForTurn(first, session, prompt),
		DryRun:         dryRun,
		Timeout:        timeout,
	}

	var result provider.TaskResult
	var err error
	if first {
		result, err = adapter.RunTask(context.Background(), req)
	} else {
		result, err = adapter.ResumeSession(context.Background(), session.ProviderSessionID, req)
	}

	session.UpdatedAt = nowUTC()
	if result.SessionID != "" {
		session.ProviderSessionID = result.SessionID
	}
	session.Status = "ready"
	if err != nil {
		session.Status = "failed"
		if session.Metadata == nil {
			session.Metadata = map[string]string{}
		}
		session.Metadata["last_error"] = enrichErrorWithStderr(err, result.StderrPath)
	}
	assistantContent, _ := project.ReadString(lastMessagePath)
	assistantContent = strings.TrimSpace(assistantContent)
	if assistantContent == "" && dryRun {
		assistantContent = "Dry-run enabled; provider execution skipped."
	}
	assistantContent, outputs, outputArtifacts := materializeAssistantOutputs(root, session, turn, prompt, assistantContent)
	artifacts := map[string]string{
		"last_message": lastMessagePath,
		"transcript":   transcriptPath,
		"stderr":       result.StderrPath,
	}
	for key, value := range outputArtifacts {
		artifacts[key] = value
	}
	failure := ""
	if err != nil {
		failure = enrichErrorWithStderr(err, result.StderrPath)
	}
	session.Messages = append(session.Messages, Message{
		Turn:              turn,
		Role:              "assistant",
		Content:           assistantContent,
		CreatedAt:         nowUTC(),
		ProviderSessionID: result.SessionID,
		Artifacts:         artifacts,
		Outputs:           copyOutputs(outputs),
		Failure:           failure,
	})
	if session.Artifacts == nil {
		session.Artifacts = map[string]any{}
	}
	session.Artifacts[turnPrefix] = artifacts
	if errSave := saveSession(root, session); errSave != nil {
		return Session{}, errSave
	}
	return session, err
}

func enrichErrorWithStderr(err error, stderrPath string) string {
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return ""
	}
	stderrPreview := firstNonEmptyLine(stderrPath)
	if stderrPreview == "" || strings.Contains(message, stderrPreview) {
		return message
	}
	return message + " - " + stderrPreview
}

func firstNonEmptyLine(path string) string {
	text, err := project.ReadString(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func assistantTurn(session Session) int {
	count := 0
	for _, message := range session.Messages {
		if message.Role == "assistant" {
			count++
		}
	}
	return count + 1
}

func saveSession(root string, session Session) error {
	return project.WriteJSON(filepath.Join(chatDir(root, session.ID), "session.json"), session)
}

func chatDir(root string, sessionID string) string {
	return project.ProjectFile(root, "chats", sessionID)
}

func latestSessionDir(root string) (string, error) {
	dir := project.ProjectFile(root, "chats")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	latest := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() > latest {
			latest = entry.Name()
		}
	}
	if latest == "" {
		return "", fmt.Errorf("no chats found")
	}
	return filepath.Join(dir, latest), nil
}

func chooseModel(override string, fallback string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}
	return fallback
}

func replyOnlyForTurn(first bool, session Session, prompt string) bool {
	if first {
		return true
	}
	_ = prompt
	return true
}

func newSessionID() string {
	return time.Now().UTC().Format("20060102T150405Z") + "-chat"
}

func visibleChatPrompt(userPrompt string, providerPrompt string) string {
	if cleaned := SanitizeVisibleChatText(userPrompt); strings.TrimSpace(cleaned) != "" {
		return cleaned
	}
	return SanitizeVisibleChatText(providerPrompt)
}

func SanitizeVisibleChatText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	looksAugmented := strings.HasPrefix(trimmed, "ARC response contract:") ||
		strings.HasPrefix(trimmed, "This is a reply-only ARC chat surface.") ||
		strings.HasPrefix(trimmed, "Attached session context:") ||
		strings.Contains(trimmed, "\nAttached session context:")
	if looksAugmented {
		for _, marker := range []string{"Current request:", "User request:"} {
			if idx := strings.LastIndex(trimmed, marker); idx >= 0 {
				candidate := strings.TrimSpace(trimmed[idx+len(marker):])
				if candidate != "" {
					return candidate
				}
			}
		}
	}
	return trimmed
}

func sanitizeSessionVisibleMessages(session *Session) bool {
	if session == nil || len(session.Messages) == 0 {
		return false
	}
	changed := false
	for i := range session.Messages {
		if session.Messages[i].Role != "user" {
			continue
		}
		cleaned := SanitizeVisibleChatText(session.Messages[i].Content)
		if cleaned == "" {
			cleaned = strings.TrimSpace(session.Messages[i].Content)
		}
		if cleaned != session.Messages[i].Content {
			session.Messages[i].Content = cleaned
			changed = true
		}
	}
	return changed
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func copyArtifacts(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyOutputs(in []Output) []Output {
	if len(in) == 0 {
		return nil
	}
	out := make([]Output, len(in))
	copy(out, in)
	return out
}
