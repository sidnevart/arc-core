package contexttool

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"agent-os/internal/contextpack"
	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/project"
)

type AssembleResult struct {
	OutputDir           string              `json:"output_dir"`
	PackJSONPath        string              `json:"pack_json_path"`
	PackMDPath          string              `json:"pack_md_path"`
	MetadataPath        string              `json:"metadata_path"`
	Pack                contextpack.Pack    `json:"pack"`
	BuiltIndex          bool                `json:"built_index"`
	MatchedTerms        []string            `json:"matched_terms"`
	QualityScore        int                 `json:"quality_score"`
	TermCoverage        int                 `json:"term_coverage"`
	MatchedSections     []string            `json:"matched_sections"`
	MemoryMatchCount    int                 `json:"memory_match_count"`
	MatchedMemoryIDs    []string            `json:"matched_memory_ids"`
	MemoryBoost         int                 `json:"memory_boost"`
	MemoryTrustBonus    int                 `json:"memory_trust_bonus"`
	MemoryRecencyBonus  int                 `json:"memory_recency_bonus"`
	SourceKinds         []string            `json:"source_kinds,omitempty"`
	SourceDiversity     int                 `json:"source_diversity"`
	DiversityBonus      int                 `json:"diversity_bonus"`
	DocFamilyDiversity  int                 `json:"doc_family_diversity"`
	CodeFamilyDiversity int                 `json:"code_family_diversity"`
	ConfigPath          string              `json:"config_path"`
	HumanConfig         HumanConfig         `json:"human_config"`
	SectionProvenance   []SectionProvenance `json:"section_provenance,omitempty"`
	Accounting          RetrievalAccounting `json:"accounting"`
	Reuse               ReuseSummary        `json:"reuse"`
}

type SectionProvenance struct {
	Title          string   `json:"title"`
	Source         string   `json:"source"`
	SourcePaths    []string `json:"source_paths,omitempty"`
	CandidateCount int      `json:"candidate_count"`
	SelectedCount  int      `json:"selected_count"`
	Truncated      bool     `json:"truncated,omitempty"`
	Notes          []string `json:"notes,omitempty"`
}

type RetrievalAccounting struct {
	CandidateTotal   int `json:"candidate_total"`
	SelectedTotal    int `json:"selected_total"`
	CandidateDocs    int `json:"candidate_docs"`
	SelectedDocs     int `json:"selected_docs"`
	CandidateFiles   int `json:"candidate_files"`
	SelectedFiles    int `json:"selected_files"`
	CandidateSymbols int `json:"candidate_symbols"`
	SelectedSymbols  int `json:"selected_symbols"`
	CandidateChanges int `json:"candidate_changes"`
	SelectedChanges  int `json:"selected_changes"`
	CandidateMemory  int `json:"candidate_memory"`
	SelectedMemory   int `json:"selected_memory"`
}

type ReuseSummary struct {
	IndexBundlePath     string `json:"index_bundle_path"`
	IndexSource         string `json:"index_source"`
	IndexReused         bool   `json:"index_reused"`
	MemoryEntriesPath   string `json:"memory_entries_path"`
	MemorySource        string `json:"memory_source"`
	MemoryEntriesCount  int    `json:"memory_entries_count"`
	ReusedArtifactCount int    `json:"reused_artifact_count"`
}

type assembleMetadata struct {
	Task                 string              `json:"task"`
	GeneratedAt          string              `json:"generated_at"`
	MatchedTerms         []string            `json:"matched_terms"`
	BuiltIndex           bool                `json:"built_index"`
	IndexPath            string              `json:"index_path"`
	MemoryPath           string              `json:"memory_path"`
	Sections             []string            `json:"sections"`
	QualityScore         int                 `json:"quality_score"`
	TermCoverage         int                 `json:"term_coverage"`
	MatchedSectionTitles []string            `json:"matched_section_titles"`
	MemoryMatchCount     int                 `json:"memory_match_count"`
	MatchedMemoryIDs     []string            `json:"matched_memory_ids"`
	MemoryBoost          int                 `json:"memory_boost"`
	MemoryTrustBonus     int                 `json:"memory_trust_bonus"`
	MemoryRecencyBonus   int                 `json:"memory_recency_bonus"`
	SourceKinds          []string            `json:"source_kinds,omitempty"`
	SourceDiversity      int                 `json:"source_diversity"`
	DiversityBonus       int                 `json:"diversity_bonus"`
	DocFamilyDiversity   int                 `json:"doc_family_diversity"`
	CodeFamilyDiversity  int                 `json:"code_family_diversity"`
	ConfigPath           string              `json:"config_path"`
	HumanConfig          HumanConfig         `json:"human_config"`
	SectionProvenance    []SectionProvenance `json:"section_provenance,omitempty"`
	Accounting           RetrievalAccounting `json:"accounting"`
	Reuse                ReuseSummary        `json:"reuse"`
}

