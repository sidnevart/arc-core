package presets

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

type PresetCatalogSyncResult struct {
	PresetID     string                    `json:"preset_id"`
	PublishedAt  string                    `json:"published_at"`
	CatalogRoot  string                    `json:"catalog_root"`
	SyncedAt     string                    `json:"synced_at"`
	Bundles      []PresetCatalogSyncBundle `json:"bundles"`
	Paths        PresetCatalogSyncPaths    `json:"paths"`
	ManifestIDs  []string                  `json:"manifest_ids,omitempty"`
	ManifestPath []string                  `json:"manifest_paths,omitempty"`
}

type PresetCatalogSyncBundle struct {
	Provider   string            `json:"provider"`
	BundleID   string            `json:"bundle_id"`
	SourceRoot string            `json:"source_root"`
	TargetRoot string            `json:"target_root"`
	Paths      PresetExportPaths `json:"paths"`
}

type PresetCatalogSyncPaths struct {
	Root         string `json:"root,omitempty"`
	JSONPath     string `json:"json_path,omitempty"`
	MarkdownPath string `json:"markdown_path,omitempty"`
}

func SyncDraftToCatalog(workspaceRoot string, presetID string, catalogRoot string) (PresetCatalogSyncResult, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetCatalogSyncResult{}, err
	}
	draft, err := LoadDraft(workspaceRoot, presetID)
	if err != nil {
		return PresetCatalogSyncResult{}, err
	}
	if strings.TrimSpace(draft.Profile.Status) != "published" {
		return PresetCatalogSyncResult{}, fmt.Errorf("preset draft must be published before catalog sync")
	}
	catalogRoot = strings.TrimSpace(catalogRoot)
	if catalogRoot == "" {
		return PresetCatalogSyncResult{}, fmt.Errorf("catalog root is required")
	}
	exported, err := ExportDraft(workspaceRoot, presetID)
	if err != nil {
		return PresetCatalogSyncResult{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	result := PresetCatalogSyncResult{
		PresetID:    draft.Profile.ID,
		PublishedAt: draft.Profile.UpdatedAt,
		CatalogRoot: catalogRoot,
		SyncedAt:    now,
		Bundles:     make([]PresetCatalogSyncBundle, 0, len(exported.Bundles)),
		Paths:       catalogSyncPaths(workspaceRoot, presetID),
		ManifestIDs: make([]string, 0, len(exported.Bundles)),
	}
	for _, bundle := range exported.Bundles {
		targetRoot := filepath.Join(catalogRoot, bundle.Manifest.ID)
		if err := copyDir(bundle.Paths.Root, targetRoot); err != nil {
			return PresetCatalogSyncResult{}, fmt.Errorf("copy bundle %s: %w", bundle.Provider, err)
		}
		if _, err := LoadManifest(filepath.Join(targetRoot, "manifest.yaml")); err != nil {
			return PresetCatalogSyncResult{}, fmt.Errorf("synced bundle validation failed for %s: %w", bundle.Provider, err)
		}
		result.Bundles = append(result.Bundles, PresetCatalogSyncBundle{
			Provider:   bundle.Provider,
			BundleID:   bundle.Manifest.ID,
			SourceRoot: bundle.Paths.Root,
			TargetRoot: targetRoot,
			Paths: PresetExportPaths{
				Root:          targetRoot,
				ManifestPath:  filepath.Join(targetRoot, "manifest.yaml"),
				ReadmePath:    filepath.Join(targetRoot, "README.md"),
				OverviewPath:  filepath.Join(targetRoot, "payload", "docs", "overview.md"),
				ProviderPath:  filepath.Join(targetRoot, "payload", providerFileName(bundle.Provider)),
				BriefJSONPath: filepath.Join(targetRoot, "brief.json"),
				EvalJSONPath:  filepath.Join(targetRoot, "evaluation.json"),
				SimJSONPath:   filepath.Join(targetRoot, "simulation.json"),
				ProfileJSON:   filepath.Join(targetRoot, "profile.json"),
			},
		})
		result.ManifestIDs = append(result.ManifestIDs, bundle.Manifest.ID)
		result.ManifestPath = append(result.ManifestPath, filepath.Join(targetRoot, "manifest.yaml"))
	}
	if err := project.WriteJSON(result.Paths.JSONPath, result); err != nil {
		return PresetCatalogSyncResult{}, err
	}
	if err := project.WriteString(result.Paths.MarkdownPath, renderCatalogSyncMarkdown(result)); err != nil {
		return PresetCatalogSyncResult{}, err
	}
	return result, nil
}

func catalogSyncPaths(workspaceRoot string, presetID string) PresetCatalogSyncPaths {
	root := filepath.Join(draftsRoot(workspaceRoot), presetID)
	return PresetCatalogSyncPaths{
		Root:         root,
		JSONPath:     filepath.Join(root, "catalog_sync.json"),
		MarkdownPath: filepath.Join(root, "catalog_sync.md"),
	}
}

func renderCatalogSyncMarkdown(result PresetCatalogSyncResult) string {
	lines := []string{
		"# Preset Catalog Sync Report",
		"",
		fmt.Sprintf("- preset: `%s`", result.PresetID),
		fmt.Sprintf("- catalog root: `%s`", result.CatalogRoot),
		fmt.Sprintf("- synced at: `%s`", result.SyncedAt),
		"",
		"## Bundles",
		"",
	}
	for _, bundle := range result.Bundles {
		lines = append(lines, fmt.Sprintf("- `%s` -> `%s`", bundle.BundleID, bundle.TargetRoot))
	}
	lines = append(lines, "", "## Manifests", "")
	for _, path := range result.ManifestPath {
		lines = append(lines, "- "+path)
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func copyDir(src string, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
