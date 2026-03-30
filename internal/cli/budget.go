package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"agent-os/internal/budget"
)

func runBudget(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("budget requires a subcommand")
	}
	switch args[0] {
	case "show":
		return runBudgetShow(args[1:])
	case "session":
		return runBudgetSession(args[1:])
	case "override":
		return runBudgetOverride(args[1:])
	default:
		return fmt.Errorf("unknown budget subcommand %q", args[0])
	}
}

func runBudgetShow(args []string) error {
	fs := flag.NewFlagSet("budget show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	override, present, err := budget.LoadProjectOverride(root)
	if err != nil {
		return err
	}
	resolution, err := budget.ResolvePolicy(root, "", "", "")
	if err != nil {
		return err
	}
	result := map[string]any{
		"project_override_path":    budget.ProjectOverridePath(root),
		"project_override_present": present,
		"project_override":         override,
		"effective_mode":           resolution.EffectiveMode,
		"effective_mode_source":    resolution.EffectiveModeSource,
		"effective_policy":         resolution.EffectivePolicy,
		"applied_override_sources": resolution.AppliedOverrideSources,
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("budget mode=%s source=%s\n", resolution.EffectiveMode, resolution.EffectiveModeSource)
	fmt.Printf("project override present=%t path=%s\n", present, budget.ProjectOverridePath(root))
	fmt.Printf("prefer_local=%t low_limit_state=%s\n", resolution.EffectivePolicy.PreferLocal, resolution.EffectivePolicy.LowLimitState)
	return nil
}

func runBudgetOverride(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("budget override requires a subcommand")
	}
	switch args[0] {
	case "set":
		return runBudgetOverrideSet(args[1:])
	case "clear":
		return runBudgetOverrideClear(args[1:])
	default:
		return fmt.Errorf("unknown budget override subcommand %q", args[0])
	}
}

func runBudgetSession(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("budget session requires a subcommand")
	}
	switch args[0] {
	case "show":
		return runBudgetSessionShow(args[1:])
	case "write":
		return runBudgetSessionWrite(args[1:])
	case "clear":
		return runBudgetSessionClear(args[1:])
	default:
		return fmt.Errorf("unknown budget session subcommand %q", args[0])
	}
}

func runBudgetOverrideSet(args []string) error {
	root, override, jsonOut, err := parseBudgetOverrideFlags("budget override set", args, true)
	if err != nil {
		return err
	}
	if err := budget.WriteProjectOverride(root, override); err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(map[string]any{
			"project_override_path": budget.ProjectOverridePath(root),
			"project_override":      override,
		})
	}
	fmt.Printf("budget override saved: %s\n", budget.ProjectOverridePath(root))
	return nil
}

func runBudgetOverrideClear(args []string) error {
	fs := flag.NewFlagSet("budget override clear", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	if err := budget.ClearProjectOverride(root); err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(map[string]any{
			"project_override_path": budget.ProjectOverridePath(root),
			"cleared":               true,
		})
	}
	fmt.Printf("budget override cleared: %s\n", budget.ProjectOverridePath(root))
	return nil
}

func runBudgetSessionShow(args []string) error {
	fs := flag.NewFlagSet("budget session show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileFlag := fs.String("file", "", "session override JSON file")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	override, present, err := budget.LoadOverride(*fileFlag)
	if err != nil {
		return err
	}
	result := map[string]any{
		"session_override_path":    strings.TrimSpace(*fileFlag),
		"session_override_present": present,
		"session_override":         override,
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("session override present=%t path=%s\n", present, strings.TrimSpace(*fileFlag))
	return nil
}

func runBudgetSessionWrite(args []string) error {
	overridePath, override, jsonOut, err := parseBudgetOverrideFlags("budget session write", args, false)
	if err != nil {
		return err
	}
	if err := budget.WriteOverride(overridePath, override); err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(map[string]any{
			"session_override_path": overridePath,
			"session_override":      override,
		})
	}
	fmt.Printf("session budget override saved: %s\n", overridePath)
	return nil
}

func runBudgetSessionClear(args []string) error {
	fs := flag.NewFlagSet("budget session clear", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fileFlag := fs.String("file", "", "session override JSON file")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	if err := budget.ClearOverride(strings.TrimSpace(*fileFlag)); err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(map[string]any{
			"session_override_path": strings.TrimSpace(*fileFlag),
			"cleared":               true,
		})
	}
	fmt.Printf("session budget override cleared: %s\n", strings.TrimSpace(*fileFlag))
	return nil
}

func parseBudgetOverrideFlags(name string, args []string, projectScoped bool) (string, budget.PolicyOverride, bool, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	fileFlag := fs.String("file", "", "session override JSON file")
	modeFlag := fs.String("mode", "", "budget mode override")
	lowLimitFlag := fs.String("low-limit-state", "", "override low-limit state")
	preferLocalFlag := fs.String("prefer-local", "", "override prefer_local with true/false")
	blockHighRiskFlag := fs.String("block-premium-high-risk", "", "override block_premium_high_risk with true/false")
	blockRequiredFlag := fs.String("block-premium-required", "", "override block_premium_required with true/false")
	requireHighRiskFlag := fs.String("require-approval-for-premium-high-risk", "", "override require_approval_for_premium_high_risk with true/false")
	requireRequiredFlag := fs.String("require-approval-for-premium-required", "", "override require_approval_for_premium_required with true/false")
	noteFlag := fs.String("note", "", "append note to the override")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return "", budget.PolicyOverride{}, false, err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return "", budget.PolicyOverride{}, false, err
	}
	override := budget.PolicyOverride{
		Mode:          strings.TrimSpace(*modeFlag),
		LowLimitState: strings.TrimSpace(*lowLimitFlag),
	}
	var err error
	if override.PreferLocal, err = parseOptionalBoolFlag(*preferLocalFlag); err != nil {
		return "", budget.PolicyOverride{}, false, err
	}
	if override.BlockPremiumHighRisk, err = parseOptionalBoolFlag(*blockHighRiskFlag); err != nil {
		return "", budget.PolicyOverride{}, false, err
	}
	if override.BlockPremiumRequired, err = parseOptionalBoolFlag(*blockRequiredFlag); err != nil {
		return "", budget.PolicyOverride{}, false, err
	}
	if override.RequireApprovalForPremiumHighRisk, err = parseOptionalBoolFlag(*requireHighRiskFlag); err != nil {
		return "", budget.PolicyOverride{}, false, err
	}
	if override.RequireApprovalForPremiumRequired, err = parseOptionalBoolFlag(*requireRequiredFlag); err != nil {
		return "", budget.PolicyOverride{}, false, err
	}
	if note := strings.TrimSpace(*noteFlag); note != "" {
		override.Notes = append(override.Notes, note)
	}
	if projectScoped {
		root, err := resolveProjectRoot(*pathFlag)
		return root, override, *jsonFlag, err
	}
	overridePath := strings.TrimSpace(*fileFlag)
	if overridePath == "" {
		return "", budget.PolicyOverride{}, false, fmt.Errorf("session override file is required")
	}
	return overridePath, override, *jsonFlag, nil
}

func parseOptionalBoolFlag(raw string) (*bool, error) {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return nil, nil
	}
	switch trimmed {
	case "true":
		value := true
		return &value, nil
	case "false":
		value := false
		return &value, nil
	default:
		return nil, fmt.Errorf("bool flag must be true or false, got %q", raw)
	}
}
