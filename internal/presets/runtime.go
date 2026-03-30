package presets

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"agent-os/internal/project"
)

type HookRunOptions struct {
	RunID               string
	RunDir              string
	Lifecycle           string
	ApproveRisky        bool
	DryRun              bool
	WorkspaceRoot       string
	AllowedMemoryScopes []string
}

type HookExecutionSummary struct {
	RunID                   string                `json:"run_id"`
	WorkspaceRoot           string                `json:"workspace_root"`
	EffectiveRuntimeCeiling string                `json:"effective_runtime_ceiling,omitempty"`
	Executions              []HookExecutionRecord `json:"executions,omitempty"`
}

type HookExecutionRecord struct {
	Name            string   `json:"name"`
	Lifecycle       string   `json:"lifecycle"`
	OwnerPresetID   string   `json:"owner_preset_id,omitempty"`
	OwnerLayer      string   `json:"owner_layer,omitempty"`
	PermissionScope string   `json:"permission_scope,omitempty"`
	DeclaredTimeout int      `json:"declared_timeout_seconds,omitempty"`
	HookPath        string   `json:"hook_path,omitempty"`
	Status          string   `json:"status"`
	ExecutionMode   string   `json:"execution_mode,omitempty"`
	StartedAt       string   `json:"started_at,omitempty"`
	FinishedAt      string   `json:"finished_at,omitempty"`
	ExitCode        int      `json:"exit_code,omitempty"`
	StdoutPath      string   `json:"stdout_path,omitempty"`
	StderrPath      string   `json:"stderr_path,omitempty"`
	SandboxDir      string   `json:"sandbox_dir,omitempty"`
	Error           string   `json:"error,omitempty"`
	Notes           []string `json:"notes,omitempty"`
}

type HookExecutionError struct {
	Lifecycle        string
	RequiresApproval bool
	Message          string
}

func (e HookExecutionError) Error() string {
	return e.Message
}

func ExecuteHooks(resolution EnvironmentResolution, opts HookRunOptions) (HookExecutionSummary, error) {
	summary := HookExecutionSummary{
		RunID:                   opts.RunID,
		WorkspaceRoot:           opts.WorkspaceRoot,
		EffectiveRuntimeCeiling: resolution.EffectiveRuntimeCeiling,
		Executions:              []HookExecutionRecord{},
	}
	hooksDir := filepath.Join(opts.WorkspaceRoot, ".arc", "hooks")
	for _, hook := range resolution.HookRegistry {
		if strings.TrimSpace(hook.Lifecycle) != strings.TrimSpace(opts.Lifecycle) {
			continue
		}
		record, err := executeHook(hooksDir, hook, resolution.EffectiveRuntimeCeiling, opts)
		summary.Executions = append(summary.Executions, record)
		if persistErr := persistHookExecutionSummary(opts.RunDir, summary); persistErr != nil {
			return summary, persistErr
		}
		if err != nil {
			return summary, err
		}
	}
	if len(summary.Executions) == 0 {
		return summary, persistHookExecutionSummary(opts.RunDir, summary)
	}
	return summary, nil
}

