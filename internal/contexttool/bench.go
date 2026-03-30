package contexttool

import (
	"crypto/sha256"
	"encoding/hex"
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
	Reuse         ReuseSummary     `json:"reuse"`
	SummaryPath   string           `json:"summary_path"`
	BaselineMD    string           `json:"baseline_md_path"`
	OptimizedMD   string           `json:"optimized_md_path"`
	BaselineJSON  string           `json:"baseline_json_path"`
	OptimizedJSON string           `json:"optimized_json_path"`
}

type BenchSummary struct {
	Task                              string   `json:"task"`
	GeneratedAt                       string   `json:"generated_at"`
	MatchedTerms                      []string `json:"matched_terms"`
	BaselineApproxTokens              int      `json:"baseline_approx_tokens"`
	OptimizedApproxTokens             int      `json:"optimized_approx_tokens"`
	BaselineQualityScore              int      `json:"baseline_quality_score"`
	OptimizedQualityScore             int      `json:"optimized_quality_score"`
	TokenReduction                    int      `json:"token_reduction"`
	TokenReductionPercent             int      `json:"token_reduction_percent"`
	BaselineSectionCount              int      `json:"baseline_section_count"`
	OptimizedSectionCount             int      `json:"optimized_section_count"`
	BaselineMemoryMatches             int      `json:"baseline_memory_matches"`
	OptimizedMemoryMatches            int      `json:"optimized_memory_matches"`
	OptimizedMemoryTrustBonus         int      `json:"optimized_memory_trust_bonus"`
	OptimizedMemoryRecencyBonus       int      `json:"optimized_memory_recency_bonus"`
	BaselineSourceDiversity           int      `json:"baseline_source_diversity"`
	OptimizedSourceDiversity          int      `json:"optimized_source_diversity"`
	OptimizedDiversityBonus           int      `json:"optimized_diversity_bonus"`
	OptimizedDocFamilyDiversity       int      `json:"optimized_doc_family_diversity"`
	OptimizedCodeFamilyDiversity      int      `json:"optimized_code_family_diversity"`
	OptimizedDocClusterDiversity      int      `json:"optimized_doc_cluster_diversity"`
	OptimizedCodeClusterDiversity     int      `json:"optimized_code_cluster_diversity"`
	OptimizedDocDominantClusterShare  int      `json:"optimized_doc_dominant_cluster_share"`
	OptimizedCodeDominantClusterShare int      `json:"optimized_code_dominant_cluster_share"`
	BaselineCandidateTotal            int      `json:"baseline_candidate_total"`
	BaselineSelectedTotal             int      `json:"baseline_selected_total"`
	OptimizedCandidateTotal           int      `json:"optimized_candidate_total"`
	OptimizedSelectedTotal            int      `json:"optimized_selected_total"`
	ReuseIndexSource                  string   `json:"reuse_index_source"`
	ReuseIndexFingerprint             string   `json:"reuse_index_fingerprint,omitempty"`
	ReuseMemorySource                 string   `json:"reuse_memory_source"`
	ReuseMemoryFingerprint            string   `json:"reuse_memory_fingerprint,omitempty"`
	ReuseArtifactCount                int      `json:"reuse_artifact_count"`
	Recommendation                    string   `json:"recommendation"`
}

