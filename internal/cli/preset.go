package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"agent-os/internal/app"
	"agent-os/internal/presets"
)

func runPreset(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("preset requires a subcommand")
	}
	switch args[0] {
	case "validate":
		return runPresetValidate(args[1:])
	case "draft":
		return runPresetDraft(args[1:])
	case "preview":
		return runPresetPreview(args[1:])
	case "install":
		return runPresetInstall(args[1:])
	case "list":
		return runPresetList(args[1:])
	case "rollback":
		return runPresetRollback(args[1:])
	default:
		return fmt.Errorf("unknown preset subcommand %q", args[0])
	}
}

func runPresetDraft(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("preset draft requires a subcommand")
	}
	switch args[0] {
	case "init":
		return runPresetDraftInit(args[1:])
	case "update":
		return runPresetDraftUpdate(args[1:])
	case "interview":
		return runPresetDraftInterview(args[1:])
	case "simulate":
		return runPresetDraftSimulate(args[1:])
	case "mark-tested":
		return runPresetDraftMarkTested(args[1:])
	case "export":
		return runPresetDraftExport(args[1:])
	case "publish":
		return runPresetDraftPublish(args[1:])
	case "catalog-sync":
		return runPresetDraftCatalogSync(args[1:])
	case "install":
		return runPresetDraftInstall(args[1:])
	case "show":
		return runPresetDraftShow(args[1:])
	case "validate":
		return runPresetDraftValidate(args[1:])
	case "list":
		return runPresetDraftList(args[1:])
	default:
		return fmt.Errorf("unknown preset draft subcommand %q", args[0])
	}
}

func runPresetDraftSimulate(args []string) error {
	fs := flag.NewFlagSet("preset draft simulate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft simulate [--path DIR] [--json] <preset-id>")
	}
	report, err := presets.SimulateDraft(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(report)
	}
	fmt.Printf("simulated preset draft %s\n", report.PresetID)
	fmt.Printf("status: %s\n", report.Status)
	fmt.Printf("report: %s\n", report.Paths.MarkdownPath)
	return nil
}

