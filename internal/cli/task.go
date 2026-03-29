package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"agent-os/internal/orchestrator"
	"agent-os/internal/project"
)

func runTask(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task requires a subcommand")
	}
	switch args[0] {
	case "plan":
		return runTaskPlan(args[1:])
	case "run":
		return runTaskRun(args[1:])
	case "verify":
		return runTaskVerify(args[1:])
	case "review":
		return runTaskReview(args[1:])
	default:
		return fmt.Errorf("unknown task subcommand %q", args[0])
	}
}

func runTaskPlan(args []string) error {
	opts, err := parseTaskOptions("task plan", args)
	if err != nil {
		return err
	}
	run, err := orchestrator.Plan(opts.Root, opts)
	if err != nil {
		return err
	}
	fmt.Printf("planned run %s\n", run.ID)
	fmt.Printf("artifacts: %d\n", len(run.Artifacts))
	return nil
}

func runTaskRun(args []string) error {
	opts, err := parseTaskOptions("task run", args)
	if err != nil {
		return err
	}
	run, err := orchestrator.RunTask(opts.Root, opts)
	if err != nil {
		return err
	}
	fmt.Printf("run %s finished with status=%s state=%s\n", run.ID, run.Status, run.CurrentState)
	return nil
}

func runTaskVerify(args []string) error {
	fs := flag.NewFlagSet("task verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	runIDFlag := fs.String("run-id", "", "run id")
	runChecksFlag := fs.Bool("run-checks", false, "execute detected checks")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	report, err := orchestrator.Verify(root, orchestrator.VerifyOptions{Root: root, RunID: *runIDFlag, RunChecks: *runChecksFlag})
	if err != nil {
		return err
	}
	fmt.Printf("verification report: %s\n", report["report"])
	return nil
}

func runTaskReview(args []string) error {
	fs := flag.NewFlagSet("task review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	runIDFlag := fs.String("run-id", "", "run id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	report, err := orchestrator.Review(root, *runIDFlag)
	if err != nil {
		return err
	}
	fmt.Printf("review report: %s\n", report["report"])
	return nil
}

func parseTaskOptions(name string, args []string) (orchestrator.TaskOptions, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	modeFlag := fs.String("mode", "", "mode override")
	providerFlag := fs.String("provider", "", "provider override")
	modelFlag := fs.String("model", "", "model override")
	dryRunFlag := fs.Bool("dry-run", false, "skip provider execution")
	runChecksFlag := fs.Bool("run-checks", false, "execute detected verification checks")
	noProviderFlag := fs.Bool("no-provider", false, "skip provider execution even if installed")
	approveRiskyFlag := fs.Bool("approve-risky", false, "allow provider execution for tasks that trigger approval gates")
	providerTimeoutFlag := fs.Duration("provider-timeout", 2*time.Minute, "timeout for provider execution")
	if err := fs.Parse(args); err != nil {
		return orchestrator.TaskOptions{}, err
	}
	if len(fs.Args()) == 0 {
		return orchestrator.TaskOptions{}, fmt.Errorf("%s requires a task description", name)
	}

	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return orchestrator.TaskOptions{}, err
	}
	proj, err := project.Load(root)
	if err != nil {
		return orchestrator.TaskOptions{}, err
	}

	modeValue := proj.Mode.Mode
	if *modeFlag != "" {
		modeValue = *modeFlag
	}
	providerValue := proj.Config.DefaultProvider
	if *providerFlag != "" {
		providerValue = *providerFlag
	}

	return orchestrator.TaskOptions{
		Root:            root,
		Task:            strings.Join(fs.Args(), " "),
		Mode:            modeValue,
		Provider:        providerValue,
		Model:           *modelFlag,
		DryRun:          *dryRunFlag,
		RunChecks:       *runChecksFlag,
		UseProvider:     !*noProviderFlag,
		ApproveRisky:    *approveRiskyFlag,
		ProviderTimeout: *providerTimeoutFlag,
	}, nil
}
