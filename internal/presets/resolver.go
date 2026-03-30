package presets

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"agent-os/internal/project"
)

type EnvironmentResolution struct {
	WorkspaceRoot            string             `json:"workspace_root"`
	CandidatePresetID        string             `json:"candidate_preset_id,omitempty"`
	CandidatePresetType      string             `json:"candidate_preset_type,omitempty"`
	Layers                   []EnvironmentLayer `json:"layers"`
	HookRegistry             []ResolvedHook     `json:"hook_registry,omitempty"`
	CommandRegistry          []ResolvedCommand  `json:"command_registry,omitempty"`
	MemoryOwnership          []MemoryScopeOwner `json:"memory_ownership,omitempty"`
	EffectiveRuntimeCeiling  string             `json:"effective_runtime_ceiling,omitempty"`
	EffectiveBudgetProfile   string             `json:"effective_budget_profile,omitempty"`
	InstalledInfrastructure  []string           `json:"installed_infrastructure,omitempty"`
	InstalledDomainPresets   []string           `json:"installed_domain_presets,omitempty"`
	InstalledSessionOverlays []string           `json:"installed_session_overlays,omitempty"`
	EnvironmentConflicts     []string           `json:"environment_conflicts,omitempty"`
	Metadata                 map[string]string  `json:"metadata,omitempty"`
}

type EnvironmentLayer struct {
	Order             int      `json:"order"`
	Kind              string   `json:"kind"`
	Name              string   `json:"name"`
	PresetID          string   `json:"preset_id,omitempty"`
	PresetType        string   `json:"preset_type,omitempty"`
	Adapter           string   `json:"adapter,omitempty"`
	RuntimePermission string   `json:"runtime_permission,omitempty"`
	BudgetProfile     string   `json:"budget_profile,omitempty"`
	MemoryScopes      []string `json:"memory_scopes,omitempty"`
	Commands          []string `json:"commands,omitempty"`
	Hooks             []string `json:"hooks,omitempty"`
	Source            string   `json:"source,omitempty"`
}

