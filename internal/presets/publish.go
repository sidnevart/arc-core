package presets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-os/internal/project"
)

type PresetPublishResult struct {
	PresetID       string                `json:"preset_id"`
	PreviousStatus string                `json:"previous_status"`
	NewStatus      string                `json:"new_status"`
	PublishedAt    string                `json:"published_at"`
	Export         PresetExportResult    `json:"export"`
	Envelope       PresetPublishEnvelope `json:"envelope"`
	Paths          PresetPublishPaths    `json:"paths"`
	ManifestPaths  []string              `json:"manifest_paths,omitempty"`
}

type PresetPublishPaths struct {
	Root             string `json:"root,omitempty"`
	JSONPath         string `json:"json_path,omitempty"`
	MarkdownPath     string `json:"markdown_path,omitempty"`
	EnvelopeJSONPath string `json:"envelope_json_path,omitempty"`
	EnvelopeMDPath   string `json:"envelope_markdown_path,omitempty"`
}

type PresetPublishEnvelope struct {
	PresetID               string                        `json:"preset_id"`
	SourceKind             string                        `json:"source_kind"`
	SourceStatus           string                        `json:"source_status"`
	PublishedAt            string                        `json:"published_at"`
	SimulationStatus       string                        `json:"simulation_status"`
	ContradictionFree      bool                          `json:"contradiction_free"`
	ManifestValidationPass bool                          `json:"manifest_validation_pass"`
	TrustNotes             []string                      `json:"trust_notes,omitempty"`
	Bundles                []PresetPublishEnvelopeBundle `json:"bundles,omitempty"`
}

type PresetPublishEnvelopeBundle struct {
	Provider          string `json:"provider"`
	BundleID          string `json:"bundle_id"`
	ManifestPath      string `json:"manifest_path"`
	ManifestSHA256    string `json:"manifest_sha256"`
	ReadmeSHA256      string `json:"readme_sha256"`
	OverviewSHA256    string `json:"overview_sha256"`
	ProviderDocSHA256 string `json:"provider_doc_sha256"`
}

func PublishDraft(workspaceRoot string, presetID string) (PresetPublishResult, error) {
	if err := project.RequireProject(workspaceRoot); err != nil {
		return PresetPublishResult{}, err
	}
	draft, err := LoadDraft(workspaceRoot, presetID)
	if err != nil {
		return PresetPublishResult{}, err
	}
	if strings.TrimSpace(draft.Profile.Status) != "tested" {
		return PresetPublishResult{}, fmt.Errorf("preset draft must be tested before publish")
	}
	report, err := LoadSimulationReport(workspaceRoot, presetID)
	if err != nil {
		return PresetPublishResult{}, fmt.Errorf("simulation report is required before publish: %w", err)
	}
	exported, err := ExportDraft(workspaceRoot, presetID)
	if err != nil {
		return PresetPublishResult{}, err
	}
	manifestPaths := make([]string, 0, len(exported.Bundles))
	for _, bundle := range exported.Bundles {
		if _, err := LoadManifest(bundle.Paths.ManifestPath); err != nil {
			return PresetPublishResult{}, fmt.Errorf("published bundle validation failed for %s: %w", bundle.Provider, err)
		}
		manifestPaths = append(manifestPaths, bundle.Paths.ManifestPath)
	}
	previous := draft.Profile.Status
	now := time.Now().UTC().Format(time.RFC3339)
	draft.Profile.Status = "published"
	draft.Profile.UpdatedAt = now
	draft.Brief = buildDraftBrief(draft.Profile, now)
	draft.Manifest = buildDraftManifest(draft.Profile)
	draft.EvaluationPack = buildDraftEvaluationPack(draft.Profile, now)
	draft.EvaluationPack.Status = "published"
	saved, err := SaveDraft(workspaceRoot, draft)
	if err != nil {
		return PresetPublishResult{}, err
	}
	result := PresetPublishResult{
		PresetID:       saved.Profile.ID,
		PreviousStatus: previous,
		NewStatus:      saved.Profile.Status,
		PublishedAt:    now,
		Export:         exported,
		Paths:          publishPaths(workspaceRoot, presetID),
		ManifestPaths:  manifestPaths,
	}
	envelope, err := buildPublishEnvelope(saved, report, exported, manifestPaths)
	if err != nil {
		return PresetPublishResult{}, err
	}
	result.Envelope = envelope
	if err := project.WriteJSON(result.Paths.JSONPath, result); err != nil {
		return PresetPublishResult{}, err
	}
	if err := project.WriteString(result.Paths.MarkdownPath, renderPublishMarkdown(result)); err != nil {
		return PresetPublishResult{}, err
	}
	if err := project.WriteJSON(result.Paths.EnvelopeJSONPath, result.Envelope); err != nil {
		return PresetPublishResult{}, err
	}
	if err := project.WriteString(result.Paths.EnvelopeMDPath, renderPublishEnvelopeMarkdown(result.Envelope)); err != nil {
		return PresetPublishResult{}, err
	}
	return result, nil
}

