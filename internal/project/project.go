package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Event struct {
	Timestamp time.Time         `json:"timestamp"`
	Command   string            `json:"command"`
	Status    string            `json:"status"`
	Details   map[string]string `json:"details,omitempty"`
}

type InitOptions struct {
	Provider         string
	EnabledProviders []string
	Mode             string
}

type ProjectConfig struct {
	Name             string
	DefaultProvider  string
	EnabledProviders []string
	CreatedAt        string
}

type ModeConfig struct {
	Mode       string
	Autonomy   string
	UpdatedAt  string
	Definition string
}

type Project struct {
	Root   string
	ArcDir string
	Config ProjectConfig
	Mode   ModeConfig
}

type Scaffold struct {
	Directories []string
	Files       map[string]string
}

func DefaultScaffold(root string, provider string, enabledProviders []string, mode string, autonomy string) Scaffold {
	enabled := strings.Join(enabledProviders, ",")

	scaffold := Scaffold{
		Directories: []string{
			".arc/provider",
			".arc/maps",
			".arc/memory",
			".arc/chats",
			".arc/presets",
			".arc/presets/backups",
			".arc/presets/reports",
			".arc/skills",
			".arc/hooks",
			".arc/plans",
			".arc/specs",
			".arc/runs",
			".arc/evals",
			".arc/index",
			".arc/cache",
		},
		Files: map[string]string{
			".arc/project.yaml": projectYAML(root, provider, enabled),
			".arc/mode.yaml":    modeYAML(mode, autonomy),
			".arc/provider/AGENTS.md": `# AGENTS

## ARC Codex Guidance

- Follow the active ARC built-in agent policy for this project.
- Prefer plans, docs, diagrams, demos, and explicit verification over guesses.
- Keep changes bounded and evidence-backed.
- If context is missing, ask for it instead of inventing it.
`,
			".arc/provider/CLAUDE.md": `# CLAUDE

## ARC Claude Guidance

- Follow the active ARC built-in agent policy for this project.
- Explain decisions clearly and surface uncertainty instead of guessing.
- Prefer structured plans, reviews, and durable artifacts over vague summaries.
- Keep the human in control when the current agent is Study or Work.
`,
			".arc/maps/REPO_MAP.md": `# REPO MAP

[TODO] Summarize the important code surfaces manually. Do not auto-fill with guesses.
`,
			".arc/maps/DOCS_MAP.md": `# DOCS MAP

[TODO] List the human-authored docs worth reading before implementation.
`,
			".arc/hooks/README.md": `# ARC Hooks

This project can host local ARC hooks here when the runtime and preset system materialize them.
`,
			".arc/memory/MEMORY_ACTIVE.md": `# MEMORY ACTIVE

[TODO] Keep only fresh facts, decisions, and active constraints.
`,
			".arc/memory/MEMORY_ARCHIVE.md": `# MEMORY ARCHIVE

[TODO] Move stale or superseded notes here instead of deleting them.
`,
			".arc/memory/DECISIONS.md": `# DECISIONS

[TODO] Record durable implementation decisions with date and rationale.
`,
			".arc/memory/OPEN_QUESTIONS.md": `# OPEN QUESTIONS

[TODO] Track unresolved questions and blockers here.
`,
			".arc/memory/entries.json":       "[]\n",
			".arc/index/symbols.json":        "{}\n",
			".arc/index/files.json":          "{}\n",
			".arc/index/dependencies.json":   "{}\n",
			".arc/index/recent_changes.json": "{}\n",
			".arc/index/docs.json":           "{}\n",
			".arc/index/tooling.json":        "{}\n",
			".arc/presets/installed.json":    "[]\n",
			".arc/evals/metrics.json": `{
  "study": {"runs": 0, "completed": 0, "blocked": 0, "failed": 0, "docs_reports": 0, "evidence_warnings": 0},
  "work": {"runs": 0, "completed": 0, "blocked": 0, "failed": 0, "docs_reports": 0, "evidence_warnings": 0},
  "hero": {"runs": 0, "completed": 0, "blocked": 0, "failed": 0, "docs_reports": 0, "evidence_warnings": 0}
}
`,
		},
	}
	for _, skill := range builtInSkillDocs() {
		scaffold.Files[filepath.Join(".arc", "skills", skill.Name, "SKILL.md")] = skill.Body
	}
	return scaffold
}

func ApplyScaffold(root string, scaffold Scaffold) ([]string, error) {
	created := make([]string, 0, len(scaffold.Directories)+len(scaffold.Files))

	for _, dir := range scaffold.Directories {
		abs := filepath.Join(root, dir)
		if _, err := os.Stat(abs); errors.Is(err, os.ErrNotExist) {
			created = append(created, abs)
		} else if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return nil, err
		}
	}

	fileNames := make([]string, 0, len(scaffold.Files))
	for name := range scaffold.Files {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)

	for _, name := range fileNames {
		abs := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, err
		}
		if _, err := os.Stat(abs); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := os.WriteFile(abs, []byte(scaffold.Files[name]), 0o644); err != nil {
			return nil, err
		}
		created = append(created, abs)
	}

	sort.Strings(created)
	return created, nil
}

func RequireProject(root string) error {
	projectFile := filepath.Join(root, ".arc", "project.yaml")
	if _, err := os.Stat(projectFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("arc project not initialized in %s; run `arc init` first", root)
		}
		return err
	}

	return nil
}

func WriteMode(root string, mode string, autonomy string) error {
	return os.WriteFile(filepath.Join(root, ".arc", "mode.yaml"), []byte(modeYAML(mode, autonomy)), 0o644)
}

func AppendEvent(root string, event Event) error {
	runsDir := filepath.Join(root, ".arc", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	logPath := filepath.Join(runsDir, "events.jsonl")
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(append(payload, '\n')); err != nil {
		return err
	}

	return nil
}

func projectYAML(root string, provider string, enabledProviders string) string {
	return strings.TrimSpace(fmt.Sprintf(`
name: "%s"
default_provider: "%s"
enabled_providers: "%s"
created_at: "%s"
`, filepath.Base(root), provider, enabledProviders, time.Now().UTC().Format(time.RFC3339))) + "\n"
}

func modeYAML(mode string, autonomy string) string {
	return strings.TrimSpace(fmt.Sprintf(`
mode: "%s"
autonomy: "%s"
updated_at: "%s"
`, mode, autonomy, time.Now().UTC().Format(time.RFC3339))) + "\n"
}
