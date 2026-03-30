package contexttool

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"agent-os/internal/contextpack"
	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/project"
)

type AssembleResult struct {
	OutputDir          string           `json:"output_dir"`
	PackJSONPath       string           `json:"pack_json_path"`
	PackMDPath         string           `json:"pack_md_path"`
	MetadataPath       string           `json:"metadata_path"`
	Pack               contextpack.Pack `json:"pack"`
	BuiltIndex         bool             `json:"built_index"`
	MatchedTerms       []string         `json:"matched_terms"`
	QualityScore       int              `json:"quality_score"`
	TermCoverage       int              `json:"term_coverage"`
	MatchedSections    []string         `json:"matched_sections"`
	MemoryMatchCount   int              `json:"memory_match_count"`
	MatchedMemoryIDs   []string         `json:"matched_memory_ids"`
	MemoryBoost        int              `json:"memory_boost"`
	MemoryTrustBonus   int              `json:"memory_trust_bonus"`
	MemoryRecencyBonus int              `json:"memory_recency_bonus"`
	ConfigPath         string           `json:"config_path"`
	HumanConfig        HumanConfig      `json:"human_config"`
}

type assembleMetadata struct {
	Task                 string      `json:"task"`
	GeneratedAt          string      `json:"generated_at"`
	MatchedTerms         []string    `json:"matched_terms"`
	BuiltIndex           bool        `json:"built_index"`
	IndexPath            string      `json:"index_path"`
	MemoryPath           string      `json:"memory_path"`
	Sections             []string    `json:"sections"`
	QualityScore         int         `json:"quality_score"`
	TermCoverage         int         `json:"term_coverage"`
	MatchedSectionTitles []string    `json:"matched_section_titles"`
	MemoryMatchCount     int         `json:"memory_match_count"`
	MatchedMemoryIDs     []string    `json:"matched_memory_ids"`
	MemoryBoost          int         `json:"memory_boost"`
	MemoryTrustBonus     int         `json:"memory_trust_bonus"`
	MemoryRecencyBonus   int         `json:"memory_recency_bonus"`
	ConfigPath           string      `json:"config_path"`
	HumanConfig          HumanConfig `json:"human_config"`
}

type retrievalSummary struct {
	QualityScore       int
	TermCoverage       int
	MatchedSections    []string
	MemoryMatchCount   int
	MatchedMemoryIDs   []string
	MemoryBoost        int
	MemoryTrustBonus   int
	MemoryRecencyBonus int
}

type memoryCandidate struct {
	score        int
	trustBonus   int
	recencyBonus int
	item         memory.Item
}

