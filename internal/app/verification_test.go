package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-os/internal/project"
)

func TestRunVerifierPresetConfigDetectsMissingSkills(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".arc", "skills")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService().SetDeveloperRole(root, "developer"); err != nil {
		t.Fatal(err)
	}

	run, err := NewService().RunVerifier(root, "preset-config-verifier")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "done" {
		t.Fatalf("expected done run, got %#v", run)
	}
	result := findVerificationResult(run, "preset-config-verifier")
	if result == nil {
		t.Fatalf("expected preset-config-verifier result, got %#v", run.Results)
	}
	if result.Verdict != "fail" {
		t.Fatalf("expected failing verifier, got %#v", result)
	}
	if !hasFindingContaining(result.Findings, "Project-local skills are missing") {
		t.Fatalf("expected missing skills finding, got %#v", result.Findings)
	}
	if _, err := os.Stat(run.SummaryPath); err != nil {
		t.Fatalf("expected summary file, got %v", err)
	}
}

func TestRunVerifierChatFlowDetectsAutoSelectedSessions(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	frontendDir := filepath.Join(root, "apps", "desktop", "wailsapp", "frontend")
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(frontendDir, "app.js"), []byte(`
if (!state.selectedSessionId && state.sessions.length) {
  await loadSessionDetail(state.sessions[0].id);
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService().SetDeveloperRole(root, "developer"); err != nil {
		t.Fatal(err)
	}

	run, err := NewService().RunVerifier(root, "chat-flow-verifier")
	if err != nil {
		t.Fatal(err)
	}
	result := findVerificationResult(run, "chat-flow-verifier")
	if result == nil {
		t.Fatalf("expected chat-flow-verifier result, got %#v", run.Results)
	}
	if result.Verdict != "fail" {
		t.Fatalf("expected fail, got %#v", result)
	}
	if !hasFindingContaining(result.Findings, "Old sessions are auto-selected") {
		t.Fatalf("expected auto-selection finding, got %#v", result.Findings)
	}
}

func TestStartVerificationProfileWritesReports(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService().SetDeveloperRole(root, "developer"); err != nil {
		t.Fatal(err)
	}

	run, err := NewService().StartVerificationProfile(root, "release-readiness")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "done" {
		t.Fatalf("expected done run, got %#v", run)
	}
	if len(run.Results) == 0 {
		t.Fatalf("expected results, got %#v", run)
	}
	for _, result := range run.Results {
		if strings.TrimSpace(result.ReportPath) == "" {
			t.Fatalf("expected report path for %#v", result)
		}
		if _, err := os.Stat(result.ReportPath); err != nil {
			t.Fatalf("expected report %q: %v", result.ReportPath, err)
		}
	}
	if _, err := os.Stat(run.SummaryPath); err != nil {
		t.Fatalf("expected summary %q: %v", run.SummaryPath, err)
	}
}

func TestRunVerifierChatUIMinimalismDetectsLegacySurfaces(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	writeDesktopSurface(t, root, "wailsapp/frontend", `
function renderChat() {
  return renderStarterScenarios();
}
const action = "chat-continue-last";
const other = "chat-fresh-thread";
`, `
body {
  overflow: auto;
}
`, `
<nav>
  <button data-screen="chat">Chat</button>
  <button data-screen="sessions">Sessions</button>
  <button data-screen="learn">Learn</button>
</nav>
`)
	writeDesktopSurface(t, root, "static", `
function renderChat() {
  return renderAgentMenu("project");
}
`, `
body {
  overflow: hidden;
}
.chat-rail-list {
  overflow: auto;
}
.message-thread-scroll {
  overflow: auto;
}
.session-panel {
  overflow: auto;
}
`, `<section id="screen-chat"></section>`)
	if _, err := NewService().SetDeveloperRole(root, "developer"); err != nil {
		t.Fatal(err)
	}

	run, err := NewService().RunVerifier(root, "chat-ui-minimalism-verifier")
	if err != nil {
		t.Fatal(err)
	}
	result := findVerificationResult(run, "chat-ui-minimalism-verifier")
	if result == nil {
		t.Fatalf("expected chat-ui-minimalism-verifier result, got %#v", run.Results)
	}
	if result.Verdict != "fail" {
		t.Fatalf("expected fail, got %#v", result)
	}
	if !hasFindingContaining(result.Findings, "Separate sessions or learn top-level screen returned") {
		t.Fatalf("expected top-level screen finding, got %#v", result.Findings)
	}
	if !hasFindingContaining(result.Findings, "Starter clutter returned to the main chat viewport") {
		t.Fatalf("expected starter clutter finding, got %#v", result.Findings)
	}
}

func TestStartVerificationProfileChatUIMinimalismPassesWithThreadFirstShell(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex", "claude"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	healthyApp := `
const label = "chat.new";
function renderChat() {
  return renderAgentMenu("project") + renderSessionDrawer() + renderThreadOutputStrip() + renderLiveWorkStatus() + renderMessageOutputs() + renderDisplayOverlay();
}
const outputA = "data-open-output";
const outputB = "data-open-output-inline";
const drawer = "chat-side-drawer";
const drawerClose = "data-close-drawer";
const launch = "data-launch-material";
const stop = "data-stop-live-app";
const tabs = ["chat.readyLabel", "chat.detailsLabel", "chat.resultPanel"];
const sending = "composerSending";
const provider = "providerHealth(";
const error = "sessionError(";
const providerCopy = "chat.providerMissing";
const failedCopy = "chat.failed";
const markdown = "renderMarkdown(";
const outputStyles = "message.failure";
const display = ["settings.displayScale", "CHAT_SCALE_LIMITS"];
function hydrate(detail) { state.sessionDetail = detail; }
`
	healthyStyles := `
body {
  overflow: hidden;
}
.chat-screen-shell {
  display: grid;
}
.chat-rail-list {
  overflow: auto;
}
.message-thread-scroll {
  overflow: auto;
}
.chat-side-drawer {
  position: absolute;
}
.chat-drawer-panel {
  position: absolute;
}
.markdown-body {
  display: grid;
}
.message-output-stack {
  display: grid;
}
.inline-banner {
  margin-top: 8px;
}
.display-dialog {
  width: 480px;
}
.display-control-row {
  display: grid;
}
`
	healthyIndex := `
<nav>
  <button data-screen="chat">Chat</button>
  <button data-screen="settings">Settings</button>
  <button data-screen="testing">Testing</button>
</nav>
<section id="screen-chat" class="chat-screen-shell"></section>
<div id="display-overlay"></div>
`
	writeDesktopSurface(t, root, "wailsapp/frontend", healthyApp, healthyStyles, healthyIndex)
	writeDesktopSurface(t, root, "static", healthyApp, healthyStyles, healthyIndex)
	if _, err := NewService().SetDeveloperRole(root, "developer"); err != nil {
		t.Fatal(err)
	}

	run, err := NewService().StartVerificationProfile(root, "chat-ui-minimalism")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "done" {
		t.Fatalf("expected done run, got %#v", run)
	}
	if run.OverallVerdict != "pass" {
		t.Fatalf("expected pass profile, got %#v", run)
	}
	if len(run.Results) != 2 {
		t.Fatalf("expected 2 verifier results, got %#v", run.Results)
	}
	for _, result := range run.Results {
		if result.Verdict != "pass" {
			t.Fatalf("expected passing verifier result, got %#v", result)
		}
	}
}

func TestRunVerifierChatUIMinimalismDetectsDuplicateTopicAgentChooser(t *testing.T) {
	root := t.TempDir()
	if _, err := project.Init(root, project.InitOptions{
		Provider:         "codex",
		EnabledProviders: []string{"codex"},
		Mode:             "work",
	}); err != nil {
		t.Fatal(err)
	}
	duplicateApp := `
const label = "chat.new";
function renderChat() {
  return renderAgentMenu("project") + renderAgentMenu("session") + renderSessionDrawer() + renderThreadOutputStrip() + renderMessageOutputs() + renderDisplayOverlay();
}
const outputA = "data-open-output";
const outputB = "data-open-output-inline";
const drawer = "chat-side-drawer";
const drawerClose = "data-close-drawer";
const launch = "data-launch-material";
const stop = "data-stop-live-app";
const sending = "composerSending";
const provider = "providerHealth(";
const error = "sessionError(";
const providerCopy = "chat.providerMissing";
const failedCopy = "chat.failed";
const markdown = "renderMarkdown(";
const outputStyles = "message.failure";
const display = ["settings.displayScale", "CHAT_SCALE_LIMITS"];
function hydrate(detail) { state.sessionDetail = detail; }
`
	healthyStyles := `
body {
  overflow: hidden;
}
.chat-screen-shell {
  display: grid;
}
.chat-rail-list {
  overflow: auto;
}
.message-thread-scroll {
  overflow: auto;
}
.chat-side-drawer {
  position: absolute;
}
.chat-drawer-panel {
  position: absolute;
}
.markdown-body {
  display: grid;
}
.message-output-stack {
  display: grid;
}
.inline-banner {
  margin-top: 8px;
}
.display-dialog {
  width: 480px;
}
.display-control-row {
  display: grid;
}
`
	healthyIndex := `
<nav>
  <button data-screen="chat">Chat</button>
  <button data-screen="settings">Settings</button>
  <button data-screen="testing">Testing</button>
</nav>
<section id="screen-chat" class="chat-screen-shell"></section>
<div id="display-overlay"></div>
`
	writeDesktopSurface(t, root, "wailsapp/frontend", duplicateApp, healthyStyles, healthyIndex)
	writeDesktopSurface(t, root, "static", duplicateApp, healthyStyles, healthyIndex)
	if _, err := NewService().SetDeveloperRole(root, "developer"); err != nil {
		t.Fatal(err)
	}

	run, err := NewService().RunVerifier(root, "chat-ui-minimalism-verifier")
	if err != nil {
		t.Fatal(err)
	}
	result := findVerificationResult(run, "chat-ui-minimalism-verifier")
	if result == nil {
		t.Fatalf("expected chat-ui-minimalism-verifier result, got %#v", run.Results)
	}
	if result.Verdict != "fail" {
		t.Fatalf("expected fail, got %#v", result)
	}
	if !hasFindingContaining(result.Findings, "Duplicate agent chooser returned to the thread") {
		t.Fatalf("expected duplicate agent chooser finding, got %#v", result.Findings)
	}
}

func findVerificationResult(run VerificationRun, verifierID string) *VerificationResult {
	for i := range run.Results {
		if run.Results[i].VerifierID == verifierID {
			return &run.Results[i]
		}
	}
	return nil
}

func hasFindingContaining(findings []VerificationFinding, needle string) bool {
	for _, finding := range findings {
		if strings.Contains(finding.Title, needle) || strings.Contains(finding.Summary, needle) {
			return true
		}
	}
	return false
}

func writeDesktopSurface(t *testing.T, root string, relative string, app string, styles string, index string) {
	t.Helper()
	base := filepath.Join(root, "apps", "desktop", relative)
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "app.js"), []byte(app), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "styles.css"), []byte(styles), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
}
