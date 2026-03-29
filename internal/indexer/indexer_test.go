package indexer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildIndexesFilesSymbolsAndDocs(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel string, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite("go.mod", "module example.com/test\n\ngo 1.23.6\n")
	mustWrite("main.go", "package main\n\ntype App struct{}\n\nfunc main() {}\n")
	mustWrite("README.md", "# Sample Project\n\n## Usage\n")

	result, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Files) < 3 {
		t.Fatalf("expected at least 3 indexed files, got %d", len(result.Files))
	}
	if len(result.Symbols) < 2 {
		t.Fatalf("expected go symbols to be indexed, got %d", len(result.Symbols))
	}
	if len(result.Docs) == 0 {
		t.Fatal("expected markdown docs to be indexed")
	}
}
