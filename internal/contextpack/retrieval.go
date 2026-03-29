package contextpack

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/project"
)

type queryProfile struct {
	Terms       []string
	Identifiers []string
}

type scoredIdentifier struct {
	value string
	score int
	index int
}

type scoredDoc struct {
	doc   indexer.DocEntry
	score int
}

type scoredSymbol struct {
	symbol indexer.SymbolEntry
	score  int
}

type scoredFile struct {
	file  indexer.FileEntry
	score int
}

type scoredMemory struct {
	item  memory.Item
	score int
}

func buildQueryProfile(task string) queryProfile {
	stopwords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "that": true, "this": true, "from": true,
		"into": true, "about": true, "does": true, "make": true, "need": true, "want": true, "user": true,
		"repo": true, "mode": true, "task": true, "project": true, "agent": true, "runtime": true,
		"что": true, "как": true, "для": true, "это": true, "или": true, "если": true, "нужно": true,
		"хочу": true, "надо": true, "сделать": true, "сейчас": true, "через": true, "чтобы": true,
		"проект": true, "режим": true, "задача": true, "агент": true, "система": true,
	}

	parts := strings.FieldsFunc(task, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-')
	})
	terms := []string{}
	seenTerms := map[string]bool{}
	seenIdentifiers := map[string]bool{}
	identifierCandidates := []scoredIdentifier{}

	for i, raw := range parts {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		normalized := strings.ToLower(raw)
		normalized = strings.Trim(normalized, "-_")
		if len([]rune(normalized)) >= 3 && !stopwords[normalized] && !seenTerms[normalized] {
			seenTerms[normalized] = true
			terms = append(terms, normalized)
		}

		if score := identifierScore(raw); score > 0 && len(raw) >= 3 && !seenIdentifiers[raw] {
			seenIdentifiers[raw] = true
			identifierCandidates = append(identifierCandidates, scoredIdentifier{
				value: raw,
				score: score,
				index: i,
			})
		}
	}

	sort.SliceStable(identifierCandidates, func(i, j int) bool {
		if identifierCandidates[i].score == identifierCandidates[j].score {
			return identifierCandidates[i].index < identifierCandidates[j].index
		}
		return identifierCandidates[i].score > identifierCandidates[j].score
	})

	identifiers := []string{}
	for _, candidate := range identifierCandidates {
		identifiers = append(identifiers, candidate.value)
		if len(identifiers) >= 5 {
			break
		}
	}

	if len(identifiers) == 0 {
		for _, term := range terms {
			if isASCIIIdentifier(term) {
				identifiers = append(identifiers, term)
			}
			if len(identifiers) >= 5 {
				break
			}
		}
	}

	if len(terms) > 8 {
		terms = terms[:8]
	}
	if len(identifiers) > 5 {
		identifiers = identifiers[:5]
	}

	return queryProfile{Terms: terms, Identifiers: identifiers}
}

func renderQuerySignals(profile queryProfile) string {
	var b strings.Builder
	b.WriteString("Extracted query terms:\n")
	if len(profile.Terms) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, term := range profile.Terms {
			b.WriteString("- " + term + "\n")
		}
	}
	b.WriteString("\nIdentifier candidates:\n")
	if len(profile.Identifiers) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, identifier := range profile.Identifiers {
			b.WriteString("- " + identifier + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func relevantExcerpt(content string, terms []string, maxChars int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	paragraphs := splitParagraphs(content)
	type candidate struct {
		text  string
		score int
		index int
	}
	candidates := []candidate{}
	for i, paragraph := range paragraphs {
		score := textScore(paragraph, terms)
		candidates = append(candidates, candidate{text: strings.TrimSpace(paragraph), score: score, index: i})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].index < candidates[j].index
		}
		return candidates[i].score > candidates[j].score
	})

	selected := []candidate{}
	for _, item := range candidates {
		if item.score <= 0 && len(selected) > 0 {
			continue
		}
		selected = append(selected, item)
		if len(selected) >= 3 {
			break
		}
	}
	if len(selected) == 0 && len(candidates) > 0 {
		selected = append(selected, candidates[0])
	}

	sort.Slice(selected, func(i, j int) bool { return selected[i].index < selected[j].index })
	out := []string{}
	for _, item := range selected {
		if item.text != "" {
			out = append(out, item.text)
		}
	}
	result := strings.Join(out, "\n\n")
	if len(result) > maxChars {
		result = strings.TrimSpace(result[:maxChars]) + "\n...[truncated]"
	}
	return result
}

