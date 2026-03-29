package cli

import (
	"flag"
	"fmt"
	"io"

	"agent-os/internal/app"
)

func runWorkspace(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("workspace requires a subcommand")
	}
	switch args[0] {
	case "summary":
		return runWorkspaceSummary(args[1:])
	case "repair":
		return runWorkspaceRepair(args[1:])
	case "verify":
		return runWorkspaceVerify(args[1:])
	default:
		return fmt.Errorf("unknown workspace subcommand %q", args[0])
	}
}

func runWorkspaceSummary(args []string) error {
	fs := flag.NewFlagSet("workspace summary", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}

	svc := app.NewService()
	summary, err := svc.WorkspaceSummary(*pathFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(summary)
	}

	fmt.Printf("workspace: %s\n", summary.Name)
	fmt.Printf("root: %s\n", summary.Root)
	fmt.Printf("mode: %s (%s)\n", summary.Mode, summary.Autonomy)
	fmt.Printf("default_provider: %s\n", summary.DefaultProvider)
	fmt.Printf("providers: %v\n", summary.EnabledProviders)
	fmt.Printf("index: files=%d symbols=%d docs=%d\n", summary.Index.Files, summary.Index.Symbols, summary.Index.Docs)
	fmt.Printf("memory: total=%d\n", summary.Memory.Total)
	if summary.LastRun != nil {
		fmt.Printf("last_run: %s %s\n", summary.LastRun.ID, summary.LastRun.Status)
	}
	return nil
}

func runWorkspaceVerify(args []string) error {
	fs := flag.NewFlagSet("workspace verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	profileFlag := fs.String("profile", "release-readiness", "verification profile id")
	verifierFlag := fs.String("verifier", "", "single verifier id")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}

	svc := app.NewService()
	var (
		run app.VerificationRun
		err error
	)
	if *verifierFlag != "" {
		run, err = svc.RunVerifier(*pathFlag, *verifierFlag)
	} else {
		run, err = svc.StartVerificationProfile(*pathFlag, *profileFlag)
	}
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(run)
	}

	fmt.Printf("verification: %s\n", run.Title)
	fmt.Printf("status: %s\n", run.Status)
	fmt.Printf("overall: %s\n", run.OverallVerdict)
	fmt.Printf("blocking_failures: %d\n", run.BlockingFailures)
	fmt.Printf("warnings: %d\n", run.WarningCount)
	if run.SummaryPath != "" {
		fmt.Printf("summary: %s\n", run.SummaryPath)
	}
	for _, result := range run.Results {
		fmt.Printf("- %s: %s (%s)\n", result.VerifierID, result.Verdict, result.Summary)
		if result.ReportPath != "" {
			fmt.Printf("  report: %s\n", result.ReportPath)
		}
	}
	return nil
}

func runWorkspaceRepair(args []string) error {
	fs := flag.NewFlagSet("workspace repair", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}

	svc := app.NewService()
	summary, err := svc.RepairWorkspace(*pathFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(summary)
	}
	fmt.Printf("repaired workspace: %s\n", summary.Root)
	fmt.Printf("mode: %s (%s)\n", summary.Mode, summary.Autonomy)
	fmt.Printf("default_provider: %s\n", summary.DefaultProvider)
	return nil
}
