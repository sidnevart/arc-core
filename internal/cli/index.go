package cli

import (
	"flag"
	"fmt"
	"io"

	"agent-os/internal/indexer"
)

func runIndex(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("index requires a subcommand")
	}
	switch args[0] {
	case "build", "refresh":
		return runIndexBuild(args[1:])
	default:
		return fmt.Errorf("unknown index subcommand %q", args[0])
	}
}

func runIndexBuild(args []string) error {
	fs := flag.NewFlagSet("index build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
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
	idx, err := indexer.Build(root)
	if err != nil {
		return err
	}
	if err := indexer.WriteIndividual(root, idx); err != nil {
		return err
	}
	if err := indexer.Save(root, idx); err != nil {
		return err
	}

	fmt.Printf("index built for %s\n", root)
	fmt.Printf("files=%d symbols=%d dependencies=%d docs=%d\n", len(idx.Files), len(idx.Symbols), len(idx.Dependencies), len(idx.Docs))
	return nil
}