type retrievalSummary struct {
	QualityScore        int
	TermCoverage        int
	MatchedSections     []string
	MemoryMatchCount    int
	MatchedMemoryIDs    []string
	MemoryBoost         int
	MemoryTrustBonus    int
	MemoryRecencyBonus  int
	SourceKinds         []string
	SourceDiversity     int
	DiversityBonus      int
	DocFamilyDiversity  int
	CodeFamilyDiversity int
	SectionProvenance   []SectionProvenance
	Accounting          RetrievalAccounting
}

type memoryCandidate struct {
	score        int
	trustBonus   int
	recencyBonus int
	item         memory.Item
}

type docCandidate struct {
	score int
	doc   indexer.DocEntry
}

type fileCandidate struct {
	score int
	file  indexer.FileEntry
}

type symbolCandidate struct {
	score  int
	symbol indexer.SymbolEntry
}

type renderedSection struct {
	Content        string
	CandidateCount int
	SelectedCount  int
	SourcePaths    []string
	Notes          []string
}

func Assemble(root string, task string) (AssembleResult, error) {
	idx, items, builtIndex, reuse, err := ensureIndexAndMemory(root)
	if err != nil {
		return AssembleResult{}, err
	}
	cfg, err := LoadHumanConfig(root)
	if err != nil {
		return AssembleResult{}, err
	}
	terms := queryTerms(task)
	pack, summary := buildPack(task, idx, items, terms)

	runID := time.Now().UTC().Format("20060102T150405Z")
	outputDir := WorkspaceFile(root, "artifacts", "assemble", runID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return AssembleResult{}, err
	}

	jsonPath := filepath.Join(outputDir, "context_pack.json")
	mdPath := filepath.Join(outputDir, "context_pack.md")
	metaPath := filepath.Join(outputDir, "metadata.json")

	if err := project.WriteJSON(jsonPath, pack); err != nil {
		return AssembleResult{}, err
	}
	if err := project.WriteString(mdPath, contextpack.Markdown(pack)); err != nil {
		return AssembleResult{}, err
	}
	sections := make([]string, 0, len(pack.Sections))
	for _, section := range pack.Sections {
		sections = append(sections, section.Title)
	}
	meta := assembleMetadata{
		Task:                 task,
		GeneratedAt:          pack.GeneratedAt,
		MatchedTerms:         terms,
		BuiltIndex:           builtIndex,
		IndexPath:            WorkspaceFile(root, "index", "bundle.json"),
		MemoryPath:           WorkspaceFile(root, "memory", "entries.json"),
		Sections:             sections,
		QualityScore:         summary.QualityScore,
		TermCoverage:         summary.TermCoverage,
		MatchedSectionTitles: summary.MatchedSections,
		MemoryMatchCount:     summary.MemoryMatchCount,
		MatchedMemoryIDs:     summary.MatchedMemoryIDs,
		MemoryBoost:          summary.MemoryBoost,
		MemoryTrustBonus:     summary.MemoryTrustBonus,
		MemoryRecencyBonus:   summary.MemoryRecencyBonus,
		SourceKinds:          summary.SourceKinds,
		SourceDiversity:      summary.SourceDiversity,
		DiversityBonus:       summary.DiversityBonus,
		DocFamilyDiversity:   summary.DocFamilyDiversity,
		CodeFamilyDiversity:  summary.CodeFamilyDiversity,
		ConfigPath:           HumanConfigPath(root),
		HumanConfig:          cfg,
		SectionProvenance:    summary.SectionProvenance,
		Accounting:           summary.Accounting,
		Reuse:                reuse,
	}
	if err := project.WriteJSON(metaPath, meta); err != nil {
		return AssembleResult{}, err
	}

	return AssembleResult{
		OutputDir:           outputDir,
		PackJSONPath:        jsonPath,
		PackMDPath:          mdPath,
		MetadataPath:        metaPath,
		Pack:                pack,
		BuiltIndex:          builtIndex,
		MatchedTerms:        terms,
		QualityScore:        summary.QualityScore,
		TermCoverage:        summary.TermCoverage,
		MatchedSections:     summary.MatchedSections,
		MemoryMatchCount:    summary.MemoryMatchCount,
		MatchedMemoryIDs:    summary.MatchedMemoryIDs,
		MemoryBoost:         summary.MemoryBoost,
		MemoryTrustBonus:    summary.MemoryTrustBonus,
		MemoryRecencyBonus:  summary.MemoryRecencyBonus,
		SourceKinds:         summary.SourceKinds,
		SourceDiversity:     summary.SourceDiversity,
		DiversityBonus:      summary.DiversityBonus,
		DocFamilyDiversity:  summary.DocFamilyDiversity,
		CodeFamilyDiversity: summary.CodeFamilyDiversity,
		ConfigPath:          HumanConfigPath(root),
		HumanConfig:         cfg,
		SectionProvenance:   summary.SectionProvenance,
		Accounting:          summary.Accounting,
		Reuse:               reuse,
	}, nil
}