func runPresetDraftMarkTested(args []string) error {
	fs := flag.NewFlagSet("preset draft mark-tested", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft mark-tested [--path DIR] [--json] <preset-id>")
	}
	result, err := presets.MarkDraftTested(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("marked preset draft %s as tested\n", result.PresetID)
	fmt.Printf("previous status: %s\n", result.PreviousStatus)
	fmt.Printf("new status: %s\n", result.NewStatus)
	fmt.Printf("simulation report: %s\n", result.Simulation.Paths.MarkdownPath)
	return nil
}

func runPresetDraftExport(args []string) error {
	fs := flag.NewFlagSet("preset draft export", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft export [--path DIR] [--json] <preset-id>")
	}
	result, err := presets.ExportDraft(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("exported preset draft %s\n", result.PresetID)
	fmt.Printf("status: %s\n", result.Status)
	for _, bundle := range result.Bundles {
		fmt.Printf("- %s -> %s\n", bundle.Provider, bundle.Paths.Root)
	}
	return nil
}

func runPresetDraftPublish(args []string) error {
	fs := flag.NewFlagSet("preset draft publish", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft publish [--path DIR] [--json] <preset-id>")
	}
	result, err := presets.PublishDraft(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("published preset draft %s\n", result.PresetID)
	fmt.Printf("previous status: %s\n", result.PreviousStatus)
	fmt.Printf("new status: %s\n", result.NewStatus)
	fmt.Printf("publish report: %s\n", result.Paths.MarkdownPath)
	return nil
}

func runPresetDraftInstall(args []string) error {
	fs := flag.NewFlagSet("preset draft install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	forceFlag := fs.Bool("force", false, "allow overwrite after previewing collisions")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft install [--path DIR] [--force] [--json] <preset-id>")
	}
	result, err := presets.InstallPublishedDraft(*pathFlag, fs.Args()[0], *forceFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("installed published preset draft %s via %s\n", result.PresetID, result.SelectedProvider)
	fmt.Printf("bundle: %s\n", result.BundleID)
	fmt.Printf("install report: %s\n", result.Install.Report)
	return nil
}

func runPresetDraftCatalogSync(args []string) error {
	fs := flag.NewFlagSet("preset draft catalog-sync", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	rootFlag := fs.String("root", "", "target preset catalog root")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft catalog-sync [--path DIR] --root DIR [--json] <preset-id>")
	}
	result, err := presets.SyncDraftToCatalog(*pathFlag, fs.Args()[0], *rootFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("synced published preset draft %s into catalog %s\n", result.PresetID, result.CatalogRoot)
	fmt.Printf("catalog sync report: %s\n", result.Paths.MarkdownPath)
	for _, bundle := range result.Bundles {
		fmt.Printf("- %s -> %s\n", bundle.BundleID, bundle.TargetRoot)
	}
	return nil
}

func runPresetDraftInterview(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("preset draft interview requires a subcommand")
	}
	switch args[0] {
	case "start":
		return runPresetDraftInterviewStart(args[1:])
	case "answer":
		return runPresetDraftInterviewAnswer(args[1:])
	case "remediate":
		return runPresetDraftInterviewRemediate(args[1:])
	case "show":
		return runPresetDraftInterviewShow(args[1:])
	default:
		return fmt.Errorf("unknown preset draft interview subcommand %q", args[0])
	}
}

func runPresetList(args []string) error {
	fs := flag.NewFlagSet("preset list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rootFlag := fs.String("root", "presets/official", "preset catalog root")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	cards, err := app.NewService().ListPresetCards(*rootFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(cards)
	}
	for _, card := range cards {
		fmt.Printf("%s %s %s %s\n", card.ID, card.Adapter, card.Category, card.Name)
	}
	return nil
}

func runPresetValidate(args []string) error {
	fs := flag.NewFlagSet("preset validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rootFlag := fs.String("root", "presets/official", "preset catalog root")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset validate [--root DIR] [--json] <preset-id>")
	}
	manifest, err := presets.LoadByID(*rootFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(manifest)
	}
	fmt.Printf("preset %s is valid\n", manifest.ID)
	return nil
}

func runPresetPreview(args []string) error {
	fs := flag.NewFlagSet("preset preview", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rootFlag := fs.String("root", "presets/official", "preset catalog root")
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset preview [--root DIR] [--path DIR] [--json] <preset-id>")
	}
	svc := app.NewService()
	preview, err := svc.PreviewPresetInstall(*pathFlag, *rootFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(preview)
	}
	fmt.Printf("preset: %s\n", preview.Manifest.Name)
	fmt.Printf("target: %s\n", preview.Target)
	fmt.Printf("conflicts: %t\n", preview.HasConflicts)
	fmt.Printf("effective runtime ceiling: %s\n", preview.Resolution.EffectiveRuntimeCeiling)
	fmt.Printf("effective budget profile: %s\n", preview.Resolution.EffectiveBudgetProfile)
	if len(preview.Resolution.Layers) > 0 {
		fmt.Println("layers:")
		for _, layer := range preview.Resolution.Layers {
			label := fmt.Sprintf("%d. %s", layer.Order, layer.Kind)
			if layer.PresetID != "" {
				label += " (" + layer.PresetID + ")"
			}
			fmt.Printf("- %s: %s\n", label, layer.Name)
		}
	}
	if preview.HasEnvironmentConflicts {
		fmt.Println("environment conflicts:")
		for _, conflict := range preview.EnvironmentConflicts {
			fmt.Printf("- %s\n", conflict)
		}
	}
	for _, op := range preview.Operations {
		fmt.Printf("- %s %s\n", op.Action, op.TargetPath)
	}
	return nil
}

func runPresetInstall(args []string) error {
	fs := flag.NewFlagSet("preset install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rootFlag := fs.String("root", "presets/official", "preset catalog root")
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	forceFlag := fs.Bool("force", false, "allow overwrite after previewing collisions")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset install [--root DIR] [--path DIR] [--force] [--json] <preset-id>")
	}
	svc := app.NewService()
	result, err := svc.InstallPreset(*pathFlag, *rootFlag, fs.Args()[0], *forceFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("installed preset %s as %s\n", result.Record.PresetID, result.Record.InstallID)
	fmt.Printf("report: %s\n", result.Report)
	if result.Record.EnvironmentReportPath != "" {
		fmt.Printf("environment report: %s\n", result.Record.EnvironmentReportPath)
	}
	if result.Record.EnvironmentJSONPath != "" {
		fmt.Printf("environment json: %s\n", result.Record.EnvironmentJSONPath)
	}
	return nil
}

func runPresetRollback(args []string) error {
	fs := flag.NewFlagSet("preset rollback", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset rollback [--path DIR] [--json] <install-id>")
	}
	svc := app.NewService()
	record, err := svc.RollbackPreset(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(record)
	}
	fmt.Printf("rolled back preset install %s\n", record.InstallID)
	return nil
}

func runPresetDraftInit(args []string) error {
	fs := flag.NewFlagSet("preset draft init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	idFlag := fs.String("id", "", "draft preset id")
	nameFlag := fs.String("name", "", "draft preset name")
	summaryFlag := fs.String("summary", "", "draft preset summary")
	goalFlag := fs.String("goal", "", "draft preset goal")
	typeFlag := fs.String("type", "domain", "preset type")
	targetAgentFlag := fs.String("target-agent", "codex", "target agent")
	providersFlag := fs.String("providers", "codex", "comma-separated compatible providers")
	versionFlag := fs.String("version", "0.1.0", "draft preset version")
	budgetFlag := fs.String("budget-profile", "balanced", "budget profile")
	autonomyFlag := fs.String("autonomy", "low", "autonomy level")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	draft, err := presets.InitDraft(presets.DraftInitOptions{
		WorkspaceRoot: *pathFlag,
		ID:            *idFlag,
		Name:          *nameFlag,
		Summary:       *summaryFlag,
		Goal:          *goalFlag,
		PresetType:    *typeFlag,
		TargetAgent:   *targetAgentFlag,
		Providers:     []string{*providersFlag},
		Version:       *versionFlag,
		BudgetProfile: *budgetFlag,
		AutonomyLevel: *autonomyFlag,
	})
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(draft)
	}
	fmt.Printf("created preset draft %s\n", draft.Profile.ID)
	fmt.Printf("profile: %s\n", draft.Paths.ProfilePath)
	fmt.Printf("brief: %s\n", draft.Paths.BriefMDPath)
	fmt.Printf("evaluation: %s\n", draft.Paths.EvaluationMD)
	return nil
}

func runPresetDraftShow(args []string) error {
	fs := flag.NewFlagSet("preset draft show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft show [--path DIR] [--json] <preset-id>")
	}
	draft, err := presets.LoadDraft(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(draft)
	}
	fmt.Printf("preset draft: %s\n", draft.Profile.Name)
	fmt.Printf("id: %s\n", draft.Profile.ID)
	fmt.Printf("status: %s\n", draft.Profile.Status)
	fmt.Printf("target agent: %s\n", draft.Profile.TargetAgent)
	fmt.Printf("providers: %s\n", strings.Join(draft.Profile.Environment.CompatibleProviders, ", "))
	fmt.Printf("workflow: %s\n", strings.Join(draft.Profile.Workflow, " -> "))
	return nil
}

func runPresetDraftInterviewStart(args []string) error {
	fs := flag.NewFlagSet("preset draft interview start", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	idFlag := fs.String("id", "", "draft preset id")
	modeFlag := fs.String("mode", "quick", "interview mode: quick|deep|import-refine")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	session, err := presets.StartInterview(presets.StartInterviewOptions{
		WorkspaceRoot: *pathFlag,
		DraftID:       *idFlag,
		Mode:          *modeFlag,
	})
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(session)
	}
	fmt.Printf("started preset interview %s for %s\n", session.ID, session.DraftID)
	if session.CurrentQuestion != nil {
		fmt.Printf("next: %s\n", session.CurrentQuestion.Prompt)
	}
	fmt.Printf("simulation ready: %t\n", session.Readiness.SimulationReady)
	fmt.Printf("save ready: %t\n", session.Readiness.SaveReady)
	return nil
}

func runPresetDraftInterviewAnswer(args []string) error {
	fs := flag.NewFlagSet("preset draft interview answer", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	sessionFlag := fs.String("session", "", "interview session id")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft interview answer [--path DIR] --session ID [--json] <answer>")
	}
	session, err := presets.AnswerInterview(presets.AnswerInterviewOptions{
		WorkspaceRoot: *pathFlag,
		SessionID:     *sessionFlag,
		Answer:        fs.Args()[0],
	})
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(session)
	}
	fmt.Printf("recorded answer for %s (%d/%d)\n", session.ID, session.AnsweredCount, session.QuestionCount)
	if session.CurrentQuestion != nil {
		fmt.Printf("next: %s\n", session.CurrentQuestion.Prompt)
	} else {
		fmt.Println("interview complete")
	}
	if len(session.Contradictions) > 0 {
		fmt.Println("contradictions:")
		for _, item := range session.Contradictions {
			fmt.Printf("- %s\n", item)
		}
	}
	if len(session.SuggestedFixes) > 0 {
		fmt.Println("suggested fixes:")
		for _, fix := range session.SuggestedFixes {
			mode := "manual"
			if fix.AutoApplicable {
				mode = "auto"
			}
			fmt.Printf("- [%s] %s\n", mode, fix.Title)
		}
	}
	fmt.Printf("simulation ready: %t\n", session.Readiness.SimulationReady)
	fmt.Printf("save ready: %t\n", session.Readiness.SaveReady)
	return nil
}

func runPresetDraftInterviewShow(args []string) error {
	fs := flag.NewFlagSet("preset draft interview show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft interview show [--path DIR] [--json] <session-id>")
	}
	session, err := presets.LoadInterview(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(session)
	}
	fmt.Printf("preset interview %s for %s (%d/%d, confidence %d%%)\n", session.ID, session.DraftID, session.AnsweredCount, session.QuestionCount, session.Confidence)
	if session.CurrentQuestion != nil {
		fmt.Printf("next: %s\n", session.CurrentQuestion.Prompt)
	}
	if len(session.Contradictions) > 0 {
		fmt.Println("contradictions:")
		for _, item := range session.Contradictions {
			fmt.Printf("- %s\n", item)
		}
	}
	if len(session.SuggestedFixes) > 0 {
		fmt.Println("suggested fixes:")
		for _, fix := range session.SuggestedFixes {
			mode := "manual"
			if fix.AutoApplicable {
				mode = "auto"
			}
			fmt.Printf("- [%s] %s\n", mode, fix.Title)
		}
	}
	fmt.Printf("simulation ready: %t\n", session.Readiness.SimulationReady)
	fmt.Printf("save ready: %t\n", session.Readiness.SaveReady)
	return nil
}

