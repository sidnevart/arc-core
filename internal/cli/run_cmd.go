package cli

import (
	"flag"
	"fmt"
	"io"

	"agent-os/internal/app"
	"agent-os/internal/orchestrator"
)

func runRun(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("run requires a subcommand")
	}
	switch args[0] {
	case "list":
		return runList(args[1:])
	case "status":
		return runStatus(args[1:])
	case "resume":
		return runResume(args[1:])
	default:
		return fmt.Errorf("unknown run subcommand %q", args[0])
	}
}

func runList(args []string) error {
	fs := flag.NewFlagSet("run list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	limitFlag := fs.Int("limit", 20, "max runs")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	svc := app.NewService()
	runs, err := svc.ListRuns(*pathFlag, *limitFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(runs)
	}
	for _, run := range runs {
		fmt.Printf("%s %s %s %s\n", run.ID, run.Status, run.Provider, run.Task)
	}
	return nil
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("run status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	runIDFlag := fs.String("run-id", "", "run id")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	run, err := orchestrator.LoadRun(root, *runIDFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		svc := app.NewService()
		detail, err := svc.RunDetail(root, *runIDFlag)
		if err != nil {
			return err
		}
		return writeJSON(detail)
	}
	fmt.Printf("run_id: %s\nstatus: %s\nstate: %s\nprovider: %s\nmode: %s\n", run.ID, run.Status, run.CurrentState, run.Provider, run.Mode)
	if run.ProviderSessionID != "" {
		fmt.Printf("provider_session_id: %s\n", run.ProviderSessionID)
	}
	for key, value := range run.Metadata {
		fmt.Printf("meta.%s=%s\n", key, value)
	}
	for name, path := range run.Artifacts {
		fmt.Printf("artifact.%s=%s\n", name, path)
	}
	return nil
}

func runResume(args []string) error {
	fs := flag.NewFlagSet("run resume", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	runIDFlag := fs.String("run-id", "", "run id")
	modelFlag := fs.String("model", "", "model override")
	dryRunFlag := fs.Bool("dry-run", false, "skip provider resume")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	prompt := ""
	if len(fs.Args()) > 0 {
		prompt = fs.Args()[0]
	}
	run, err := orchestrator.Resume(root, *runIDFlag, prompt, *modelFlag, *dryRunFlag)
	if err != nil {
		return err
	}
	fmt.Printf("resumed run %s status=%s\n", run.ID, run.Status)
	return nil
}
