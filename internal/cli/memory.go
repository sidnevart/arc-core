package cli

import (
	"flag"
	"fmt"
	"io"

	"agent-os/internal/memory"
)

func runMemory(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("memory requires a subcommand")
	}
	switch args[0] {
	case "status":
		return runMemoryStatus(args[1:])
	case "compact":
		return runMemoryCompact(args[1:])
	default:
		return fmt.Errorf("unknown memory subcommand %q", args[0])
	}
}

func runMemoryStatus(args []string) error {
	fs := flag.NewFlagSet("memory status", flag.ContinueOnError)
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
	items, err := memory.Load(root)
	if err != nil {
		return err
	}
	status := memory.Status(items)
	if *jsonFlag {
		return writeJSON(status)
	}
	fmt.Printf("memory total=%d\n", status.Total)
	for key, value := range status.ByStatus {
		fmt.Printf("status.%s=%d\n", key, value)
	}
	for key, value := range status.ByKind {
		fmt.Printf("kind.%s=%d\n", key, value)
	}
	return nil
}

func runMemoryCompact(args []string) error {
	root, err := parsePathOnly("memory compact", args)
	if err != nil {
		return err
	}
	status, err := memory.Compact(root)
	if err != nil {
		return err
	}
	fmt.Printf("memory compacted total=%d\n", status.Total)
	return nil
}

func parsePathOnly(name string, args []string) (string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return "", err
	}
	return resolveProjectRoot(*pathFlag)
}
