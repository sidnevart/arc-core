package contextpack

import (
	"fmt"
	"strings"
	"time"

	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/mode"
)

type Section struct {
	Title        string `json:"title"`
	Source       string `json:"source"`
	Content      string `json:"content"`
	ApproxTokens int    `json:"approx_tokens"`
}

type Pack struct {
	Task         string    `json:"task"`
	Mode         string    `json:"mode"`
	GeneratedAt  string    `json:"generated_at"`
	ApproxTokens int       `json:"approx_tokens"`
	Sections     []Section `json:"sections"`
}

func Build(root string, task string, modeDef mode.Definition, idx indexer.Result, items []memory.Item) Pack {
	profile := buildQueryProfile(task)
	sections := []Section{}
	add := func(title string, source string, content string, maxChars int) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		if len(content) > maxChars {
			content = strings.TrimSpace(content[:maxChars]) + "\n...[truncated]"
		}
		sections = append(sections, Section{
			Title:        title,
			Source:       source,
			Content:      content,
			ApproxTokens: len(content) / 4,
		})
	}

	add("Task Brief", "task", task, 1500*4)
	add("Mode Policy", "mode", mode.Markdown(modeDef), 800*4)
	add("Query Signals", "task analysis", renderQuerySignals(profile), 600*4)

	add("Project Guidance", ".arc/provider/AGENTS.md", readRelevantFile(root, "provider/AGENTS.md", profile.Terms, 1200*4), 1200*4)
	add("Repo Map", ".arc/maps/REPO_MAP.md", readRelevantFile(root, "maps/REPO_MAP.md", profile.Terms, 1800*4), 1800*4)
	add("Docs Map", ".arc/maps/DOCS_MAP.md", readRelevantFile(root, "maps/DOCS_MAP.md", profile.Terms, 1800*4), 1800*4)
	add("Decisions", ".arc/memory/DECISIONS.md", readRelevantFile(root, "memory/DECISIONS.md", profile.Terms, 800*4), 800*4)
	add("Open Questions", ".arc/memory/OPEN_QUESTIONS.md", readRelevantFile(root, "memory/OPEN_QUESTIONS.md", profile.Terms, 800*4), 800*4)

	add("Relevant Docs", "docs index", renderRelevantDocs(root, idx, profile), 1800*4)
	add("Relevant Code Surfaces", "index", renderRelevantCodeSummary(idx, profile), 1800*4)
	add("Grep Hits", "rg", renderRGHits(root, profile), 1200*4)
	add("AST Hits", "ast-grep", renderASTHits(root, profile), 1200*4)
	add("Relevant Memory", "memory entries", renderRelevantMemory(items, profile), 1200*4)
	add("Index Summary", "index", renderIndexSummary(idx), 2500*4)
	add("Memory Summary", "memory entries", renderMemorySummary(items), 800*4)

	total := 0
	for _, section := range sections {
		total += section.ApproxTokens
	}

	return Pack{
		Task:         task,
		Mode:         modeDef.Name,
		GeneratedAt:  nowUTC(),
		ApproxTokens: total,
		Sections:     sections,
	}
}

func Markdown(pack Pack) string {
	var b strings.Builder
	b.WriteString("# Context Pack\n\n")
	b.WriteString(fmt.Sprintf("- task: %s\n", pack.Task))
	b.WriteString(fmt.Sprintf("- mode: %s\n", pack.Mode))
	b.WriteString(fmt.Sprintf("- generated_at: %s\n", pack.GeneratedAt))
	b.WriteString(fmt.Sprintf("- approx_tokens: %d\n\n", pack.ApproxTokens))

	for _, section := range pack.Sections {
		b.WriteString("## " + section.Title + "\n\n")
		b.WriteString("- source: " + section.Source + "\n")
		b.WriteString(fmt.Sprintf("- approx_tokens: %d\n\n", section.ApproxTokens))
		b.WriteString(section.Content + "\n\n")
	}
	return b.String()
}

func renderIndexSummary(idx indexer.Result) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Files indexed: %d\n", len(idx.Files)))
	b.WriteString(fmt.Sprintf("Symbols indexed: %d\n", len(idx.Symbols)))
	b.WriteString(fmt.Sprintf("Dependencies indexed: %d\n", len(idx.Dependencies)))
	b.WriteString(fmt.Sprintf("Docs indexed: %d\n", len(idx.Docs)))
	if len(idx.Recent) > 0 {
		b.WriteString("\nRecent changes:\n")
		for i, change := range idx.Recent {
			if i >= 10 {
				break
			}
			b.WriteString(fmt.Sprintf("- %s %s %s\n", change.Hash, change.Date, change.Subject))
		}
	}
	if len(idx.Symbols) > 0 {
		b.WriteString("\nTop symbols:\n")
		for i, symbol := range idx.Symbols {
			if i >= 20 {
				break
			}
			b.WriteString(fmt.Sprintf("- %s (%s) in %s:%d\n", symbol.Name, symbol.Kind, symbol.Path, symbol.Line))
		}
	}
	return strings.TrimSpace(b.String())
}

func renderMemorySummary(items []memory.Item) string {
	if len(items) == 0 {
		return "No structured memory entries yet."
	}
	var b strings.Builder
	for i, item := range items {
		if i >= 12 {
			break
		}
		b.WriteString(fmt.Sprintf("- %s [%s/%s]: %s\n", item.ID, item.Kind, item.Status, item.Summary))
	}
	return strings.TrimSpace(b.String())
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
