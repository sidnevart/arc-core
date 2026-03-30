package cli

import (
	"path/filepath"
	"testing"

	"agent-os/internal/budget"
	"agent-os/internal/project"
)

func TestBudgetOverrideSetAndClear(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}

	if err := Run([]string{
		"budget", "override", "set",
		"--path", root,
		"--mode", "deep_work",
		"--prefer-local", "false",
		"--note", "cli test",
	}); err != nil {
		t.Fatal(err)
	}

	override, present, err := budget.LoadProjectOverride(root)
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("expected project override to be present")
	}
	if override.Mode != "deep_work" {
		t.Fatalf("mode = %q, want deep_work", override.Mode)
	}
	if override.PreferLocal == nil || *override.PreferLocal {
		t.Fatalf("prefer_local = %#v, want false", override.PreferLocal)
	}

	if err := Run([]string{"budget", "override", "clear", "--path", root}); err != nil {
		t.Fatal(err)
	}
	_, present, err = budget.LoadProjectOverride(root)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("expected project override to be cleared")
	}
}

func TestBudgetShowSucceedsWithProjectOverride(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if err := budget.WriteProjectOverride(root, budget.PolicyOverride{Mode: "deep_work"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"budget", "show", "--path", root, "--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestBudgetSessionWriteShowAndClear(t *testing.T) {
	root := t.TempDir()
	sessionPath := filepath.Join(root, "session-budget.json")
	if err := Run([]string{
		"budget", "session", "write",
		"--file", sessionPath,
		"--mode", "emergency_low_limit",
		"--block-premium-required", "true",
		"--note", "session cli test",
	}); err != nil {
		t.Fatal(err)
	}

	override, present, err := budget.LoadOverride(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("expected session override to be present")
	}
	if override.Mode != "emergency_low_limit" {
		t.Fatalf("mode = %q, want emergency_low_limit", override.Mode)
	}
	if override.BlockPremiumRequired == nil || !*override.BlockPremiumRequired {
		t.Fatalf("block_premium_required = %#v, want true", override.BlockPremiumRequired)
	}

	if err := Run([]string{"budget", "session", "show", "--file", sessionPath, "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"budget", "session", "clear", "--file", sessionPath}); err != nil {
		t.Fatal(err)
	}
	_, present, err = budget.LoadOverride(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("expected session override to be cleared")
	}
}