func publishPaths(workspaceRoot string, presetID string) PresetPublishPaths {
	root := filepath.Join(draftsRoot(workspaceRoot), presetID)
	return PresetPublishPaths{
		Root:             root,
		JSONPath:         filepath.Join(root, "publish.json"),
		MarkdownPath:     filepath.Join(root, "publish.md"),
		EnvelopeJSONPath: filepath.Join(root, "publish_envelope.json"),
		EnvelopeMDPath:   filepath.Join(root, "publish_envelope.md"),
	}
}

func renderPublishMarkdown(result PresetPublishResult) string {
	lines := []string{
		"# Preset Publish Report",
		"",
		fmt.Sprintf("- preset: `%s`", result.PresetID),
		fmt.Sprintf("- previous status: `%s`", result.PreviousStatus),
		fmt.Sprintf("- new status: `%s`", result.NewStatus),
		fmt.Sprintf("- published at: `%s`", result.PublishedAt),
		"",
		"## Exported Bundles",
		"",
	}
	for _, bundle := range result.Export.Bundles {
		lines = append(lines, fmt.Sprintf("- `%s`: %s", bundle.Provider, bundle.Paths.Root))
	}
	lines = append(lines, "", "## Validated Manifests", "")
	for _, path := range result.ManifestPaths {
		lines = append(lines, "- "+path)
	}
	lines = append(lines, "", "## Publish Envelope", "")
	lines = append(lines, "- "+result.Paths.EnvelopeJSONPath)
	lines = append(lines, "- "+result.Paths.EnvelopeMDPath)
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func buildPublishEnvelope(draft PresetDraft, report PresetSimulationReport, exported PresetExportResult, manifestPaths []string) (PresetPublishEnvelope, error) {
	envelope := PresetPublishEnvelope{
		PresetID:               draft.Profile.ID,
		SourceKind:             "local_tested_draft",
		SourceStatus:           "tested",
		PublishedAt:            draft.Profile.UpdatedAt,
		SimulationStatus:       report.Status,
		ContradictionFree:      len(report.Contradictions) == 0,
		ManifestValidationPass: len(manifestPaths) == len(exported.Bundles),
		TrustNotes: []string{
			"Published from a local ARC draft after passing simulation.",
			"Provider-specific bundles were re-exported and revalidated during publish.",
			"Bundle fingerprints are recorded so later catalog sync or review can compare exact payloads.",
		},
		Bundles: make([]PresetPublishEnvelopeBundle, 0, len(exported.Bundles)),
	}
	sort.Slice(exported.Bundles, func(i, j int) bool {
		return exported.Bundles[i].Provider < exported.Bundles[j].Provider
	})
	for _, bundle := range exported.Bundles {
		manifestSHA, err := fileSHA256(bundle.Paths.ManifestPath)
		if err != nil {
			return PresetPublishEnvelope{}, err
		}
		readmeSHA, err := fileSHA256(bundle.Paths.ReadmePath)
		if err != nil {
			return PresetPublishEnvelope{}, err
		}
		overviewSHA, err := fileSHA256(bundle.Paths.OverviewPath)
		if err != nil {
			return PresetPublishEnvelope{}, err
		}
		providerSHA, err := fileSHA256(bundle.Paths.ProviderPath)
		if err != nil {
			return PresetPublishEnvelope{}, err
		}
		envelope.Bundles = append(envelope.Bundles, PresetPublishEnvelopeBundle{
			Provider:          bundle.Provider,
			BundleID:          bundle.Manifest.ID,
			ManifestPath:      bundle.Paths.ManifestPath,
			ManifestSHA256:    manifestSHA,
			ReadmeSHA256:      readmeSHA,
			OverviewSHA256:    overviewSHA,
			ProviderDocSHA256: providerSHA,
		})
	}
	return envelope, nil
}

func renderPublishEnvelopeMarkdown(envelope PresetPublishEnvelope) string {
	lines := []string{
		"# Preset Publish Envelope",
		"",
		fmt.Sprintf("- preset: `%s`", envelope.PresetID),
		fmt.Sprintf("- source kind: `%s`", envelope.SourceKind),
		fmt.Sprintf("- source status: `%s`", envelope.SourceStatus),
		fmt.Sprintf("- published at: `%s`", envelope.PublishedAt),
		fmt.Sprintf("- simulation status: `%s`", envelope.SimulationStatus),
		fmt.Sprintf("- contradiction free: `%t`", envelope.ContradictionFree),
		fmt.Sprintf("- manifest validation pass: `%t`", envelope.ManifestValidationPass),
		"",
		"## Trust Notes",
		"",
	}
	for _, note := range envelope.TrustNotes {
		lines = append(lines, "- "+note)
	}
	lines = append(lines, "", "## Bundle Fingerprints", "")
	for _, bundle := range envelope.Bundles {
		lines = append(lines, "### "+bundle.BundleID, "")
		lines = append(lines, "- provider: `"+bundle.Provider+"`")
		lines = append(lines, "- manifest: `"+bundle.ManifestSHA256+"`")
		lines = append(lines, "- readme: `"+bundle.ReadmeSHA256+"`")
		lines = append(lines, "- overview: `"+bundle.OverviewSHA256+"`")
		lines = append(lines, "- provider doc: `"+bundle.ProviderDocSHA256+"`")
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
