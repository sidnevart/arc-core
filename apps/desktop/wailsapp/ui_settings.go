package wailsapp

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	chatScaleDefault     = 100
	chatScaleMin         = 60
	chatScaleMax         = 130
	chatScaleStep        = 5
	chatScaleEventName   = "arc:chat-scale"
	openDisplayEventName = "arc:open-display"
	openProjectsEventName = "arc:open-projects"
	openSettingsEventName = "arc:open-settings"
	setLocaleEventName    = "arc:set-locale"
)

type desktopUISettings struct {
	ChatDensity      string `json:"chat_density,omitempty"`
	ChatScalePercent int    `json:"chat_scale_percent,omitempty"`
}

func defaultDesktopUISettings() desktopUISettings {
	return desktopUISettings{
		ChatScalePercent: chatScaleDefault,
	}
}

func normalizeChatScalePercent(value int) int {
	if value <= 0 {
		return chatScaleDefault
	}
	if value < chatScaleMin {
		value = chatScaleMin
	}
	if value > chatScaleMax {
		value = chatScaleMax
	}
	remainder := value % chatScaleStep
	if remainder == 0 {
		return value
	}
	if remainder >= chatScaleStep/2 {
		value += chatScaleStep - remainder
	} else {
		value -= remainder
	}
	if value < chatScaleMin {
		return chatScaleMin
	}
	if value > chatScaleMax {
		return chatScaleMax
	}
	return value
}

func desktopUISettingsPath() string {
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "ARC Desktop", "ui-settings.json")
	}
	return filepath.Join(os.TempDir(), "arc-desktop-ui-settings.json")
}

func loadDesktopUISettings() desktopUISettings {
	return loadDesktopUISettingsFile(desktopUISettingsPath())
}

func loadDesktopUISettingsFile(path string) desktopUISettings {
	settings := defaultDesktopUISettings()
	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}
	var decoded desktopUISettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		return settings
	}
	if decoded.ChatScalePercent == 0 {
		switch decoded.ChatDensity {
		case "compact":
			decoded.ChatScalePercent = 90
		case "comfortable":
			decoded.ChatScalePercent = 110
		default:
			decoded.ChatScalePercent = chatScaleDefault
		}
	}
	decoded.ChatScalePercent = normalizeChatScalePercent(decoded.ChatScalePercent)
	return decoded
}

func saveDesktopUISettingsFile(path string, settings desktopUISettings) error {
	settings.ChatScalePercent = normalizeChatScalePercent(settings.ChatScalePercent)
	settings.ChatDensity = ""
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
