package liveapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"agent-os/internal/project"
)

type App struct {
	ID             string   `json:"id"`
	Root           string   `json:"root"`
	SessionID      string   `json:"session_id,omitempty"`
	Title          string   `json:"title"`
	Origin         string   `json:"origin,omitempty"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	Port           int      `json:"port,omitempty"`
	PreviewURL     string   `json:"preview_url,omitempty"`
	SourcePath     string   `json:"source_path,omitempty"`
	StartedAt      string   `json:"started_at,omitempty"`
	UpdatedAt      string   `json:"updated_at,omitempty"`
	StoppedAt      string   `json:"stopped_at,omitempty"`
	ExpiresAt      string   `json:"expires_at,omitempty"`
	AutoStopPolicy string   `json:"auto_stop_policy,omitempty"`
	StopReason     string   `json:"stop_reason,omitempty"`
	StdoutPath     string   `json:"stdout_path,omitempty"`
	StderrPath     string   `json:"stderr_path,omitempty"`
	Command        []string `json:"command,omitempty"`
	PID            int      `json:"pid,omitempty"`
}

type StartOptions struct {
	Root           string
	SessionID      string
	Title          string
	Origin         string
	Type           string
	SourcePath     string
	AutoStopAfter  time.Duration
	AutoStopPolicy string
}

var processRegistry = struct {
	mu   sync.Mutex
	proc map[string]*exec.Cmd
}{
	proc: map[string]*exec.Cmd{},
}

const stoppedRuntimeRetention = 72 * time.Hour

func StartStaticPreview(opts StartOptions) (App, error) {
	if err := project.RequireProject(opts.Root); err != nil {
		return App{}, err
	}
	_ = pruneStoppedRuntimeDirs(opts.Root, stoppedRuntimeRetention)
	sourcePath, err := filepath.Abs(strings.TrimSpace(opts.SourcePath))
	if err != nil {
		return App{}, err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return App{}, err
	}

	serveDir := sourcePath
	route := "/"
	if !info.IsDir() {
		serveDir = filepath.Dir(sourcePath)
		route = "/" + filepath.Base(sourcePath)
	}

	port, err := freePort()
	if err != nil {
		return App{}, err
	}
	id := newID()
	runDir := appDir(opts.Root, id)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return App{}, err
	}
	stdoutPath := filepath.Join(runDir, "stdout.log")
	stderrPath := filepath.Join(runDir, "stderr.log")
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		return App{}, err
	}
	defer stdoutFile.Close()
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		return App{}, err
	}
	defer stderrFile.Close()

	command := []string{"python3", "-m", "http.server", strconv.Itoa(port), "--bind", "127.0.0.1", "--directory", serveDir}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = io.MultiWriter(stdoutFile)
	cmd.Stderr = io.MultiWriter(stderrFile)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return App{}, err
	}

	expiresAt := ""
	if opts.AutoStopAfter <= 0 {
		opts.AutoStopAfter = 20 * time.Minute
	}
	expiresAt = time.Now().UTC().Add(opts.AutoStopAfter).Format(time.RFC3339)
	if opts.AutoStopPolicy == "" {
		opts.AutoStopPolicy = "idle_20m"
	}

	app := App{
		ID:             id,
		Root:           opts.Root,
		SessionID:      strings.TrimSpace(opts.SessionID),
		Title:          defaultIfEmpty(opts.Title, "Локальное демо"),
		Origin:         strings.TrimSpace(opts.Origin),
		Type:           defaultIfEmpty(opts.Type, "demo"),
		Status:         "starting",
		Port:           port,
		PreviewURL:     fmt.Sprintf("http://127.0.0.1:%d%s", port, route),
		SourcePath:     sourcePath,
		StartedAt:      nowUTC(),
		UpdatedAt:      nowUTC(),
		ExpiresAt:      expiresAt,
		AutoStopPolicy: opts.AutoStopPolicy,
		StdoutPath:     stdoutPath,
		StderrPath:     stderrPath,
		Command:        append([]string{}, command...),
		PID:            cmd.Process.Pid,
	}
	if err := save(opts.Root, app); err != nil {
		_ = killProcess(cmd)
		return App{}, err
	}
	registerProcess(app.ID, cmd)

	go func(root string, appID string, cmd *exec.Cmd) {
		waitErr := cmd.Wait()
		current, err := Load(root, appID)
		if err != nil {
			unregisterProcess(appID)
			return
		}
		current.UpdatedAt = nowUTC()
		if current.Status != "stopped" {
			if waitErr == nil {
				current.Status = "stopped"
				current.StopReason = defaultIfEmpty(current.StopReason, "process_exited")
			} else {
				current.Status = "failed"
				current.StopReason = waitErr.Error()
			}
			current.StoppedAt = nowUTC()
			_ = save(root, current)
		}
		unregisterProcess(appID)
	}(opts.Root, app.ID, cmd)

	if err := waitReady(app.PreviewURL, 8*time.Second); err != nil {
		_, _ = Stop(opts.Root, app.ID, "startup_failed")
		return App{}, err
	}
	app.Status = "ready"
	app.UpdatedAt = nowUTC()
	if err := save(opts.Root, app); err != nil {
		_, _ = Stop(opts.Root, app.ID, "save_failed")
		return App{}, err
	}
	return app, nil
}

func List(root string) ([]App, error) {
	_ = pruneStoppedRuntimeDirs(root, stoppedRuntimeRetention)
	dir := appsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []App{}, nil
		}
		return nil, err
	}
	out := make([]App, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		app, err := Load(root, entry.Name())
		if err != nil {
			continue
		}
		app = refresh(root, app)
		out = append(out, app)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out, nil
}

func Load(root string, id string) (App, error) {
	var app App
	if err := project.ReadJSON(filepath.Join(appDir(root, id), "app.json"), &app); err != nil {
		return App{}, err
	}
	return app, nil
}

func Stop(root string, id string, reason string) (App, error) {
	app, err := Load(root, id)
	if err != nil {
		return App{}, err
	}
	if app.Status == "stopped" || app.Status == "failed" {
		return app, nil
	}
	pid := app.PID
	if cmd := getRegisteredProcess(id); cmd != nil {
		_ = killProcess(cmd)
	} else if pid > 0 {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
	waitStopped(pid, 2*time.Second)
	app.Status = "stopped"
	app.StopReason = defaultIfEmpty(reason, "stopped_by_user")
	app.StoppedAt = nowUTC()
	app.UpdatedAt = nowUTC()
	app.PID = 0
	if err := save(root, app); err != nil {
		return App{}, err
	}
	unregisterProcess(id)
	return app, nil
}

func refresh(root string, app App) App {
	if app.Status != "ready" && app.Status != "starting" {
		return app
	}
	if app.ExpiresAt != "" {
		if expiry, err := time.Parse(time.RFC3339, app.ExpiresAt); err == nil && time.Now().UTC().After(expiry) {
			stopped, stopErr := Stop(root, app.ID, "auto_stopped")
			if stopErr == nil {
				return stopped
			}
		}
	}
	if !processAlive(app.PID) {
		app.Status = "stopped"
		app.StopReason = defaultIfEmpty(app.StopReason, "process_exited")
		app.StoppedAt = defaultIfEmpty(app.StoppedAt, nowUTC())
		app.UpdatedAt = nowUTC()
		_ = save(root, app)
	}
	return app
}

func pruneStoppedRuntimeDirs(root string, olderThan time.Duration) error {
	if olderThan <= 0 {
		return nil
	}
	dir := appsDir(root)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		app, err := Load(root, entry.Name())
		if err != nil {
			continue
		}
		if app.Status != "stopped" && app.Status != "failed" {
			continue
		}
		if processAlive(app.PID) {
			continue
		}
		reference := pruneReferenceTime(app)
		if reference.IsZero() || reference.After(cutoff) {
			continue
		}
		_ = os.RemoveAll(appDir(root, app.ID))
	}
	return nil
}

func pruneReferenceTime(app App) time.Time {
	for _, raw := range []string{app.StoppedAt, app.UpdatedAt, app.StartedAt} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if ts, err := time.Parse(time.RFC3339, raw); err == nil {
			return ts.UTC()
		}
	}
	return time.Time{}
}

func save(root string, app App) error {
	return project.WriteJSON(filepath.Join(appDir(root, app.ID), "app.json"), app)
}

func appsDir(root string) string {
	return project.ProjectFile(root, "live_apps")
}

func appDir(root string, id string) string {
	return project.ProjectFile(root, "live_apps", id)
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("unexpected listener address")
	}
	return addr.Port, nil
}

func waitReady(url string, timeout time.Duration) error {
	client := http.Client{Timeout: 1500 * time.Millisecond}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("preview did not become ready: %s", url)
}

func registerProcess(id string, cmd *exec.Cmd) {
	processRegistry.mu.Lock()
	defer processRegistry.mu.Unlock()
	processRegistry.proc[id] = cmd
}

func unregisterProcess(id string) {
	processRegistry.mu.Lock()
	defer processRegistry.mu.Unlock()
	delete(processRegistry.proc, id)
}

func getRegisteredProcess(id string) *exec.Cmd {
	processRegistry.mu.Lock()
	defer processRegistry.mu.Unlock()
	return processRegistry.proc[id]
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func waitStopped(pid int, timeout time.Duration) {
	if pid <= 0 || timeout <= 0 {
		return
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func killProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		_ = cmd.Process.Kill()
		return err
	}
	return nil
}

func newID() string {
	return time.Now().UTC().Format("20060102T150405Z") + "-live"
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func defaultIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func Marshal(app App) string {
	data, _ := json.Marshal(app)
	return string(data)
}