func loadMemory(root string) ([]memory.Item, error) {
	path := WorkspaceFile(root, "memory", "entries.json")
	var items []memory.Item
	if err := project.ReadJSON(path, &items); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return items, nil
}

func buildPack(task string, idx indexer.Result, items []memory.Item, terms []string) (contextpack.Pack, retrievalSummary) {
	sections := []contextpack.Section{}
	provenance := []SectionProvenance{}
	accounting := RetrievalAccounting{}
	memoryMatches := selectRelevantMemory(items, terms)
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
	add("Query Signals", "task analysis", renderedSection{Content: renderQuerySignals(task, terms), CandidateCount: max(1, len(terms)), SelectedCount: max(1, len(terms))}, 600*4)

	docsRendered := renderRelevantDocs(idx, terms)
	accounting.CandidateDocs = docsRendered.CandidateCount
	accounting.SelectedDocs = docsRendered.SelectedCount
	add("Relevant Docs", ".context/index/docs.json", docsRendered, 1800*4)

	codeRendered := renderRelevantCode(idx, terms)
	accounting.CandidateFiles = countNoteValue(codeRendered.Notes, "candidate_files")
	accounting.SelectedFiles = countNoteValue(codeRendered.Notes, "selected_files")
	accounting.CandidateSymbols = countNoteValue(codeRendered.Notes, "candidate_symbols")
	accounting.SelectedSymbols = countNoteValue(codeRendered.Notes, "selected_symbols")
	add("Relevant Code Surfaces", ".context/index/{files,symbols}.json", codeRendered, 1800*4)

	changesRendered := renderRecentChanges(idx)
	accounting.CandidateChanges = changesRendered.CandidateCount
	accounting.SelectedChanges = changesRendered.SelectedCount
	add("Recent Changes", ".context/index/recent_changes.json", changesRendered, 1200*4)

	memoryRendered := renderRelevantMemory(memoryMatches)
	accounting.CandidateMemory = memoryRendered.CandidateCount
	accounting.SelectedMemory = memoryRendered.SelectedCount
	add("Relevant Memory", ".context/memory/entries.json", memoryRendered, 1200*4)

	add("Index Summary", ".context/index/bundle.json", renderedSection{Content: renderIndexSummary(idx), CandidateCount: 1, SelectedCount: 1, SourcePaths: []string{".context/index/bundle.json"}}, 2500*4)
	add("Memory Summary", ".context/memory/entries.json", renderedSection{Content: renderMemorySummary(items), CandidateCount: len(items), SelectedCount: min(10, len(items)), SourcePaths: []string{".context/memory/entries.json"}}, 800*4)

	total := 0
	for _, section := range sections {
		total += section.ApproxTokens
	}

	pack := contextpack.Pack{
		Task:         task,
		Mode:         "context-tool",
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		ApproxTokens: total,
		Sections:     sections,
	}
	return pack, summarizePack(pack, terms, memoryMatches, provenance, accounting)
}

