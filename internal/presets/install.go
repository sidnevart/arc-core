package presets

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-os/internal/project"
)

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
)

type PreviewOptions struct {
	WorkspaceRoot string
	CatalogRoot   string
	PresetID      string
}

type InstallOptions struct {
	WorkspaceRoot  string
	CatalogRoot    string
	PresetID       string
	AllowOverwrite bool
}

type RollbackOptions struct {
	WorkspaceRoot string
	InstallID     string
}

type FileOperation struct {
	TargetPath string `json:"target_path"`
	SourcePath string `json:"source_path"`
	Action     Action `json:"action"`
	Collision  bool   `json:"collision"`
}

type InstallPreview struct {
	Manifest                Manifest              `json:"manifest"`
	Target                  string                `json:"target"`
	Operations              []FileOperation       `json:"operations"`
	Resolution              EnvironmentResolution `json:"resolution"`
	HasConflicts            bool                  `json:"has_conflicts"`
	Conflicts               []string              `json:"conflicts,omitempty"`
	HasFileConflicts        bool                  `json:"has_file_conflicts,omitempty"`
	FileConflicts           []string              `json:"file_conflicts,omitempty"`
	HasEnvironmentConflicts bool                  `json:"has_environment_conflicts,omitempty"`
	EnvironmentConflicts    []string              `json:"environment_conflicts,omitempty"`
}

type InstalledRecord struct {
	InstallID             string          `json:"install_id"`
	PresetID              string          `json:"preset_id"`
	Version               string          `json:"version"`
	ManifestPath          string          `json:"manifest_path,omitempty"`
	Manifest              Manifest        `json:"manifest,omitempty"`
	Target                string          `json:"target"`
	InstalledAt           string          `json:"installed_at"`
	Status                string          `json:"status"`
	AllowOverwrite        bool            `json:"allow_overwrite"`
	Operations            []FileOperation `json:"operations"`
	ReportPath            string          `json:"report_path"`
	EnvironmentReportPath string          `json:"environment_report_path,omitempty"`
	EnvironmentJSONPath   string          `json:"environment_json_path,omitempty"`
	BackupDir             string          `json:"backup_dir,omitempty"`
	RolledBackAt          string          `json:"rolled_back_at,omitempty"`
	RollbackSummary       string          `json:"rollback_summary,omitempty"`
}

type InstallResult struct {
	Preview  InstallPreview  `json:"preview"`
	Record   InstalledRecord `json:"record"`
	Report   string          `json:"report"`
	Warnings []string        `json:"warnings,omitempty"`
}

func PreviewInstall(opts PreviewOptions) (InstallPreview, error) {
	if err := project.RequireProject(opts.WorkspaceRoot); err != nil {
		return InstallPreview{}, err
	}
	manifest, err := LoadByID(opts.CatalogRoot, opts.PresetID)
	if err != nil {
		return InstallPreview{}, err
	}
	operations, conflicts, err := plannedOperations(opts.WorkspaceRoot, manifest)
	if err != nil {
		return InstallPreview{}, err
	}
	resolution, err := ResolveEnvironment(opts.WorkspaceRoot, opts.CatalogRoot, &manifest)
	if err != nil {
		return InstallPreview{}, err
	}
	environmentConflicts := append([]string{}, resolution.EnvironmentConflicts...)
	allConflicts := append([]string{}, conflicts...)
	allConflicts = append(allConflicts, environmentConflicts...)
	return InstallPreview{
		Manifest:                manifest,
		Target:                  opts.WorkspaceRoot,
		Operations:              operations,
		Resolution:              resolution,
		HasConflicts:            len(allConflicts) > 0,
		Conflicts:               allConflicts,
		HasFileConflicts:        len(conflicts) > 0,
		FileConflicts:           conflicts,
		HasEnvironmentConflicts: len(environmentConflicts) > 0,
		EnvironmentConflicts:    environmentConflicts,
	}, nil
}