func renderRelevantDocs(root string, idx indexer.Result, profile queryProfile) string {
	scored := []scoredDoc{}
	for _, doc := range idx.Docs {
		score := textScore(doc.Path+" "+doc.Title+" "+strings.Join(doc.Headings, " "), profile.Terms)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredDoc{doc: doc, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].doc.Path < scored[j].doc.Path
		}
		return scored[i].score > scored[j].score
	})

	var b strings.Builder
	limit := minInt(len(scored), 5)
	for i := 0; i < limit; i++ {
		path := filepath.Join(root, scored[i].doc.Path)
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		excerpt := relevantExcerpt(string(content), profile.Terms, 500)
		if excerpt == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", scored[i].doc.Path, scored[i].doc.Title))
		b.WriteString("  " + indentMultiline(excerpt, "  ") + "\n")
	}
	return strings.TrimSpace(b.String())
}

func renderRelevantCodeSummary(idx indexer.Result, profile queryProfile) string {
	symbols := []scoredSymbol{}
	for _, symbol := range idx.Symbols {
		score := textScore(symbol.Name+" "+symbol.Path+" "+symbol.Kind, profile.Terms)
		if score <= 0 {
			continue
		}
		symbols = append(symbols, scoredSymbol{symbol: symbol, score: score})
	}
	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].score == symbols[j].score {
			return symbols[i].symbol.Path+symbols[i].symbol.Name < symbols[j].symbol.Path+symbols[j].symbol.Name
		}
		return symbols[i].score > symbols[j].score
	})

	files := []scoredFile{}
	for _, file := range idx.Files {
		score := textScore(file.Path+" "+file.Kind, profile.Terms)
		if score <= 0 {
			continue
		}
		files = append(files, scoredFile{file: file, score: score})
	}
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].score == files[j].score {
			return files[i].file.Path < files[j].file.Path
		}
		return files[i].score > files[j].score
	})

	var b strings.Builder
	if len(files) > 0 {
		b.WriteString("Relevant files:\n")
		for i := 0; i < minInt(len(files), 8); i++ {
			b.WriteString(fmt.Sprintf("- %s [%s]\n", files[i].file.Path, files[i].file.Kind))
		}
	}
	if len(symbols) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("Relevant symbols:\n")
		for i := 0; i < minInt(len(symbols), 12); i++ {
			s := symbols[i].symbol
			b.WriteString(fmt.Sprintf("- %s (%s) in %s:%d\n", s.Name, s.Kind, s.Path, s.Line))
		}
	}
	return strings.TrimSpace(b.String())
}

func renderRelevantMemory(items []memory.Item, profile queryProfile) string {
	scored := []scoredMemory{}
	for _, item := range items {
		if item.Status == "archived" {
			continue
		}
		score := textScore(item.Summary+" "+strings.Join(item.Tags, " "), profile.Terms)
		if item.Status == "active" {
			score += 4
		}
		if item.Kind == "decision" || item.Kind == "constraint" {
			score += 2
		}
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredMemory{item: item, score: score})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].item.CreatedAt > scored[j].item.CreatedAt
		}
		return scored[i].score > scored[j].score
	})

	var b strings.Builder
	for i := 0; i < minInt(len(scored), 8); i++ {
		item := scored[i].item
		b.WriteString(fmt.Sprintf("- %s [%s/%s]: %s\n", item.ID, item.Kind, item.Status, item.Summary))
	}
	return strings.TrimSpace(b.String())
}

