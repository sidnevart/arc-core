package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/indexer"
	"agent-os/internal/mode"
	"agent-os/internal/project"
	"agent-os/internal/provider"
)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	pathFlag := fs.String("path", ".", "project path")
	providerFlag := fs.String("provider", "codex,claude", "enabled providers: comma-separated values")
	modeFlag := fs.String("mode", "work", "default mode: study, work, hero")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireNoArgs(fs.Args()); err != nil {
		return err
	}
	if err := requireMode(*modeFlag); err != nil {
		return err
	}
	root, err := filepath.Abs(*pathFlag)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	enabledProviders := splitProviderList(*providerFlag)
	defaultProvider := defaultProvider(enabledProviders)

	if _, err := project.EnsureHome(); err != nil {
		return err
	}

	created, err := project.Init(root, project.InitOptions{
		Provider:         defaultProvider,
		EnabledProviders: enabledProviders,
		Mode:             *modeFlag,
	})
	if err != nil {
		return err
	}

	for _, name := range enabledProviders {
		if adapter, err := provider.Get(name); err == nil {
			_ = adapter.ApplyProjectScaffold(root)
		}
	}

	if idx, err := indexer.Build(root); err == nil {
		_ = indexer.WriteIndividual(root, idx)
		_ = indexer.Save(root, idx)
	}

	event := project.Event{
		Timestamp: time.Now().UTC(),
		Command:   "init",
		Status:    "ok",
		Details: map[string]string{
			"provider":          defaultProvider,
			"enabled_providers": strings.Join(enabledProviders, ","),
			"mode":              *modeFlag,
			"path":              root,
		},
	}
	if err := project.AppendEvent(root, event); err != nil {
		return err
	}

	fmt.Printf("initialized arc project scaffold in %s\n", root)
	fmt.Printf("default provider: %s\n", defaultProvider)
	fmt.Printf("mode: %s (%s autonomy)\n", *modeFlag, mode.ByName(*modeFlag).Autonomy)
	if len(created) > 0 {
		fmt.Println()
		fmt.Println("created:")
		for _, item := range created {
			fmt.Printf("  - %s\n", strings.TrimPrefix(item, root+string(filepath.Separator)))
		}
	}

	return nil
}

func splitProviderList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part != "codex" && part != "claude" {
			continue
		}
		if !seen[part] {
			seen[part] = true
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{"codex"}
	}
	return out
}

func defaultProvider(enabled []string) string {
	for _, name := range enabled {
		if name == "codex" {
			return name
		}
	}
	return enabled[0]
}