func Install(opts InstallOptions) (InstallResult, error) {
	preview, err := PreviewInstall(PreviewOptions{
		WorkspaceRoot: opts.WorkspaceRoot,
		CatalogRoot:   opts.CatalogRoot,
		PresetID:      opts.PresetID,
	})
	if err != nil {
		return InstallResult{}, err
	}
	if preview.HasEnvironmentConflicts {
		return InstallResult{}, fmt.Errorf("preset has environment conflicts; review preview before installing")
	}
	if preview.HasFileConflicts && !opts.AllowOverwrite {
		return InstallResult{}, fmt.Errorf("preset has file collisions; rerun with overwrite approval")
	}

	installID := time.Now().UTC().Format("20060102T150405Z") + "-" + preview.Manifest.ID
	backupDir := project.ProjectFile(opts.WorkspaceRoot, "presets", "backups", installID)
	reportPath := project.ProjectFile(opts.WorkspaceRoot, "presets", "reports", installID+".md")
	environmentReportPath := project.ProjectFile(opts.WorkspaceRoot, "presets", "reports", installID+".environment.md")
	environmentJSONPath := project.ProjectFile(opts.WorkspaceRoot, "presets", "reports", installID+".environment.json")

	applied := []FileOperation{}
	for _, operation := range preview.Operations {
		if operation.Collision {
			if err := backupFile(opts.WorkspaceRoot, operation.TargetPath, backupDir); err != nil {
				_ = rollbackApplied(opts.WorkspaceRoot, applied, backupDir)
				return InstallResult{}, err
			}
		}
		if err := writeTargetFile(opts.WorkspaceRoot, operation); err != nil {
			_ = rollbackApplied(opts.WorkspaceRoot, applied, backupDir)
			return InstallResult{}, err
		}
		applied = append(applied, operation)
	}

	record := InstalledRecord{
		InstallID:             installID,
		PresetID:              preview.Manifest.ID,
		Version:               preview.Manifest.Version,
		ManifestPath:          preview.Manifest.Path,
		Manifest:              preview.Manifest,
		Target:                opts.WorkspaceRoot,
		InstalledAt:           time.Now().UTC().Format(time.RFC3339),
		Status:                "installed",
		AllowOverwrite:        opts.AllowOverwrite,
		Operations:            applied,
		ReportPath:            reportPath,
		EnvironmentReportPath: environmentReportPath,
		EnvironmentJSONPath:   environmentJSONPath,
	}
	if hasAnyCollision(applied) {
		record.BackupDir = backupDir
	}

	if err := project.WriteJSON(environmentJSONPath, preview.Resolution); err != nil {
		_ = rollbackApplied(opts.WorkspaceRoot, applied, backupDir)
		return InstallResult{}, err
	}
	if err := project.WriteString(environmentReportPath, RenderEnvironmentResolutionMarkdown(preview.Resolution)); err != nil {
		_ = rollbackApplied(opts.WorkspaceRoot, applied, backupDir)
		_ = os.Remove(environmentJSONPath)
		return InstallResult{}, err
	}
	report := renderInstallReport(preview, record)
	if err := project.WriteString(reportPath, report); err != nil {
		_ = rollbackApplied(opts.WorkspaceRoot, applied, backupDir)
		_ = os.Remove(environmentJSONPath)
		_ = os.Remove(environmentReportPath)
		return InstallResult{}, err
	}
	if err := appendInstalledRecord(opts.WorkspaceRoot, record); err != nil {
		_ = rollbackApplied(opts.WorkspaceRoot, applied, backupDir)
		_ = os.Remove(reportPath)
		_ = os.Remove(environmentJSONPath)
		_ = os.Remove(environmentReportPath)
		return InstallResult{}, err
	}
	return InstallResult{
		Preview: preview,
		Record:  record,
		Report:  reportPath,
	}, nil
}

func Rollback(opts RollbackOptions) (InstalledRecord, error) {
	records, err := loadInstalledRecords(opts.WorkspaceRoot)
	if err != nil {
		return InstalledRecord{}, err
	}
	index := -1
	for i := range records {
		if records[i].InstallID == opts.InstallID {
			index = i
			break
		}
	}
	if index == -1 {
		return InstalledRecord{}, fmt.Errorf("install %q not found", opts.InstallID)
	}
	record := records[index]
	if record.Status == "rolled_back" {
		return record, nil
	}
	if err := rollbackApplied(opts.WorkspaceRoot, record.Operations, record.BackupDir); err != nil {
		return InstalledRecord{}, err
	}
	record.Status = "rolled_back"
	record.RolledBackAt = time.Now().UTC().Format(time.RFC3339)
	record.RollbackSummary = "Preset files restored from backup or removed if newly created."
	records[index] = record
	if err := saveInstalledRecords(opts.WorkspaceRoot, records); err != nil {
		return InstalledRecord{}, err
	}
	return record, nil
}