func Bench(root string, task string) (BenchResult, error) {
	idx, items, builtIndex, reuse, err := ensureIndexAndMemory(root)
	if err != nil {
		return BenchResult{}, err
	}
	terms := queryTerms(task)
	baseline, baselineSummary := buildBaselinePack(task, idx, items, terms)
	optimized, optimizedSummary := buildPack(task, idx, items, terms)

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
		Task:                              task,
		GeneratedAt:                       time.Now().UTC().Format(time.RFC3339),
		MatchedTerms:                      terms,
		BaselineApproxTokens:              baseline.ApproxTokens,
		OptimizedApproxTokens:             optimized.ApproxTokens,
		BaselineQualityScore:              baselineSummary.QualityScore,
		OptimizedQualityScore:             optimizedSummary.QualityScore,
		TokenReduction:                    reduction,
		TokenReductionPercent:             reductionPercent,
		BaselineSectionCount:              len(baseline.Sections),
		OptimizedSectionCount:             len(optimized.Sections),
		BaselineMemoryMatches:             baselineSummary.MemoryMatchCount,
		OptimizedMemoryMatches:            optimizedSummary.MemoryMatchCount,
		OptimizedMemoryTrustBonus:         optimizedSummary.MemoryTrustBonus,
		OptimizedMemoryRecencyBonus:       optimizedSummary.MemoryRecencyBonus,
		BaselineSourceDiversity:           baselineSummary.SourceDiversity,
		OptimizedSourceDiversity:          optimizedSummary.SourceDiversity,
		OptimizedDiversityBonus:           optimizedSummary.DiversityBonus,
		OptimizedDocFamilyDiversity:       optimizedSummary.DocFamilyDiversity,
		OptimizedCodeFamilyDiversity:      optimizedSummary.CodeFamilyDiversity,
		OptimizedDocClusterDiversity:      optimizedSummary.DocClusterDiversity,
		OptimizedCodeClusterDiversity:     optimizedSummary.CodeClusterDiversity,
		OptimizedDocDominantClusterShare:  optimizedSummary.DocDominantClusterShare,
		OptimizedCodeDominantClusterShare: optimizedSummary.CodeDominantClusterShare,
		BaselineCandidateTotal:            baselineSummary.Accounting.CandidateTotal,
		BaselineSelectedTotal:             baselineSummary.Accounting.SelectedTotal,
		OptimizedCandidateTotal:           optimizedSummary.Accounting.CandidateTotal,
		OptimizedSelectedTotal:            optimizedSummary.Accounting.SelectedTotal,
		ReuseIndexSource:                  reuse.IndexSource,
		ReuseIndexFingerprint:             reuse.IndexFingerprint,
		ReuseMemorySource:                 reuse.MemorySource,
		ReuseMemoryFingerprint:            reuse.MemoryFingerprint,
		ReuseArtifactCount:                reuse.ReusedArtifactCount,
		Recommendation:                    benchRecommendation(reduction, reductionPercent),
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
		Reuse:         reuse,
		SummaryPath:   summaryPath,
		BaselineMD:    baselineMD,
		OptimizedMD:   optimizedMD,
		BaselineJSON:  baselineJSON,
		OptimizedJSON: optimizedJSON,
	}, nil
}

func ensureIndexAndMemory(root string) (indexer.Result, []memory.Item, bool, ReuseSummary, error) {
	if _, err := Init(root); err != nil {
		return indexer.Result{}, nil, false, ReuseSummary{}, err
	}
	reuse := ReuseSummary{
		IndexBundlePath:   WorkspaceFile(root, "index", "bundle.json"),
		MemoryEntriesPath: WorkspaceFile(root, "memory", "entries.json"),
	}
	idx, err := LoadIndex(root)
	builtIndex := false
	if err != nil {
		if !os.IsNotExist(err) {
			return indexer.Result{}, nil, false, ReuseSummary{}, err
		}
		idx, err = BuildIndex(root)
		if err != nil {
			return indexer.Result{}, nil, false, ReuseSummary{}, err
		}
		builtIndex = true
		reuse.IndexSource = "rebuilt"
		reuse.IndexReused = false
	} else {
		reuse.IndexSource = "reused_existing"
		reuse.IndexReused = true
		reuse.ReusedArtifactCount++
	}
	if reuse.IndexBundlePath != "" {
		reuse.IndexFingerprint = fingerprintFile(reuse.IndexBundlePath)
	}
	items, err := loadMemory(root)
	if err != nil {
		return indexer.Result{}, nil, false, ReuseSummary{}, err
	}
	reuse.MemoryEntriesCount = len(items)
	if len(items) > 0 {
		reuse.MemorySource = "reused_existing"
		reuse.ReusedArtifactCount++
	} else {
		reuse.MemorySource = "empty_workspace"
	}
	if reuse.MemoryEntriesPath != "" {
		reuse.MemoryFingerprint = fingerprintFile(reuse.MemoryEntriesPath)
	}
	return idx, items, builtIndex, reuse, nil
}

func fingerprintFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8])
}

