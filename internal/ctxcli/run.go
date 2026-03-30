package ctxcli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agent-os/internal/contexttool"
	"agent-os/internal/memory"
)

func Run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "index":
		return runIndex(args[1:])
	case "assemble":
		return runAssemble(args[1:])
	case "bench":
		return runBench(args[1:])
	case "memory":
		return runMemory(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown ctx command %q", args[0])
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("ctx init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	root, err := filepath.Abs(*pathFlag)
	if err != nil {
		return err
	}
	created, err := contexttool.Init(root)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(map[string]any{
			"root":          root,
			"workspace_dir": contexttool.WorkspaceDir(root),
			"created":       created,
		})
	}
	fmt.Printf("ctx workspace ready at %s\n", contexttool.WorkspaceDir(root))
	fmt.Printf("created=%d\n", len(created))
	return nil
}

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
	fs := flag.NewFlagSet("ctx index build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	idx, err := contexttool.BuildIndex(root)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(map[string]any{
			"root":          root,
			"workspace_dir": contexttool.WorkspaceDir(root),
			"files":         len(idx.Files),
			"symbols":       len(idx.Symbols),
			"dependencies":  len(idx.Dependencies),
			"docs":          len(idx.Docs),
		})
	}
	fmt.Printf("ctx index built for %s\n", root)
	fmt.Printf("files=%d symbols=%d dependencies=%d docs=%d\n", len(idx.Files), len(idx.Symbols), len(idx.Dependencies), len(idx.Docs))
	return nil
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("ctx doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	report, err := contexttool.Doctor(*pathFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(report)
	}
	fmt.Printf("ctx doctor for %s\n", report.Root)
	fmt.Printf("workspace=%s schema=%s\n", report.WorkspaceDir, report.SchemaVersion)
	for _, check := range report.Checks {
		fmt.Printf("- [%s] %s", check.Status, check.Name)
		if check.Details != "" {
			fmt.Printf(": %s", check.Details)
		}
		fmt.Println()
	}
	if len(report.Warnings) > 0 {
		fmt.Println("warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("- %s\n", warning)
		}
	}
	return nil
}

func runAssemble(args []string) error {
	fs := flag.NewFlagSet("ctx assemble", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return fmt.Errorf("ctx assemble requires a task description")
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	result, err := contexttool.Assemble(root, joinArgs(fs.Args()))
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("ctx assemble created %s\n", result.OutputDir)
	fmt.Printf("sections=%d approx_tokens=%d built_index=%t\n", len(result.Pack.Sections), result.Pack.ApproxTokens, result.BuiltIndex)
	fmt.Printf("pack_md=%s\n", result.PackMDPath)
	return nil
}

func runBench(args []string) error {
	fs := flag.NewFlagSet("ctx bench", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return fmt.Errorf("ctx bench requires a task description")
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	result, err := contexttool.Bench(root, joinArgs(fs.Args()))
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(result)
	}
	fmt.Printf("ctx bench created %s\n", result.OutputDir)
	fmt.Printf("baseline_tokens=%d optimized_tokens=%d reduction=%d (%d%%)\n",
		result.Summary.BaselineApproxTokens,
		result.Summary.OptimizedApproxTokens,
		result.Summary.TokenReduction,
		result.Summary.TokenReductionPercent,
	)
	fmt.Printf("summary=%s\n", result.SummaryPath)
	return nil
}

func runMemory(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("memory requires a subcommand")
	}
	switch args[0] {
	case "add":
		return runMemoryAdd(args[1:])
	case "list":
		return runMemoryList(args[1:])
	case "search":
		return runMemorySearch(args[1:])
	case "status":
		return runMemoryStatus(args[1:])
	case "compact":
		return runMemoryCompact(args[1:])
	default:
		return fmt.Errorf("unknown memory subcommand %q", args[0])
	}
}

