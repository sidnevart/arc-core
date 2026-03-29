package cli

import (
	"fmt"
	"path/filepath"

	"agent-os/internal/memory"
	"agent-os/internal/project"
)

func runQuestions(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("questions requires a subcommand")
	}
	switch args[0] {
	case "show":
		root, err := parsePathOnly("questions show", args[1:])
		if err != nil {
			return err
		}
		items, err := memory.Load(root)
		if err != nil {
			return err
		}
		questions := memory.UnknownQuestions(items)
		seen := map[string]bool{}
		for _, item := range questions {
			if seen[item.Summary] {
				continue
			}
			seen[item.Summary] = true
			fmt.Printf("- %s: %s\n", item.ID, item.Summary)
		}
		if latest, err := project.LatestRunDir(root); err == nil {
			path := filepath.Join(latest, "question_bundle.md")
			if content, err := project.ReadString(path); err == nil {
				fmt.Println()
				fmt.Println(content)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown questions subcommand %q", args[0])
	}
}