func buildBaselinePack(task string, idx indexer.Result, items []memory.Item, terms []string) (contextpack.Pack, retrievalSummary) {
	sections := []contextpack.Section{}
	provenance := []SectionProvenance{}
	accounting := RetrievalAccounting{}
	add := func(title string, source string, rendered renderedSection, maxChars int) {
		content := strings.TrimSpace(rendered.Content)
		if content == "" {
			return
		}
		truncated := false
		if len(content) > maxChars {
			content = strings.TrimSpace(content[:maxChars]) + "\n...[truncated]"
			truncated = true
		}
		sections = append(sections, contextpack.Section{
			Title:        title,
			Source:       source,
			Content:      content,
			ApproxTokens: len(content) / 4,
		})
		provenance = append(provenance, SectionProvenance{
			Title:          title,
			Source:         source,
			SourcePaths:    rendered.SourcePaths,
			CandidateCount: rendered.CandidateCount,
			SelectedCount:  rendered.SelectedCount,
			Truncated:      truncated,
			Notes:          rendered.Notes,
		})
		accounting.CandidateTotal += rendered.CandidateCount
		accounting.SelectedTotal += rendered.SelectedCount
	}

	add("Task Brief", "task", renderedSection{Content: task, CandidateCount: 1, SelectedCount: 1, Notes: []string{"user task input"}}, 1500*4)
	allDocs := renderAllDocs(idx)
	accounting.CandidateDocs = allDocs.CandidateCount
	accounting.SelectedDocs = allDocs.SelectedCount
	add("All Docs Snapshot", ".context/index/docs.json", allDocs, 5000*4)
	allCode := renderAllCode(idx)
	accounting.CandidateFiles = countNoteValue(allCode.Notes, "candidate_files")
	accounting.SelectedFiles = countNoteValue(allCode.Notes, "selected_files")
	accounting.CandidateSymbols = countNoteValue(allCode.Notes, "candidate_symbols")
	accounting.SelectedSymbols = countNoteValue(allCode.Notes, "selected_symbols")
	add("All Code Snapshot", ".context/index/{files,symbols}.json", allCode, 5000*4)
	changes := renderRecentChanges(idx)
	accounting.CandidateChanges = changes.CandidateCount
	accounting.SelectedChanges = changes.SelectedCount
	add("Recent Changes", ".context/index/recent_changes.json", changes, 1200*4)
	add("Dependencies Snapshot", ".context/index/dependencies.json", renderDependencies(idx), 1600*4)
	add("Memory Snapshot", ".context/memory/entries.json", renderedSection{Content: renderMemorySummary(items), CandidateCount: len(items), SelectedCount: min(10, len(items)), SourcePaths: []string{".context/memory/entries.json"}}, 1200*4)
	add("Index Summary", ".context/index/bundle.json", renderedSection{Content: renderIndexSummary(idx), CandidateCount: 1, SelectedCount: 1, SourcePaths: []string{".context/index/bundle.json"}}, 2500*4)

	total := 0
	for _, section := range sections {
		total += section.ApproxTokens
	}
	pack := contextpack.Pack{
		Task:         task,
		Mode:         "context-tool-baseline",
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		ApproxTokens: total,
		Sections:     sections,
	}
	return pack, summarizePack(pack, terms, nil, provenance, accounting)
}

func renderAllDocs(idx indexer.Result) renderedSection {
	if len(idx.Docs) == 0 {
		return renderedSection{Content: "No docs found in the current index.", SourcePaths: []string{".context/index/docs.json"}}
	}
	docs := append([]indexer.DocEntry(nil), idx.Docs...)
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	var b strings.Builder
	limit := min(40, len(docs))
	sourcePaths := make([]string, 0, limit)
	selected := 0
	for i := 0; i < limit; i++ {
		doc := docs[i]
		if shouldIgnorePath(doc.Path) {
			continue
		}
		selected++
		sourcePaths = append(sourcePaths, doc.Path)
		b.WriteString(fmt.Sprintf("- %s — %s\n", doc.Path, doc.Title))
		for _, heading := range doc.Headings {
			if strings.TrimSpace(heading) == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  heading: %s\n", heading))
			break
		}
	}
	return renderedSection{
		Content:        strings.TrimSpace(b.String()),
		CandidateCount: len(docs),
		SelectedCount:  selected,
		SourcePaths:    sourcePaths,
	}
}

func renderAllCode(idx indexer.Result) renderedSection {
	var b strings.Builder
	files := append([]indexer.FileEntry(nil), idx.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	b.WriteString("Files:\n")
	fileCount := 0
	sourcePaths := []string{}
	for _, file := range files {
		if shouldIgnorePath(file.Path) {
			continue
		}
		sourcePaths = append(sourcePaths, file.Path)
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
		sourcePaths = append(sourcePaths, fmt.Sprintf("%s:%s", symbol.Path, symbol.Name))
		b.WriteString(fmt.Sprintf("- %s (%s) in %s:%d\n", symbol.Name, symbol.Kind, symbol.Path, symbol.Line))
		symbolCount++
		if symbolCount >= 40 {
			break
		}
	}
	return renderedSection{
		Content:        strings.TrimSpace(b.String()),
		CandidateCount: len(files) + len(symbols),
		SelectedCount:  fileCount + symbolCount,
		SourcePaths:    sourcePaths,
		Notes: []string{
			fmt.Sprintf("candidate_files=%d", len(files)),
			fmt.Sprintf("selected_files=%d", fileCount),
			fmt.Sprintf("candidate_symbols=%d", len(symbols)),
			fmt.Sprintf("selected_symbols=%d", symbolCount),
		},
	}
}

func renderDependencies(idx indexer.Result) renderedSection {
	if len(idx.Dependencies) == 0 {
		return renderedSection{Content: "No dependencies found in the current index.", SourcePaths: []string{".context/index/dependencies.json"}}
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
	return renderedSection{
		Content:        strings.TrimSpace(b.String()),
		CandidateCount: len(deps),
		SelectedCount:  limit,
		SourcePaths:    []string{".context/index/dependencies.json"},
	}
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
