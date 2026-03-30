package presets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-os/internal/project"
)

func TestExecuteHooksSandboxedExecUsesDedicatedSandbox(t *testing.T) {
	workspace := t.TempDir()
	runDir := filepath.Join(workspace, ".arc", "runs", "test-run")
	if err := os.MkdirAll(filepath.Join(workspace, ".arc", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\n" +
		"echo \"pwd=$(pwd)\"\n" +
		"echo \"sandbox=$ARC_HOOK_SANDBOX_DIR\"\n" +
		"echo \"workspace=$ARC_WORKSPACE_ROOT\"\n"
	if err := os.WriteFile(filepath.Join(workspace, ".arc", "hooks", "sandboxed.sh"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	summary, err := ExecuteHooks(EnvironmentResolution{
		EffectiveRuntimeCeiling: "sandboxed_exec",
		HookRegistry: []ResolvedHook{
			{
				Name:            "sandboxed",
				Lifecycle:       "before_run",
				OwnerPresetID:   "infra-runtime",
				OwnerLayer:      "installed_infrastructure",
				PermissionScope: "sandboxed_exec",
				TimeoutSeconds:  5,
			},
		},
	}, HookRunOptions{
		RunID:               "test-run",
		RunDir:              runDir,
		Lifecycle:           "before_run",
		WorkspaceRoot:       workspace,
		AllowedMemoryScopes: []string{"project", "runs/test-run"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Executions) != 1 {
		t.Fatalf("expected one execution, got %#v", summary.Executions)
	}
	record := summary.Executions[0]
	if record.Status != "executed" {
		t.Fatalf("expected executed status, got %#v", record)
	}
	if record.ExecutionMode != "sandboxed_local_subprocess" {
		t.Fatalf("expected sandboxed execution mode, got %#v", record)
	}
	if record.SandboxDir == "" {
		t.Fatalf("expected sandbox dir in record")
	}
	output, err := os.ReadFile(record.StdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	stdout := string(output)
	sandboxCanonical, err := filepath.EvalSymlinks(record.SandboxDir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "pwd="+sandboxCanonical) {
		t.Fatalf("expected hook to run from sandbox dir, got %q", stdout)
	}
	if !strings.Contains(stdout, "sandbox="+record.SandboxDir) {
		t.Fatalf("expected sandbox env to be exposed, got %q", stdout)
	}
	if !strings.Contains(stdout, "workspace="+workspace) {
		t.Fatalf("expected workspace root env to be exposed, got %q", stdout)
	}
	var profiles []HookSandboxProfile
	if err := project.ReadJSON(filepath.Join(runDir, "hook_sandbox_profile.json"), &profiles); err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected one sandbox profile, got %#v", profiles)
	}
	if !profiles[0].WorkspaceRootExposed || profiles[0].ParentEnvInherited {
		t.Fatalf("unexpected sandbox profile: %#v", profiles[0])
	}
	if profiles[0].MemoryWritePath != "arc hook memory add" {
		t.Fatalf("memory write path = %q, want arc hook memory add", profiles[0].MemoryWritePath)
	}
}

func TestExecuteHooksSkipsMissingOverlayHook(t *testing.T) {
	workspace := t.TempDir()
	runDir := filepath.Join(workspace, ".arc", "runs", "test-run")
	if err := os.MkdirAll(filepath.Join(workspace, ".arc", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	summary, err := ExecuteHooks(EnvironmentResolution{
		EffectiveRuntimeCeiling: "read_only",
		HookRegistry: []ResolvedHook{
			{
				Name:            "missing-overlay",
				Lifecycle:       "before_run",
				OwnerPresetID:   "overlay",
				OwnerLayer:      "project_overlay",
				PermissionScope: "read_only",
				TimeoutSeconds:  5,
			},
		},
	}, HookRunOptions{
		RunID:         "test-run",
		RunDir:        runDir,
		Lifecycle:     "before_run",
		WorkspaceRoot: workspace,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Executions) != 1 || summary.Executions[0].Status != "missing_skipped" {
		t.Fatalf("unexpected summary: %#v", summary.Executions)
	}
}

func TestExecuteHooksFailsMissingInfrastructureHook(t *testing.T) {
	workspace := t.TempDir()
	runDir := filepath.Join(workspace, ".arc", "runs", "test-run")
	if err := os.MkdirAll(filepath.Join(workspace, ".arc", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	summary, err := ExecuteHooks(EnvironmentResolution{
		EffectiveRuntimeCeiling: "read_only",
		HookRegistry: []ResolvedHook{
			{
				Name:            "missing-required",
				Lifecycle:       "before_run",
				OwnerPresetID:   "infra",
				OwnerLayer:      "installed_infrastructure",
				PermissionScope: "read_only",
				TimeoutSeconds:  5,
			},
		},
	}, HookRunOptions{
		RunID:         "test-run",
		RunDir:        runDir,
		Lifecycle:     "before_run",
		WorkspaceRoot: workspace,
	})
	if err == nil {
		t.Fatal("expected missing infrastructure hook to fail")
	}
	if len(summary.Executions) != 1 || summary.Executions[0].Status != "missing_required" {
		t.Fatalf("unexpected summary: %#v", summary.Executions)
	}
}

func TestExecuteHooksRejectsNonShellSandboxedHook(t *testing.T) {
	workspace := t.TempDir()
	runDir := filepath.Join(workspace, ".arc", "runs", "test-run")
	if err := os.MkdirAll(filepath.Join(workspace, ".arc", "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".arc", "hooks", "binary-hook"), []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	summary, err := ExecuteHooks(EnvironmentResolution{
		EffectiveRuntimeCeiling: "sandboxed_exec",
		HookRegistry: []ResolvedHook{
			{
				Name:            "binary-hook",
				Lifecycle:       "before_run",
				OwnerPresetID:   "infra",
				OwnerLayer:      "installed_infrastructure",
				PermissionScope: "sandboxed_exec",
				TimeoutSeconds:  5,
			},
		},
	}, HookRunOptions{
		RunID:         "test-run",
		RunDir:        runDir,
		Lifecycle:     "before_run",
		WorkspaceRoot: workspace,
	})
	if err == nil {
		t.Fatal("expected sandboxed hook without shell extension to fail")
	}
	if len(summary.Executions) != 1 || summary.Executions[0].Status != "missing_required" {
		t.Fatalf("unexpected summary: %#v", summary.Executions)
	}
}

func TestHookExecutionSummaryPersistsMarkdown(t *testing.T) {
	workspace := t.TempDir()
	runDir := filepath.Join(workspace, "run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	summary := HookExecutionSummary{
		RunID:                   "run-1",
		WorkspaceRoot:           workspace,
		EffectiveRuntimeCeiling: "read_only",
		Executions: []HookExecutionRecord{{
			Name:      "sample",
			Lifecycle: "before_run",
			Status:    "missing_skipped",
		}},
	}
	if err := persistHookExecutionSummary(runDir, summary); err != nil {
		t.Fatal(err)
	}
	var persisted HookExecutionSummary
	if err := project.ReadJSON(filepath.Join(runDir, "hook_execution.json"), &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted.Executions) != 1 {
		t.Fatalf("expected persisted execution summary")
	}
}