func executeHook(hooksDir string, hook ResolvedHook, runtimeCeiling string, opts HookRunOptions) (HookExecutionRecord, error) {
	record := HookExecutionRecord{
		Name:            hook.Name,
		Lifecycle:       hook.Lifecycle,
		OwnerPresetID:   hook.OwnerPresetID,
		OwnerLayer:      hook.OwnerLayer,
		PermissionScope: hook.PermissionScope,
		DeclaredTimeout: hook.TimeoutSeconds,
		Status:          "pending",
	}
	if runtimePermissionRank(hook.PermissionScope) == 0 {
		record.Status = "blocked"
		record.Error = "hook permission scope is none"
		return record, HookExecutionError{
			Lifecycle: opts.Lifecycle,
			Message:   fmt.Sprintf("hook %s is blocked because permission scope is none", hook.Name),
		}
	}
	if runtimePermissionRank(hook.PermissionScope) > runtimePermissionRank(runtimeCeiling) {
		record.Status = "blocked"
		record.Error = "hook permission scope exceeds effective runtime ceiling"
		return record, HookExecutionError{
			Lifecycle: opts.Lifecycle,
			Message:   fmt.Sprintf("hook %s exceeds effective runtime ceiling %s", hook.Name, runtimeCeiling),
		}
	}
	if hook.PermissionScope == "risky_exec_requires_approval" && !opts.ApproveRisky {
		record.Status = "blocked_requires_approval"
		record.Error = "hook requires risky approval"
		return record, HookExecutionError{
			Lifecycle:        opts.Lifecycle,
			RequiresApproval: true,
			Message:          fmt.Sprintf("hook %s requires risky approval", hook.Name),
		}
	}
	if opts.DryRun && runtimePermissionRank(hook.PermissionScope) > runtimePermissionRank("preview_only") {
		record.Status = "skipped_for_dry_run"
		record.Notes = []string{"dry-run only executes hooks up to preview_only"}
		return record, nil
	}

	sandboxed := hook.PermissionScope == "sandboxed_exec" || hook.PermissionScope == "risky_exec_requires_approval"
	hookPath, invokeArgs, err := discoverHookPath(hooksDir, hook.Name, sandboxed)
	if err != nil {
		record.ExecutionMode = hookExecutionMode(hook.PermissionScope)
		record.Status = "missing_required"
		if canSoftSkipMissingHook(hook.OwnerLayer) {
			record.Status = "missing_skipped"
			record.Notes = []string{"missing hook implementation was skipped because the owner layer is overlay-scoped"}
			record.Error = err.Error()
			return record, nil
		}
		record.Error = err.Error()
		return record, err
	}
	record.HookPath = hookPath
	record.ExecutionMode = hookExecutionMode(hook.PermissionScope)
	record.StartedAt = time.Now().UTC().Format(time.RFC3339)

	stdoutPath, stderrPath := hookLogPaths(opts.RunDir, hook.Lifecycle, hook.Name)
	if err := os.MkdirAll(filepath.Dir(stdoutPath), 0o755); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		return record, err
	}
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		return record, err
	}
	defer stdoutFile.Close()
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		return record, err
	}
	defer stderrFile.Close()

	timeout := time.Duration(hook.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, invokeArgs[0], invokeArgs[1:]...)
	if sandboxed {
		sandboxDir, notes, err := prepareHookSandbox(opts.RunDir, hook)
		if err != nil {
			record.Status = "failed"
			record.Error = err.Error()
			return record, err
		}
		record.SandboxDir = sandboxDir
		record.Notes = append(record.Notes, notes...)
		cmd.Dir = sandboxDir
		cmd.Env = hookEnvironment(opts, hook, sandboxDir, true)
	} else {
		cmd.Dir = opts.WorkspaceRoot
		cmd.Env = hookEnvironment(opts, hook, "", false)
	}
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	err = cmd.Run()
	record.StdoutPath = stdoutPath
	record.StderrPath = stderrPath
	record.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if ctx.Err() == context.DeadlineExceeded {
		record.Status = "timed_out"
		record.Error = fmt.Sprintf("hook timed out after %ds", hook.TimeoutSeconds)
		return record, HookExecutionError{
			Lifecycle: opts.Lifecycle,
			Message:   fmt.Sprintf("hook %s timed out after %ds", hook.Name, hook.TimeoutSeconds),
		}
	}
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			record.ExitCode = exitErr.ExitCode()
		}
		return record, err
	}
	record.Status = "executed"
	record.ExitCode = 0
	return record, nil
}

func hookExecutionMode(permissionScope string) string {
	switch permissionScope {
	case "sandboxed_exec":
		return "sandboxed_local_subprocess"
	case "risky_exec_requires_approval":
		return "approved_sandboxed_local_subprocess"
	default:
		return "local_subprocess"
	}
}

func canSoftSkipMissingHook(ownerLayer string) bool {
	switch strings.TrimSpace(ownerLayer) {
	case "project_overlay", "installed_session_overlay", "candidate_session_overlay":
		return true
	default:
		return false
	}
}

func prepareHookSandbox(runDir string, hook ResolvedHook) (string, []string, error) {
	slug := sanitizeHookName(hook.Lifecycle + "-" + hook.Name)
	sandboxDir := filepath.Join(runDir, "hooks", slug+".sandbox")
	homeDir := filepath.Join(sandboxDir, "home")
	tmpDir := filepath.Join(sandboxDir, "tmp")
	for _, dir := range []string{sandboxDir, homeDir, tmpDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", nil, err
		}
	}
	return sandboxDir, []string{
		"sandboxed_exec hooks run from a dedicated working directory under the run artifacts",
		"sandboxed_exec hooks receive a sanitized environment and must use ARC_WORKSPACE_ROOT explicitly to reach the repo",
	}, nil
}

