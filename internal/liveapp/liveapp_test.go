package liveapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-os/internal/project"
)

func TestStartAndStopStaticPreview(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatalf("init project: %v", err)
	}

	sourceDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	app, err := StartStaticPreview(StartOptions{
		Root:           root,
		SessionID:      "session-1",
		Title:          "Demo",
		Origin:         "lesson",
		Type:           "demo",
		SourcePath:     sourceDir,
		AutoStopAfter:  2 * time.Minute,
		AutoStopPolicy: "idle_2m",
	})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("sandbox does not allow opening localhost listeners")
		}
		t.Fatalf("start preview: %v", err)
	}
	if app.Status != "ready" {
		t.Fatalf("expected ready app, got %q", app.Status)
	}
	if !strings.Contains(app.PreviewURL, "127.0.0.1") {
		t.Fatalf("expected preview url, got %q", app.PreviewURL)
	}

	list, err := List(root)
	if err != nil {
		t.Fatalf("list apps: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one app, got %d", len(list))
	}

	stopped, err := Stop(root, app.ID, "test_stop")
	if err != nil {
		t.Fatalf("stop app: %v", err)
	}
	if stopped.Status != "stopped" {
		t.Fatalf("expected stopped app, got %q", stopped.Status)
	}
	if stopped.PID != 0 {
		t.Fatalf("expected stopped app to clear pid, got %d", stopped.PID)
	}

	loaded, err := Load(root, app.ID)
	if err != nil {
		t.Fatalf("load stopped app: %v", err)
	}
	if loaded.Status != "stopped" {
		t.Fatalf("expected persisted stopped app, got %q", loaded.Status)
	}
	if loaded.PID != 0 {
		t.Fatalf("expected persisted stopped app to clear pid, got %d", loaded.PID)
	}
}

func TestListPrunesOldStoppedRuntimeDirs(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatalf("init project: %v", err)
	}

	old := App{
		ID:        "old-live",
		Root:      root,
		Status:    "stopped",
		Title:     "Old",
		StoppedAt: time.Now().UTC().Add(-stoppedRuntimeRetention - time.Hour).Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Add(-stoppedRuntimeRetention - time.Hour).Format(time.RFC3339),
	}
	if err := os.MkdirAll(appDir(root, old.ID), 0o755); err != nil {
		t.Fatalf("mkdir old app dir: %v", err)
	}
	if err := save(root, old); err != nil {
		t.Fatalf("save old app: %v", err)
	}
	recent := App{
		ID:        "recent-live",
		Root:      root,
		Status:    "stopped",
		Title:     "Recent",
		StoppedAt: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
	}
	if err := os.MkdirAll(appDir(root, recent.ID), 0o755); err != nil {
		t.Fatalf("mkdir recent app dir: %v", err)
	}
	if err := save(root, recent); err != nil {
		t.Fatalf("save recent app: %v", err)
	}

	list, err := List(root)
	if err != nil {
		t.Fatalf("list apps: %v", err)
	}
	if len(list) != 1 || list[0].ID != recent.ID {
		t.Fatalf("expected only recent app to remain, got %#v", list)
	}
	if _, err := os.Stat(appDir(root, old.ID)); !os.IsNotExist(err) {
		t.Fatalf("expected old app dir to be pruned, got err=%v", err)
	}
}
