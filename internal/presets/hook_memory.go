package presets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/memory"
)

type HookMemoryEvent struct {
	Timestamp     string      `json:"timestamp"`
	RunID         string      `json:"run_id"`
	HookName      string      `json:"hook_name"`
	HookLifecycle string      `json:"hook_lifecycle"`
	OwnerPresetID string      `json:"owner_preset_id,omitempty"`
	AllowedScopes []string    `json:"allowed_scopes,omitempty"`
	Item          memory.Item `json:"item"`
}

func AddHookMemory(root string, item memory.Item, runID string, hookName string, hookLifecycle string, ownerPresetID string, allowedScopes []string) (HookMemoryEvent, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = fmt.Sprintf("hookmem-%d", time.Now().UTC().UnixNano())
	}
	if strings.TrimSpace(item.Source) == "" {
		item.Source = "hook"
	}
	if strings.TrimSpace(item.Confidence) == "" {
		item.Confidence = "medium"
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "active"
	}
	if strings.TrimSpace(item.CreatedAt) == "" {
		item.CreatedAt = now
	}
	if strings.TrimSpace(item.LastVerifiedAt) == "" {
		item.LastVerifiedAt = now
	}
	if err := memory.AddAllowed(root, item, allowedScopes); err != nil {
		return HookMemoryEvent{}, err
	}
	event := HookMemoryEvent{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		RunID:         runID,
		HookName:      hookName,
		HookLifecycle: hookLifecycle,
		OwnerPresetID: ownerPresetID,
		AllowedScopes: append([]string{}, allowedScopes...),
		Item:          item,
	}
	if err := appendHookMemoryEvent(root, runID, event); err != nil {
		return HookMemoryEvent{}, err
	}
	return event, nil
}

func appendHookMemoryEvent(root string, runID string, event HookMemoryEvent) error {
	path := filepath.Join(root, ".arc", "runs", runID, "hook_memory_events.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(payload, '\n')); err != nil {
		return err
	}
	return nil
}
