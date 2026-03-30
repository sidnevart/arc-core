package contexttool

import (
	"agent-os/internal/indexer"
	"agent-os/internal/project"
	"strings"
)

func BuildIndex(root string) (indexer.Result, error) {
	if _, err := Init(root); err != nil {
		return indexer.Result{}, err
	}
	cfg, err := LoadHumanConfig(root)
	if err != nil {
		return indexer.Result{}, err
	}
	idx, err := indexer.Build(root)
	if err != nil {
		return indexer.Result{}, err
	}
	idx = applyHumanConfigToIndex(idx, cfg)
	if err := SaveIndex(root, idx); err != nil {
		return indexer.Result{}, err
	}
	return idx, nil
}

func applyHumanConfigToIndex(idx indexer.Result, cfg HumanConfig) indexer.Result {
	idx.Files = filterFileEntries(idx.Files, cfg)
	idx.Symbols = filterSymbolEntries(idx.Symbols, cfg)
	idx.Docs = filterDocEntries(idx.Docs, cfg)
	return idx
}

func filterFileEntries(files []indexer.FileEntry, cfg HumanConfig) []indexer.FileEntry {
	out := make([]indexer.FileEntry, 0, len(files))
	for _, file := range files {
		if !configAllowsPath(file.Path, cfg.IncludePaths, cfg.ExcludePaths) {
			continue
		}
		out = append(out, file)
	}
	return out
}

func filterSymbolEntries(symbols []indexer.SymbolEntry, cfg HumanConfig) []indexer.SymbolEntry {
	out := make([]indexer.SymbolEntry, 0, len(symbols))
	for _, symbol := range symbols {
		if !configAllowsPath(symbol.Path, cfg.IncludePaths, cfg.ExcludePaths) {
			continue
		}
		out = append(out, symbol)
	}
	return out
}

func filterDocEntries(docs []indexer.DocEntry, cfg HumanConfig) []indexer.DocEntry {
	out := make([]indexer.DocEntry, 0, len(docs))
	for _, doc := range docs {
		if !configAllowsPath(doc.Path, cfg.IncludePaths, cfg.ExcludePaths) {
			continue
		}
		if len(cfg.DocsPaths) > 0 && !matchesAnyConfigPath(doc.Path, cfg.DocsPaths) {
			continue
		}
		out = append(out, doc)
	}
	return out
}

func configAllowsPath(path string, includePaths []string, excludePaths []string) bool {
	if matchesAnyConfigPath(path, excludePaths) {
		return false
	}
	if len(includePaths) == 0 {
		return true
	}
	return matchesAnyConfigPath(path, includePaths)
}

func matchesAnyConfigPath(path string, prefixes []string) bool {
	normalizedPath := normalizeConfigMatchPath(path)
	for _, prefix := range prefixes {
		prefix = normalizeConfigMatchPath(prefix)
		if prefix == "." || prefix == "" {
			return true
		}
		if normalizedPath == prefix || strings.HasPrefix(normalizedPath, prefix+"/") {
			return true
		}
	}
	return false
}

func normalizeConfigMatchPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." {
		return "."
	}
	value = strings.TrimPrefix(value, "./")
	value = strings.TrimPrefix(value, "/")
	return strings.TrimSuffix(value, "/")
}

func SaveIndex(root string, idx indexer.Result) error {
	if err := project.WriteJSON(WorkspaceFile(root, "index", "bundle.json"), idx); err != nil {
		return err
	}
	if err := project.WriteJSON(WorkspaceFile(root, "index", "files.json"), idx.Files); err != nil {
		return err
	}
	if err := project.WriteJSON(WorkspaceFile(root, "index", "symbols.json"), idx.Symbols); err != nil {
		return err
	}
	if err := project.WriteJSON(WorkspaceFile(root, "index", "dependencies.json"), idx.Dependencies); err != nil {
		return err
	}
	if err := project.WriteJSON(WorkspaceFile(root, "index", "recent_changes.json"), idx.Recent); err != nil {
		return err
	}
	if err := project.WriteJSON(WorkspaceFile(root, "index", "docs.json"), idx.Docs); err != nil {
		return err
	}
	return project.WriteJSON(WorkspaceFile(root, "index", "tooling.json"), idx.Tooling)
}

func LoadIndex(root string) (indexer.Result, error) {
	var idx indexer.Result
	if err := project.ReadJSON(WorkspaceFile(root, "index", "bundle.json"), &idx); err != nil {
		return indexer.Result{}, err
	}
	return idx, nil
}
