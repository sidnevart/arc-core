package wailsapp

import "embed"

// Assets contains the native Wails frontend bundle for ARC Desktop.
//
//go:embed frontend/*
var Assets embed.FS
