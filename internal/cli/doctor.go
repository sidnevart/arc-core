package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"agent-os/internal/app"
	"agent-os/internal/project"
	"agent-os/internal/provider"
)

type toolCheck struct {
	Name        string
	Required    bool
	Description string
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}

	checks := []toolCheck{
		{Name: "git", Required: true, Description: "version control"},
		{Name: "rg", Required: true, Description: "fast code search"},
		{Name: "ast-grep", Required: true, Description: "AST search"},
		{Name: "codex", Required: true, Description: "Codex runtime"},
		{Name: "claude", Required: true, Description: "Claude runtime"},
	}

	if *jsonFlag {
		home, err := project.EnsureHome()
		if err != nil {
			return err
		}
		projectRoot, _ := project.DiscoverRoot(".")
		payload := map[string]any{
			"home":           home,
			"project_root":   projectRoot,
			"providers":      app.NewService().ProviderHealth(context.Background()),
			"required_tools": checks,
		}
		return writeJSON(payload)
	}

	fmt.Println("arc doctor")
	fmt.Println()

	home, err := project.EnsureHome()
	if err != nil {
		return err
	}
	fmt.Println(project.HomeSummary(home))
	if root, err := project.DiscoverRoot("."); err == nil {
		fmt.Println("project root:", root)
	}
	fmt.Println()

	missingRequired := false
	for _, check := range checks {
		path, err := exec.LookPath(check.Name)
		status := "OK"
		if err != nil {
			status = "MISSING"
			if check.Required {
				missingRequired = true
			}
		}

		if path == "" {
			path = "-"
		}

		requirement := "required"
		if !check.Required {
			requirement = "optional"
		}

		fmt.Printf("%-10s %-8s %-8s %s\n", check.Name, status, requirement, check.Description)
		fmt.Printf("  path: %s\n", path)
	}

	fmt.Println()
	fmt.Println("providers")
	for _, name := range []string{"codex", "claude"} {
		adapter, _ := provider.Get(name)
		status := adapter.CheckInstalled(context.Background())
		fmt.Printf("%-10s installed=%t", status.Name, status.Installed)
		if status.BinaryPath != "" {
			fmt.Printf(" path=%s", status.BinaryPath)
		}
		fmt.Println()
		if len(status.Capabilities) > 0 {
			fmt.Printf("  capabilities: %s\n", strings.Join(status.Capabilities, ", "))
		}
		for _, note := range status.Notes {
			fmt.Printf("  note: %s\n", note)
		}
	}

	fmt.Println()
	fmt.Println("remediation")
	fmt.Println("- install `ast-grep` to unlock AST-aware search in index and context collection")
	fmt.Println("- install `claude` to enable the second provider adapter and live non-interactive Claude runs")
	fmt.Println("- run `arc init` in a repository to create `.arc/` project files and `~/.arc/` global defaults")

	if missingRequired {
		return fmt.Errorf("required tools are missing")
	}

	fmt.Println()
	fmt.Println("doctor check passed")
	return nil
}
