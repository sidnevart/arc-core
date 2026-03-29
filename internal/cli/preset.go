package cli

import (
	"flag"
	"fmt"
	"io"

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
