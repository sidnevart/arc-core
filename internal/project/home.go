package project

import (
	"fmt"
	"os"
	"path/filepath"
)

type Home struct {
	Root string
}

func DefaultHome() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, ".arc"), nil
}

func EnsureHome() (Home, error) {
	root, err := DefaultHome()
	if err != nil {
		return Home{}, err
	}

	dirs := []string{
		root,
		filepath.Join(root, "providers"),
		filepath.Join(root, "modes"),
		filepath.Join(root, "skills"),
		filepath.Join(root, "templates"),
		filepath.Join(root, "memory"),
		filepath.Join(root, "cache"),
		filepath.Join(root, "sessions"),
		filepath.Join(root, "logs"),
		filepath.Join(root, "evals"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return Home{}, err
		}
	}

	files := map[string]string{
		filepath.Join(root, "config.yaml"): `default_provider: "codex"
default_mode: "work"
autonomy: "medium"
`,
		filepath.Join(root, "providers", "codex.yaml"): `binary: "codex"
enabled: "true"
`,
		filepath.Join(root, "providers", "claude.yaml"): `binary: "claude"
enabled: "true"
`,
		filepath.Join(root, "memory", "GLOBAL_MEMORY.md"): "# GLOBAL MEMORY\n\n[TODO] Add your durable personal operating preferences here.\n",
	}

	for _, def := range builtInModeDocs() {
		files[filepath.Join(root, "modes", def.Name+".md")] = def.Markdown
	}
	for _, skill := range builtInSkillDocs() {
		files[filepath.Join(root, "skills", skill.Name, "SKILL.md")] = skill.Body
	}

	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return Home{}, err
		}
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return Home{}, err
		}
	}

	return Home{Root: root}, nil
}

type namedContent struct {
	Name     string
	Markdown string
	Body     string
}

func builtInModeDocs() []namedContent {
	return []namedContent{
		{
			Name:     "study",
			Markdown: "# Study Mode\n\nGoal: improve understanding, not replace thinking.\n\nAutonomy: low\n\nRules:\n- Ask clarifying questions first.\n- Use the help ladder.\n- Record knowledge gaps.\n",
		},
		{
			Name:     "work",
			Markdown: "# Work Mode\n\nGoal: improve engineering judgment while still moving tasks forward.\n\nAutonomy: medium\n\nRules:\n- Map the system first.\n- State unknowns before coding.\n- Critique the approach before implementation.\n",
		},
		{
			Name:     "hero",
			Markdown: "# Hero Mode\n\nGoal: close ticket-sized work autonomously with guardrails.\n\nAutonomy: high\n\nRules:\n- Do not invent unknowns.\n- Gather evidence first.\n- Produce spec, verification, review, and docs artifacts.\n",
		},
	}
}

func builtInSkillDocs() []namedContent {
	templates := map[string]string{
		"plan-task": `---
name: plan-task
description: Use when planning a task into concrete specs, unknowns, and artifact outputs.
---

# Plan Task

Build a concise task spec, identify unknowns, and define the next executable slice.
`,
		"review-spec": `---
name: review-spec
description: Use when checking a draft spec for missing assumptions, unclear requirements, and risk.
---

# Review Spec

Find holes, contradictions, and unsupported assumptions in a draft task spec.
`,
		"request-context": `---
name: request-context
description: Use when missing context blocks implementation and a precise question bundle is needed.
---

# Request Context

Gather and package only the context still missing after local search.
`,
		"implement-feature": `---
name: implement-feature
description: Use when converting an approved spec into code, tests, and implementation notes.
---

# Implement Feature

Implement the smallest useful slice and record what changed.
`,
		"verify-changes": `---
name: verify-changes
description: Use when validating a change with tests, static checks, and explicit residual risks.
---

# Verify Changes

Run the strongest practical checks and summarize what remains unverified.
`,
		"review-code": `---
name: review-code
description: Use when performing an independent bug- and regression-focused review.
---

# Review Code

Focus on defects, regressions, missing tests, and unsupported assumptions.
`,
		"write-docs": `---
name: write-docs
description: Use when generating docs deltas after implementation or verification.
---

# Write Docs

Capture the operator-facing and maintainer-facing documentation changes implied by the run.
`,
		"teach-challenge": `---
name: teach-challenge
description: Use for study-mode help that should challenge the learner instead of dumping the answer.
---

# Teach Challenge

Guide the user up the help ladder and log knowledge gaps.
`,
		"build-visualizer": `---
name: build-visualizer
description: Use when a flow, diagram, or small demo would clarify the system or concept.
---

# Build Visualizer

Create a compact visual aid or demo if it materially improves understanding.
`,
		"compact-memory": `---
name: compact-memory
description: Use when compressing run memory into durable facts, decisions, and open questions.
---

# Compact Memory

Retain only stable facts, decisions, and unresolved questions.
`,
	}

	skills := make([]namedContent, 0, len(templates))
	for name, body := range templates {
		skills = append(skills, namedContent{Name: name, Body: body})
	}
	return skills
}

func HomeSummary(home Home) string {
	return fmt.Sprintf("global home: %s", home.Root)
}