func hookEnvironment(opts HookRunOptions, hook ResolvedHook, sandboxDir string, sandboxed bool) []string {
	base := []string{}
	if sandboxed {
		base = append(base,
			"PATH="+os.Getenv("PATH"),
			"HOME="+filepath.Join(sandboxDir, "home"),
			"TMPDIR="+filepath.Join(sandboxDir, "tmp"),
			"LANG="+fallbackEnv("LANG", "C.UTF-8"),
			"LC_ALL="+fallbackEnv("LC_ALL", "C.UTF-8"),
		)
	} else {
		base = append(base, os.Environ()...)
	}
	base = append(base,
		"ARC_WORKSPACE_ROOT="+opts.WorkspaceRoot,
		"ARC_RUN_ID="+opts.RunID,
		"ARC_RUN_DIR="+opts.RunDir,
		"ARC_HOOK_NAME="+hook.Name,
		"ARC_HOOK_LIFECYCLE="+hook.Lifecycle,
		"ARC_HOOK_OWNER_PRESET="+hook.OwnerPresetID,
		"ARC_RUNTIME_PERMISSION="+hook.PermissionScope,
		"ARC_ALLOWED_MEMORY_SCOPES="+strings.Join(opts.AllowedMemoryScopes, ","),
		"ARC_HOOK_MEMORY_ADD_CMD=arc hook memory add",
	)
	if sandboxed {
		base = append(base, "ARC_HOOK_SANDBOX_DIR="+sandboxDir)
	}
	return base
}

func fallbackEnv(key string, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func hookLogPaths(runDir string, lifecycle string, name string) (string, string) {
	slug := sanitizeHookName(lifecycle + "-" + name)
	return filepath.Join(runDir, "hooks", slug+".stdout.log"), filepath.Join(runDir, "hooks", slug+".stderr.log")
}

var hookNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeHookName(value string) string {
	value = hookNameSanitizer.ReplaceAllString(strings.TrimSpace(value), "_")
	value = strings.Trim(value, "._-")
	if value == "" {
		return "hook"
	}
	return value
}

func discoverHookPath(hooksDir string, name string, sandboxed bool) (string, []string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return "", nil, fmt.Errorf("hook %q has invalid name for local execution", name)
	}
	candidates := []string{
		filepath.Join(hooksDir, name),
		filepath.Join(hooksDir, name+".sh"),
		filepath.Join(hooksDir, name+".bash"),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		switch filepath.Ext(candidate) {
		case ".sh", ".bash":
			return candidate, []string{"/bin/sh", candidate}, nil
		default:
			if sandboxed {
				return "", nil, fmt.Errorf("hook %q requires a .sh or .bash script when permission scope is sandboxed_exec", name)
			}
			return candidate, []string{candidate}, nil
		}
	}
	return "", nil, fmt.Errorf("hook %q is declared but no script exists under %s", name, hooksDir)
}

func persistHookExecutionSummary(runDir string, latest HookExecutionSummary) error {
	path := filepath.Join(runDir, "hook_execution.json")
	current := HookExecutionSummary{
		RunID:                   latest.RunID,
		WorkspaceRoot:           latest.WorkspaceRoot,
		EffectiveRuntimeCeiling: latest.EffectiveRuntimeCeiling,
		Executions:              []HookExecutionRecord{},
	}
	if _, err := os.Stat(path); err == nil {
		if err := project.ReadJSON(path, &current); err != nil {
			return err
		}
	}
	if current.RunID == "" {
		current.RunID = latest.RunID
	}
	if current.WorkspaceRoot == "" {
		current.WorkspaceRoot = latest.WorkspaceRoot
	}
	if latest.EffectiveRuntimeCeiling != "" {
		current.EffectiveRuntimeCeiling = latest.EffectiveRuntimeCeiling
	}
	current.Executions = append(current.Executions, latest.Executions...)
	if err := project.WriteJSON(path, current); err != nil {
		return err
	}
	return project.WriteString(filepath.Join(runDir, "hook_execution.md"), RenderHookExecutionMarkdown(current))
}

func RenderHookExecutionMarkdown(summary HookExecutionSummary) string {
	var b strings.Builder
	b.WriteString("# Hook Execution\n\n")
	b.WriteString("- run_id: " + summary.RunID + "\n")
	b.WriteString("- workspace_root: " + summary.WorkspaceRoot + "\n")
	b.WriteString("- effective_runtime_ceiling: " + summary.EffectiveRuntimeCeiling + "\n\n")
	if len(summary.Executions) == 0 {
		b.WriteString("No hooks executed.\n")
		return b.String()
	}
	for _, execution := range summary.Executions {
		b.WriteString(fmt.Sprintf("- %s (%s): %s", execution.Name, execution.Lifecycle, execution.Status))
		if execution.OwnerPresetID != "" {
			b.WriteString(" [" + execution.OwnerPresetID + "]")
		}
		b.WriteString("\n")
		if execution.Error != "" {
			b.WriteString("  - error: " + execution.Error + "\n")
		}
		if execution.HookPath != "" {
			b.WriteString("  - hook_path: " + execution.HookPath + "\n")
		}
		if execution.StdoutPath != "" {
			b.WriteString("  - stdout: " + execution.StdoutPath + "\n")
		}
		if execution.StderrPath != "" {
			b.WriteString("  - stderr: " + execution.StderrPath + "\n")
		}
	}
	return b.String()
}
