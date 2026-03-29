//go:build wails

package main

import (
	"log"

	"agent-os/apps/desktop/wailsapp"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

func main() {
	app := wailsapp.New()

	err := wails.Run(&options.App{
		Title:            "ARC Desktop",
		Width:            1520,
		Height:           980,
		MinWidth:         1200,
		MinHeight:        760,
		Frameless:        false,
		DisableResize:    false,
		AssetServer:      &assetserver.Options{Assets: wailsapp.Assets},
		Menu:             app.ApplicationMenu(),
		OnStartup:        app.Startup,
		OnBeforeClose:    app.BeforeClose,
		Bind:             []any{app},
		Mac:              &mac.Options{TitleBar: mac.TitleBarDefault()},
		EnableDefaultContextMenu: true,
	})
	if err != nil {
		log.Fatal(err)
	}
}