func runMemoryAdd(args []string) error {
	fs := flag.NewFlagSet("ctx memory add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	idFlag := fs.String("id", "", "memory id")
	scopeFlag := fs.String("scope", "project", "memory scope")
	kindFlag := fs.String("kind", "note", "memory kind")
	sourceFlag := fs.String("source", "human", "memory source")
	confidenceFlag := fs.String("confidence", "medium", "memory confidence")
	statusFlag := fs.String("status", "active", "memory status")
	tagsFlag := fs.String("tags", "", "comma-separated tags")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return fmt.Errorf("ctx memory add requires a summary")
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	item, err := contexttool.AddMemory(root, memory.Item{
		ID:         *idFlag,
		Scope:      *scopeFlag,
		Kind:       *kindFlag,
		Source:     *sourceFlag,
		Confidence: *confidenceFlag,
		Status:     *statusFlag,
		Tags:       splitCSV(*tagsFlag),
		Summary:    joinArgs(fs.Args()),
	})
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(item)
	}
	fmt.Printf("ctx memory added %s\n", item.ID)
	fmt.Printf("kind=%s scope=%s tags=%d\n", item.Kind, item.Scope, len(item.Tags))
	return nil
}

func runMemoryList(args []string) error {
	fs := flag.NewFlagSet("ctx memory list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	items, err := contexttool.ListMemory(root)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(items)
	}
	for _, item := range items {
		fmt.Printf("%s [%s/%s] %s\n", item.ID, item.Kind, item.Status, item.Summary)
	}
	fmt.Printf("total=%d\n", len(items))
	return nil
}

func runMemorySearch(args []string) error {
	fs := flag.NewFlagSet("ctx memory search", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	limitFlag := fs.Int("limit", 10, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return fmt.Errorf("ctx memory search requires a query")
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	items, err := contexttool.SearchMemory(root, joinArgs(fs.Args()), *limitFlag)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(items)
	}
	for _, item := range items {
		fmt.Printf("%s [%s/%s] %s\n", item.ID, item.Kind, item.Status, item.Summary)
	}
	fmt.Printf("total=%d\n", len(items))
	return nil
}

func runMemoryStatus(args []string) error {
	fs := flag.NewFlagSet("ctx memory status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	report, err := contexttool.MemoryStatus(root)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(report)
	}
	fmt.Printf("total=%d\n", report.Summary.Total)
	fmt.Printf("entries=%s\n", report.EntriesPath)
	fmt.Printf("active=%s\n", report.ActivePath)
	fmt.Printf("archive=%s\n", report.ArchivePath)
	fmt.Printf("questions=%s\n", report.QuestionsPath)
	if len(report.Summary.ByKind) > 0 {
		fmt.Println("by_kind:")
		for _, key := range sortedKeys(report.Summary.ByKind) {
			fmt.Printf("- %s=%d\n", key, report.Summary.ByKind[key])
		}
	}
	if len(report.Summary.ByStatus) > 0 {
		fmt.Println("by_status:")
		for _, key := range sortedKeys(report.Summary.ByStatus) {
			fmt.Printf("- %s=%d\n", key, report.Summary.ByStatus[key])
		}
	}
	return nil
}

func runMemoryCompact(args []string) error {
	fs := flag.NewFlagSet("ctx memory compact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pathFlag := fs.String("path", ".", "project path")
	jsonFlag := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("unexpected arguments: %v", fs.Args())
	}
	root, err := contexttool.ResolveRoot(*pathFlag)
	if err != nil {
		return err
	}
	report, err := contexttool.CompactMemory(root)
	if err != nil {
		return err
	}
	if *jsonFlag {
		return writeJSON(report)
	}
	fmt.Printf("ctx memory compact updated %s\n", report.EntriesPath)
	fmt.Printf("total=%d\n", report.Summary.Total)
	for _, key := range sortedKeys(report.Summary.ByStatus) {
		fmt.Printf("- %s=%d\n", key, report.Summary.ByStatus[key])
	}
	return nil
}

func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	out := args[0]
	for _, arg := range args[1:] {
		out += " " + arg
	}
	return out
}

func writeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printUsage() {
	fmt.Println("ctx - Standalone Context Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ctx init [--path DIR] [--json]")
	fmt.Println("  ctx doctor [--path DIR] [--json]")
	fmt.Println("  ctx index build [--path DIR] [--json]")
	fmt.Println("  ctx index refresh [--path DIR] [--json]")
	fmt.Println("  ctx memory add [--path DIR] [--kind note] [--tags a,b] [--json] <summary>")
	fmt.Println("  ctx memory list [--path DIR] [--json]")
	fmt.Println("  ctx memory search [--path DIR] [--limit N] [--json] <query>")
	fmt.Println("  ctx memory status [--path DIR] [--json]")
	fmt.Println("  ctx memory compact [--path DIR] [--json]")
	fmt.Println("  ctx assemble [--path DIR] [--json] <task>")
	fmt.Println("  ctx bench [--path DIR] [--json] <task>")
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

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
