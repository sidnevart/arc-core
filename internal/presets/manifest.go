package presets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Manifest struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Tagline             string        `json:"tagline"`
	ShortDescription    []string      `json:"short_description,omitempty"`
	Goal                string        `json:"goal"`
	Adapter             string        `json:"adapter"`
	Category            string        `json:"category"`
	Persona             string        `json:"persona"`
	Version             string        `json:"version"`
	Files               []string      `json:"files"`
	SafetyNotes         []string      `json:"safety_notes"`
	PresetType          string        `json:"preset_type,omitempty"`
	CompatibleProviders []string      `json:"compatible_providers,omitempty"`
	RequiredModules     []string      `json:"required_modules,omitempty"`
	Permissions         Permissions   `json:"permissions,omitempty"`
	Hooks               []Hook        `json:"hooks,omitempty"`
	Commands            []Command     `json:"commands,omitempty"`
	MemoryScopes        []string      `json:"memory_scopes,omitempty"`
	RuntimePolicy       RuntimePolicy `json:"runtime_policy,omitempty"`
	QualityGates        []string      `json:"quality_gates,omitempty"`
	MetricsExpectations []string      `json:"metrics_expectations,omitempty"`
	BudgetProfile       string        `json:"budget_profile,omitempty"`
	Author              Author        `json:"author"`
	Path                string        `json:"-"`
}

type Author struct {
	Name   string `json:"name"`
	Handle string `json:"handle"`
}

type Permissions struct {
	Runtime string `json:"runtime,omitempty"`
}

type Hook struct {
	Name            string `json:"name"`
	Lifecycle       string `json:"lifecycle"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty"`
	PermissionScope string `json:"permission_scope,omitempty"`
}

type Command struct {
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
}

type RuntimePolicy struct {
	AutoStopPolicy string `json:"auto_stop_policy,omitempty"`
}

func List(root string) ([]Manifest, error) {
	entries := []Manifest{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "manifest.yaml" {
			return nil
		}
		manifest, err := LoadManifest(path)
		if err != nil {
			return err
		}
		entries = append(entries, manifest)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Category == entries[j].Category {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Category < entries[j].Category
	})
	return entries, nil
}

func LoadByID(root string, id string) (Manifest, error) {
	manifests, err := List(root)
	if err != nil {
		return Manifest{}, err
	}
	for _, manifest := range manifests {
		if manifest.ID == id {
			return manifest, nil
		}
	}
	return Manifest{}, fmt.Errorf("preset %q not found", id)
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s must be JSON-compatible YAML: %w", path, err)
	}
	manifest.Path = path
	normalizeManifest(&manifest)
	if manifest.ID == "" || manifest.Name == "" || manifest.Adapter == "" {
		return Manifest{}, fmt.Errorf("manifest %s is missing required fields", path)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest %s is invalid: %w", path, err)
	}
	return manifest, nil
}

func normalizeManifest(manifest *Manifest) {
	if manifest == nil {
		return
	}
	if strings.TrimSpace(manifest.PresetType) == "" {
		manifest.PresetType = "domain"
	}
}

