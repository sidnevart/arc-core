package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"agent-os/internal/orchestrator"
	"agent-os/internal/project"
)

func runLearn(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("learn requires a topic or subcommand")
	}
	switch args[0] {
	case "quiz":
		return runLearnQuiz(args[1:])
	case "prove":
		return runLearnProve(args[1:])
	default:
		return runLearnTopic(args)
	}
}

func runLearnTopic(args []string) error {
	root, err := resolveProjectRoot(".")
	if err != nil {
		return err
	}
	opts := orchestrator.TaskOptions{
		Root:        root,
		Task:        strings.Join(args, " "),
		Mode:        "study",
		Provider:    "codex",
		DryRun:      true,
		UseProvider: false,
	}
	run, err := orchestrator.Plan(root, opts)
	if err != nil {
		return err
	}
	fmt.Printf("study plan created: %s\n", run.ID)
	return nil
}

func runLearnQuiz(args []string) error {
	fs := flag.NewFlagSet("learn quiz", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	runs, err := orchestrator.ListRuns(root)
	if err != nil {
		return err
	}
	latestStudy := ""
	for _, run := range runs {
		if run.Mode == "study" {
			latestStudy = run.ID
			break
		}
	}
	if latestStudy == "" {
		return fmt.Errorf("no study-mode run found")
	}
	path := filepath.Join(project.ProjectFile(root, "runs"), latestStudy, "practice_tasks.md")
	content := "# Practice Tasks\n\n- Explain the main concept in your own words.\n- Solve one analogous mini-problem.\n- List one thing that is still unclear.\n"
	if err := project.WriteString(path, content); err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func runLearnProve(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("learn prove requires a claim")
	}
	root, err := resolveProjectRoot(".")
	if err != nil {
		return err
	}
	opts := orchestrator.TaskOptions{
		Root:        root,
		Task:        "Prove or justify: " + strings.Join(args, " "),
		Mode:        "study",
		Provider:    "codex",
		DryRun:      true,
		UseProvider: false,
	}
	run, err := orchestrator.Plan(root, opts)
	if err != nil {
		return err
	}
	fmt.Printf("study proof plan created: %s\n", run.ID)
	return nil
}