func Assemble(root string, task string) (AssembleResult, error) {
	idx, items, builtIndex, err := ensureIndexAndMemory(root)
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
		ConfigPath:           HumanConfigPath(root),
		HumanConfig:          cfg,
	}
	if err := project.WriteJSON(metaPath, meta); err != nil {
		return AssembleResult{}, err
	}

	return AssembleResult{
		OutputDir:          outputDir,
		PackJSONPath:       jsonPath,
		PackMDPath:         mdPath,
		MetadataPath:       metaPath,
		Pack:               pack,
		BuiltIndex:         builtIndex,
		MatchedTerms:       terms,
		QualityScore:       summary.QualityScore,
		TermCoverage:       summary.TermCoverage,
		MatchedSections:    summary.MatchedSections,
		MemoryMatchCount:   summary.MemoryMatchCount,
		MatchedMemoryIDs:   summary.MatchedMemoryIDs,
		MemoryBoost:        summary.MemoryBoost,
		MemoryTrustBonus:   summary.MemoryTrustBonus,
		MemoryRecencyBonus: summary.MemoryRecencyBonus,
		ConfigPath:         HumanConfigPath(root),
		HumanConfig:        cfg,
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
	memoryMatches := selectRelevantMemory(items, terms)
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
	add("Query Signals", "task analysis", renderQuerySignals(task, terms), 600*4)
	add("Relevant Docs", ".context/index/docs.json", renderRelevantDocs(idx, terms), 1800*4)
	add("Relevant Code Surfaces", ".context/index/{files,symbols}.json", renderRelevantCode(idx, terms), 1800*4)
	add("Recent Changes", ".context/index/recent_changes.json", renderRecentChanges(idx), 1200*4)
	add("Relevant Memory", ".context/memory/entries.json", renderRelevantMemory(memoryMatches), 1200*4)
	add("Index Summary", ".context/index/bundle.json", renderIndexSummary(idx), 2500*4)
	add("Memory Summary", ".context/memory/entries.json", renderMemorySummary(items), 800*4)

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
	return pack, summarizePack(pack, terms, memoryMatches)
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

func renderRelevantDocs(idx indexer.Result, terms []string) string {
	type candidate struct {
		score int
		doc   indexer.DocEntry
	}
	candidates := []candidate{}
	for _, doc := range idx.Docs {
		if shouldIgnorePath(doc.Path) {
			continue
		}
		score := scoreDoc(doc, terms)
		if score == 0 && len(terms) > 0 {
			continue
		}
		candidates = append(candidates, candidate{score: score, doc: doc})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].doc.Path < candidates[j].doc.Path
		}
		return candidates[i].score > candidates[j].score
	})
	if len(candidates) == 0 {
		return "No strongly matching docs found in the current index."
	}
	var b strings.Builder
	limit := min(8, len(candidates))
	for i := 0; i < limit; i++ {
		doc := candidates[i].doc
		b.WriteString(fmt.Sprintf("- %s — %s\n", doc.Path, doc.Title))
		for _, heading := range doc.Headings {
			if strings.TrimSpace(heading) == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  headings: %s\n", heading))
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func renderRelevantCode(idx indexer.Result, terms []string) string {
	type scoredFile struct {
		score int
		file  indexer.FileEntry
	}
	type scoredSymbol struct {
		score  int
		symbol indexer.SymbolEntry
	}
	files := []scoredFile{}
	for _, file := range idx.Files {
		if shouldIgnorePath(file.Path) {
			continue
		}
		score := scoreWeightedText(file.Path, terms, 3)
		score += scoreWeightedText(filepath.Base(file.Path), terms, 6)
		score += coverageBonus(file.Path, terms, 8)
		if score == 0 && len(terms) > 0 {
			continue
		}
		files = append(files, scoredFile{score: score, file: file})
	}
	symbols := []scoredSymbol{}
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
		symbols = append(symbols, scoredSymbol{score: score, symbol: symbol})
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
		return "No strongly matching code surfaces found in the current index."
	}
	if len(files) > 0 {
		b.WriteString("Files:\n")
		for i := 0; i < min(10, len(files)); i++ {
			file := files[i].file
			b.WriteString(fmt.Sprintf("- %s (%s)\n", file.Path, file.Kind))
		}
	}
	if len(symbols) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Symbols:\n")
		for i := 0; i < min(10, len(symbols)); i++ {
			symbol := symbols[i].symbol
			b.WriteString(fmt.Sprintf("- %s (%s) in %s:%d\n", symbol.Name, symbol.Kind, symbol.Path, symbol.Line))
		}
	}
	return strings.TrimSpace(b.String())
}

func renderRecentChanges(idx indexer.Result) string {
	if len(idx.Recent) == 0 {
		return "No git history captured in the current index."
	}
	var b strings.Builder
	for i := 0; i < min(10, len(idx.Recent)); i++ {
		change := idx.Recent[i]
		b.WriteString(fmt.Sprintf("- %s %s %s\n", change.Hash, change.Date, change.Subject))
	}
	return strings.TrimSpace(b.String())
}

func renderRelevantMemory(candidates []memoryCandidate) string {
	if len(candidates) == 0 {
		return "No context-tool memory entries yet."
	}
	var b strings.Builder
	for i := 0; i < min(8, len(candidates)); i++ {
		item := candidates[i].item
		b.WriteString(fmt.Sprintf("- %s [%s/%s]: %s\n", item.ID, item.Kind, item.Status, item.Summary))
	}
	return strings.TrimSpace(b.String())
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

func summarizePack(pack contextpack.Pack, terms []string, memoryMatches []memoryCandidate) retrievalSummary {
	if len(terms) == 0 {
		memBoost, trustBonus, recencyBonus := memoryBonuses(memoryMatches)
		return retrievalSummary{
			QualityScore:       len(pack.Sections)*10 + memBoost,
			MemoryMatchCount:   len(memoryMatches),
			MatchedMemoryIDs:   matchedMemoryIDs(memoryMatches),
			MemoryBoost:        memBoost,
			MemoryTrustBonus:   trustBonus,
			MemoryRecencyBonus: recencyBonus,
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
		QualityScore:       covered*100 + len(matchedSections)*12 + bonus + memBoost,
		TermCoverage:       covered,
		MatchedSections:    matchedSections,
		MemoryMatchCount:   len(memoryMatches),
		MatchedMemoryIDs:   matchedMemoryIDs(memoryMatches),
		MemoryBoost:        memBoost,
		MemoryTrustBonus:   trustBonus,
		MemoryRecencyBonus: recencyBonus,
	}
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