func validateManifest(manifest Manifest) error {
	if !allowedPresetType(manifest.PresetType) {
		return fmt.Errorf("unsupported preset_type %q", manifest.PresetType)
	}
	if !allowedRuntimePermission(manifest.Permissions.Runtime) {
		return fmt.Errorf("unsupported permissions.runtime %q", manifest.Permissions.Runtime)
	}
	if err := validateStringList(manifest.CompatibleProviders, "compatible_providers"); err != nil {
		return err
	}
	if err := validateStringList(manifest.RequiredModules, "required_modules"); err != nil {
		return err
	}
	if err := validateStringList(manifest.QualityGates, "quality_gates"); err != nil {
		return err
	}
	if err := validateStringList(manifest.MetricsExpectations, "metrics_expectations"); err != nil {
		return err
	}
	seenCommandNames := map[string]struct{}{}
	for _, cmd := range manifest.Commands {
		name := strings.TrimSpace(cmd.Name)
		if name == "" {
			return fmt.Errorf("command entry is missing name")
		}
		if _, exists := seenCommandNames[name]; exists {
			return fmt.Errorf("duplicate command %q", name)
		}
		seenCommandNames[name] = struct{}{}
	}
	seenHookKeys := map[string]struct{}{}
	for _, hook := range manifest.Hooks {
		name := strings.TrimSpace(hook.Name)
		lifecycle := strings.TrimSpace(hook.Lifecycle)
		if name == "" || lifecycle == "" {
			return fmt.Errorf("hook entries require name and lifecycle")
		}
		if !allowedHookLifecycle(lifecycle) {
			return fmt.Errorf("unsupported hook lifecycle %q", lifecycle)
		}
		if hook.TimeoutSeconds <= 0 || hook.TimeoutSeconds > 600 {
			return fmt.Errorf("hook %q must declare timeout_seconds between 1 and 600", name)
		}
		if strings.TrimSpace(hook.PermissionScope) == "" {
			return fmt.Errorf("hook %q must declare permission_scope", name)
		}
		if !allowedRuntimePermission(hook.PermissionScope) || strings.TrimSpace(hook.PermissionScope) == "" {
			return fmt.Errorf("hook %q declares unsupported permission_scope %q", name, hook.PermissionScope)
		}
		if runtimePermissionRank(hook.PermissionScope) > runtimePermissionRank(manifest.Permissions.Runtime) {
			return fmt.Errorf("hook %q permission_scope %q exceeds preset runtime permission %q", name, hook.PermissionScope, manifest.Permissions.Runtime)
		}
		key := lifecycle + ":" + name
		if _, exists := seenHookKeys[key]; exists {
			return fmt.Errorf("duplicate hook %q", key)
		}
		seenHookKeys[key] = struct{}{}
	}
	seenScopes := map[string]struct{}{}
	for _, scope := range manifest.MemoryScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return fmt.Errorf("memory_scopes cannot contain empty entries")
		}
		if !allowedMemoryScope(scope) {
			return fmt.Errorf("unsupported memory scope %q", scope)
		}
		if _, exists := seenScopes[scope]; exists {
			return fmt.Errorf("duplicate memory scope %q", scope)
		}
		seenScopes[scope] = struct{}{}
	}
	if len(manifest.ShortDescription) > 3 {
		return fmt.Errorf("short_description supports at most 3 lines")
	}
	for _, line := range manifest.ShortDescription {
		if strings.TrimSpace(line) == "" {
			return fmt.Errorf("short_description cannot contain empty lines")
		}
	}
	if !allowedBudgetProfile(manifest.BudgetProfile) {
		return fmt.Errorf("unsupported budget_profile %q", manifest.BudgetProfile)
	}
	if manifest.PresetType != "infrastructure" {
		if len(manifest.RequiredModules) > 0 {
			return fmt.Errorf("required_modules require preset_type=infrastructure")
		}
		for _, scope := range manifest.MemoryScopes {
			if strings.TrimSpace(scope) == "system" {
				return fmt.Errorf("memory scope %q requires preset_type=infrastructure", scope)
			}
		}
		switch strings.TrimSpace(manifest.Permissions.Runtime) {
		case "sandboxed_exec", "risky_exec_requires_approval":
			return fmt.Errorf("permissions.runtime %q requires preset_type=infrastructure", manifest.Permissions.Runtime)
		}
	}
	return nil
}

func validateStringList(values []string, field string) error {
	seen := map[string]struct{}{}
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			return fmt.Errorf("%s cannot contain empty entries", field)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate %s entry %q", field, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func allowedPresetType(value string) bool {
	switch strings.TrimSpace(value) {
	case "infrastructure", "domain", "session_overlay":
		return true
	default:
		return false
	}
}

func allowedRuntimePermission(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "none", "read_only", "preview_only", "sandboxed_exec", "risky_exec_requires_approval":
		return true
	default:
		return false
	}
}

func runtimePermissionRank(value string) int {
	switch strings.TrimSpace(value) {
	case "read_only":
		return 1
	case "preview_only":
		return 2
	case "sandboxed_exec":
		return 3
	case "risky_exec_requires_approval":
		return 4
	default:
		return 0
	}
}

func allowedHookLifecycle(value string) bool {
	switch strings.TrimSpace(value) {
	case "before_context_assembly", "after_context_assembly", "before_run", "after_run", "before_persist_memory", "before_launch_runtime":
		return true
	default:
		return false
	}
}

func allowedMemoryScope(value string) bool {
	switch value {
	case "system", "project", "session", "archive", "run_artifacts":
		return true
	}
	return strings.HasPrefix(value, "presets/") || strings.HasPrefix(value, "runs/")
}

func allowedBudgetProfile(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "ultra_safe", "balanced", "deep_work", "emergency_low_limit":
		return true
	default:
		return false
	}
}
