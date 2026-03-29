package memory

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-os/internal/project"
)

type Item struct {
	ID             string   `json:"id"`
	Scope          string   `json:"scope"`
	Kind           string   `json:"kind"`
	Source         string   `json:"source"`
	Confidence     string   `json:"confidence"`
	CreatedAt      string   `json:"created_at"`
	LastVerifiedAt string   `json:"last_verified_at"`
	Status         string   `json:"status"`
	Tags           []string `json:"tags"`
	Summary        string   `json:"summary"`
}

type Summary struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
	ByKind   map[string]int `json:"by_kind"`
}

func Load(root string) ([]Item, error) {
	path := project.ProjectFile(root, "memory", "entries.json")
	var items []Item
	if err := project.ReadJSON(path, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func Save(root string, items []Item) error {
	items = dedupe(items)
	if err := project.WriteJSON(project.ProjectFile(root, "memory", "entries.json"), items); err != nil {
		return err
	}
	return syncMarkdown(root, items)
}

func Add(root string, item Item) error {
	items, err := Load(root)
	if err != nil {
		return err
	}
	items = append(items, item)
	return Save(root, items)
}

func Compact(root string) (Summary, error) {
	items, err := Load(root)
	if err != nil {
		return Summary{}, err
	}

	now := time.Now().UTC()
	for i := range items {
		last, err := time.Parse(time.RFC3339, items[i].LastVerifiedAt)
		if err == nil && now.Sub(last) > 30*24*time.Hour && items[i].Status == "active" {
			items[i].Status = "stale"
		}
	}

	if err := Save(root, items); err != nil {
		return Summary{}, err
	}
	return Status(items), nil
}

func Status(items []Item) Summary {
	summary := Summary{
		Total:    len(items),
		ByStatus: map[string]int{},
		ByKind:   map[string]int{},
	}
	for _, item := range items {
		summary.ByStatus[item.Status]++
		summary.ByKind[item.Kind]++
	}
	return summary
}

func UnknownQuestions(items []Item) []Item {
	out := []Item{}
	for _, item := range items {
		if item.Kind == "question" && item.Status != "archived" {
			out = append(out, item)
		}
	}
	return out
}

func syncMarkdown(root string, items []Item) error {
	active := []Item{}
	archive := []Item{}
	decisions := []Item{}
	questions := []Item{}

	for _, item := range items {
		if item.Kind == "decision" {
			decisions = append(decisions, item)
		}
		if item.Kind == "question" {
			questions = append(questions, item)
		}
		if item.Status == "archived" || item.Status == "stale" {
			archive = append(archive, item)
		} else {
			active = append(active, item)
		}
	}

	sort.Slice(active, func(i, j int) bool { return active[i].CreatedAt > active[j].CreatedAt })
	sort.Slice(archive, func(i, j int) bool { return archive[i].CreatedAt > archive[j].CreatedAt })
	sort.Slice(decisions, func(i, j int) bool { return decisions[i].CreatedAt > decisions[j].CreatedAt })
	sort.Slice(questions, func(i, j int) bool { return questions[i].CreatedAt > questions[j].CreatedAt })

	if err := project.WriteString(project.ProjectFile(root, "memory", "MEMORY_ACTIVE.md"), renderItems("MEMORY ACTIVE", active)); err != nil {
		return err
	}
	if err := project.WriteString(project.ProjectFile(root, "memory", "MEMORY_ARCHIVE.md"), renderItems("MEMORY ARCHIVE", archive)); err != nil {
		return err
	}
	if err := project.WriteString(project.ProjectFile(root, "memory", "DECISIONS.md"), renderItems("DECISIONS", decisions)); err != nil {
		return err
	}
	return project.WriteString(project.ProjectFile(root, "memory", "OPEN_QUESTIONS.md"), renderItems("OPEN QUESTIONS", questions))
}

func renderItems(title string, items []Item) string {
	if len(items) == 0 {
		return fmt.Sprintf("# %s\n\nNo items.\n", title)
	}
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

func dedupe(items []Item) []Item {
	indexByID := map[string]int{}
	out := make([]Item, 0, len(items))
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
