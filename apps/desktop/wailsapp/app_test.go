package wailsapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeDirectoryDialogStart(t *testing.T) {
	dir := t.TempDir()
	got := normalizeDirectoryDialogStart(dir)
	if got != dir {
		t.Fatalf("expected existing dir, got %q", got)
	}
}

func TestNormalizeDirectoryDialogStartFallsBackToUsableDirectory(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing", "child")
	got := normalizeDirectoryDialogStart(missing)
	if got == "" || got == missing {
		t.Fatalf("expected fallback directory, got %q", got)
	}
}

func TestNormalizeChatScalePercent(t *testing.T) {
	if got := normalizeChatScalePercent(55); got != chatScaleMin {
		t.Fatalf("expected min clamp %d, got %d", chatScaleMin, got)
	}
	if got := normalizeChatScalePercent(107); got != 110 {
		t.Fatalf("expected step-rounded 110, got %d", got)
	}
	if got := normalizeChatScalePercent(1000); got != chatScaleMax {
		t.Fatalf("expected max clamp %d, got %d", chatScaleMax, got)
	}
}

func TestDesktopUISettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ui-settings.json")
	input := desktopUISettings{ChatScalePercent: 115}
	if err := saveDesktopUISettingsFile(path, input); err != nil {
		t.Fatalf("saveDesktopUISettingsFile failed: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings file: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal settings file: %v", err)
	}
	if decoded["chat_density"] != nil {
		t.Fatalf("expected legacy chat_density to be omitted, got %#v", decoded["chat_density"])
	}
	if decoded["chat_scale_percent"] != float64(115) {
		t.Fatalf("expected saved chat scale 115, got %#v", decoded["chat_scale_percent"])
	}

	output := loadDesktopUISettingsFile(path)
	if output.ChatScalePercent != 115 {
		t.Fatalf("expected loaded scale 115, got %d", output.ChatScalePercent)
	}
}
