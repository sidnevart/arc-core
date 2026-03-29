package contextpack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-os/internal/indexer"
	"agent-os/internal/memory"
	"agent-os/internal/mode"
	"agent-os/internal/project"
)

func TestBuildQueryProfileExtractsTermsAndIdentifiers(t *testing.T) {
	profile := buildQueryProfile("Refactor provider timeout handling in Codex RunTask and review context pack")

	if len(profile.Terms) == 0 {
		t.Fatal("expected query terms to be extracted")
	}
	if !contains(profile.Terms, "provider") || !contains(profile.Terms, "timeout") {
		t.Fatalf("expected relevant terms, got %#v", profile.Terms)
	}
	if !contains(profile.Identifiers, "Codex") || !contains(profile.Identifiers, "RunTask") {
		t.Fatalf("expected identifier candidates, got %#v", profile.Identifiers)
	}
}

func TestBuildIncludesRelevantQueryDrivenSections(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "hero",
	}); err != nil {
		t.Fatal(err)
	}

	mustWrite := func(rel string, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("go.mod", "module example.com/context\n\ngo 1.23.6\n")
	mustWrite("internal/provider/codex.go", "package provider\n\nfunc RunTaskTimeout() {}\n")
	mustWrite("README.md", "# Provider Timeout Guide\n\nThe codex adapter supports provider timeout handling.\n")
	if err := memory.Save(root, []memory.Item{
		{
			ID:             "decision-timeout",
			Scope:          "project",
			Kind:           "decision",
			Source:         "test",
			Confidence:     "high",
			CreatedAt:      "2026-03-27T00:00:00Z",
			LastVerifiedAt: "2026-03-27T00:00:00Z",
			Status:         "active",
			Tags:           []string{"provider", "timeout"},
			Summary:        "Keep provider timeout deterministic and surface failures in run artifacts.",
		},
	}); err != nil {
		t.Fatal(err)
	}

	idx, err := indexer.Build(root)
	if err != nil {
		t.Fatal(err)
	}

	pack := Build(root, "Refactor provider timeout handling for codex adapter", mode.ByName("work"), idx, mustLoadMemory(t, root))

	if !hasSection(pack, "Query Signals") {
		t.Fatal("expected Query Signals section")
	}
	codeSection := sectionContent(pack, "Relevant Code Surfaces")
	if !strings.Contains(codeSection, "internal/provider/codex.go") {
		t.Fatalf("expected relevant code section to mention codex adapter, got:\n%s", codeSection)
	}
	memorySection := sectionContent(pack, "Relevant Memory")
	if !strings.Contains(memorySection, "provider timeout deterministic") {
		t.Fatalf("expected relevant memory section, got:\n%s", memorySection)
	}
}

func mustLoadMemory(t *testing.T, root string) []memory.Item {
	t.Helper()
	items, err := memory.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func hasSection(pack Pack, title string) bool {
	for _, section := range pack.Sections {
		if section.Title == title {
			return true
		}
	}
	return false
}

func sectionContent(pack Pack, title string) string {
	for _, section := range pack.Sections {
		if section.Title == title {
			return section.Content
		}
	}
	return ""
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
