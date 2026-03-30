package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"agent-os/internal/memory"
	"agent-os/internal/presets"
)

func runHook(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("hook requires a subcommand")
	}
	switch args[0] {
	case "memory":
		return runHookMemory(args[1:])
	default:
		return fmt.Errorf("unknown hook subcommand %q", args[0])
	}
}

func runHookMemory(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("hook memory requires a subcommand")
	}
	switch args[0] {
	case "add":
		return runHookMemoryAdd(args[1:])
	default:
		return fmt.Errorf("unknown hook memory subcommand %q", args[0])
	}
}

func runHookMemoryAdd(args []string) error {
	fs := flag.NewFlagSet("hook memory add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", fallbackEnv("ARC_WORKSPACE_ROOT", "."), "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	idFlag := fs.String("id", "", "memory id")
	scopeFlag := fs.String("scope", "", "memory scope")
	kindFlag := fs.String("kind", "note", "memory kind")
	sourceFlag := fs.String("source", "hook", "memory source")
	confidenceFlag := fs.String("confidence", "medium", "memory confidence")
	statusFlag := fs.String("status", "active", "memory status")
	tagsFlag := fs.String("tags", "", "comma-separated tags")
	runIDFlag := fs.String("run-id", fallbackEnv("ARC_RUN_ID", ""), "run id")
	hookNameFlag := fs.String("hook-name", fallbackEnv("ARC_HOOK_NAME", ""), "hook name")
	lifecycleFlag := fs.String("hook-lifecycle", fallbackEnv("ARC_HOOK_LIFECYCLE", ""), "hook lifecycle")
	ownerPresetFlag := fs.String("owner-preset", fallbackEnv("ARC_HOOK_OWNER_PRESET", ""), "owner preset id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*scopeFlag) == "" {
		return fmt.Errorf("hook memory add requires --scope")
	}
	if len(fs.Args()) == 0 {
		return fmt.Errorf("hook memory add requires a summary")
	}
	root, err := resolveProjectRoot(*pathFlag)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*runIDFlag) == "" {
		return fmt.Errorf("hook memory add requires ARC_RUN_ID or --run-id")
	}
	allowedScopes := splitCSV(fallbackEnv("ARC_ALLOWED_MEMORY_SCOPES", ""))
	if len(allowedScopes) == 0 {
		return fmt.Errorf("hook memory add requires ARC_ALLOWED_MEMORY_SCOPES")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := memory.Item{
		ID:             strings.TrimSpace(*idFlag),
		Scope:          strings.TrimSpace(*scopeFlag),
		Kind:           strings.TrimSpace(*kindFlag),
		Source:         strings.TrimSpace(*sourceFlag),
		Confidence:     strings.TrimSpace(*confidenceFlag),
		Status:         strings.TrimSpace(*statusFlag),
		Tags:           splitCSV(*tagsFlag),
		Summary:        strings.Join(fs.Args(), " "),
		CreatedAt:      now,
		LastVerifiedAt: now,
	}
	event, err := presets.AddHookMemory(root, item, *runIDFlag, *hookNameFlag, *lifecycleFlag, *ownerPresetFlag, allowedScopes)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(event)
	}
	fmt.Printf("hook memory added %s\n", event.Item.ID)
	fmt.Printf("scope=%s run=%s\n", event.Item.Scope, event.RunID)
	return nil
}

func fallbackEnv(key string, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
