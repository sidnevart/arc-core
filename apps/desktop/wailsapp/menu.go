package wailsapp

import (
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/menu"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ApplicationMenu() *menu.Menu {
	items := []*menu.MenuItem{
		menu.AppMenu(),
		menu.SubMenu("ARC", a.workspaceMenu()),
		menu.EditMenu(),
		menu.SubMenu("View", a.viewMenu()),
		menu.WindowMenu(),
	}
	return menu.NewMenuFromItems(items[0], items[1:]...)
}

func (a *App) workspaceMenu() *menu.Menu {
	workspaceMenu := menu.NewMenu()
	workspaceMenu.AddText("Projects…", nil, func(*menu.CallbackData) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, openProjectsEventName)
		}
	})
	workspaceMenu.AddText("Settings", nil, func(*menu.CallbackData) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, openSettingsEventName)
		}
	})
	languageMenu := workspaceMenu.AddSubmenu("Language")
	languageMenu.AddText("Русский", nil, func(*menu.CallbackData) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, setLocaleEventName, "ru")
		}
	})
	languageMenu.AddText("English", nil, func(*menu.CallbackData) {
		if a.ctx != nil {
			wailsruntime.EventsEmit(a.ctx, setLocaleEventName, "en")
		}
	})
	return workspaceMenu
}

func (a *App) viewMenu() *menu.Menu {
	viewMenu := menu.NewMenu()
	viewMenu.AddText("Display...", nil, func(*menu.CallbackData) {
		a.openDisplayPanel()
	})
	scaleMenu := viewMenu.AddSubmenu("Chat Scale")

	a.settingsMu.Lock()
	currentScale := normalizeChatScalePercent(a.uiSettings.ChatScalePercent)
	a.chatScaleMenuItems = map[int]*menu.MenuItem{}
	a.settingsMu.Unlock()

	for _, option := range []int{90, 100, 110} {
		value := option
		item := scaleMenu.AddRadio(fmt.Sprintf("%d%%", value), currentScale == value, nil, func(*menu.CallbackData) {
			a.applyChatScalePercent(value)
		})
		a.settingsMu.Lock()
		a.chatScaleMenuItems[value] = item
		a.settingsMu.Unlock()
	}
	return viewMenu
}

func (a *App) ChatScalePercent() int {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	return normalizeChatScalePercent(a.uiSettings.ChatScalePercent)
}

func (a *App) SetChatScalePercent(value int) int {
	a.applyChatScalePercent(value)
	return a.ChatScalePercent()
}

func (a *App) openDisplayPanel() {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, openDisplayEventName)
	}
}

func (a *App) applyChatScalePercent(value int) {
	next := normalizeChatScalePercent(value)

	a.settingsMu.Lock()
	a.uiSettings.ChatScalePercent = next
	for key, item := range a.chatScaleMenuItems {
		if item != nil {
			item.SetChecked(key == next)
		}
	}
	savePath := desktopUISettingsPath()
	err := saveDesktopUISettingsFile(savePath, a.uiSettings)
	a.settingsMu.Unlock()

	if err != nil {
		a.logf("error", "failed to save desktop UI settings: "+err.Error())
	} else {
		a.logf("info", fmt.Sprintf("desktop chat scale set to %d%%", next))
	}

	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, chatScaleEventName, next)
		wailsruntime.MenuUpdateApplicationMenu(a.ctx)
	}
}