func runPresetDraftInterviewRemediate(args []string) error {
	fs := flag.NewFlagSet("preset draft interview remediate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft interview remediate [--path DIR] [--json] <session-id>")
	}
	result, err := presets.RemediateInterview(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("remediated interview %s\n", result.Session.ID)
	if len(result.AppliedFixes) == 0 {
		fmt.Println("no auto-applicable fixes")
	} else {
		for _, fix := range result.AppliedFixes {
			fmt.Printf("- applied: %s\n", fix.Title)
		}
	}
	fmt.Printf("simulation ready: %t\n", result.Session.Readiness.SimulationReady)
	fmt.Printf("save ready: %t\n", result.Session.Readiness.SaveReady)
	return nil
}

func runPresetDraftUpdate(args []string) error {
	fs := flag.NewFlagSet("preset draft update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	idFlag := fs.String("id", "", "draft preset id")
	nameFlag := fs.String("name", "", "draft preset name")
	summaryFlag := fs.String("summary", "", "draft preset summary")
	goalFlag := fs.String("goal", "", "draft preset goal")
	typeFlag := fs.String("type", "", "preset type")
	targetAgentFlag := fs.String("target-agent", "", "target agent")
	providersFlag := fs.String("providers", "", "comma-separated compatible providers")
	versionFlag := fs.String("version", "", "draft preset version")
	budgetFlag := fs.String("budget-profile", "", "budget profile")
	autonomyFlag := fs.String("autonomy", "", "autonomy level")
	nonGoalsFlag := fs.String("non-goals", "", "comma-separated non-goals")
	inputsFlag := fs.String("inputs", "", "comma-separated inputs")
	outputsFlag := fs.String("outputs", "", "comma-separated outputs")
	workflowFlag := fs.String("workflow", "", "comma-separated workflow steps")
	qualityGatesFlag := fs.String("quality-gates", "", "comma-separated quality gates")
	statusFlag := fs.String("status", "", "draft status")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	if strings.TrimSpace(*idFlag) == "" {
		return fmt.Errorf("usage: arc preset draft update --path DIR --id PRESET-ID [--summary TEXT] [--goal TEXT] [--providers a,b] [--outputs a,b] [--workflow a,b] [--quality-gates a,b] [--json]")
	}
	draft, err := presets.UpdateDraft(presets.DraftUpdateOptions{
		WorkspaceRoot: *pathFlag,
		ID:            *idFlag,
		Name:          *nameFlag,
		Summary:       *summaryFlag,
		Goal:          *goalFlag,
		PresetType:    *typeFlag,
		TargetAgent:   *targetAgentFlag,
		Providers:     []string{*providersFlag},
		Version:       *versionFlag,
		BudgetProfile: *budgetFlag,
		AutonomyLevel: *autonomyFlag,
		NonGoals:      []string{*nonGoalsFlag},
		Inputs:        []string{*inputsFlag},
		Outputs:       []string{*outputsFlag},
		Workflow:      []string{*workflowFlag},
		QualityGates:  []string{*qualityGatesFlag},
		Status:        *statusFlag,
	})
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(draft)
	}
	fmt.Printf("updated preset draft %s\n", draft.Profile.ID)
	fmt.Printf("summary: %s\n", draft.Profile.Summary)
	fmt.Printf("goal: %s\n", draft.Profile.Goal)
	fmt.Printf("workflow: %s\n", strings.Join(draft.Profile.Workflow, " -> "))
	return nil
}

func runPresetDraftValidate(args []string) error {
	fs := flag.NewFlagSet("preset draft validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: arc preset draft validate [--path DIR] [--json] <preset-id>")
	}
	draft, err := presets.LoadDraft(*pathFlag, fs.Args()[0])
	if err != nil {
		return err
	}
	if err := presets.ValidateDraft(draft); err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(map[string]any{
			"id":            draft.Profile.ID,
			"status":        "valid",
			"quality_gates": draft.Profile.QualityGates,
			"paths":         draft.Paths,
		})
	}
	fmt.Printf("preset draft %s is valid\n", draft.Profile.ID)
	return nil
}

func runPresetDraftList(args []string) error {
	fs := flag.NewFlagSet("preset draft list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	items, err := presets.ListDrafts(*pathFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(items)
	}
	for _, item := range items {
		fmt.Printf("%s %s %s %s\n", item.ID, item.TargetAgent, item.PresetType, item.Name)
	}
	return nil
}
