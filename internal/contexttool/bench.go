package contexttool

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-os/internal/contextpack"
	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/project"
)

type BenchResult struct {
	OutputDir     string           `json:"output_dir"`
	Baseline      contextpack.Pack `json:"baseline"`
	Optimized     contextpack.Pack `json:"optimized"`
	Summary       BenchSummary     `json:"summary"`
	BuiltIndex    bool             `json:"built_index"`
	MatchedTerms  []string         `json:"matched_terms"`
	SummaryPath   string           `json:"summary_path"`
	BaselineMD    string           `json:"baseline_md_path"`
	OptimizedMD   string           `json:"optimized_md_path"`
	BaselineJSON  string           `json:"baseline_json_path"`
	OptimizedJSON string           `json:"optimized_json_path"`
}

type BenchSummary struct {
	Task                        string   `json:"task"`
	GeneratedAt                 string   `json:"generated_at"`
	MatchedTerms                []string `json:"matched_terms"`
	BaselineApproxTokens        int      `json:"baseline_approx_tokens"`
	OptimizedApproxTokens       int      `json:"optimized_approx_tokens"`
	BaselineQualityScore        int      `json:"baseline_quality_score"`
	OptimizedQualityScore       int      `json:"optimized_quality_score"`
	TokenReduction              int      `json:"token_reduction"`
	TokenReductionPercent       int      `json:"token_reduction_percent"`
	BaselineSectionCount        int      `json:"baseline_section_count"`
	OptimizedSectionCount       int      `json:"optimized_section_count"`
	BaselineMemoryMatches       int      `json:"baseline_memory_matches"`
	OptimizedMemoryMatches      int      `json:"optimized_memory_matches"`
	OptimizedMemoryTrustBonus   int      `json:"optimized_memory_trust_bonus"`
	OptimizedMemoryRecencyBonus int      `json:"optimized_memory_recency_bonus"`
	Recommendation              string   `json:"recommendation"`
}

func Bench(root string, task string) (BenchResult, error) {
	idx, items, builtIndex, err := ensureIndexAndMemory(root)
	if err != nil {
		return BenchResult{}, err
	}
	terms := queryTerms(task)
	baseline := buildBaselinePack(task, idx, items)
	optimized, optimizedSummary := buildPack(task, idx, items, terms)
	baselineSummary := summarizePack(baseline, terms, nil)

	runID := time.Now().UTC().Format("20060102T150405Z")
	outputDir := WorkspaceFile(root, "benchmarks", runID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return BenchResult{}, err
	}

	baselineJSON := filepath.Join(outputDir, "baseline_pack.json")
	baselineMD := filepath.Join(outputDir, "baseline_pack.md")
	optimizedJSON := filepath.Join(outputDir, "optimized_pack.json")
	optimizedMD := filepath.Join(outputDir, "optimized_pack.md")
	summaryPath := filepath.Join(outputDir, "summary.json")

	if err := project.WriteJSON(baselineJSON, baseline); err != nil {
		return BenchResult{}, err
	}
	if err := project.WriteString(baselineMD, contextpack.Markdown(baseline)); err != nil {
		return BenchResult{}, err
	}
	if err := project.WriteJSON(optimizedJSON, optimized); err != nil {
		return BenchResult{}, err
	}
	if err := project.WriteString(optimizedMD, contextpack.Markdown(optimized)); err != nil {
		return BenchResult{}, err
	}

	reduction := baseline.ApproxTokens - optimized.ApproxTokens
	reductionPercent := 0
	if baseline.ApproxTokens > 0 {
		reductionPercent = reduction * 100 / baseline.ApproxTokens
	}
	summary := BenchSummary{
		Task:                        task,
		GeneratedAt:                 time.Now().UTC().Format(time.RFC3339),
		MatchedTerms:                terms,
		BaselineApproxTokens:        baseline.ApproxTokens,
		OptimizedApproxTokens:       optimized.ApproxTokens,
		BaselineQualityScore:        baselineSummary.QualityScore,
		OptimizedQualityScore:       optimizedSummary.QualityScore,
		TokenReduction:              reduction,
		TokenReductionPercent:       reductionPercent,
		BaselineSectionCount:        len(baseline.Sections),
		OptimizedSectionCount:       len(optimized.Sections),
		BaselineMemoryMatches:       baselineSummary.MemoryMatchCount,
		OptimizedMemoryMatches:      optimizedSummary.MemoryMatchCount,
		OptimizedMemoryTrustBonus:   optimizedSummary.MemoryTrustBonus,
		OptimizedMemoryRecencyBonus: optimizedSummary.MemoryRecencyBonus,
		Recommendation:              benchRecommendation(reduction, reductionPercent),
	}
	if err := project.WriteJSON(summaryPath, summary); err != nil {
		return BenchResult{}, err
	}

	return BenchResult{
		OutputDir:     outputDir,
		Baseline:      baseline,
		Optimized:     optimized,
		Summary:       summary,
		BuiltIndex:    builtIndex,
		MatchedTerms:  terms,
		SummaryPath:   summaryPath,
		BaselineMD:    baselineMD,
		OptimizedMD:   optimizedMD,
		BaselineJSON:  baselineJSON,
		OptimizedJSON: optimizedJSON,
	}, nil
}

