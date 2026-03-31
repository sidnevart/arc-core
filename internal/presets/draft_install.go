package presets

import (
	"fmt"
	"strings"

	"agent-os/internal/project"
)

type DraftInstallResult struct {
	PresetID         string             `json:"preset_id"`
	SelectedProvider string             `json:"selected_provider"`
	BundleID         string             `json:"bundle_id"`
	Export           PresetExportResult `json:"export"`
	Install          InstallResult      `json:"install"`
}

func InstallPublishedDraft(workspaceRoot string, presetID string, allowOverwrite bool) (DraftInstallResult, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return DraftInstallResult{}, err
	}
	draft, err := LoadDraft(workspaceRoot, presetID)
	if err != nil {
		return DraftInstallResult{}, err
	}
	if strings.TrimSpace(draft.Profile.Status) != "published" {
		return DraftInstallResult{}, fmt.Errorf("preset draft must be published before install")
	}
	proj, err := project.Load(workspaceRoot)
	if err != nil {
		return DraftInstallResult{}, err
	}
	exported, err := ExportDraft(workspaceRoot, presetID)
	if err != nil {
		return DraftInstallResult{}, err
	}
	selectedProvider, bundleID, err := selectDraftInstallBundle(proj.Config.DefaultProvider, draft.Profile.TargetAgent, exported)
	if err != nil {
		return DraftInstallResult{}, err
	}
	installResult, err := Install(InstallOptions{
		WorkspaceRoot:  workspaceRoot,
		CatalogRoot:    exportsRoot(workspaceRoot),
		PresetID:       bundleID,
		AllowOverwrite: allowOverwrite,
	})
	if err != nil {
		return DraftInstallResult{}, err
	}
	return DraftInstallResult{
		PresetID:         draft.Profile.ID,
		SelectedProvider: selectedProvider,
		BundleID:         bundleID,
		Export:           exported,
		Install:          installResult,
	}, nil
}

func selectDraftInstallBundle(defaultProvider string, targetAgent string, exported PresetExportResult) (string, string, error) {
	if len(exported.Bundles) == 0 {
		return "", "", fmt.Errorf("no exported bundles available")
	}
	preferred := []string{}
	if value := strings.TrimSpace(defaultProvider); value != "" {
		preferred = append(preferred, value)
	}
	if value := strings.TrimSpace(targetAgent); value != "" && value != "arc" {
		preferred = append(preferred, value)
	}
	for _, candidate := range preferred {
		for _, bundle := range exported.Bundles {
			if bundle.Provider == candidate {
				return bundle.Provider, bundle.Manifest.ID, nil
			}
		}
	}
	first := exported.Bundles[0]
	return first.Provider, first.Manifest.ID, nil
}
