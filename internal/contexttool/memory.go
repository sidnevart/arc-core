package contexttool

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-os/internal/memory"
	"agent-os/internal/project"
)

type MemoryStatusReport struct {
	Summary       memory.Summary `json:"summary"`
	EntriesPath   string         `json:"entries_path"`
	ActivePath    string         `json:"active_path"`
	ArchivePath   string         `json:"archive_path"`
	QuestionsPath string         `json:"questions_path"`
	MostRecent    []memory.Item  `json:"most_recent,omitempty"`
}

func ListMemory(root string) ([]memory.Item, error) {
	if _, err := Init(root); err != nil {
		return nil, err
	}
	items, err := loadMemory(root)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func AddMemory(root string, item memory.Item) (memory.Item, error) {
	if _, err := Init(root); err != nil {
		return memory.Item{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item = normalizeMemoryItem(item, now)
	items, err := loadMemory(root)
	if err != nil {
		return memory.Item{}, err
	}
	items = append(items, item)
	if err := saveMemory(root, items); err != nil {
		return memory.Item{}, err
	}
	return item, nil
}

func SearchMemory(root string, query string, limit int) ([]memory.Item, error) {
	if _, err := Init(root); err != nil {
		return nil, err
	}
	items, err := loadMemory(root)
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return ListMemory(root)
	}
	terms := strings.Fields(query)
	type candidate struct {
		score int
		item  memory.Item
	}
	now := time.Now().UTC()
	candidates := make([]candidate, 0, len(items))
	for _, item := range items {
		scored := scoreMemoryItem(item, terms, now)
		if scored.score == 0 {
			continue
		}
		candidates = append(candidates, candidate{score: scored.score, item: item})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			if candidates[i].item.CreatedAt == candidates[j].item.CreatedAt {
				return candidates[i].item.ID > candidates[j].item.ID
			}
			return candidates[i].item.CreatedAt > candidates[j].item.CreatedAt
		}
		return candidates[i].score > candidates[j].score
	})
	if limit <= 0 || limit > len(candidates) {
		limit = len(candidates)
	}
	out := make([]memory.Item, 0, limit)
	for _, candidate := range candidates[:limit] {
		out = append(out, candidate.item)
	}
	return out, nil
}

func MemoryStatus(root string) (MemoryStatusReport, error) {
	items, err := ListMemory(root)
	if err != nil {
		return MemoryStatusReport{}, err
	}
	report := MemoryStatusReport{
		Summary:       memory.Status(items),
		EntriesPath:   WorkspaceFile(root, "memory", "entries.json"),
		ActivePath:    WorkspaceFile(root, "memory", "MEMORY_ACTIVE.md"),
		ArchivePath:   WorkspaceFile(root, "memory", "MEMORY_ARCHIVE.md"),
		QuestionsPath: WorkspaceFile(root, "memory", "OPEN_QUESTIONS.md"),
	}
	if len(items) > 5 {
		report.MostRecent = items[:5]
	} else {
		report.MostRecent = items
	}
	return report, nil
}

func CompactMemory(root string) (MemoryStatusReport, error) {
	items, err := ListMemory(root)
	if err != nil {
		return MemoryStatusReport{}, err
	}
	now := time.Now().UTC()
	for i := range items {
		last, err := time.Parse(time.RFC3339, items[i].LastVerifiedAt)
		if err == nil && now.Sub(last) > 30*24*time.Hour && items[i].Status == "active" {
			items[i].Status = "stale"
		}
	}
	if err := saveMemory(root, items); err != nil {
		return MemoryStatusReport{}, err
	}
	return MemoryStatus(root)
}

func saveMemory(root string, items []memory.Item) error {
	deduped := dedupeMemory(items)
	if err := project.WriteJSON(WorkspaceFile(root, "memory", "entries.json"), deduped); err != nil {
		return err
	}
	return syncMemoryMarkdown(root, deduped)
}

func normalizeMemoryItem(item memory.Item, now string) memory.Item {
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = fmt.Sprintf("ctx-%d", time.Now().UTC().UnixNano())
	}
	item.Scope = fallback(item.Scope, "project")
	item.Kind = fallback(item.Kind, "note")
	item.Source = fallback(item.Source, "human")
	item.Confidence = fallback(item.Confidence, "medium")
	item.Status = fallback(item.Status, "active")
	item.Summary = strings.TrimSpace(item.Summary)
	if item.CreatedAt == "" {
		item.CreatedAt = now
	}
	if item.LastVerifiedAt == "" {
		item.LastVerifiedAt = item.CreatedAt
	}
	item.Tags = normalizeTags(item.Tags)
	return item
}

func fallback(value string, defaultValue string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	return value
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func dedupeMemory(items []memory.Item) []memory.Item {
	indexByID := map[string]int{}
	out := make([]memory.Item, 0, len(items))
	for _, item := range items {
		if item.ID == "" {
			out = append(out, item)
			continue
		}
		if idx, ok := indexByID[item.ID]; ok {
			out[idx] = item
			continue
		}
		indexByID[item.ID] = len(out)
		out = append(out, item)
	}
	return out
}

func syncMemoryMarkdown(root string, items []memory.Item) error {
	active := make([]memory.Item, 0, len(items))
	archive := make([]memory.Item, 0, len(items))
	questions := make([]memory.Item, 0, len(items))
	for _, item := range items {
		if item.Status != "archived" && item.Status != "stale" {
			active = append(active, item)
		} else {
			archive = append(archive, item)
		}
		if item.Kind == "question" && item.Status != "archived" {
			questions = append(questions, item)
		}
	}
	if err := project.WriteString(WorkspaceFile(root, "memory", "MEMORY_ACTIVE.md"), renderMemoryItems("CTX MEMORY ACTIVE", active)); err != nil {
		return err
	}
	if err := project.WriteString(WorkspaceFile(root, "memory", "MEMORY_ARCHIVE.md"), renderMemoryItems("CTX MEMORY ARCHIVE", archive)); err != nil {
		return err
	}
	return project.WriteString(WorkspaceFile(root, "memory", "OPEN_QUESTIONS.md"), renderMemoryItems("CTX OPEN QUESTIONS", questions))
}

func renderMemoryItems(title string, items []memory.Item) string {
	if len(items) == 0 {
		return fmt.Sprintf("# %s\n\nNo items.\n", title)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	var b strings.Builder
	b.WriteString("# " + title + "\n\n")
	for _, item := range items {
		b.WriteString("## " + item.ID + "\n\n")
		b.WriteString("- scope: " + item.Scope + "\n")
		b.WriteString("- kind: " + item.Kind + "\n")
		b.WriteString("- source: " + item.Source + "\n")
		b.WriteString("- confidence: " + item.Confidence + "\n")
		b.WriteString("- created_at: " + item.CreatedAt + "\n")
		b.WriteString("- last_verified_at: " + item.LastVerifiedAt + "\n")
		b.WriteString("- status: " + item.Status + "\n")
		if len(item.Tags) > 0 {
			b.WriteString("- tags: " + strings.Join(item.Tags, ", ") + "\n")
		}
		b.WriteString("\n" + item.Summary + "\n\n")
	}
	return b.String()
}