func renderQuerySignals(task string, terms []string) string {
	var b strings.Builder
	b.WriteString("Task:\n")
	b.WriteString("- " + strings.TrimSpace(task) + "\n")
	if len(terms) == 0 {
		b.WriteString("\nDerived terms:\n- no strong keywords detected\n")
		return b.String()
	}
	b.WriteString("\nDerived terms:\n")
	for _, term := range terms {
		b.WriteString("- " + term + "\n")
	}
	return strings.TrimSpace(b.String())
}

func renderRelevantDocs(idx indexer.Result, terms []string) renderedSection {
	candidates := []docCandidate{}
	for _, doc := range idx.Docs {
		if shouldIgnorePath(doc.Path) {
			continue
		}
		score := scoreDoc(doc, terms)
		if score == 0 && len(terms) > 0 {
			continue
		}
		candidates = append(candidates, docCandidate{score: score, doc: doc})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].doc.Path < candidates[j].doc.Path
		}
		return candidates[i].score > candidates[j].score
	})
	if len(candidates) == 0 {
		return renderedSection{
			Content:        "No strongly matching docs found in the current index.",
			SourcePaths:    []string{".context/index/docs.json"},
			CandidateCount: 0,
			SelectedCount:  0,
		}
	}
	var b strings.Builder
	selected := selectDocCandidates(candidates, 8)
	limit := len(selected)
	sourcePaths := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		doc := selected[i].doc
		sourcePaths = append(sourcePaths, doc.Path)
		b.WriteString(fmt.Sprintf("- %s — %s\n", doc.Path, doc.Title))
		for _, heading := range doc.Headings {
			if strings.TrimSpace(heading) == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  headings: %s\n", heading))
			break
		}
	}
	return renderedSection{
		Content:        strings.TrimSpace(b.String()),
		CandidateCount: len(candidates),
		SelectedCount:  limit,
		SourcePaths:    sourcePaths,
	}
}

func renderRelevantCode(idx indexer.Result, terms []string) renderedSection {
	files := []fileCandidate{}
	for _, file := range idx.Files {
		if shouldIgnorePath(file.Path) {
			continue
		}
		score := scoreWeightedText(file.Path, terms, 3)
		score += scoreWeightedText(filepath.Base(file.Path), terms, 6)
		score += coverageBonus(file.Path, terms, 8)
		score += codeFileKindBonus(file.Kind)
		if score == 0 && len(terms) > 0 {
			continue
		}
		files = append(files, fileCandidate{score: score, file: file})
	}
	symbols := []symbolCandidate{}
	for _, symbol := range idx.Symbols {
		if shouldIgnorePath(symbol.Path) {
			continue
		}
		score := scoreWeightedText(symbol.Name, terms, 8)
		score += scoreWeightedText(symbol.Path, terms, 2)
		score += scoreWeightedText(symbol.Kind, terms, 1)
		score += coverageBonus(symbol.Name+" "+symbol.Path+" "+symbol.Kind, terms, 10)
		if score == 0 && len(terms) > 0 {
			continue
		}
		symbols = append(symbols, symbolCandidate{score: score, symbol: symbol})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].score == files[j].score {
			return files[i].file.Path < files[j].file.Path
		}
		return files[i].score > files[j].score
	})
	sort.Slice(symbols, func(i, j int) bool {
		if symbols[i].score == symbols[j].score {
			return symbols[i].symbol.Path+symbols[i].symbol.Name < symbols[j].symbol.Path+symbols[j].symbol.Name
		}
		return symbols[i].score > symbols[j].score
	})
	var b strings.Builder
	if len(files) == 0 && len(symbols) == 0 {
		return renderedSection{
			Content:        "No strongly matching code surfaces found in the current index.",
			SourcePaths:    []string{".context/index/files.json", ".context/index/symbols.json"},
			CandidateCount: 0,
			SelectedCount:  0,
		}
	}
	sourcePaths := []string{}
	selectedFileItems := selectDiverseFiles(files, 10)
	selectedFiles := len(selectedFileItems)
	selectedSymbols := min(10, len(symbols))
	if len(selectedFileItems) > 0 {
		b.WriteString("Files:\n")
		for i := 0; i < selectedFiles; i++ {
			file := selectedFileItems[i].file
			sourcePaths = append(sourcePaths, file.Path)
			b.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, file.Kind))
		}
	}
	if len(symbols) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Symbols:\n")
		for i := 0; i < selectedSymbols; i++ {
			symbol := symbols[i].symbol
			sourcePaths = append(sourcePaths, fmt.Sprintf("%s:%s", symbol.Path, symbol.Name))
			b.WriteString(fmt.Sprintf("- %s (%s) in %s:%d\n", symbol.Name, symbol.Kind, symbol.Path, symbol.Line))
		}
	}
	return renderedSection{
		Content:        strings.TrimSpace(b.String()),
		CandidateCount: len(files) + len(symbols),
		SelectedCount:  selectedFiles + selectedSymbols,
		SourcePaths:    sourcePaths,
		Notes: []string{
			fmt.Sprintf("candidate_files=%d", len(files)),
			fmt.Sprintf("selected_files=%d", selectedFiles),
			fmt.Sprintf("candidate_symbols=%d", len(symbols)),
			fmt.Sprintf("selected_symbols=%d", selectedSymbols),
		},
	}
}

