package cli

import (
	"flag"
	"fmt"
	"io"

	"agent-os/internal/orchestrator"
)

func runDocs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("docs requires a subcommand")
	}
	switch args[0] {
	case "generate":
		return runDocsGenerate(args[1:])
	default:
		return fmt.Errorf("unknown docs subcommand %q", args[0])
	}
}

func runDocsGenerate(args []string) error {
	fs := flag.NewFlagSet("docs generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	runIDFlag := fs.String("run-id", "", "run id")
	applyFlag := fs.Bool("apply", false, "apply docs updates to project maps")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	result, err := orchestrator.GenerateDocsWithApply(root, *runIDFlag, *applyFlag)
	if err != nil {
		return err
	}
	fmt.Printf("docs delta: %s\n", result["report"])
	if applied := result["applied"]; applied != "" {
		fmt.Printf("applied: %s\n", applied)
	}
	return nil
}