func renderRGHits(root string, profile queryProfile) string {
	if len(profile.Terms) == 0 {
		return ""
	}
	if _, err := exec.LookPath("rg"); err != nil {
		return ""
	}

	args := []string{"-n", "-S", "-F"}
	for _, term := range profile.Terms[:minInt(len(profile.Terms), 5)] {
		args = append(args, "-e", term)
	}
	args = append(args, ".")

	cmd := exec.Command("rg", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return ""
		}
		return ""
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return ""
	}
	var b strings.Builder
	seen := map[string]bool{}
	count := 0
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		key := parts[0] + ":" + parts[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		b.WriteString(fmt.Sprintf("- %s:%s %s\n", parts[0], parts[1], strings.TrimSpace(parts[2])))
		count++
		if count >= 12 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func renderASTHits(root string, profile queryProfile) string {
	if len(profile.Identifiers) == 0 {
		return ""
	}
	if _, err := exec.LookPath("ast-grep"); err != nil {
		return ""
	}

	searchRoots := []string{}
	for _, rel := range []string{"internal", "cmd"} {
		if info, err := os.Stat(filepath.Join(root, rel)); err == nil && info.IsDir() {
			searchRoots = append(searchRoots, rel)
		}
	}
	if len(searchRoots) == 0 {
		searchRoots = []string{"."}
	}

	var b strings.Builder
	for _, identifier := range profile.Identifiers[:minInt(len(profile.Identifiers), 3)] {
		args := []string{"scan", "--pattern", identifier}
		args = append(args, searchRoots...)
		cmd := exec.Command("ast-grep", args...)
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if err != nil || len(output) == 0 {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			continue
		}
		b.WriteString("- pattern: " + identifier + "\n")
		for i, line := range lines {
			if i >= 8 {
				break
			}
			b.WriteString("  " + line + "\n")
		}
	}
	return strings.TrimSpace(b.String())
}

func textScore(text string, terms []string) int {
	if len(terms) == 0 || text == "" {
		return 0
	}
	normalized := strings.ToLower(text)
	score := 0
	for _, term := range terms {
		if term == "" {
			continue
		}
		count := strings.Count(normalized, term)
		if count > 0 {
			score += 10 * count
			if strings.Contains(normalized, "/"+term) || strings.Contains(normalized, term+".") {
				score += 5
			}
		}
	}
	return score
}

func splitParagraphs(content string) []string {
	parts := strings.Split(content, "\n\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{content}
	}
	return out
}

func indentMultiline(value string, prefix string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	for i, line := range lines {
		lines[i] = prefix + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func isIdentifierLike(value string) bool {
	if value == "" {
		return false
	}
	if strings.ContainsAny(value, "-") {
		return false
	}
	if !isASCIIIdentifier(value) {
		return false
	}
	return true
}

func identifierScore(value string) int {
	if !isIdentifierLike(value) {
		return 0
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	innerUpper := false

	for i, r := range value {
		if unicode.IsUpper(r) {
			hasUpper = true
			if i > 0 {
				innerUpper = true
			}
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	switch {
	case innerUpper:
		return 40
	case strings.Contains(value, "_") || hasDigit:
		return 35
	case hasUpper && hasLower:
		return 25
	case hasUpper:
		return 20
	default:
		return 10
	}
}

func isASCIIIdentifier(value string) bool {
	for i, r := range value {
		switch {
		case i == 0 && (unicode.IsLetter(r) || r == '_'):
		case i > 0 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'):
		default:
			return false
		}
	}
	return true
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func readRelevantFile(root string, rel string, terms []string, maxChars int) string {
	content, err := project.ReadString(project.ProjectFile(root, strings.Split(rel, "/")...))
	if err != nil {
		return ""
	}
	return relevantExcerpt(content, terms, maxChars)
}
