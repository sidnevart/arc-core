package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-os/internal/memory"
	"agent-os/internal/project"
)

func TestHookMemoryAddWritesAllowedEntryAndAuditEvent(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARC_RUN_ID", "run-1")
	t.Setenv("ARC_HOOK_NAME", "remember")
	t.Setenv("ARC_HOOK_LIFECYCLE", "before_run")
	t.Setenv("ARC_HOOK_OWNER_PRESET", "infra-runtime")
	t.Setenv("ARC_ALLOWED_MEMORY_SCOPES", "project,runs/run-1")

	if err := Run([]string{
		"hook", "memory", "add",
		"--path", root,
		"--scope", "runs/run-1",
		"--kind", "fact",
		"--tags", "hook,policy",
		"remember", "this", "fact",
	}); err != nil {
		t.Fatal(err)
	}

	items, err := memory.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one memory item, got %d", len(items))
	}
	if items[0].Scope != "runs/run-1" {
		t.Fatalf("unexpected scope: %#v", items[0])
	}
	eventPath := filepath.Join(root, ".arc", "runs", "run-1", "hook_memory_events.jsonl")
	data, err := os.ReadFile(eventPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\"hook_name\":\"remember\"") {
		t.Fatalf("expected hook audit event, got %s", string(data))
	}
}

func TestHookMemoryAddRejectsDisallowedScope(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARC_RUN_ID", "run-2")
	t.Setenv("ARC_ALLOWED_MEMORY_SCOPES", "project")

	err := Run([]string{
		"hook", "memory", "add",
		"--path", root,
		"--scope", "runs/run-2",
		"should", "fail",
	})
	if err == nil {
		t.Fatal("expected disallowed hook memory scope to fail")
	}
}
