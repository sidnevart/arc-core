package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-os/internal/chat"
	"agent-os/internal/liveapp"
	"agent-os/internal/project"
)

type desktopAccessConfig struct {
	Role      string `json:"role"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type testingScenario struct {
	ID      string
	Title   string
	Summary string
	AgentID string
	Steps   []TestingStep
}

var testingMu sync.Mutex

func testingScenarios() []testingScenario {
	return []testingScenario{
		{
			ID:      "full-product-tour",
			Title:   "Полный тур возможностей",
			Summary: "Показывает встроенных агентов, учебные материалы, документы, демо, поиск по сессиям и policy-checks без ручного тыканья по всему приложению.",
			AgentID: "work",
			Steps: []TestingStep{
				{ID: "study-session", Title: "Study: объяснение и диаграмма", Summary: "Создаёт учебную сессию с объяснением, диаграммой и материалами по клеточному дыханию.", Status: "pending"},
				{ID: "study-demo", Title: "Study: мини-симуляция", Summary: "Поднимает встроенную мини-симуляцию и проверяет встроенный preview.", Status: "pending"},
				{ID: "work-session", Title: "Work: документы и план", Summary: "Создаёт рабочую сессию с планом, документом и summary того, что было подготовлено.", Status: "pending"},
				{ID: "hero-demo", Title: "Hero: автономное демо", Summary: "Показывает более автономный сценарий с демо и готовым summary.", Status: "pending"},
				{ID: "session-search", Title: "Сессии: поиск и продолжение", Summary: "Проверяет поиск по прошлым сессиям и подтверждает, что контекст можно продолжить.", Status: "pending"},
				{ID: "policy-check", Title: "Политики агентов", Summary: "Проверяет, что Study / Work / Hero имеют разные ограничения и ожидаемое поведение.", Status: "pending"},
			},
		},
		{
			ID:      "study-biology-tour",
			Title:   "Study: биология с диаграммой и симуляцией",
			Summary: "Отдельный учебный сценарий по клеточному дыханию с уроком, диаграммой и встроенной симуляцией.",
			AgentID: "study",
			Steps: []TestingStep{
				{ID: "study-session", Title: "Создать учебную сессию", Summary: "Подготовить объяснение, заметки и диаграмму.", Status: "pending"},
				{ID: "study-demo", Title: "Поднять симуляцию", Summary: "Открыть lesson demo внутри ARC.", Status: "pending"},
				{ID: "policy-check", Title: "Проверить ограничения Study", Summary: "Убедиться, что Study не уходит в автономное выполнение задачи.", Status: "pending"},
			},
		},
	}
}

func (Service) DeveloperAccessState(root string) (DeveloperAccessState, error) {
	role := strings.TrimSpace(os.Getenv("ARC_DESKTOP_ROLE"))
	source := "env"
	if role == "" {
		source = "default_user"
		role = "user"
		if resolved, err := project.DiscoverRoot(root); err == nil {
			var cfg desktopAccessConfig
			if err := project.ReadJSON(project.ProjectFile(resolved, "desktop_access.json"), &cfg); err == nil && strings.TrimSpace(cfg.Role) != "" {
				role = strings.TrimSpace(cfg.Role)
				source = "project_config"
			}
		}
	}
	role = normalizeDeveloperRole(role)
	canUseTesting := role == "developer" || role == "owner"
	state := DeveloperAccessState{
		Role:              role,
		CanUseTesting:     canUseTesting,
		CanManageRole:     true,
		AllowedRoles:      []string{"user", "developer", "owner"},
		VisibleScreens:    []string{"chat", "sessions", "learn", "settings"},
		AvailableFeatures: []string{"sessions", "materials", "live_apps"},
		Source:            source,
	}
	if canUseTesting {
		state.VisibleScreens = append(state.VisibleScreens, "testing")
		state.AvailableFeatures = append(state.AvailableFeatures, "testing")
	}
	return state, nil
}

func (Service) SetDeveloperRole(root string, role string) (DeveloperAccessState, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return DeveloperAccessState{}, err
	}
	role = normalizeDeveloperRole(role)
	config := desktopAccessConfig{
		Role:      role,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := project.WriteJSON(project.ProjectFile(resolved, "desktop_access.json"), config); err != nil {
		return DeveloperAccessState{}, err
	}
	return Service{}.DeveloperAccessState(resolved)
}

func normalizeDeveloperRole(role string) string {
	switch strings.TrimSpace(strings.ToLower(role)) {
	case "owner":
		return "owner"
	case "developer":
		return "developer"
	default:
		return "user"
	}
}

func (Service) TestingScenarios(root string) ([]TestingScenarioSummary, error) {
	access, err := Service{}.DeveloperAccessState(root)
	if err != nil {
		return nil, err
	}
	if !access.CanUseTesting {
		return nil, fmt.Errorf("режим тестирования доступен только для developer role")
	}
	scenarios := testingScenarios()
	out := make([]TestingScenarioSummary, 0, len(scenarios))
	for _, scenario := range scenarios {
		out = append(out, TestingScenarioSummary{
			ID:      scenario.ID,
			Title:   scenario.Title,
			Summary: scenario.Summary,
			AgentID: scenario.AgentID,
			Steps:   len(scenario.Steps),
		})
	}
	return out, nil
}

func (s Service) StartTestingScenario(root string, scenarioID string, agentID string, stepMode bool) (TestingRun, error) {
	access, err := s.DeveloperAccessState(root)
	if err != nil {
		return TestingRun{}, err
	}
	if !access.CanUseTesting {
		return TestingRun{}, fmt.Errorf("режим тестирования доступен только для developer role")
	}
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return TestingRun{}, err
	}
	scenario, ok := scenarioByID(scenarioID)
	if !ok {
		return TestingRun{}, fmt.Errorf("unknown testing scenario %q", scenarioID)
	}
	if strings.TrimSpace(agentID) == "" {
		agentID = scenario.AgentID
	}
	run := TestingRun{
		ID:          newTestingRunID(),
		ScenarioID:  scenario.ID,
		Title:       scenario.Title,
		Summary:     scenario.Summary,
		AgentID:     normalizeAgentMode(agentID),
		Status:      "paused",
		StepMode:    stepMode,
		CurrentStep: -1,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		Steps:       cloneTestingSteps(scenario.Steps),
	}
	if err := saveTestingRun(resolved, run); err != nil {
		return TestingRun{}, err
	}
	return run, nil
}

func (Service) TestingStatus(root string, runID string) (TestingRun, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return TestingRun{}, err
	}
	return loadTestingRun(resolved, runID)
}

func (s Service) TestingControl(root string, runID string, action string) (TestingRun, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return TestingRun{}, err
	}
	action = strings.TrimSpace(strings.ToLower(action))
	switch action {
	case "next":
		return s.executeNextTestingStep(resolved, runID)
	case "quit":
		run, err := loadTestingRun(resolved, runID)
		if err != nil {
			return TestingRun{}, err
		}
		run.StepMode = false
		run.Status = "running"
		run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := saveTestingRun(resolved, run); err != nil {
			return TestingRun{}, err
		}
		go s.continueTestingRun(resolved, run.ID)
		return run, nil
	case "end":
		run, err := loadTestingRun(resolved, runID)
		if err != nil {
			return TestingRun{}, err
		}
		run.Status = "ended"
		run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := saveTestingRun(resolved, run); err != nil {
			return TestingRun{}, err
		}
		return run, nil
	default:
		return TestingRun{}, fmt.Errorf("unknown testing action %q", action)
	}
}

func (s Service) continueTestingRun(root string, runID string) {
	for {
		run, err := loadTestingRun(root, runID)
		if err != nil {
			return
		}
		if run.Status == "done" || run.Status == "failed" || run.Status == "ended" || run.StepMode {
			return
		}
		if run.CurrentStep >= len(run.Steps)-1 {
			run.Status = "done"
			run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			_ = saveTestingRun(root, run)
			return
		}
		if _, err := s.executeNextTestingStep(root, runID); err != nil {
			return
		}
		time.Sleep(180 * time.Millisecond)
	}
}

func (s Service) executeNextTestingStep(root string, runID string) (TestingRun, error) {
	testingMu.Lock()
	defer testingMu.Unlock()

	run, err := loadTestingRun(root, runID)
	if err != nil {
		return TestingRun{}, err
	}
	if run.Status == "done" || run.Status == "failed" || run.Status == "ended" {
		return run, nil
	}
	nextIndex := run.CurrentStep + 1
	if nextIndex >= len(run.Steps) {
		run.Status = "done"
		run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := saveTestingRun(root, run); err != nil {
			return TestingRun{}, err
		}
		return run, nil
	}
	run.Status = "running"
	run.Steps[nextIndex].Status = "running"
	run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := saveTestingRun(root, run); err != nil {
		return TestingRun{}, err
	}

	step, sessionID, err := s.performTestingStep(root, run, nextIndex)
	run.CurrentStep = nextIndex
	run.SessionID = firstNonEmpty(sessionID, run.SessionID)
	run.Steps[nextIndex] = step
	run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		run.Status = "failed"
		run.LastError = err.Error()
		run.Steps[nextIndex].Status = "failed"
		run.Steps[nextIndex].Details = firstNonEmpty(run.Steps[nextIndex].Details, err.Error())
	} else if nextIndex >= len(run.Steps)-1 {
		run.Status = "done"
	} else if run.StepMode {
		run.Status = "paused"
	} else {
		run.Status = "running"
	}
	if err := saveTestingRun(root, run); err != nil {
		return TestingRun{}, err
	}
	return run, err
}

func (s Service) performTestingStep(root string, run TestingRun, index int) (TestingStep, string, error) {
	switch run.ScenarioID {
	case "study-biology-tour":
		return s.performStudyTourStep(root, run, index)
	case "full-product-tour":
		return s.performFullTourStep(root, run, index)
	default:
		return run.Steps[index], run.SessionID, fmt.Errorf("unknown scenario %q", run.ScenarioID)
	}
}

func (s Service) performStudyTourStep(root string, run TestingRun, index int) (TestingStep, string, error) {
	switch index {
	case 0:
		return s.createStudySessionStep(root, run.Steps[index], "study")
	case 1:
		return s.launchLessonTestingDemo(root, run.Steps[index], run.SessionID)
	case 2:
		return s.policyCheckStep(run.Steps[index], "study"), run.SessionID, nil
	default:
		return run.Steps[index], run.SessionID, fmt.Errorf("unexpected step index %d", index)
	}
}

func (s Service) performFullTourStep(root string, run TestingRun, index int) (TestingStep, string, error) {
	switch index {
	case 0:
		return s.createStudySessionStep(root, run.Steps[index], "study")
	case 1:
		return s.launchLessonTestingDemo(root, run.Steps[index], run.SessionID)
	case 2:
		return s.createWorkSessionStep(root, run.Steps[index])
	case 3:
		return s.createHeroSessionStep(root, run.Steps[index])
	case 4:
		return s.sessionSearchStep(root, run.Steps[index])
	case 5:
		return s.policyCheckStep(run.Steps[index], "all"), run.SessionID, nil
	default:
		return run.Steps[index], run.SessionID, fmt.Errorf("unexpected step index %d", index)
	}
}

func (s Service) createStudySessionStep(root string, step TestingStep, agent string) (TestingStep, string, error) {
	providerName := projectDefaultProvider(root)
	session, err := chat.Create(chat.CreateOptions{
		Root:     root,
		Provider: providerName,
		Mode:     agent,
		Model:    "",
		Prompt:   "Объясни клеточное дыхание простыми словами через схему, короткий текст и мини-симуляцию.",
	})
	if err != nil {
		return step, "", err
	}
	lessonDir, _, err := ensureLessonDemo(root, "cellular-respiration")
	if err != nil {
		return step, session.ID, err
	}
	notesPath := filepath.Join(lessonDir, "notes.md")
	diagramPath := filepath.Join(lessonDir, "diagram.svg")
	questionsPath := filepath.Join(lessonDir, "self-check.md")
	if _, err := chat.MergeMetadata(root, session.ID, map[string]string{
		"lesson_id":         "cellular-respiration",
		"testing_scenario":  "study-biology-tour",
		"testing_step_mode": "true",
	}); err != nil {
		return step, session.ID, err
	}
	message := "Я подготовил учебный пакет по клеточному дыханию: краткое объяснение, диаграмму пути глюкозы к ATP и вопросы для самопроверки."
	if _, err := chat.AppendAssistantMessage(root, session.ID, message, map[string]string{
		"notes":     notesPath,
		"diagram":   diagramPath,
		"selfcheck": questionsPath,
	}); err != nil {
		return step, session.ID, err
	}
	step.Status = "done"
	step.SessionID = session.ID
	step.Details = "Создана учебная сессия Study с диаграммой, заметками и блоком самопроверки."
	return step, session.ID, nil
}

func (s Service) launchLessonTestingDemo(root string, step TestingStep, sessionID string) (TestingStep, string, error) {
	if strings.TrimSpace(sessionID) == "" {
		return step, sessionID, fmt.Errorf("testing session is missing")
	}
	appDetail, err := s.LaunchLessonDemo(root, sessionID, "cellular-respiration")
	if err != nil {
		return step, sessionID, err
	}
	if _, err := chat.AppendAssistantMessage(root, sessionID, "Мини-симуляция запущена и встроена прямо в ARC. Её можно открыть внутри приложения или отдельно.", map[string]string{}); err != nil {
		return step, sessionID, err
	}
	step.Status = "done"
	step.SessionID = sessionID
	step.LiveAppID = appDetail.ID
	step.PreviewURL = appDetail.PreviewURL
	step.Details = "Мини-симуляция клеточного дыхания поднята локально и доступна как embedded preview."
	return step, sessionID, nil
}

func (s Service) createWorkSessionStep(root string, step TestingStep) (TestingStep, string, error) {
	providerName := projectDefaultProvider(root)
	session, err := chat.Create(chat.CreateOptions{
		Root:     root,
		Provider: providerName,
		Mode:     "work",
		Prompt:   "Подготовь рабочий план, документацию и понятный summary для команды.",
	})
	if err != nil {
		return step, "", err
	}
	dir := testingArtifactsDir(root, session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return step, session.ID, err
	}
	planPath := filepath.Join(dir, "team-plan.md")
	docPath := filepath.Join(dir, "handoff-notes.md")
	if err := os.WriteFile(planPath, []byte("# План работы\n\n1. Сформулировать цель.\n2. Подготовить безопасный шаг.\n3. Проверить результат.\n"), 0o644); err != nil {
		return step, session.ID, err
	}
	if err := os.WriteFile(docPath, []byte("# Документ для команды\n\nЭтот документ показывает, как ARC объясняет, планирует и передаёт следующий шаг человеку.\n"), 0o644); err != nil {
		return step, session.ID, err
	}
	if _, err := chat.AppendAssistantMessage(root, session.ID, "Я собрал рабочий план и короткий документ для команды. Их можно открыть в материалах сессии.", map[string]string{
		"plan": planPath,
		"doc":  docPath,
	}); err != nil {
		return step, session.ID, err
	}
	step.Status = "done"
	step.SessionID = session.ID
	step.Details = "Создана рабочая сессия Work с планом и документом, которые показываются как материалы."
	return step, session.ID, nil
}

func (s Service) createHeroSessionStep(root string, step TestingStep) (TestingStep, string, error) {
	providerName := projectDefaultProvider(root)
	session, err := chat.Create(chat.CreateOptions{
		Root:     root,
		Provider: providerName,
		Mode:     "hero",
		Prompt:   "Сделай демонстрационное мини-приложение и покажи готовый результат.",
	})
	if err != nil {
		return step, "", err
	}
	dir := testingArtifactsDir(root, session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return step, session.ID, err
	}
	indexPath := filepath.Join(dir, "index.html")
	summaryPath := filepath.Join(dir, "delivery-summary.md")
	if err := os.WriteFile(indexPath, []byte(heroDemoHTML()), 0o644); err != nil {
		return step, session.ID, err
	}
	if err := os.WriteFile(summaryPath, []byte("# Готовый результат\n\nHero подготовил мини-демо, которое можно смотреть прямо в ARC.\n"), 0o644); err != nil {
		return step, session.ID, err
	}
	appDetail, err := liveapp.StartStaticPreview(liveapp.StartOptions{
		Root:           root,
		SessionID:      session.ID,
		Title:          "Hero demo",
		Origin:         "testing:" + session.ID,
		Type:           "demo",
		SourcePath:     dir,
		AutoStopAfter:  15 * time.Minute,
		AutoStopPolicy: "testing_15m",
	})
	if err != nil {
		return step, session.ID, err
	}
	if _, err := chat.AppendAssistantMessage(root, session.ID, "Hero подготовил готовое демо и отдельный summary результата.", map[string]string{
		"demo":    indexPath,
		"summary": summaryPath,
	}); err != nil {
		return step, session.ID, err
	}
	step.Status = "done"
	step.SessionID = session.ID
	step.LiveAppID = appDetail.ID
	step.PreviewURL = appDetail.PreviewURL
	step.Details = "Создана Hero-сессия с готовым демо, которое открывается прямо внутри ARC."
	return step, session.ID, nil
}

func (s Service) sessionSearchStep(root string, step TestingStep) (TestingStep, string, error) {
	results, err := s.ListSessions(root, 20, "клеточное", "", "")
	if err != nil {
		return step, "", err
	}
	if len(results) == 0 {
		return step, "", fmt.Errorf("search did not find the study session")
	}
	step.Status = "done"
	step.SessionID = results[0].ID
	step.Details = fmt.Sprintf("Поиск по сессиям нашёл %d результат(ов). Их можно продолжать или подтягивать в текущий контекст.", len(results))
	return step, results[0].ID, nil
}

func (Service) policyCheckStep(step TestingStep, agent string) TestingStep {
	policies := []AllowedSessionActions{}
	if agent == "all" {
		policies = []AllowedSessionActions{
			allowedActionsForMode("study"),
			allowedActionsForMode("work"),
			allowedActionsForMode("hero"),
		}
	} else {
		policies = []AllowedSessionActions{allowedActionsForMode(agent)}
	}
	lines := []string{}
	for _, policy := range policies {
		lines = append(lines, fmt.Sprintf("%s: explain=%t, plan=%t, safe=%t, do=%t, unlock=%t", policy.AgentName, policy.CanExplain, policy.CanPlan, policy.CanSafeRun, policy.CanDo, policy.DoRequiresUnlock))
	}
	step.Status = "done"
	step.Details = strings.Join(lines, "\n")
	return step
}

func scenarioByID(id string) (testingScenario, bool) {
	for _, scenario := range testingScenarios() {
		if scenario.ID == id {
			return scenario, true
		}
	}
	return testingScenario{}, false
}

func cloneTestingSteps(steps []TestingStep) []TestingStep {
	out := make([]TestingStep, len(steps))
	copy(out, steps)
	return out
}

func newTestingRunID() string {
	return time.Now().UTC().Format("20060102T150405Z") + "-testing"
}

func testingRunPath(root string, runID string) string {
	return project.ProjectFile(root, "testing", "runs", runID+".json")
}

func saveTestingRun(root string, run TestingRun) error {
	return project.WriteJSON(testingRunPath(root, run.ID), run)
}

func loadTestingRun(root string, runID string) (TestingRun, error) {
	var run TestingRun
	if err := project.ReadJSON(testingRunPath(root, runID), &run); err != nil {
		return TestingRun{}, err
	}
	return run, nil
}

func testingArtifactsDir(root string, sessionID string) string {
	return project.ProjectFile(root, "testing", "artifacts", sessionID)
}

func projectDefaultProvider(root string) string {
	proj, err := project.Load(root)
	if err != nil {
		return "codex"
	}
	if strings.TrimSpace(proj.Config.DefaultProvider) == "" {
		return "codex"
	}
	return proj.Config.DefaultProvider
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func heroDemoHTML() string {
	return `<!doctype html>
<html lang="ru">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <title>Hero Demo</title>
    <style>
      body{margin:0;font-family:Avenir Next,Segoe UI,sans-serif;background:#f5f4ef;color:#102117}
      .wrap{max-width:980px;margin:0 auto;padding:28px;display:grid;gap:18px}
      .hero,.card{background:#fff;border:1px solid #dde6de;border-radius:24px;padding:22px;box-shadow:0 18px 44px rgba(18,33,23,.08)}
      h1,h2{margin:0 0 12px}
      .grid{display:grid;grid-template-columns:repeat(3,1fr);gap:14px}
      .metric{background:#f7faf8;border-radius:18px;padding:16px}
      @media (max-width:820px){.grid{grid-template-columns:1fr}}
    </style>
  </head>
  <body>
    <div class="wrap">
      <section class="hero">
        <h1>Hero demo</h1>
        <p>Это автономное мини-демо показывает, как ARC может поднять результат прямо внутри приложения и дать отдельную ссылку для просмотра.</p>
      </section>
      <section class="card">
        <h2>Что уже готово</h2>
        <div class="grid">
          <div class="metric"><strong>Демо</strong><div>Запущено локально</div></div>
          <div class="metric"><strong>Результат</strong><div>Готов к просмотру</div></div>
          <div class="metric"><strong>Управление</strong><div>Можно остановить из ARC</div></div>
        </div>
      </section>
    </div>
  </body>
</html>`
}