func renderRecentChanges(idx indexer.Result) renderedSection {
	if len(idx.Recent) == 0 {
		return renderedSection{
			Content:        "No git history captured in the current index.",
			SourcePaths:    []string{".context/index/recent_changes.json"},
			CandidateCount: 0,
			SelectedCount:  0,
		}
	}
	var b strings.Builder
	limit := min(10, len(idx.Recent))
	for i := 0; i < limit; i++ {
		change := idx.Recent[i]
		b.WriteString(fmt.Sprintf("- %s %s %s\n", change.Hash, change.Date, change.Subject))
	}
	return renderedSection{
		Content:        strings.TrimSpace(b.String()),
		CandidateCount: len(idx.Recent),
		SelectedCount:  limit,
		SourcePaths:    []string{".context/index/recent_changes.json"},
	}
}

func renderRelevantMemory(candidates []memoryCandidate) renderedSection {
	if len(candidates) == 0 {
		return renderedSection{
			Content:        "No context-tool memory entries yet.",
			SourcePaths:    []string{".context/memory/entries.json"},
			CandidateCount: 0,
			SelectedCount:  0,
		}
	}
	var b strings.Builder
	limit := min(8, len(candidates))
	sourcePaths := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		item := candidates[i].item
		sourcePaths = append(sourcePaths, item.ID)
		b.WriteString(fmt.Sprintf("- %s [%s/%s]: %s\n", item.ID, item.Kind, item.Status, item.Summary))
	}
	return renderedSection{
		Content:        strings.TrimSpace(b.String()),
		CandidateCount: len(candidates),
		SelectedCount:  limit,
		SourcePaths:    sourcePaths,
	}
}

func selectRelevantMemory(items []memory.Item, terms []string) []memoryCandidate {
	if len(items) == 0 {
		return nil
	}
	now := time.Now().UTC()
	candidates := []memoryCandidate{}
	for _, item := range items {
		candidate := scoreMemoryItem(item, terms, now)
		if candidate.score == 0 && len(terms) > 0 {
			continue
		}
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].item.ID < candidates[j].item.ID
		}
		return candidates[i].score > candidates[j].score
	})
	return candidates
}

func renderIndexSummary(idx indexer.Result) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Files indexed: %d\n", len(idx.Files)))
	b.WriteString(fmt.Sprintf("Symbols indexed: %d\n", len(idx.Symbols)))
	b.WriteString(fmt.Sprintf("Dependencies indexed: %d\n", len(idx.Dependencies)))
	b.WriteString(fmt.Sprintf("Docs indexed: %d\n", len(idx.Docs)))
	if len(idx.Recent) > 0 {
		b.WriteString(fmt.Sprintf("Recent changes captured: %d\n", len(idx.Recent)))
	}
	if idx.Tooling.Git || idx.Tooling.RG || idx.Tooling.AstGrep {
		b.WriteString("\nTooling:\n")
		b.WriteString(fmt.Sprintf("- git: %t\n", idx.Tooling.Git))
		b.WriteString(fmt.Sprintf("- rg: %t\n", idx.Tooling.RG))
		b.WriteString(fmt.Sprintf("- ast-grep: %t\n", idx.Tooling.AstGrep))
	}
	return strings.TrimSpace(b.String())
}