func ListInstalled(workspaceRoot string) ([]InstalledRecord, error) {
	return loadInstalledRecords(workspaceRoot)
}

func plannedOperations(workspaceRoot string, manifest Manifest) ([]FileOperation, []string, error) {
	base := filepath.Dir(manifest.Path)
	operations := make([]FileOperation, 0, len(manifest.Files))
	conflicts := []string{}
	for _, rel := range manifest.Files {
		source := filepath.Join(base, "payload", filepath.FromSlash(rel))
		if _, err := os.Stat(source); err != nil {
			return nil, nil, fmt.Errorf("preset payload missing for %s", rel)
		}
		target := filepath.Join(workspaceRoot, filepath.FromSlash(rel))
		action := ActionCreate
		collision := false
		if _, err := os.Stat(target); err == nil {
			action = ActionUpdate
			collision = true
			conflicts = append(conflicts, rel)
		}
		operations = append(operations, FileOperation{
			TargetPath: rel,
			SourcePath: source,
			Action:     action,
			Collision:  collision,
		})
	}
	sort.Slice(operations, func(i, j int) bool { return operations[i].TargetPath < operations[j].TargetPath })
	sort.Strings(conflicts)
	return operations, conflicts, nil
}

func compositionConflictsForManifest(manifest Manifest) []string {
	conflicts := []string{}
	if manifest.PresetType != "infrastructure" {
		if len(manifest.RequiredModules) > 0 {
			conflicts = append(conflicts, fmt.Sprintf("preset %s cannot declare required_modules unless preset_type=infrastructure", manifest.ID))
		}
		for _, scope := range manifest.MemoryScopes {
			scope = strings.TrimSpace(scope)
			if scope == "system" {
				conflicts = append(conflicts, fmt.Sprintf("preset %s cannot claim system memory scope unless preset_type=infrastructure", manifest.ID))
			}
		}
		switch strings.TrimSpace(manifest.Permissions.Runtime) {
		case "sandboxed_exec", "risky_exec_requires_approval":
			conflicts = append(conflicts, fmt.Sprintf("preset %s cannot request runtime permission %s unless preset_type=infrastructure", manifest.ID, manifest.Permissions.Runtime))
		}
	}
	return conflicts
}

func pairwiseCompositionConflicts(installed Manifest, candidate Manifest) []string {
	conflicts := []string{}
	for _, cmd := range candidate.Commands {
		if duplicateCommandName(installed, cmd.Name) {
			conflicts = append(conflicts, fmt.Sprintf("command collision: %s already declares command %s", installed.ID, cmd.Name))
		}
	}
	for _, hook := range candidate.Hooks {
		if duplicateHook(installed, hook) {
			conflicts = append(conflicts, fmt.Sprintf("hook collision: %s already declares %s/%s", installed.ID, hook.Lifecycle, hook.Name))
		}
	}
	for _, scope := range candidate.MemoryScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" || memoryScopeCanBeShared(scope) {
			continue
		}
		if hasMemoryScope(installed, scope) {
			conflicts = append(conflicts, fmt.Sprintf("memory scope collision: %s already claims %s", installed.ID, scope))
		}
	}
	if installed.PresetType == "infrastructure" && candidate.PresetType != "infrastructure" {
		if len(candidate.RequiredModules) > 0 {
			conflicts = append(conflicts, fmt.Sprintf("preset %s cannot require infrastructure modules while %s is installed as infrastructure", candidate.ID, installed.ID))
		}
	}
	return conflicts
}

func duplicateCommandName(manifest Manifest, name string) bool {
	name = strings.TrimSpace(name)
	for _, cmd := range manifest.Commands {
		if strings.TrimSpace(cmd.Name) == name {
			return true
		}
	}
	return false
}

func duplicateHook(manifest Manifest, hook Hook) bool {
	name := strings.TrimSpace(hook.Name)
	lifecycle := strings.TrimSpace(hook.Lifecycle)
	for _, existing := range manifest.Hooks {
		if strings.TrimSpace(existing.Name) == name && strings.TrimSpace(existing.Lifecycle) == lifecycle {
			return true
		}
	}
	return false
}