func ensureIndexAndMemory(root string) (indexer.Result, []memory.Item, bool, error) {
	if _, err := Init(root); err != nil {
		return indexer.Result{}, nil, false, err
	}
	idx, err := LoadIndex(root)
	builtIndex := false
	if err != nil {
		if !os.IsNotExist(err) {
			return indexer.Result{}, nil, false, err
		}
		idx, err = BuildIndex(root)
		if err != nil {
			return indexer.Result{}, nil, false, err
		}
		builtIndex = true
	}
	items, err := loadMemory(root)
	if err != nil {
		return indexer.Result{}, nil, false, err
	}
	return idx, items, builtIndex, nil
}

func buildBaselinePack(task string, idx indexer.Result, items []memory.Item) contextpack.Pack {
	sections := []contextpack.Section{}
	add := func(title string, source string, content string, maxChars int) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		if len(content) > maxChars {
			content = strings.TrimSpace(content[:maxChars]) + "\n...[truncated]"
		}
		sections = append(sections, contextpack.Section{
			Title:        title,
			Source:       source,
			Content:      content,
			ApproxTokens: len(content) / 4,
		})
	}

	add("Task Brief", "task", task, 1500*4)
	add("All Docs Snapshot", ".context/index/docs.json", renderAllDocs(idx), 5000*4)
	add("All Code Snapshot", ".context/index/{files,symbols}.json", renderAllCode(idx), 5000*4)
	add("Recent Changes", ".context/index/recent_changes.json", renderRecentChanges(idx), 1200*4)
	add("Dependencies Snapshot", ".context/index/dependencies.json", renderDependencies(idx), 1600*4)
	add("Memory Snapshot", ".context/memory/entries.json", renderMemorySummary(items), 1200*4)
	add("Index Summary", ".context/index/bundle.json", renderIndexSummary(idx), 2500*4)

	total := 0
	for _, section := range sections {
		total += section.ApproxTokens
	}
	return contextpack.Pack{
		Task:         task,
		Mode:         "context-tool-baseline",
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		ApproxTokens: total,
		Sections:     sections,
	}
}

func renderAllDocs(idx indexer.Result) string {
	if len(idx.Docs) == 0 {
		return "No docs found in the current index."
	}
	docs := append([]indexer.DocEntry(nil), idx.Docs...)
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	var b strings.Builder
	limit := min(40, len(docs))
	for i := 0; i < limit; i++ {
		doc := docs[i]
		if shouldIgnorePath(doc.Path) {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s — %s\n", doc.Path, doc.Title))
		for _, heading := range doc.Headings {
			if strings.TrimSpace(heading) == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  heading: %s\n", heading))
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func renderAllCode(idx indexer.Result) string {
	var b strings.Builder
	files := append([]indexer.FileEntry(nil), idx.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	b.WriteString("Files:\n")
	fileCount := 0
	for _, file := range files {
		if shouldIgnorePath(file.Path) {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, file.Kind))
		fileCount++
		if fileCount >= 40 {
			break
		}
	}
	symbols := append([]indexer.SymbolEntry(nil), idx.Symbols...)
	sort.Slice(symbols, func(i, j int) bool {
		return symbols[i].Path+symbols[i].Name < symbols[j].Path+symbols[j].Name
	})
	b.WriteString("\nSymbols:\n")
	symbolCount := 0
	for _, symbol := range symbols {
		if shouldIgnorePath(symbol.Path) {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s (%s) in %s:%d\n", symbol.Name, symbol.Kind, symbol.Path, symbol.Line))
		symbolCount++
		if symbolCount >= 40 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func renderDependencies(idx indexer.Result) string {
	if len(idx.Dependencies) == 0 {
		return "No dependencies found in the current index."
	}
	deps := append([]indexer.DependencyEntry(nil), idx.Dependencies...)
	sort.Slice(deps, func(i, j int) bool {
		if deps[i].Ecosystem == deps[j].Ecosystem {
			return deps[i].Name < deps[j].Name
		}
		return deps[i].Ecosystem < deps[j].Ecosystem
	})
	var b strings.Builder
	limit := min(40, len(deps))
	for i := 0; i < limit; i++ {
		dep := deps[i]
		b.WriteString(fmt.Sprintf("- [%s] %s %s\n", dep.Ecosystem, dep.Name, dep.Version))
	}
	return strings.TrimSpace(b.String())
}

func benchRecommendation(reduction int, reductionPercent int) string {
	switch {
	case reduction <= 0:
		return "Current optimized assembly is not smaller than baseline yet; improve retrieval ranking before switching orchestrator integration."
	case reductionPercent >= 30:
		return "Optimized assembly is meaningfully smaller than baseline; safe to deepen retrieval and start planning orchestrator integration."
	default:
		return "Optimized assembly is smaller, but the gain is still modest; improve retrieval precision before relying on it as the main orchestrator path."
	}
}