func renderMemorySummary(items []memory.Item) string {
	if len(items) == 0 {
		return "No context-tool memory entries yet."
	}
	var b strings.Builder
	for i, item := range items {
		if i >= 10 {
			break
		}
		b.WriteString(fmt.Sprintf("- %s [%s/%s]: %s\n", item.ID, item.Kind, item.Status, item.Summary))
	}
	return strings.TrimSpace(b.String())
}

func queryTerms(task string) []string {
	stop := map[string]struct{}{
		"what": {}, "with": {}, "that": {}, "this": {}, "from": {}, "into": {}, "and": {},
		"для": {}, "как": {}, "что": {}, "это": {}, "или": {}, "над": {}, "под": {}, "при": {},
	}
	seen := map[string]struct{}{}
	out := []string{}
	var current []rune
	flush := func() {
		if len(current) == 0 {
			return
		}
		token := strings.ToLower(string(current))
		current = current[:0]
		if len([]rune(token)) < 3 {
			return
		}
		if _, ok := stop[token]; ok {
			return
		}
		if _, ok := seen[token]; ok {
			return
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	for _, r := range task {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			current = append(current, r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func scoreDoc(doc indexer.DocEntry, terms []string) int {
	score := scoreWeightedText(doc.Path, terms, 2)
	score += scoreWeightedText(doc.Title, terms, 6)
	for _, heading := range doc.Headings {
		score += scoreWeightedText(heading, terms, 4)
	}
	score += coverageBonus(doc.Path+" "+doc.Title+" "+strings.Join(doc.Headings, " "), terms, 10)
	return score
}

func scoreText(text string, terms []string) int {
	if len(terms) == 0 {
		return 1
	}
	lower := strings.ToLower(text)
	score := 0
	for _, term := range terms {
		if strings.Contains(lower, strings.ToLower(term)) {
			score += 3
		}
	}
	return score
}

func scoreWeightedText(text string, terms []string, weight int) int {
	if len(terms) == 0 {
		return weight
	}
	lower := strings.ToLower(text)
	score := 0
	for _, term := range terms {
		termLower := strings.ToLower(term)
		if strings.Contains(lower, termLower) {
			score += weight
		}
	}
	return score
}

func coverageBonus(text string, terms []string, unit int) int {
	if len(terms) == 0 {
		return 0
	}
	lower := strings.ToLower(text)
	covered := 0
	for _, term := range terms {
		if strings.Contains(lower, strings.ToLower(term)) {
			covered++
		}
	}
	if covered <= 1 {
		return 0
	}
	return covered * unit
}

func summarizePack(pack contextpack.Pack, terms []string, memoryMatches []memoryCandidate, provenance []SectionProvenance, accounting RetrievalAccounting) retrievalSummary {
	sourceKinds, diversityBonus := sourceDiversitySignals(provenance)
	docFamilyDiversity := sectionFamilyDiversity(provenance, "Relevant Docs")
	codeFamilyDiversity := sectionFamilyDiversity(provenance, "Relevant Code Surfaces")
	if len(terms) == 0 {
		memBoost, trustBonus, recencyBonus := memoryBonuses(memoryMatches)
		return retrievalSummary{
			QualityScore:        len(pack.Sections)*10 + memBoost + diversityBonus,
			MemoryMatchCount:    len(memoryMatches),
			MatchedMemoryIDs:    matchedMemoryIDs(memoryMatches),
			MemoryBoost:         memBoost,
			MemoryTrustBonus:    trustBonus,
			MemoryRecencyBonus:  recencyBonus,
			SourceKinds:         sourceKinds,
			SourceDiversity:     len(sourceKinds),
			DiversityBonus:      diversityBonus,
			DocFamilyDiversity:  docFamilyDiversity,
			CodeFamilyDiversity: codeFamilyDiversity,
			SectionProvenance:   provenance,
			Accounting:          accounting,
		}
	}
	lowerSections := make([]string, 0, len(pack.Sections))
	for _, section := range pack.Sections {
		lowerSections = append(lowerSections, strings.ToLower(section.Title+"\n"+section.Source+"\n"+section.Content))
	}
	covered := 0
	matchedSections := []string{}
	seenSection := map[string]bool{}
	for _, term := range terms {
		termLower := strings.ToLower(term)
		for _, section := range pack.Sections {
			blob := strings.ToLower(section.Title + "\n" + section.Source + "\n" + section.Content)
			if strings.Contains(blob, termLower) {
				if !seenSection[section.Title] {
					seenSection[section.Title] = true
					matchedSections = append(matchedSections, section.Title)
				}
			}
		}
		for _, blob := range lowerSections {
			if strings.Contains(blob, termLower) {
				covered++
				break
			}
		}
	}
	bonus := 0
	for _, section := range pack.Sections {
		switch section.Title {
		case "Relevant Docs":
			bonus += 15
		case "Relevant Code Surfaces":
			bonus += 20
		case "Relevant Memory":
			bonus += 10
		case "Query Signals":
			bonus += 10
		}
	}
	memBoost, trustBonus, recencyBonus := memoryBonuses(memoryMatches)
	return retrievalSummary{
		QualityScore:        covered*100 + len(matchedSections)*12 + bonus + memBoost + diversityBonus,
		TermCoverage:        covered,
		MatchedSections:     matchedSections,
		MemoryMatchCount:    len(memoryMatches),
		MatchedMemoryIDs:    matchedMemoryIDs(memoryMatches),
		MemoryBoost:         memBoost,
		MemoryTrustBonus:    trustBonus,
		MemoryRecencyBonus:  recencyBonus,
		SourceKinds:         sourceKinds,
		SourceDiversity:     len(sourceKinds),
		DiversityBonus:      diversityBonus,
		DocFamilyDiversity:  docFamilyDiversity,
		CodeFamilyDiversity: codeFamilyDiversity,
		SectionProvenance:   provenance,
		Accounting:          accounting,
	}
}

func selectDocCandidates(candidates []docCandidate, limit int) []docCandidate {
	if len(candidates) <= limit {
		return append([]docCandidate(nil), candidates...)
	}
	selected := make([]docCandidate, 0, limit)
	familyCounts := map[string]int{}
	used := map[int]bool{}
	for pass := 0; pass < 2 && len(selected) < limit; pass++ {
		for i, candidate := range candidates {
			if used[i] {
				continue
			}
			family := pathFamily(candidate.doc.Path)
			if pass == 0 && familyCounts[family] >= 2 {
				continue
			}
			selected = append(selected, candidate)
			used[i] = true
			familyCounts[family]++
			if len(selected) >= limit {
				break
			}
		}
	}
	return selected
}

func selectDiverseFiles(files []fileCandidate, limit int) []fileCandidate {
	if len(files) <= limit {
		return append([]fileCandidate(nil), files...)
	}
	selected := make([]fileCandidate, 0, limit)
	familyCounts := map[string]int{}
	used := map[int]bool{}
	for pass := 0; pass < 2 && len(selected) < limit; pass++ {
		for i, candidate := range files {
			if used[i] {
				continue
			}
			family := pathFamily(candidate.file.Path)
			if pass == 0 && familyCounts[family] >= 2 {
				continue
			}
			selected = append(selected, candidate)
			used[i] = true
			familyCounts[family]++
			if len(selected) >= limit {
				break
			}
		}
	}
	return selected
}

func sourceDiversitySignals(provenance []SectionProvenance) ([]string, int) {
	if len(provenance) == 0 {
		return nil, 0
	}
	seen := map[string]bool{}
	ordered := []string{}
	for _, section := range provenance {
		if section.SelectedCount <= 0 {
			continue
		}
		kind := sectionSourceKind(section.Title)
		if kind == "" || seen[kind] {
			continue
		}
		seen[kind] = true
		ordered = append(ordered, kind)
	}
	if len(ordered) == 0 {
		return nil, 0
	}
	bonus := len(ordered) * 14
	if seen["docs"] && seen["code"] {
		bonus += 12
	}
	if seen["memory"] && (seen["docs"] || seen["code"]) {
		bonus += 8
	}
	if seen["changes"] && (seen["docs"] || seen["code"]) {
		bonus += 6
	}
	return ordered, bonus
}

func sectionFamilyDiversity(provenance []SectionProvenance, title string) int {
	for _, section := range provenance {
		if section.Title != title || len(section.SourcePaths) == 0 {
			continue
		}
		seen := map[string]bool{}
		for _, sourcePath := range section.SourcePaths {
			family := pathFamilyFromSourcePath(sourcePath)
			if family == "" {
				continue
			}
			seen[family] = true
		}
		return len(seen)
	}
	return 0
}

func sectionSourceKind(title string) string {
	switch title {
	case "Relevant Docs", "All Docs Snapshot":
		return "docs"
	case "Relevant Code Surfaces", "All Code Snapshot":
		return "code"
	case "Recent Changes":
		return "changes"
	case "Relevant Memory", "Memory Summary", "Memory Snapshot":
		return "memory"
	case "Dependencies Snapshot":
		return "dependencies"
	case "Index Summary":
		return "index"
	case "Task Brief", "Query Signals":
		return "task"
	default:
		return ""
	}
}

func pathFamilyFromSourcePath(value string) string {
	if idx := strings.Index(value, ":"); idx > 0 {
		value = value[:idx]
	}
	return pathFamily(value)
}

func pathFamily(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	if len(parts) == 1 {
		return parts[0]
	}
	switch parts[0] {
	case "internal", "cmd", "apps", "content", "memory_bank", "backups":
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}
	return parts[0]
}

func codeFileKindBonus(kind string) int {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "go", "javascript", "typescript", "python", "shell", "json", "yaml", "toml":
		return 8
	case "config":
		return 3
	case "doc":
		return -4
	default:
		return 0
	}
}

func countNoteValue(notes []string, prefix string) int {
	for _, note := range notes {
		if !strings.HasPrefix(note, prefix+"=") {
			continue
		}
		value := strings.TrimPrefix(note, prefix+"=")
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return parsed
	}
	return 0
}

func scoreMemoryItem(item memory.Item, terms []string, now time.Time) memoryCandidate {
	score := scoreText(item.Summary+" "+strings.Join(item.Tags, " ")+" "+item.Kind+" "+item.Scope, terms)
	for _, tag := range item.Tags {
		score += scoreText(tag, terms) * 2
	}
	trustBonus := 0
	switch item.Kind {
	case "decision":
		trustBonus += 6
	case "question":
		trustBonus += 3
	}
	switch item.Status {
	case "active":
		trustBonus += 4
	case "stale":
		trustBonus -= 2
	}
	switch item.Source {
	case "human":
		trustBonus += 4
	}
	switch strings.ToLower(strings.TrimSpace(item.Confidence)) {
	case "high":
		trustBonus += 6
	case "medium":
		trustBonus += 3
	}
	recencyBonus := scoreMemoryRecency(item, now)
	score += trustBonus + recencyBonus
	return memoryCandidate{
		score:        score,
		trustBonus:   trustBonus,
		recencyBonus: recencyBonus,
		item:         item,
	}
}

func matchedMemoryIDs(matches []memoryCandidate) []string {
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match.item.ID)
	}
	return out
}

func memoryBonuses(matches []memoryCandidate) (int, int, int) {
	if len(matches) == 0 {
		return 0, 0, 0
	}
	boost := 0
	trustBonus := 0
	recencyBonus := 0
	limit := min(3, len(matches))
	for i := 0; i < limit; i++ {
		boost += 15 + matches[i].trustBonus + matches[i].recencyBonus
		trustBonus += matches[i].trustBonus
		recencyBonus += matches[i].recencyBonus
	}
	return boost, trustBonus, recencyBonus
}

func scoreMemoryRecency(item memory.Item, now time.Time) int {
	ts := strings.TrimSpace(item.LastVerifiedAt)
	if ts == "" {
		ts = strings.TrimSpace(item.CreatedAt)
	}
	if ts == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return 0
	}
	age := now.Sub(parsed)
	switch {
	case age <= 7*24*time.Hour:
		return 6
	case age <= 30*24*time.Hour:
		return 3
	case age > 180*24*time.Hour:
		return -2
	default:
		return 0
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func shouldIgnorePath(path string) bool {
	switch {
	case strings.HasPrefix(path, ".arc/"):
		return true
	case strings.HasPrefix(path, "backups/repo-splits/"):
		return true
	default:
		return false
	}
}
