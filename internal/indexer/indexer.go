package indexer

import (
	"bufio"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-os/internal/project"
)

type FileEntry struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Kind    string `json:"kind"`
	ModTime string `json:"mod_time"`
}

type SymbolEntry struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Language string `json:"language"`
	Line     int    `json:"line"`
}

type DependencyEntry struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Source    string `json:"source"`
}

type ChangeEntry struct {
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Author  string `json:"author"`
	Subject string `json:"subject"`
}

type DocEntry struct {
	Path     string   `json:"path"`
	Title    string   `json:"title"`
	Headings []string `json:"headings,omitempty"`
}

type ToolingStatus struct {
	Git     bool `json:"git"`
	RG      bool `json:"rg"`
	AstGrep bool `json:"ast_grep"`
}

type Result struct {
	Files        []FileEntry       `json:"files"`
	Symbols      []SymbolEntry     `json:"symbols"`
	Dependencies []DependencyEntry `json:"dependencies"`
	Recent       []ChangeEntry     `json:"recent_changes"`
	Docs         []DocEntry        `json:"docs"`
	Tooling      ToolingStatus     `json:"tooling"`
	GeneratedAt  string            `json:"generated_at"`
}

func Build(root string) (Result, error) {
	result := Result{GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	tooling := ToolingStatus{}
	if _, err := exec.LookPath("git"); err == nil {
		tooling.Git = true
	}
	if _, err := exec.LookPath("rg"); err == nil {
		tooling.RG = true
	}
	if _, err := exec.LookPath("ast-grep"); err == nil {
		tooling.AstGrep = true
	}
	result.Tooling = tooling

	fset := token.NewFileSet()
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		if rel == "." {
			return nil
		}

		if d.IsDir() && shouldSkipDir(rel) {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(rel))
		result.Files = append(result.Files, FileEntry{
			Path:    rel,
			Size:    info.Size(),
			Kind:    kindForFile(rel),
			ModTime: info.ModTime().UTC().Format(time.RFC3339),
		})

		if ext == ".go" {
			src, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err == nil {
				ast.Inspect(src, func(node ast.Node) bool {
					switch typed := node.(type) {
					case *ast.FuncDecl:
						result.Symbols = append(result.Symbols, SymbolEntry{Name: typed.Name.Name, Kind: "function", Path: rel, Language: "go", Line: fset.Position(typed.Pos()).Line})
					case *ast.TypeSpec:
						result.Symbols = append(result.Symbols, SymbolEntry{Name: typed.Name.Name, Kind: "type", Path: rel, Language: "go", Line: fset.Position(typed.Pos()).Line})
					case *ast.ValueSpec:
						for _, name := range typed.Names {
							result.Symbols = append(result.Symbols, SymbolEntry{Name: name.Name, Kind: "value", Path: rel, Language: "go", Line: fset.Position(typed.Pos()).Line})
						}
					}
					return true
				})
			}
		}

		if isDocFile(rel) {
			doc := parseDoc(path)
			doc.Path = rel
			result.Docs = append(result.Docs, doc)
		}

		if filepath.Base(rel) == "go.mod" {
			result.Dependencies = append(result.Dependencies, parseGoMod(path)...)
		}
		if filepath.Base(rel) == "package.json" {
			result.Dependencies = append(result.Dependencies, parsePackageJSON(path)...)
		}

		return nil
	})

	if result.Tooling.Git {
		result.Recent = gitRecentChanges(root)
	}

	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].Path < result.Files[j].Path })
	sort.Slice(result.Symbols, func(i, j int) bool {
		return result.Symbols[i].Path+result.Symbols[i].Name < result.Symbols[j].Path+result.Symbols[j].Name
	})
	sort.Slice(result.Dependencies, func(i, j int) bool { return result.Dependencies[i].Name < result.Dependencies[j].Name })
	sort.Slice(result.Docs, func(i, j int) bool { return result.Docs[i].Path < result.Docs[j].Path })
	return result, nil
}

func Save(root string, result Result) error {
	return project.WriteJSON(project.ProjectFile(root, "index", "bundle.json"), result)
}

func WriteIndividual(root string, result Result) error {
	if err := project.WriteJSON(project.ProjectFile(root, "index", "files.json"), result.Files); err != nil {
		return err
	}
	if err := project.WriteJSON(project.ProjectFile(root, "index", "symbols.json"), result.Symbols); err != nil {
		return err
	}
	if err := project.WriteJSON(project.ProjectFile(root, "index", "dependencies.json"), result.Dependencies); err != nil {
		return err
	}
	if err := project.WriteJSON(project.ProjectFile(root, "index", "recent_changes.json"), result.Recent); err != nil {
		return err
	}
	if err := project.WriteJSON(project.ProjectFile(root, "index", "docs.json"), result.Docs); err != nil {
		return err
	}
	return project.WriteJSON(project.ProjectFile(root, "index", "tooling.json"), result.Tooling)
}

func shouldSkipDir(rel string) bool {
	parts := strings.Split(rel, string(filepath.Separator))
	for _, part := range parts {
		switch part {
		case ".git", "node_modules", "vendor":
			return true
		}
	}
	if strings.HasPrefix(rel, ".arc/runs") || strings.HasPrefix(rel, ".arc/evals") || strings.HasPrefix(rel, ".arc/cache") {
		return true
	}
	return false
}

func kindForFile(rel string) string {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go":
		return "go"
	case ".md", ".txt", ".adoc":
		return "doc"
	case ".json", ".yaml", ".yml", ".toml":
		return "config"
	default:
		return "file"
	}
}

func isDocFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt", ".adoc":
		return true
	default:
		return false
	}
}

func parseDoc(path string) DocEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return DocEntry{Title: filepath.Base(path)}
	}
	title := filepath.Base(path)
	headings := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") && title == filepath.Base(path) {
			title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
		if strings.HasPrefix(line, "#") {
			headings = append(headings, strings.TrimSpace(strings.TrimLeft(line, "#")))
		}
		if len(headings) >= 10 {
			break
		}
	}
	return DocEntry{Title: title, Headings: headings}
}

func parseGoMod(path string) []DependencyEntry {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	deps := []DependencyEntry{}
	scanner := bufio.NewScanner(file)
	inBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "require (") {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if strings.HasPrefix(line, "require ") {
			line = strings.TrimPrefix(line, "require ")
		} else if !inBlock {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			deps = append(deps, DependencyEntry{Ecosystem: "go", Name: fields[0], Version: fields[1], Source: filepath.Base(path)})
		}
	}
	return deps
}

func parsePackageJSON(path string) []DependencyEntry {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var payload struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil
	}
	deps := []DependencyEntry{}
	for name, version := range payload.Dependencies {
		deps = append(deps, DependencyEntry{Ecosystem: "npm", Name: name, Version: version, Source: filepath.Base(path)})
	}
	for name, version := range payload.DevDependencies {
		deps = append(deps, DependencyEntry{Ecosystem: "npm-dev", Name: name, Version: version, Source: filepath.Base(path)})
	}
	return deps
}

func gitRecentChanges(root string) []ChangeEntry {
	cmd := exec.Command("git", "-C", root, "log", "-n", "20", "--date=short", "--pretty=format:%h\t%ad\t%an\t%s")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	changes := []ChangeEntry{}
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		changes = append(changes, ChangeEntry{
			Hash:    parts[0],
			Date:    parts[1],
			Author:  parts[2],
			Subject: parts[3],
		})
	}
	return changes
}