type ResolvedHook struct {
	Name            string `json:"name"`
	Lifecycle       string `json:"lifecycle"`
	PermissionScope string `json:"permission_scope"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	OwnerPresetID   string `json:"owner_preset_id,omitempty"`
	OwnerLayer      string `json:"owner_layer"`
}

type ResolvedCommand struct {
	Name          string `json:"name"`
	OwnerPresetID string `json:"owner_preset_id,omitempty"`
	OwnerLayer    string `json:"owner_layer"`
}

type MemoryScopeOwner struct {
	Scope         string `json:"scope"`
	OwnerPresetID string `json:"owner_preset_id,omitempty"`
	OwnerLayer    string `json:"owner_layer"`
	Shared        bool   `json:"shared"`
}

type MemoryPolicy struct {
	RunID          string             `json:"run_id,omitempty"`
	AllowedScopes  []string           `json:"allowed_scopes"`
	OwnedScopes    []MemoryScopeOwner `json:"owned_scopes,omitempty"`
	SharedScopes   []string           `json:"shared_scopes,omitempty"`
	SystemWritable bool               `json:"system_writable"`
}

func ResolveEnvironment(workspaceRoot string, catalogRoot string, candidate *Manifest) (EnvironmentResolution, error) {
	resolved, err := project.DiscoverRoot(workspaceRoot)
	if err != nil {
		return EnvironmentResolution{}, err
	}
	proj, err := project.Load(resolved)
	if err != nil {
		return EnvironmentResolution{}, err
	}

	records, err := loadInstalledRecords(resolved)
	if err != nil {
		return EnvironmentResolution{}, err
	}
	installed, err := resolveInstalledManifests(records, catalogRoot, candidate)
	if err != nil {
		return EnvironmentResolution{}, err
	}

	resolution := EnvironmentResolution{
		WorkspaceRoot:            resolved,
		Layers:                   []EnvironmentLayer{},
		HookRegistry:             []ResolvedHook{},
		CommandRegistry:          []ResolvedCommand{},
		MemoryOwnership:          []MemoryScopeOwner{},
		EnvironmentConflicts:     []string{},
		InstalledInfrastructure:  []string{},
		InstalledDomainPresets:   []string{},
		InstalledSessionOverlays: []string{},
		Metadata: map[string]string{
			"project_mode":     proj.Mode.Mode,
			"default_provider": proj.Config.DefaultProvider,
		},
	}
	if candidate != nil {
		resolution.CandidatePresetID = candidate.ID
		resolution.CandidatePresetType = candidate.PresetType
	}

	resolution.Layers = append(resolution.Layers, EnvironmentLayer{
		Order:  1,
		Kind:   "arc_base",
		Name:   "ARC Base Rules",
		Source: ".arc",
	})
	resolution.Layers = append(resolution.Layers, EnvironmentLayer{
		Order:   2,
		Kind:    "provider_adapter",
		Name:    providerLayerName(candidate, proj.Config.DefaultProvider),
		Adapter: providerLayerAdapter(candidate, proj.Config.DefaultProvider),
		Source:  "provider adapter",
	})

	order := 3
	maxRuntimeRank := 0
	effectiveBudget := ""
	for _, manifest := range installed {
		if manifest.PresetType == "infrastructure" {
			resolution.Layers = append(resolution.Layers, layerFromManifest(order, "installed_infrastructure", manifest))
			order++
			resolution.InstalledInfrastructure = append(resolution.InstalledInfrastructure, manifest.ID)
			maxRuntimeRank = max(maxRuntimeRank, runtimePermissionRank(manifest.Permissions.Runtime))
			if strings.TrimSpace(manifest.BudgetProfile) != "" {
				effectiveBudget = manifest.BudgetProfile
			}
			resolution.HookRegistry = append(resolution.HookRegistry, resolvedHooks(manifest, "installed_infrastructure")...)
			resolution.CommandRegistry = append(resolution.CommandRegistry, resolvedCommands(manifest, "installed_infrastructure")...)
			resolution.MemoryOwnership = append(resolution.MemoryOwnership, memoryOwners(manifest, "installed_infrastructure")...)
		}
	}
	if candidate != nil && candidate.PresetType == "infrastructure" {
		resolution.Layers = append(resolution.Layers, layerFromManifest(order, "candidate_infrastructure", *candidate))
		order++
		maxRuntimeRank = max(maxRuntimeRank, runtimePermissionRank(candidate.Permissions.Runtime))
		if strings.TrimSpace(candidate.BudgetProfile) != "" {
			effectiveBudget = candidate.BudgetProfile
		}
		resolution.HookRegistry = append(resolution.HookRegistry, resolvedHooks(*candidate, "candidate_infrastructure")...)
		resolution.CommandRegistry = append(resolution.CommandRegistry, resolvedCommands(*candidate, "candidate_infrastructure")...)
		resolution.MemoryOwnership = append(resolution.MemoryOwnership, memoryOwners(*candidate, "candidate_infrastructure")...)
	}
	for _, manifest := range installed {
		if manifest.PresetType == "domain" {
			resolution.Layers = append(resolution.Layers, layerFromManifest(order, "installed_domain", manifest))
			order++
			resolution.InstalledDomainPresets = append(resolution.InstalledDomainPresets, manifest.ID)
			maxRuntimeRank = max(maxRuntimeRank, runtimePermissionRank(manifest.Permissions.Runtime))
			if strings.TrimSpace(manifest.BudgetProfile) != "" {
				effectiveBudget = manifest.BudgetProfile
			}
			resolution.HookRegistry = append(resolution.HookRegistry, resolvedHooks(manifest, "installed_domain")...)
			resolution.CommandRegistry = append(resolution.CommandRegistry, resolvedCommands(manifest, "installed_domain")...)
			resolution.MemoryOwnership = append(resolution.MemoryOwnership, memoryOwners(manifest, "installed_domain")...)
		}
	}
	if candidate != nil && candidate.PresetType == "domain" {
		resolution.Layers = append(resolution.Layers, layerFromManifest(order, "candidate_domain", *candidate))
		order++
		maxRuntimeRank = max(maxRuntimeRank, runtimePermissionRank(candidate.Permissions.Runtime))
		if strings.TrimSpace(candidate.BudgetProfile) != "" {
			effectiveBudget = candidate.BudgetProfile
		}
		resolution.HookRegistry = append(resolution.HookRegistry, resolvedHooks(*candidate, "candidate_domain")...)
		resolution.CommandRegistry = append(resolution.CommandRegistry, resolvedCommands(*candidate, "candidate_domain")...)
		resolution.MemoryOwnership = append(resolution.MemoryOwnership, memoryOwners(*candidate, "candidate_domain")...)
	}
	for _, manifest := range installed {
		if manifest.PresetType == "session_overlay" {
			resolution.Layers = append(resolution.Layers, layerFromManifest(order, "installed_session_overlay", manifest))
			order++
			resolution.InstalledSessionOverlays = append(resolution.InstalledSessionOverlays, manifest.ID)
			maxRuntimeRank = max(maxRuntimeRank, runtimePermissionRank(manifest.Permissions.Runtime))
			if strings.TrimSpace(manifest.BudgetProfile) != "" {
				effectiveBudget = manifest.BudgetProfile
			}
			resolution.HookRegistry = append(resolution.HookRegistry, resolvedHooks(manifest, "installed_session_overlay")...)
			resolution.CommandRegistry = append(resolution.CommandRegistry, resolvedCommands(manifest, "installed_session_overlay")...)
			resolution.MemoryOwnership = append(resolution.MemoryOwnership, memoryOwners(manifest, "installed_session_overlay")...)
		}
	}
	if candidate != nil && candidate.PresetType == "session_overlay" {
		resolution.Layers = append(resolution.Layers, layerFromManifest(order, "candidate_session_overlay", *candidate))
		order++
		maxRuntimeRank = max(maxRuntimeRank, runtimePermissionRank(candidate.Permissions.Runtime))
		if strings.TrimSpace(candidate.BudgetProfile) != "" {
			effectiveBudget = candidate.BudgetProfile
		}
		resolution.HookRegistry = append(resolution.HookRegistry, resolvedHooks(*candidate, "candidate_session_overlay")...)
		resolution.CommandRegistry = append(resolution.CommandRegistry, resolvedCommands(*candidate, "candidate_session_overlay")...)
		resolution.MemoryOwnership = append(resolution.MemoryOwnership, memoryOwners(*candidate, "candidate_session_overlay")...)
	}
	resolution.Layers = append(resolution.Layers, EnvironmentLayer{
		Order:  order,
		Kind:   "project_overlay",
		Name:   "Project Overlay",
		Source: filepath.Join(resolved, ".arc"),
	})

	resolution.EffectiveRuntimeCeiling = runtimePermissionFromRank(maxRuntimeRank)
	if effectiveBudget == "" {
		if candidate != nil && strings.TrimSpace(candidate.BudgetProfile) != "" {
			effectiveBudget = candidate.BudgetProfile
		} else {
			effectiveBudget = "balanced"
		}
	}
	resolution.EffectiveBudgetProfile = effectiveBudget
	if candidate != nil {
		conflicts := compositionConflictsForManifest(*candidate)
		for _, manifest := range installed {
			conflicts = append(conflicts, pairwiseCompositionConflicts(manifest, *candidate)...)
		}
		resolution.EnvironmentConflicts = dedupeStrings(conflicts)
	}
	return resolution, nil
}

func resolveInstalledManifests(records []InstalledRecord, catalogRoot string, candidate *Manifest) ([]Manifest, error) {
	out := []Manifest{}
	seen := map[string]struct{}{}
	for _, record := range records {
		if record.Status != "installed" {
			continue
		}
		if candidate != nil && record.PresetID == candidate.ID {
			continue
		}
		if _, ok := seen[record.PresetID]; ok {
			continue
		}
		manifest, err := manifestFromInstalledRecord(record, catalogRoot)
		if err != nil {
			return nil, err
		}
		out = append(out, manifest)
		seen[record.PresetID] = struct{}{}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PresetType == out[j].PresetType {
			return out[i].ID < out[j].ID
		}
		return out[i].PresetType < out[j].PresetType
	})
	return out, nil
}

func manifestFromInstalledRecord(record InstalledRecord, catalogRoot string) (Manifest, error) {
	if strings.TrimSpace(record.Manifest.ID) != "" {
		manifest := record.Manifest
		normalizeManifest(&manifest)
		if manifest.Path == "" {
			manifest.Path = strings.TrimSpace(record.ManifestPath)
		}
		if err := validateManifest(manifest); err != nil {
			return Manifest{}, fmt.Errorf("installed preset %q snapshot is invalid: %w", record.PresetID, err)
		}
		return manifest, nil
	}
	if strings.TrimSpace(catalogRoot) == "" {
		return Manifest{}, fmt.Errorf("installed preset %q has no manifest snapshot and no catalog root was provided", record.PresetID)
	}
	return LoadByID(catalogRoot, record.PresetID)
}

func layerFromManifest(order int, kind string, manifest Manifest) EnvironmentLayer {
	return EnvironmentLayer{
		Order:             order,
		Kind:              kind,
		Name:              manifest.Name,
		PresetID:          manifest.ID,
		PresetType:        manifest.PresetType,
		Adapter:           manifest.Adapter,
		RuntimePermission: fallbackRuntimePermission(manifest.Permissions.Runtime),
		BudgetProfile:     strings.TrimSpace(manifest.BudgetProfile),
		MemoryScopes:      append([]string{}, manifest.MemoryScopes...),
		Commands:          manifestCommandNames(manifest),
		Hooks:             manifestHookNames(manifest),
		Source:            manifest.Path,
	}
}

func resolvedHooks(manifest Manifest, layer string) []ResolvedHook {
	out := make([]ResolvedHook, 0, len(manifest.Hooks))
	for _, hook := range manifest.Hooks {
		out = append(out, ResolvedHook{
			Name:            hook.Name,
			Lifecycle:       hook.Lifecycle,
			PermissionScope: hook.PermissionScope,
			TimeoutSeconds:  hook.TimeoutSeconds,
			OwnerPresetID:   manifest.ID,
			OwnerLayer:      layer,
		})
	}
	return out
}

func resolvedCommands(manifest Manifest, layer string) []ResolvedCommand {
	out := make([]ResolvedCommand, 0, len(manifest.Commands))
	for _, cmd := range manifest.Commands {
		out = append(out, ResolvedCommand{
			Name:          cmd.Name,
			OwnerPresetID: manifest.ID,
			OwnerLayer:    layer,
		})
	}
	return out
}

func memoryOwners(manifest Manifest, layer string) []MemoryScopeOwner {
	out := make([]MemoryScopeOwner, 0, len(manifest.MemoryScopes))
	for _, scope := range manifest.MemoryScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		out = append(out, MemoryScopeOwner{
			Scope:         scope,
			OwnerPresetID: manifest.ID,
			OwnerLayer:    layer,
			Shared:        memoryScopeCanBeShared(scope),
		})
	}
	return out
}

func providerLayerName(candidate *Manifest, fallback string) string {
	adapter := providerLayerAdapter(candidate, fallback)
	if adapter == "" {
		return "Provider Adapter"
	}
	return fmt.Sprintf("%s Adapter", strings.Title(adapter))
}

func providerLayerAdapter(candidate *Manifest, fallback string) string {
	if candidate != nil && strings.TrimSpace(candidate.Adapter) != "" {
		return candidate.Adapter
	}
	return strings.TrimSpace(fallback)
}

func manifestCommandNames(manifest Manifest) []string {
	out := make([]string, 0, len(manifest.Commands))
	for _, cmd := range manifest.Commands {
		if strings.TrimSpace(cmd.Name) != "" {
			out = append(out, cmd.Name)
		}
	}
	return out
}

func manifestHookNames(manifest Manifest) []string {
	out := make([]string, 0, len(manifest.Hooks))
	for _, hook := range manifest.Hooks {
		if strings.TrimSpace(hook.Name) != "" {
			out = append(out, fmt.Sprintf("%s:%s", hook.Lifecycle, hook.Name))
		}
	}
	return out
}

func fallbackRuntimePermission(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "none"
	}
	return value
}

func runtimePermissionFromRank(rank int) string {
	switch rank {
	case 1:
		return "read_only"
	case 2:
		return "preview_only"
	case 3:
		return "sandboxed_exec"
	case 4:
		return "risky_exec_requires_approval"
	default:
		return "none"
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func RenderEnvironmentResolutionMarkdown(resolution EnvironmentResolution) string {
	var b strings.Builder
	b.WriteString("# Environment Resolution\n\n")
	b.WriteString("- workspace_root: " + resolution.WorkspaceRoot + "\n")
	if resolution.CandidatePresetID != "" {
		b.WriteString("- candidate_preset_id: " + resolution.CandidatePresetID + "\n")
	}
	b.WriteString("- effective_runtime_ceiling: " + resolution.EffectiveRuntimeCeiling + "\n")
	b.WriteString("- effective_budget_profile: " + resolution.EffectiveBudgetProfile + "\n\n")
	b.WriteString("## Layers\n\n")
	for _, layer := range resolution.Layers {
		b.WriteString(fmt.Sprintf("- %d. %s", layer.Order, layer.Kind))
		if layer.PresetID != "" {
			b.WriteString(fmt.Sprintf(" (%s)", layer.PresetID))
		}
		b.WriteString(": " + layer.Name + "\n")
	}
	if len(resolution.EnvironmentConflicts) > 0 {
		b.WriteString("\n## Environment Conflicts\n\n")
		for _, conflict := range resolution.EnvironmentConflicts {
			b.WriteString("- " + conflict + "\n")
		}
	}
	if len(resolution.HookRegistry) > 0 {
		b.WriteString("\n## Hook Registry\n\n")
		for _, hook := range resolution.HookRegistry {
			b.WriteString(fmt.Sprintf("- %s (%s, scope=%s, timeout=%ds)\n", hook.Name, hook.Lifecycle, hook.PermissionScope, hook.TimeoutSeconds))
		}
	}
	return b.String()
}

func BuildMemoryPolicy(resolution EnvironmentResolution, runID string) MemoryPolicy {
	allowed := []string{"project", "session", "archive", "run_artifacts"}
	shared := []string{"project", "session", "archive", "run_artifacts"}
	if strings.TrimSpace(runID) != "" {
		allowed = append(allowed, "runs/"+runID)
	}
	owned := make([]MemoryScopeOwner, 0, len(resolution.MemoryOwnership))
	systemWritable := false
	for _, owner := range resolution.MemoryOwnership {
		scope := strings.TrimSpace(owner.Scope)
		if scope == "" {
			continue
		}
		owned = append(owned, owner)
		allowed = append(allowed, scope)
		if owner.Shared {
			shared = append(shared, scope)
		}
		if scope == "system" {
			systemWritable = true
		}
	}
	return MemoryPolicy{
		RunID:          strings.TrimSpace(runID),
		AllowedScopes:  dedupeStrings(allowed),
		OwnedScopes:    owned,
		SharedScopes:   dedupeStrings(shared),
		SystemWritable: systemWritable,
	}
}

func RenderMemoryPolicyMarkdown(policy MemoryPolicy) string {
	var b strings.Builder
	b.WriteString("# Memory Policy\n\n")
	if policy.RunID != "" {
		b.WriteString("- run_id: " + policy.RunID + "\n")
	}
	b.WriteString("- system_writable: " + fmt.Sprintf("%t", policy.SystemWritable) + "\n\n")
	b.WriteString("## Allowed Scopes\n\n")
	for _, scope := range policy.AllowedScopes {
		b.WriteString("- " + scope + "\n")
	}
	if len(policy.OwnedScopes) > 0 {
		b.WriteString("\n## Owned Scopes\n\n")
		for _, owner := range policy.OwnedScopes {
			line := "- " + owner.Scope + " (" + owner.OwnerLayer
			if owner.OwnerPresetID != "" {
				line += ":" + owner.OwnerPresetID
			}
			line += ")"
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}
