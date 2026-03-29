<p align="center">
  <img src="./assets/arc-logo.svg" alt="ARC logo" width="220" />
</p>

# ARC

Project console for local AI work on real codebases.

English: this file  
Русский: [README.ru.md](./README.ru.md)

## What ARC is

ARC helps you work with AI on a project in a way that stays understandable:

- start from a project, not from a blank chat
- keep one local place for runs, chats, memory, maps, and artifacts
- switch between CLI and desktop without losing context
- inspect what changed before trusting the result

ARC is local-first. Your project stays on disk. The CLI is the runtime. The desktop app is the simpler visual layer on top.

## What is in this repo

- [cmd/arc](./cmd/arc) — the main CLI
- [cmd/arc-desktop-wails](./cmd/arc-desktop-wails) — native desktop launcher
- [apps/desktop](./apps/desktop) — desktop product zone
- [internal/app](./internal/app) — runtime service layer shared by CLI and desktop
- [internal/liveapp](./internal/liveapp) — managed live preview runtime for demos and miniapps

Related repos:

- `arc-docs` — public docs site and end-user guides
- `arc-presets` — preset catalog and preset-spec artifacts
- `arc-editorial` — posts, articles, and rollout content

## Quick start

### 1. Build the CLI

```bash
make build
```

### 2. Check your environment

```bash
./bin/arc doctor
```

### 3. Initialize ARC in your project

```bash
./bin/arc init --path .
```

### 4. Make your first plan

```bash
./bin/arc task plan --path . --mode work --provider codex "Review this project and suggest the next safe change"
```

### 5. Build the native desktop app

```bash
go build -tags wails ./cmd/arc-desktop-wails
```

Canonical `.app` packaging path:

```bash
make desktop-wails-package
```

If you want to validate the raw Wails CLI path separately:

```bash
cd cmd/arc-desktop-wails
GOCACHE=/tmp/agent-os-gocache CGO_ENABLED=1 GOOS=darwin GOARCH=$(uname -m) wails build -platform darwin/$(uname -m) -tags wails -clean -nopackage
```

### 6. Use the desktop preview only when you really need it

```bash
make build-desktop
make desktop-preview
```

The browser preview is now a temporary debug surface. The native desktop app is the main testing path.

### 7. Run desktop tests

```bash
make test-desktop
```

## Main user flows

### CLI

Use the CLI if you want direct control or terminal-first workflows:

- `arc doctor`
- `arc init --path .`
- `arc mode set work --path .`
- `arc task plan --path . ...`
- `arc task run --path . ...`
- `arc run list --path .`
- `arc chat start ...`
- `arc preset list`

Beginner guide: see `arc-docs` for the published CLI guide.

### Desktop

Use the desktop app if you want a simpler, more visual flow:

- `Welcome / Open Project` — start through a recent/open chooser
- `Chat` — the main working surface
- `Study / Work / Hero` — built-in agent presets for different styles of work
- past conversations live as topics in the left rail instead of a separate `Sessions` screen
- demos, plans, and checks open inside the session instead of a separate task manager
- managed local runtime for simple demos and mini-sites directly inside the app, without requiring Docker

Beginner guide: see `arc-docs` for the published desktop guide.

Current default:

- use the native `.app` for regular testing
- start browser preview only when you need a temporary debug fallback
- start docs locally only when you are actively checking docs

## Documentation

Product docs live in the separate `arc-docs` repository.

## Current status

Today the repo already includes:

- working CLI runtime
- local runs, memory, docs maps, and artifacts
- desktop preview
- native Wails desktop shell
- preset preview, install, and rollback flows
- local chat sessions with provider transcripts

Known limits:

- live `codex` runs still depend on the stability of the local `codex` binary
- verification is strongest in Go repositories right now
- the desktop app is already usable, but still moving toward a simpler consumer-grade UX

## For contributors

If you change the product behavior, update the durable project context in [memory_bank](./memory_bank).

Branding contract: the canonical source image for the UI and desktop icon should live in [assets/branding](./assets/branding).
