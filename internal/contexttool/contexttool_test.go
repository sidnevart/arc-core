package contexttool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-os/internal/contextpack"
	"agent-os/internal/indexer"
	"agent-os/internal/memory"
)

func TestInitCreatesWorkspace(t *testing.T) {
	root := t.TempDir()
	created, err := Init(root)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if len(created) == 0 {
		t.Fatalf("Init() created nothing")
	}
	for _, path := range []string{
		HumanConfigPath(root),
		WorkspaceFile(root, "config.json"),
		WorkspaceFile(root, "memory", "entries.json"),
		WorkspaceFile(root, "README.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestLoadHumanConfigReadsManagedConfig(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	content := `project_name: "demo"
include_paths:
  - "internal"
exclude_paths:
  - "vendor"
docs_paths:
  - "docs"
memory_paths:
  - ".context/memory"
language_hints:
  - "go"
  - "markdown"
metrics_enabled: false
`
	if err := os.WriteFile(HumanConfigPath(root), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cfg, err := LoadHumanConfig(root)
	if err != nil {
		t.Fatalf("LoadHumanConfig() error = %v", err)
	}
	if cfg.ProjectName != "demo" {
		t.Fatalf("ProjectName = %q, want demo", cfg.ProjectName)
	}
	if cfg.MetricsEnabled {
		t.Fatalf("expected metrics_enabled=false")
	}
	if len(cfg.IncludePaths) != 1 || cfg.IncludePaths[0] != "internal" {
		t.Fatalf("unexpected include paths: %#v", cfg.IncludePaths)
	}
	if len(cfg.LanguageHints) != 2 {
		t.Fatalf("unexpected language hints: %#v", cfg.LanguageHints)
	}
}

func TestLoadHumanConfigKeepsDefaultMetricsWhenFieldIsOmitted(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	content := `project_name: "demo"
include_paths:
  - "internal"
`
	if err := os.WriteFile(HumanConfigPath(root), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cfg, err := LoadHumanConfig(root)
	if err != nil {
		t.Fatalf("LoadHumanConfig() error = %v", err)
	}
	if !cfg.MetricsEnabled {
		t.Fatalf("expected default metrics_enabled=true when field is omitted")
	}
}

func TestBuildIndexWritesContextArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Demo\n\nContext tool docs.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	idx, err := BuildIndex(root)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}
	if len(idx.Files) == 0 {
		t.Fatalf("expected files to be indexed")
	}
	if _, err := os.Stat(WorkspaceFile(root, "index", "bundle.json")); err != nil {
		t.Fatalf("expected context index bundle: %v", err)
	}
	for _, file := range idx.Files {
		if strings.HasPrefix(file.Path, ".context/") {
			t.Fatalf("index unexpectedly includes .context path %q", file.Path)
		}
	}
}

func TestBuildIndexRespectsHumanConfigPathFilters(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Root\n\nShould be filtered from docs.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "guide.md"), []byte("# Guide\n\nContext docs.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "logic.go"), []byte("package internal\n\nfunc Logic() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	config := `project_name: "demo"
include_paths:
  - "internal"
  - "docs"
docs_paths:
  - "docs"
exclude_paths:
  - ".context"
language_hints:
  - "go"
metrics_enabled: true
`
	if err := os.WriteFile(HumanConfigPath(root), []byte(config), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	idx, err := BuildIndex(root)
	if err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}
	for _, file := range idx.Files {
		if file.Path == "main.go" || file.Path == "README.md" {
			t.Fatalf("unexpected filtered file in index: %s", file.Path)
		}
	}
	if len(idx.Docs) != 1 || idx.Docs[0].Path != "docs/guide.md" {
		t.Fatalf("unexpected docs after config filter: %#v", idx.Docs)
	}
}

func TestAssembleWritesArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Auth\n\nToken budgeting and context assembly.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth.go"), []byte("package auth\n\nfunc BudgetGuard() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	result, err := Assemble(root, "explain token budgeting")
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if len(result.Pack.Sections) == 0 {
		t.Fatalf("expected non-empty pack sections")
	}
	if _, err := os.Stat(result.PackJSONPath); err != nil {
		t.Fatalf("expected pack json: %v", err)
	}
	if _, err := os.Stat(result.PackMDPath); err != nil {
		t.Fatalf("expected pack markdown: %v", err)
	}
	if !result.BuiltIndex {
		t.Fatalf("expected assemble to build missing index")
	}
	if result.QualityScore <= 0 {
		t.Fatalf("expected positive quality score")
	}
	if result.TermCoverage <= 0 {
		t.Fatalf("expected positive term coverage")
	}
	var meta assembleMetadata
	data, err := os.ReadFile(result.MetadataPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(meta.SectionProvenance) == 0 {
		t.Fatalf("expected section provenance metadata")
	}
	if meta.Accounting.CandidateTotal <= 0 || meta.Accounting.SelectedTotal <= 0 {
		t.Fatalf("expected positive accounting totals: %#v", meta.Accounting)
	}
	if meta.Accounting.CandidateDocs < meta.Accounting.SelectedDocs {
		t.Fatalf("candidate docs should be >= selected docs: %#v", meta.Accounting)
	}
	if meta.Reuse.IndexSource == "" || meta.Reuse.MemorySource == "" {
		t.Fatalf("expected reuse summary in metadata: %#v", meta.Reuse)
	}
	if meta.Reuse.IndexFingerprint == "" {
		t.Fatalf("expected index fingerprint in reuse summary: %#v", meta.Reuse)
	}
	if meta.Reuse.MemoryFingerprint == "" {
		t.Fatalf("expected memory fingerprint in reuse summary: %#v", meta.Reuse)
	}
	if meta.SourceDiversity <= 0 || meta.DiversityBonus <= 0 {
		t.Fatalf("expected positive diversity metadata: diversity=%d bonus=%d", meta.SourceDiversity, meta.DiversityBonus)
	}
}

func TestResolveRootDiscoversWorkspaceFromNestedDir(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	nested := filepath.Join(root, "nested", "child")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	resolved, err := ResolveRoot(nested)
	if err != nil {
		t.Fatalf("ResolveRoot() error = %v", err)
	}
	if resolved != root {
		t.Fatalf("ResolveRoot() = %q, want %q", resolved, root)
	}
}

func TestBenchWritesComparisonArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Context\n\nBudgeted context management.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	result, err := Bench(root, "explain context management benchmarking")
	if err != nil {
		t.Fatalf("Bench() error = %v", err)
	}
	if _, err := os.Stat(result.SummaryPath); err != nil {
		t.Fatalf("expected summary artifact: %v", err)
	}
	if _, err := os.Stat(result.BaselineMD); err != nil {
		t.Fatalf("expected baseline markdown artifact: %v", err)
	}
	if _, err := os.Stat(result.OptimizedMD); err != nil {
		t.Fatalf("expected optimized markdown artifact: %v", err)
	}
	if result.Summary.Recommendation == "" {
		t.Fatalf("expected benchmark recommendation to be populated")
	}
	if result.Summary.BaselineApproxTokens == 0 || result.Summary.OptimizedApproxTokens == 0 {
		t.Fatalf("expected non-zero token counts in benchmark summary")
	}
	if result.Summary.BaselineCandidateTotal <= 0 || result.Summary.OptimizedCandidateTotal <= 0 {
		t.Fatalf("expected candidate totals in benchmark summary: %#v", result.Summary)
	}
	if result.Summary.OptimizedCandidateTotal < result.Summary.OptimizedSelectedTotal {
		t.Fatalf("optimized candidate total should be >= selected total: %#v", result.Summary)
	}
	if result.Reuse.IndexSource == "" || result.Summary.ReuseIndexSource == "" {
		t.Fatalf("expected reuse evidence in bench result: %#v / %#v", result.Reuse, result.Summary)
	}
	if result.Reuse.IndexFingerprint == "" || result.Summary.ReuseIndexFingerprint == "" {
		t.Fatalf("expected index fingerprint evidence in bench result: %#v / %#v", result.Reuse, result.Summary)
	}
	if result.Reuse.MemoryFingerprint == "" || result.Summary.ReuseMemoryFingerprint == "" {
		t.Fatalf("expected memory fingerprint evidence in bench result: %#v / %#v", result.Reuse, result.Summary)
	}
}

func TestAddMemoryPersistsWorkspaceEntriesAndMarkdown(t *testing.T) {
	root := t.TempDir()
	item, err := AddMemory(root, memory.Item{
		Kind:    "decision",
		Tags:    []string{"ctx", "budget", "ctx"},
		Summary: "Context memory should bias assembly toward durable facts.",
	})
	if err != nil {
		t.Fatalf("AddMemory() error = %v", err)
	}
	if item.ID == "" {
		t.Fatalf("expected generated memory id")
	}
	items, err := ListMemory(root)
	if err != nil {
		t.Fatalf("ListMemory() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one memory item, got %d", len(items))
	}
	if len(items[0].Tags) != 2 {
		t.Fatalf("expected normalized tags, got %v", items[0].Tags)
	}
	for _, path := range []string{
		WorkspaceFile(root, "memory", "entries.json"),
		WorkspaceFile(root, "memory", "MEMORY_ACTIVE.md"),
		WorkspaceFile(root, "memory", "MEMORY_ARCHIVE.md"),
		WorkspaceFile(root, "memory", "OPEN_QUESTIONS.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected memory artifact %s: %v", path, err)
		}
	}
}

func TestSearchMemoryMatchesSummaryAndTags(t *testing.T) {
	root := t.TempDir()
	if _, err := AddMemory(root, memory.Item{
		ID:      "ctx-budget",
		Kind:    "decision",
		Tags:    []string{"budget", "routing"},
		Summary: "Budget routing should stay local-first only for clearly local tasks.",
	}); err != nil {
		t.Fatalf("AddMemory() error = %v", err)
	}
	if _, err := AddMemory(root, memory.Item{
		ID:      "ctx-preset",
		Kind:    "decision",
		Tags:    []string{"preset", "environment"},
		Summary: "Preset environments should enforce memory scopes.",
	}); err != nil {
		t.Fatalf("AddMemory() error = %v", err)
	}
	results, err := SearchMemory(root, "budget routing", 10)
	if err != nil {
		t.Fatalf("SearchMemory() error = %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected non-empty search results")
	}
	if results[0].ID != "ctx-budget" {
		t.Fatalf("expected ctx-budget first, got %q", results[0].ID)
	}
}

func TestMemoryStatusReportsPathsAndCounts(t *testing.T) {
	root := t.TempDir()
	if _, err := AddMemory(root, memory.Item{Kind: "decision", Summary: "Decision one"}); err != nil {
		t.Fatalf("AddMemory() error = %v", err)
	}
	if _, err := AddMemory(root, memory.Item{Kind: "question", Summary: "Question one"}); err != nil {
		t.Fatalf("AddMemory() error = %v", err)
	}
	report, err := MemoryStatus(root)
	if err != nil {
		t.Fatalf("MemoryStatus() error = %v", err)
	}
	if report.Summary.Total != 2 {
		t.Fatalf("expected total=2, got %d", report.Summary.Total)
	}
	if report.EntriesPath == "" || report.ActivePath == "" || report.ArchivePath == "" || report.QuestionsPath == "" {
		t.Fatalf("expected populated artifact paths: %#v", report)
	}
	if len(report.MostRecent) != 2 {
		t.Fatalf("expected two recent items, got %d", len(report.MostRecent))
	}
}

func TestCompactMemoryMarksOldActiveEntriesStale(t *testing.T) {
	root := t.TempDir()
	old := time.Now().UTC().Add(-45 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := AddMemory(root, memory.Item{
		ID:             "old-memory",
		Kind:           "decision",
		Status:         "active",
		CreatedAt:      old,
		LastVerifiedAt: old,
		Summary:        "Old memory item that should become stale.",
	}); err != nil {
		t.Fatalf("AddMemory() error = %v", err)
	}
	report, err := CompactMemory(root)
	if err != nil {
		t.Fatalf("CompactMemory() error = %v", err)
	}
	if report.Summary.ByStatus["stale"] != 1 {
		t.Fatalf("expected one stale item after compaction, got %#v", report.Summary.ByStatus)
	}
	content, err := os.ReadFile(WorkspaceFile(root, "memory", "MEMORY_ARCHIVE.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "old-memory") {
		t.Fatalf("expected archive markdown to include compacted item")
	}
}

func TestDoctorReportsWorkspaceAndIndexHealth(t *testing.T) {
	root := t.TempDir()
	if _, err := Init(root); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Demo\n\nContext.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := BuildIndex(root); err != nil {
		t.Fatalf("BuildIndex() error = %v", err)
	}
	report, err := Doctor(root)
	if err != nil {
		t.Fatalf("Doctor() error = %v", err)
	}
	if report.SchemaVersion == "" {
		t.Fatalf("expected schema version")
	}
	if report.ConfigPath == "" {
		t.Fatalf("expected config path")
	}
	if report.HumanConfig.ProjectName == "" {
		t.Fatalf("expected human config")
	}
	if len(report.Checks) == 0 {
		t.Fatalf("expected doctor checks")
	}
	foundIndex := false
	for _, check := range report.Checks {
		if check.Name == "index_bundle" && check.Status == "ok" {
			foundIndex = true
			break
		}
	}
	if !foundIndex {
		t.Fatalf("expected successful index_bundle check, got %#v", report.Checks)
	}
}

func TestAssembleIncludesRelevantMemoryInPack(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# Context\n\nContext assembly for preset environments.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := AddMemory(root, memory.Item{
		Kind:    "decision",
		Tags:    []string{"preset", "environment"},
		Summary: "Preset environment rules must stay isolated from domain memory scopes.",
	}); err != nil {
		t.Fatalf("AddMemory() error = %v", err)
	}
	result, err := Assemble(root, "explain preset environment memory scopes")
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	found := false
	for _, section := range result.Pack.Sections {
		if section.Title == "Relevant Memory" && strings.Contains(section.Content, "Preset environment rules must stay isolated") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected relevant memory section to include added memory entry")
	}
	if !contains(result.MatchedSections, "Relevant Memory") {
		t.Fatalf("expected relevant memory section to contribute to matched sections: %v", result.MatchedSections)
	}
	if result.MemoryMatchCount != 1 {
		t.Fatalf("expected one memory match, got %d", result.MemoryMatchCount)
	}
	if result.MemoryBoost <= 0 {
		t.Fatalf("expected positive memory boost")
	}
	if result.MemoryTrustBonus <= 0 {
		t.Fatalf("expected positive memory trust bonus")
	}
	if result.MemoryRecencyBonus <= 0 {
		t.Fatalf("expected positive memory recency bonus")
	}
}

func TestSelectRelevantMemoryPrefersRecentTrustedDecision(t *testing.T) {
	now := time.Now().UTC()
	candidates := selectRelevantMemory([]memory.Item{
		{
			ID:             "old-note",
			Kind:           "note",
			Source:         "human",
			Confidence:     "low",
			Status:         "active",
			CreatedAt:      now.Add(-400 * 24 * time.Hour).Format(time.RFC3339),
			LastVerifiedAt: now.Add(-400 * 24 * time.Hour).Format(time.RFC3339),
			Tags:           []string{"preset", "memory"},
			Summary:        "Old note about preset memory rules.",
		},
		{
			ID:             "fresh-decision",
			Kind:           "decision",
			Source:         "human",
			Confidence:     "high",
			Status:         "active",
			CreatedAt:      now.Add(-2 * 24 * time.Hour).Format(time.RFC3339),
			LastVerifiedAt: now.Add(-2 * 24 * time.Hour).Format(time.RFC3339),
			Tags:           []string{"preset", "memory"},
			Summary:        "Recent decision about preset memory rules.",
		},
	}, []string{"preset", "memory", "rules"})
	if len(candidates) != 2 {
		t.Fatalf("expected two memory candidates, got %d", len(candidates))
	}
	if candidates[0].item.ID != "fresh-decision" {
		t.Fatalf("expected fresh trusted decision first, got %s", candidates[0].item.ID)
	}
	if candidates[0].trustBonus <= candidates[1].trustBonus {
		t.Fatalf("expected trust bonus ordering to favor fresh decision")
	}
	if candidates[0].recencyBonus <= candidates[1].recencyBonus {
		t.Fatalf("expected recency bonus ordering to favor fresh decision")
	}
}

func TestScoreDocPrefersTitleAndHeadingCoverage(t *testing.T) {
	terms := []string{"preset", "environment", "memory"}
	weak := scoreDoc(indexer.DocEntry{
		Path:  "docs/preset-notes.md",
		Title: "Generic notes",
	}, terms)
	strong := scoreDoc(indexer.DocEntry{
		Path:     "docs/runtime.md",
		Title:    "Preset Environment Memory Rules",
		Headings: []string{"Environment memory scopes", "Preset isolation"},
	}, terms)
	if strong <= weak {
		t.Fatalf("expected title+heading coverage score %d to beat weak score %d", strong, weak)
	}
}

func TestRenderRelevantCodePrefersMatchingSymbolName(t *testing.T) {
	idx := indexer.Result{
		Files: []indexer.FileEntry{
			{Path: "internal/presets/runtime.go", Kind: "go"},
			{Path: "internal/random/file.go", Kind: "go"},
		},
		Symbols: []indexer.SymbolEntry{
			{Name: "ApplyPresetEnvironmentRules", Kind: "function", Path: "internal/presets/runtime.go", Line: 12},
			{Name: "HandleStuff", Kind: "function", Path: "internal/random/file.go", Line: 9},
		},
	}
	rendered := renderRelevantCode(idx, []string{"preset", "environment", "rules"})
	if !strings.Contains(rendered.Content, "ApplyPresetEnvironmentRules") {
		t.Fatalf("expected matching symbol to appear in rendered code surfaces: %s", rendered.Content)
	}
	if strings.Index(rendered.Content, "ApplyPresetEnvironmentRules") > strings.Index(rendered.Content, "HandleStuff") && strings.Contains(rendered.Content, "HandleStuff") {
		t.Fatalf("expected stronger symbol match to rank ahead of weak symbol match: %s", rendered.Content)
	}
}

func TestRenderRelevantDocsSpreadsAcrossFamilies(t *testing.T) {
	idx := indexer.Result{
		Docs: []indexer.DocEntry{
			{Path: "apps/docs/docs/context-tool.md", Title: "Context Tool", Headings: []string{"Context selection"}},
			{Path: "apps/docs/docs/provider-budgeting.md", Title: "Budget", Headings: []string{"Context selection"}},
			{Path: "rfcs/context-management-tool.md", Title: "Standalone Context Management Tool", Headings: []string{"Context selection"}},
			{Path: "memory_bank/active_context.md", Title: "Active Context", Headings: []string{"Context selection"}},
		},
	}
	rendered := renderRelevantDocs(idx, []string{"context", "selection"})
	families := map[string]bool{}
	for _, path := range rendered.SourcePaths {
		families[pathFamily(path)] = true
	}
	if len(families) < 2 {
		t.Fatalf("expected docs selection to span at least two families, got %v", rendered.SourcePaths)
	}
}

func TestRenderRelevantCodePrefersCodeKindsOverDocs(t *testing.T) {
	idx := indexer.Result{
		Files: []indexer.FileEntry{
			{Path: "apps/docs/docs/context-tool.md", Kind: "doc"},
			{Path: "internal/contexttool/assemble.go", Kind: "go"},
			{Path: "internal/orchestrator/orchestrator.go", Kind: "go"},
		},
		Symbols: []indexer.SymbolEntry{
			{Name: "ChooseContextPack", Kind: "function", Path: "internal/orchestrator/orchestrator.go", Line: 10},
		},
	}
	rendered := renderRelevantCode(idx, []string{"context", "tool"})
	if !strings.Contains(rendered.Content, "internal/contexttool/assemble.go") {
		t.Fatalf("expected code file to outrank docs in code surfaces: %s", rendered.Content)
	}
}

func TestRenderRelevantCodeSpreadsAcrossClustersWithinFamily(t *testing.T) {
	idx := indexer.Result{
		Files: []indexer.FileEntry{
			{Path: "internal/contexttool/assemble.go", Kind: "go"},
			{Path: "internal/contexttool/index.go", Kind: "go"},
			{Path: "internal/contexttool/memory.go", Kind: "go"},
			{Path: "internal/orchestrator/orchestrator.go", Kind: "go"},
			{Path: "internal/presets/runtime.go", Kind: "go"},
			{Path: "internal/budget/budget.go", Kind: "go"},
		},
	}
	rendered := renderRelevantCode(idx, []string{"context"})
	clusters := map[string]bool{}
	for _, sourcePath := range rendered.SourcePaths {
		clusters[pathClusterFromSourcePath(sourcePath)] = true
	}
	if len(clusters) < 3 {
		t.Fatalf("expected code selection to span at least three clusters, got %v", rendered.SourcePaths)
	}
	if share := sectionDominantClusterShare([]SectionProvenance{{
		Title:       "Relevant Code Surfaces",
		SourcePaths: rendered.SourcePaths,
	}}, "Relevant Code Surfaces"); share > 50 {
		t.Fatalf("expected dominant cluster share <= 50, got %d from %v", share, rendered.SourcePaths)
	}
}

func TestSummarizePackRewardsSourceDiversity(t *testing.T) {
	pack := contextpack.Pack{
		Task: "explain context selection",
		Sections: []contextpack.Section{
			{Title: "Task Brief", Content: "explain context selection"},
			{Title: "Relevant Docs", Content: "context selection docs"},
			{Title: "Relevant Code Surfaces", Content: "context selection code"},
			{Title: "Relevant Memory", Content: "context selection memory"},
		},
	}
	provenance := []SectionProvenance{
		{Title: "Task Brief", SelectedCount: 1},
		{Title: "Relevant Docs", SelectedCount: 2},
		{Title: "Relevant Code Surfaces", SelectedCount: 3},
		{Title: "Relevant Memory", SelectedCount: 1},
	}
	summary := summarizePack(pack, []string{"context", "selection"}, nil, provenance, RetrievalAccounting{})
	if summary.SourceDiversity < 3 {
		t.Fatalf("expected source diversity >= 3, got %d", summary.SourceDiversity)
	}
	if summary.DiversityBonus <= 0 {
		t.Fatalf("expected positive diversity bonus")
	}
	if !contains(summary.SourceKinds, "docs") || !contains(summary.SourceKinds, "code") || !contains(summary.SourceKinds, "memory") {
		t.Fatalf("expected docs/code/memory source kinds, got %v", summary.SourceKinds)
	}
}

func TestSummarizePackTracksClusterDiversityAndDominance(t *testing.T) {
	pack := contextpack.Pack{
		Task: "explain context selection",
		Sections: []contextpack.Section{
			{Title: "Relevant Docs", Content: "docs"},
			{Title: "Relevant Code Surfaces", Content: "code"},
		},
	}
	provenance := []SectionProvenance{
		{
			Title:         "Relevant Docs",
			SelectedCount: 4,
			SourcePaths: []string{
				"apps/docs/docs/context-tool.md",
				"apps/docs/docs/provider-budgeting.md",
				"rfcs/context-management-tool.md",
				"memory_bank/active_context.md",
			},
		},
		{
			Title:         "Relevant Code Surfaces",
			SelectedCount: 4,
			SourcePaths: []string{
				"internal/contexttool/assemble.go",
				"internal/contexttool/index.go",
				"internal/orchestrator/orchestrator.go",
				"internal/presets/runtime.go",
			},
		},
	}
	summary := summarizePack(pack, []string{"context", "selection"}, nil, provenance, RetrievalAccounting{})
	if summary.DocClusterDiversity < 3 {
		t.Fatalf("expected doc cluster diversity >= 3, got %d", summary.DocClusterDiversity)
	}
	if summary.CodeClusterDiversity < 3 {
		t.Fatalf("expected code cluster diversity >= 3, got %d", summary.CodeClusterDiversity)
	}
	if summary.DocDominantClusterShare <= 0 || summary.DocDominantClusterShare > 50 {
		t.Fatalf("unexpected doc dominant cluster share %d", summary.DocDominantClusterShare)
	}
	if summary.CodeDominantClusterShare <= 0 || summary.CodeDominantClusterShare > 50 {
		t.Fatalf("unexpected code dominant cluster share %d", summary.CodeDominantClusterShare)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