func hasMemoryScope(manifest Manifest, scope string) bool {
	scope = strings.TrimSpace(scope)
	for _, existing := range manifest.MemoryScopes {
		if strings.TrimSpace(existing) == scope {
			return true
		}
	}
	return false
}

func memoryScopeCanBeShared(scope string) bool {
	switch strings.TrimSpace(scope) {
	case "project", "session", "run_artifacts", "archive":
		return true
	default:
		return false
	}
}

func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func writeTargetFile(workspaceRoot string, operation FileOperation) error {
	data, err := os.ReadFile(operation.SourcePath)
	if err != nil {
		return err
	}
	target := filepath.Join(workspaceRoot, filepath.FromSlash(operation.TargetPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func backupFile(workspaceRoot string, relativeTarget string, backupDir string) error {
	source := filepath.Join(workspaceRoot, filepath.FromSlash(relativeTarget))
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	target := filepath.Join(backupDir, filepath.FromSlash(relativeTarget))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func rollbackApplied(workspaceRoot string, operations []FileOperation, backupDir string) error {
	for i := len(operations) - 1; i >= 0; i-- {
		op := operations[i]
		target := filepath.Join(workspaceRoot, filepath.FromSlash(op.TargetPath))
		if op.Collision {
			backup := filepath.Join(backupDir, filepath.FromSlash(op.TargetPath))
			data, err := os.ReadFile(backup)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return err
			}
			continue
		}
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func appendInstalledRecord(workspaceRoot string, record InstalledRecord) error {
	records, err := loadInstalledRecords(workspaceRoot)
	if err != nil {
		return err
	}
	records = append(records, record)
	return saveInstalledRecords(workspaceRoot, records)
}

func loadInstalledRecords(workspaceRoot string) ([]InstalledRecord, error) {
	path := project.ProjectFile(workspaceRoot, "presets", "installed.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := project.WriteJSON(path, []InstalledRecord{}); err != nil {
			return nil, err
		}
	}
	var records []InstalledRecord
	if err := project.ReadJSON(path, &records); err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool { return records[i].InstalledAt > records[j].InstalledAt })
	return records, nil
}

func saveInstalledRecords(workspaceRoot string, records []InstalledRecord) error {
	return project.WriteJSON(project.ProjectFile(workspaceRoot, "presets", "installed.json"), records)
}

func renderInstallReport(preview InstallPreview, record InstalledRecord) string {
	var b strings.Builder
	b.WriteString("# Preset Install Report\n\n")
	b.WriteString("- install_id: " + record.InstallID + "\n")
	b.WriteString("- preset_id: " + record.PresetID + "\n")
	b.WriteString("- version: " + record.Version + "\n")
	b.WriteString("- target: " + record.Target + "\n")
	b.WriteString("- installed_at: " + record.InstalledAt + "\n")
	b.WriteString("- overwrite_allowed: " + fmt.Sprintf("%t", record.AllowOverwrite) + "\n\n")
	b.WriteString("## Operations\n\n")
	for _, op := range preview.Operations {
		line := "- " + string(op.Action) + " " + op.TargetPath
		if op.Collision {
			line += " (collision backed up)"
		}
		b.WriteString(line + "\n")
	}
	if len(preview.EnvironmentConflicts) > 0 {
		b.WriteString("\n## Environment Conflicts\n\n")
		for _, conflict := range preview.EnvironmentConflicts {
			b.WriteString("- " + conflict + "\n")
		}
	}
	if len(preview.Resolution.Layers) > 0 {
		b.WriteString("\n## Environment Resolution\n\n")
		b.WriteString("- effective_runtime_ceiling: " + preview.Resolution.EffectiveRuntimeCeiling + "\n")
		b.WriteString("- effective_budget_profile: " + preview.Resolution.EffectiveBudgetProfile + "\n")
		b.WriteString("- layers: " + fmt.Sprintf("%d", len(preview.Resolution.Layers)) + "\n")
	}
	if len(preview.Manifest.SafetyNotes) > 0 {
		b.WriteString("\n## Safety Notes\n\n")
		for _, note := range preview.Manifest.SafetyNotes {
			b.WriteString("- " + note + "\n")
		}
	}
	return b.String()
}

func hasAnyCollision(operations []FileOperation) bool {
	for _, op := range operations {
		if op.Collision {
			return true
		}
	}
	return false
}
