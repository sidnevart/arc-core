package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-os/internal/project"
)

type verifierSpec struct {
	Definition VerifierDefinition
	ReportName string
	Run        func(root string) VerificationResult
}

func verificationSpecs() []verifierSpec {
	return []verifierSpec{
		{
			Definition: VerifierDefinition{
				ID:       "tech-verifier",
				Title:    "Technical requirements",
				Summary:  "Проверяет scaffold, provider resolution, базовые runtime предпосылки и системные технические риски.",
				Domain:   "technical",
				Blocking: true,
			},
			ReportName: "tech_verification_report.md",
			Run:        runTechVerifier,
		},
		{
			Definition: VerifierDefinition{
				ID:       "ux-verifier",
				Title:    "UX and UI",
				Summary:  "Проверяет, что UI не свален в одну поверхность, а основные user-facing states и экраны читаются быстро.",
				Domain:   "ux",
				Blocking: false,
			},
			ReportName: "ux_verification_report.md",
			Run:        runUXVerifier,
		},
		{
			Definition: VerifierDefinition{
				ID:       "business-verifier",
				Title:    "Business logic",
				Summary:  "Проверяет, что built-in agents реально enforce-ят задуманную продуктовую логику ARC.",
				Domain:   "business",
				Blocking: true,
			},
			ReportName: "business_verification_report.md",
			Run:        runBusinessVerifier,
		},
		{
			Definition: VerifierDefinition{
				ID:       "chat-flow-verifier",
				Title:    "Chat flow",
				Summary:  "Проверяет new-vs-continue semantics, auto-selection ловушки и понятность chat lifecycle.",
				Domain:   "chat",
				Blocking: true,
			},
			ReportName: "chat_verification_report.md",
			Run:        runChatFlowVerifier,
		},
		{
			Definition: VerifierDefinition{
				ID:       "preset-config-verifier",
				Title:    "Preset and config health",
				Summary:  "Проверяет согласованность built-in agents, installed presets, scaffold и provider guidance.",
				Domain:   "preset_config",
				Blocking: true,
			},
			ReportName: "preset_verification_report.md",
			Run:        runPresetConfigVerifier,
		},
		{
			Definition: VerifierDefinition{
				ID:       "testing-verifier",
				Title:    "Testing mode",
				Summary:  "Проверяет dev-only testing harness, сценарии и controls для product tours.",
				Domain:   "testing",
				Blocking: false,
			},
			ReportName: "testing_verification_report.md",
			Run:        runTestingVerifier,
		},
		{
			Definition: VerifierDefinition{
				ID:       "chat-ui-minimalism-verifier",
				Title:    "Chat UI minimalism",
				Summary:  "Проверяет thread-first shell, компактные controls и отсутствие legacy starter/session surfaces в основном chat viewport.",
				Domain:   "chat_ui",
				Blocking: true,
			},
			ReportName: "chat_ui_minimalism_report.md",
			Run:        runChatUIMinimalismVerifier,
		},
	}
}

func verificationProfiles() []VerificationProfile {
	return []VerificationProfile{
		{
			ID:          "pre-ux-overhaul",
			Title:       "Baseline before UX/system work",
			Summary:     "Фиксирует текущую техническую, UX и product baseline перед следующей волной реализации.",
			VerifierIDs: []string{"ux-verifier", "chat-flow-verifier", "business-verifier"},
		},
		{
			ID:          "chat-reliability",
			Title:       "Chat reliability",
			Summary:     "Проверяет session routing, provider resolution и понятность текущего chat path.",
			VerifierIDs: []string{"chat-flow-verifier", "tech-verifier"},
		},
		{
			ID:          "preset-system-health",
			Title:       "Preset system health",
			Summary:     "Проверяет built-in agents, scaffold, provider guidance и install state проекта.",
			VerifierIDs: []string{"preset-config-verifier", "business-verifier"},
		},
		{
			ID:          "runtime-demo-health",
			Title:       "Runtime and demo health",
			Summary:     "Проверяет managed local runtime, preview assumptions и продуктовые риски demo flow.",
			VerifierIDs: []string{"tech-verifier", "ux-verifier"},
		},
		{
			ID:          "testing-mode-health",
			Title:       "Testing mode health",
			Summary:     "Проверяет dev-only testing harness, visibility и step-by-step controls.",
			VerifierIDs: []string{"testing-verifier", "ux-verifier"},
		},
		{
			ID:          "chat-ui-minimalism",
			Title:       "Chat UI minimalism",
			Summary:     "Проверяет минималистичный chat-first shell: topics слева, широкий thread по центру, drawer по требованию и отсутствие starter clutter.",
			VerifierIDs: []string{"chat-ui-minimalism-verifier", "chat-flow-verifier"},
		},
		{
			ID:          "release-readiness",
			Title:       "Release readiness",
			Summary:     "Полный verification pass по техническому, UX, chat, preset и testing слоям.",
			VerifierIDs: []string{"tech-verifier", "ux-verifier", "business-verifier", "chat-flow-verifier", "chat-ui-minimalism-verifier", "preset-config-verifier", "testing-verifier"},
		},
	}
}

func verifierByID(id string) (verifierSpec, bool) {
	for _, spec := range verificationSpecs() {
		if spec.Definition.ID == id {
			return spec, true
		}
	}
	return verifierSpec{}, false
}

func verificationProfileByID(id string) (VerificationProfile, bool) {
	for _, profile := range verificationProfiles() {
		if profile.ID == id {
			return profile, true
		}
	}
	return VerificationProfile{}, false
}

func (Service) Verifiers(root string) ([]VerifierDefinition, error) {
	access, err := Service{}.DeveloperAccessState(root)
	if err != nil {
		return nil, err
	}
	if !access.CanUseTesting {
		return nil, fmt.Errorf("verification доступен только для developer role")
	}
	out := make([]VerifierDefinition, 0, len(verificationSpecs()))
	for _, spec := range verificationSpecs() {
		out = append(out, spec.Definition)
	}
	return out, nil
}

func (Service) VerificationProfiles(root string) ([]VerificationProfile, error) {
	access, err := Service{}.DeveloperAccessState(root)
	if err != nil {
		return nil, err
	}
	if !access.CanUseTesting {
		return nil, fmt.Errorf("verification доступен только для developer role")
	}
	return verificationProfiles(), nil
}

func (s Service) StartVerificationProfile(root string, profileID string) (VerificationRun, error) {
	profile, ok := verificationProfileByID(strings.TrimSpace(profileID))
	if !ok {
		return VerificationRun{}, fmt.Errorf("unknown verification profile %q", profileID)
	}
	return s.runVerification(root, profile.ID, "", profile.Title, profile.Summary, profile.VerifierIDs)
}

func (s Service) RunVerifier(root string, verifierID string) (VerificationRun, error) {
	spec, ok := verifierByID(strings.TrimSpace(verifierID))
	if !ok {
		return VerificationRun{}, fmt.Errorf("unknown verifier %q", verifierID)
	}
	return s.runVerification(root, "", spec.Definition.ID, spec.Definition.Title, spec.Definition.Summary, []string{spec.Definition.ID})
}

func (Service) VerificationStatus(root string, runID string) (VerificationRun, error) {
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return VerificationRun{}, err
	}
	return loadVerificationRun(resolved, runID)
}

func (s Service) runVerification(root string, profileID string, verifierID string, title string, summary string, verifierIDs []string) (VerificationRun, error) {
	access, err := s.DeveloperAccessState(root)
	if err != nil {
		return VerificationRun{}, err
	}
	if !access.CanUseTesting {
		return VerificationRun{}, fmt.Errorf("verification доступен только для developer role")
	}
	resolved, err := project.DiscoverRoot(root)
	if err != nil {
		return VerificationRun{}, err
	}

	run := VerificationRun{
		ID:             newVerificationRunID(),
		ProfileID:      profileID,
		VerifierID:     verifierID,
		Title:          title,
		Summary:        summary,
		Status:         "running",
		OverallVerdict: "pass",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	if err := saveVerificationRun(resolved, run); err != nil {
		return VerificationRun{}, err
	}

	results := make([]VerificationResult, 0, len(verifierIDs))
	for _, id := range verifierIDs {
		spec, ok := verifierByID(id)
		if !ok {
			continue
		}
		result := spec.Run(resolved)
		reportPath := verificationReportPath(resolved, run.ID, spec.ReportName)
		result.ReportPath = reportPath
		if err := project.WriteString(reportPath, renderVerificationReport(run, result)); err != nil {
			run.Status = "failed"
			run.LastError = err.Error()
			run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			_ = saveVerificationRun(resolved, run)
			return run, err
		}
		results = append(results, result)
	}

	run.Results = results
	run.Status = "done"
	run.OverallVerdict = overallVerificationVerdict(results)
	run.BlockingFailures = countBlockingFailures(results)
	run.WarningCount = countVerificationWarnings(results)
	run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	run.SummaryPath = verificationSummaryPath(resolved, run.ID)
	if err := project.WriteJSON(run.SummaryPath, verificationSummaryDocument(run)); err != nil {
		run.Status = "failed"
		run.LastError = err.Error()
		run.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		_ = saveVerificationRun(resolved, run)
		return run, err
	}
	if err := saveVerificationRun(resolved, run); err != nil {
		return VerificationRun{}, err
	}
	return run, nil
}

func runTechVerifier(root string) VerificationResult {
	findings := []VerificationFinding{}

	if hasTODO(readIfExists(filepath.Join(root, ".arc", "provider", "AGENTS.md"))) {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Codex guidance is still TODO",
			Summary:        "Проектный `.arc/provider/AGENTS.md` всё ещё шаблонный и не задаёт реальных правил для Codex.",
			Evidence:       []string{filepath.Join(root, ".arc", "provider", "AGENTS.md")},
			Recommendation: "Заполнить provider guidance и убрать TODO-заглушку из scaffolded файла.",
		})
	}
	if hasTODO(readIfExists(filepath.Join(root, ".arc", "provider", "CLAUDE.md"))) {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Claude guidance is still TODO",
			Summary:        "Проектный `.arc/provider/CLAUDE.md` всё ещё шаблонный и не задаёт реальных правил для Claude.",
			Evidence:       []string{filepath.Join(root, ".arc", "provider", "CLAUDE.md")},
			Recommendation: "Заполнить Claude guidance и включить его в project repair flow.",
		})
	}
	if !dirExists(filepath.Join(root, ".arc", "skills")) {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Project skills scaffold is missing",
			Summary:        "В текущем проекте нет `.arc/skills`, хотя актуальный scaffold ARC уже ожидает project-local skills.",
			Evidence:       []string{filepath.Join(root, ".arc", "skills")},
			Recommendation: "Добавить repair/migration path для старых проектов и материализовать missing `.arc/skills`.",
		})
	}

	providers := Service{}.ProviderHealth(context.Background())
	for _, item := range providers {
		if !item.Installed {
			severity := "warn"
			recommendation := "Поставить provider binary или показать пользователю понятный recovery flow."
			if strings.EqualFold(item.Name, projectDefaultProvider(root)) {
				severity = "fail"
				recommendation = "Default provider проекта должен быть доступен для chat/run flow."
			}
			findings = append(findings, VerificationFinding{
				Severity:       severity,
				Title:          fmt.Sprintf("Provider %s unavailable", item.Name),
				Summary:        "Один из провайдеров недоступен в текущем окружении.",
				Evidence:       nonEmptyStrings(item.BinaryPath, strings.Join(item.Notes, " | ")),
				Recommendation: recommendation,
			})
		}
	}

	for _, path := range []string{
		filepath.Join(root, "internal", "provider", "codex.go"),
		filepath.Join(root, "internal", "provider", "claude.go"),
	} {
		text := readIfExists(path)
		if strings.Contains(text, "exec.LookPath(") {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          "Provider resolution relies on LookPath only",
				Summary:        "Native desktop на macOS уже упирался в GUI PATH mismatch; простого `exec.LookPath` недостаточно.",
				Evidence:       []string{path},
				Recommendation: "Добавить desktop-aware binary resolution с known candidate paths и явным recovery hint в UI.",
			})
		}
	}

	return finalizeVerificationResult(VerificationResult{
		VerifierID: "tech-verifier",
		Title:      "Technical requirements",
		Domain:     "technical",
		Blocking:   true,
		Summary:    "Проверяет scaffold, provider resolution и базовую техническую готовность.",
	}, findings)
}

func runUXVerifier(root string) VerificationResult {
	findings := []VerificationFinding{}
	for _, surface := range []struct {
		app    string
		styles string
		index  string
		name   string
	} {
		{
			name:   "native",
			app:    filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "app.js"),
			styles: filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "styles.css"),
			index:  filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "index.html"),
		},
		{
			name:   "preview",
			app:    filepath.Join(root, "apps", "desktop", "static", "app.js"),
			styles: filepath.Join(root, "apps", "desktop", "static", "styles.css"),
			index:  filepath.Join(root, "apps", "desktop", "static", "index.html"),
		},
	} {
		appText := readIfExists(surface.app)
		stylesText := readIfExists(surface.styles)
		indexText := readIfExists(surface.index)
		if appText == "" || stylesText == "" || indexText == "" {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          "Frontend surface missing",
				Summary:        "Не найден один из desktop/frontend surfaces, поэтому UX baseline неполный.",
				Evidence:       nonEmptyStrings(surface.app, surface.styles, surface.index),
				Recommendation: "Держать native и preview surfaces рядом и не терять второй UI слой.",
			})
			continue
		}

		for _, token := range []string{"chat.readyLabel", "chat.detailsLabel", "chat.resultPanel"} {
			if !strings.Contains(appText, token) {
				findings = append(findings, VerificationFinding{
					Severity:       "fail",
					Title:          "Inspector tabs are incomplete",
					Summary:        "Chat workspace должен сохранять понятный surface `Детали чата` с секциями `Материалы чата / Файлы`, даже если drawer открывается только по требованию.",
					Evidence:       []string{surface.app, token},
					Recommendation: "Держать единый `Детали чата` drawer с материалами темы и обозревателем файлов, не возвращая пустые или дублирующие вкладки.",
				})
			}
		}
		if strings.Contains(indexText, `data-screen="sessions"`) || strings.Contains(indexText, `data-screen="learn"`) {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Legacy top-level navigation returned",
				Summary:        "В актуальном desktop IA верхняя навигация должна оставаться `Chat / Settings / Testing`, без отдельных top-level `Sessions` и `Learn`.",
				Evidence:       []string{surface.index},
				Recommendation: "Держать topics и learning affordances внутри chat/product flow, а не как primary navigation.",
			})
		}
		if !containsAll(appText, "renderSessionDrawer(", "renderThreadOutputStrip(") || !containsAll(stylesText, ".chat-screen-shell", ".message-thread-scroll", ".chat-rail-list", ".chat-side-drawer") {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          "Thread and drawer split looks incomplete",
				Summary:        "UX baseline ARC теперь опирается на split `topics / thread` плюс on-demand drawer; если drawer contract пропал, shell быстро станет перегруженным или потеряет документы/детали.",
				Evidence:       []string{surface.app, surface.styles},
				Recommendation: "Сохранять desktop shell с topics rail, доминирующим thread и отдельным drawer для документов и деталей.",
			})
		}
		if !containsAll(appText, "renderMarkdown(", "renderMessageOutputs(", "message.failure") || !containsAll(stylesText, ".markdown-body", ".message-output-stack", ".inline-banner") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Markdown and inline output rendering are incomplete",
				Summary:        "Нормальный chat path должен рендерить assistant markdown, inline visual outputs и failure states прямо внутри thread, а не скатываться к plain text.",
				Evidence:       []string{surface.app, surface.styles},
				Recommendation: "Держать markdown-first bubbles, inline output stack и явные failure banners в thread.",
			})
		}
		if !containsAll(indexText, `id="display-overlay"`) || !containsAll(appText, "renderDisplayOverlay(", "settings.displayScale", "CHAT_SCALE_LIMITS") || !containsAll(stylesText, ".display-dialog", ".display-control-row") {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          "Display scale surface looks incomplete",
				Summary:        "Процентный Display scale должен быть доступен через отдельную surface, а не только как внутренняя настройка без UI.",
				Evidence:       []string{surface.index, surface.app, surface.styles},
				Recommendation: "Держать display overlay/dialog и percentage scale controls синхронизированными на native и preview surfaces.",
			})
		}
	}

	return finalizeVerificationResult(VerificationResult{
		VerifierID: "ux-verifier",
		Title:      "UX and UI",
		Domain:     "ux",
		Blocking:   false,
		Summary:    "Проверяет, что chat workspace остаётся широким thread-first shell без возвращения legacy navigation и постоянного right inspector.",
	}, findings)
}

func runBusinessVerifier(root string) VerificationResult {
	findings := []VerificationFinding{}
	agents := builtInAgents()
	if len(agents) != 3 {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Built-in agents list is incomplete",
			Summary:        "ARC должен иметь явные first-party agents `Study / Work / Hero`.",
			Recommendation: "Сохранять built-in agents как ядро продукта и не прятать их за сырой mode-конфиг.",
		})
	}

	study := allowedActionsForMode("study")
	if study.CanDo || study.CanSafeRun {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Study autonomy is too permissive",
			Summary:        "Study не должен quietly уходить в safe/full run как обычный исполнитель.",
			Evidence:       []string{fmt.Sprintf("study: safe=%t do=%t", study.CanSafeRun, study.CanDo)},
			Recommendation: "Оставить Study teaching-first и трансформировать неподходящие действия в объяснение, план и practice flow.",
		})
	}

	work := allowedActionsForMode("work")
	if !work.DoRequiresUnlock {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Work lost autonomy unlock gate",
			Summary:        "Work должен требовать explicit unlock перед более автономным действием.",
			Evidence:       []string{fmt.Sprintf("work: do=%t unlock=%t", work.CanDo, work.DoRequiresUnlock)},
			Recommendation: "Сохранять collaborative-by-default semantics и явный unlock только на конкретный запуск.",
		})
	}

	hero := allowedActionsForMode("hero")
	if !hero.CanDo {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Hero cannot execute autonomously",
			Summary:        "Hero должен уметь брать bounded delivery path под guardrails ARC.",
			Evidence:       []string{fmt.Sprintf("hero: do=%t", hero.CanDo)},
			Recommendation: "Сохранять Hero как autonomous preset, а не просто ещё один explanatory mode.",
		})
	}

	return finalizeVerificationResult(VerificationResult{
		VerifierID: "business-verifier",
		Title:      "Business logic",
		Domain:     "business",
		Blocking:   true,
		Summary:    "Проверяет, что built-in agents соблюдают ключевую продуктовую семантику ARC.",
	}, findings)
}

func runChatFlowVerifier(root string) VerificationResult {
	findings := []VerificationFinding{}
	for _, path := range []string{
		filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "app.js"),
		filepath.Join(root, "apps", "desktop", "static", "app.js"),
	} {
		text := readIfExists(path)
		if text == "" {
			continue
		}
		if strings.Contains(text, "if (!state.selectedSessionId && state.sessions.length)") || strings.Contains(text, "await loadSessionDetail(state.sessions[0].id)") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Old sessions are auto-selected",
				Summary:        "После открытия проекта UI всё ещё может форсированно подхватить старую сессию, из-за чего сообщения идут «не туда».",
				Evidence:       []string{path},
				Recommendation: "Default state должен быть `Новый разговор`, а продолжение последней сессии — только по явному CTA.",
			})
		}
		if !containsAll(text, "composerSending", "renderLiveWorkStatus(") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Async reply lifecycle is not visible in chat",
				Summary:        "После отправки сообщения пользователь должен видеть явный pending/live state прямо в thread, а не просто ждать молча.",
				Evidence:       []string{path},
				Recommendation: "Держать локальный sending state в composer и отдельный inline work/failure block в переписке.",
			})
		}
		if !containsAll(text, "providerHealth(", "sessionError(", `"chat.providerMissing"`, `"chat.failed"`) {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Provider and failure states are not surfaced clearly",
				Summary:        "Chat path обязан явно показывать недоступный provider, failed reply и recovery next step, а не оставлять разговор в подвешенном состоянии.",
				Evidence:       []string{path},
				Recommendation: "Проверять provider до отправки и показывать ошибки/last_error прямо внутри thread lifecycle.",
			})
		}
		if !strings.Contains(text, "state.sessionDetail = detail") {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          "Pending chat detail is not hydrated immediately",
				Summary:        "Если UI не подхватывает running detail сразу после chat start/send, разговор будет ощущаться «глухим» до следующего poll.",
				Evidence:       []string{path},
				Recommendation: "Сразу гидрировать pending session detail после async send/start, а уже потом полагаться на poll/update цикл.",
			})
		}
		if !containsAll(text, "renderMessageOutputs(", "data-launch-material", "data-stop-live-app") {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          "Rich output actions are not clearly wired",
				Summary:        "Если normal chat не держит inline outputs и actions для demo/simulation lifecycle, пользователь не увидит обещанные схемы и миниприложения.",
				Evidence:       []string{path},
				Recommendation: "Сохранять inline output rendering и lifecycle actions для launch/restart/stop внутри chat flow.",
			})
		}
		if !strings.Contains(text, "\"chat.new\"") {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          "No explicit new conversation affordance",
				Summary:        "Chat UX должен явно показывать, как начать новый разговор, а не только продолжать выбранный.",
				Evidence:       []string{path},
				Recommendation: "Сохранять отдельное понятное действие `Новый разговор` рядом с composer.",
			})
		}
	}

	return finalizeVerificationResult(VerificationResult{
		VerifierID: "chat-flow-verifier",
		Title:      "Chat flow",
		Domain:     "chat",
		Blocking:   true,
		Summary:    "Проверяет new-vs-continue semantics и ловушки текущего chat routing.",
	}, findings)
}

func runChatUIMinimalismVerifier(root string) VerificationResult {
	findings := []VerificationFinding{}
	for _, surface := range []struct {
		app    string
		styles string
		index  string
		name   string
	}{
		{
			name:   "native",
			app:    filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "app.js"),
			styles: filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "styles.css"),
			index:  filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "index.html"),
		},
		{
			name:   "preview",
			app:    filepath.Join(root, "apps", "desktop", "static", "app.js"),
			styles: filepath.Join(root, "apps", "desktop", "static", "styles.css"),
			index:  filepath.Join(root, "apps", "desktop", "static", "index.html"),
		},
	} {
		appText := readIfExists(surface.app)
		stylesText := readIfExists(surface.styles)
		indexText := readIfExists(surface.index)
		if appText == "" || stylesText == "" || indexText == "" {
			findings = append(findings, VerificationFinding{
				Severity:       "warn",
				Title:          fmt.Sprintf("%s chat UI surface missing", strings.Title(surface.name)),
				Summary:        "Нельзя полноценно проверить chat minimalism, если один из frontend surfaces не materialized.",
				Evidence:       nonEmptyStrings(surface.app, surface.styles, surface.index),
				Recommendation: "Держать native и preview фронтенды materialized и проверяемыми одновременно.",
			})
			continue
		}
		if strings.Contains(indexText, `data-screen="sessions"`) || strings.Contains(indexText, `data-screen="learn"`) {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Separate sessions or learn top-level screen returned",
				Summary:        "Chat minimalism ломается, когда `Sessions` или `Learn` снова становятся primary navigation surface.",
				Evidence:       []string{surface.index},
				Recommendation: "Оставить topics в левой rail и не возвращать top-level `Sessions`/`Learn` в основном desktop IA.",
			})
		}
		if strings.Contains(appText, "chat-continue-last") || strings.Contains(appText, "chat-fresh-thread") || strings.Contains(appText, "renderStarterScenarios(") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Starter clutter returned to the main chat viewport",
				Summary:        "Первый viewport не должен снова заполняться starter scenarios, `Continue last`, `Fresh thread` и похожими onboarding/promotional controls.",
				Evidence:       []string{surface.app},
				Recommendation: "Оставить базовый path как `type -> send`, а новый разговор держать только компактным rail control.",
			})
		}
		if !strings.Contains(appText, `renderAgentMenu("project"`) {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Project agent picker is missing from primary chat chrome",
				Summary:        "В актуальном ARC UX должен остаться один видимый chooser агента — проектный picker в верхнем chrome.",
				Evidence:       []string{surface.app},
				Recommendation: "Держать project agent picker в верхнем chrome и не прятать его глубоко в secondary settings.",
			})
		}
		if strings.Contains(appText, `renderAgentMenu("session"`) {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Duplicate agent chooser returned to the thread",
				Summary:        "После упрощения chat UX в основном интерфейсе не должно быть второго chooser-а для topic/session agent.",
				Evidence:       []string{surface.app},
				Recommendation: "Оставить только project agent chooser, а agent snapshot темы показывать максимум как read-only metadata.",
			})
		}
		if strings.Contains(appText, "renderActionMenu(") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Legacy action selector returned to normal chat UI",
				Summary:        "Обычный chat flow больше не должен показывать Explain / Plan / Safe / Do как видимый control. Агент должен понимать намерение из самого сообщения.",
				Evidence:       []string{surface.app},
				Recommendation: "Убрать visible action selector из обычного чата и оставить эти режимы только во внутреннем routing/testing layer.",
			})
		}
		if !containsAll(appText, "renderThreadOutputStrip(", "renderMessageOutputs(", "data-open-output-inline", "data-open-output") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Hybrid output routing regressed",
				Summary:        "Актуальный chat-native UX ожидает inline message outputs в thread и document/detail routing через inspector.",
				Evidence:       []string{surface.app},
				Recommendation: "Держать visual outputs inline в thread через structured message outputs, а документы/детали открывать через drawer tabs.",
			})
		}
		if !containsAll(appText, "renderSessionDrawer(", "data-close-drawer", "chat-side-drawer") || !containsAll(stylesText, "body {", "overflow: hidden;", ".chat-rail-list", ".message-thread-scroll", ".chat-side-drawer", ".chat-drawer-panel") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Fixed-shell scroll model regressed",
				Summary:        "Desktop chat shell должен держать global page без длинного scroll и давать независимый scroll только rail/thread/drawer.",
				Evidence:       []string{surface.styles},
				Recommendation: "Сохранять fixed-shell layout с локальными scroll containers для topics, thread и drawer.",
			})
		}
		if strings.Contains(stylesText, "data-chat-density") {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Legacy density presets leaked into the chat shell",
				Summary:        "После перехода к percentage-based Display scale chat shell не должен зависеть от старых compact/default/comfortable density selectors.",
				Evidence:       []string{surface.styles},
				Recommendation: "Оставить numeric chat scale tokens и убрать legacy data-chat-density selectors из frontend surfaces.",
			})
		}
	}

	return finalizeVerificationResult(VerificationResult{
		VerifierID: "chat-ui-minimalism-verifier",
		Title:      "Chat UI minimalism",
		Domain:     "chat_ui",
		Blocking:   true,
		Summary:    "Проверяет thread-first chat shell, один visible agent picker, drawer по требованию и отсутствие clutter в основном chat viewport.",
	}, findings)
}

func runPresetConfigVerifier(root string) VerificationResult {
	findings := []VerificationFinding{}
	builtInPresetCount := 0
	for _, agent := range builtInAgents() {
		expected := filepath.Join(root, "presets", "official", agent.ID)
		if pathExists(expected) {
			builtInPresetCount++
			continue
		}
		findings = append(findings, VerificationFinding{
			Severity:       "warn",
			Title:          fmt.Sprintf("Built-in agent %s is not backed by preset files", agent.Name),
			Summary:        "Built-in agent уже есть в UI и policy layer, но не материализован как preset-like artifact в `presets/official`.",
			Evidence:       []string{expected},
			Recommendation: "Либо сделать built-in agents machine-readable preset definitions, либо честно отделить их от preset install model.",
		})
	}

	installedPath := filepath.Join(root, ".arc", "presets", "installed.json")
	var installed []map[string]any
	if err := project.ReadJSON(installedPath, &installed); err != nil {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Installed presets state is unreadable",
			Summary:        "Проектный install state не читается, поэтому preset lifecycle нельзя считать надёжным.",
			Evidence:       []string{installedPath, err.Error()},
			Recommendation: "Держать `.arc/presets/installed.json` валидным и включить его в repair flow.",
		})
	} else if len(installed) == 0 && builtInPresetCount == 0 {
		findings = append(findings, VerificationFinding{
			Severity:       "warn",
			Title:          "No installed presets recorded",
			Summary:        "Проект не хранит никаких installed presets, хотя UI уже опирается на модель built-in agents и preset ecosystem.",
			Evidence:       []string{installedPath},
			Recommendation: "Согласовать built-in agents с preset/install model и добавить project repair path для старых workspaces.",
		})
	}

	if !dirExists(filepath.Join(root, ".arc", "skills")) {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "Project-local skills are missing",
			Summary:        "Старый project scaffold не мигрирован: `.arc/skills` отсутствует, хотя ARC уже рассчитывает на local skills layer.",
			Evidence:       []string{filepath.Join(root, ".arc", "skills")},
			Recommendation: "Добавить repair/migration path для старых workspaces и привести `.arc` к текущему scaffold.",
		})
	}

	if !dirExists(filepath.Join(root, ".arc", "hooks")) {
		findings = append(findings, VerificationFinding{
			Severity:       "warn",
			Title:          "Hooks layer is not scaffolded",
			Summary:        "В проекте нет materialized hooks directory, хотя продуктовая модель ARC уже подразумевает hookable ecosystem.",
			Evidence:       []string{filepath.Join(root, ".arc", "hooks")},
			Recommendation: "Либо добавить hooks в scaffold/install flow, либо временно убрать эти обещания из user-facing описаний.",
		})
	}

	return finalizeVerificationResult(VerificationResult{
		VerifierID: "preset-config-verifier",
		Title:      "Preset and config health",
		Domain:     "preset_config",
		Blocking:   true,
		Summary:    "Проверяет согласованность built-in agents, preset state и project scaffold.",
	}, findings)
}

func runTestingVerifier(root string) VerificationResult {
	findings := []VerificationFinding{}
	access, _ := Service{}.DeveloperAccessState(root)
	if access.CanUseTesting && access.Role == "developer" && access.Source == "default_local_dev" {
		findings = append(findings, VerificationFinding{
			Severity:       "warn",
			Title:          "Testing defaults to developer mode locally",
			Summary:        "Сейчас developer access включается локально по умолчанию, без отдельной auth/capability handoff для обычного пользователя.",
			Evidence:       []string{"internal/app/testing.go: DeveloperAccessState"},
			Recommendation: "Сохранить dev-only gating, но вынести его в явный capability/auth-ready слой до публичного desktop release.",
		})
	}

	scenarios := testingScenarios()
	if len(scenarios) == 0 {
		findings = append(findings, VerificationFinding{
			Severity:       "fail",
			Title:          "No testing scenarios configured",
			Summary:        "Dev-only testing harness должен иметь хотя бы несколько product walkthrough scenarios.",
			Recommendation: "Держать сценарии для diagrams, docs, demos, sessions и policy-checks.",
		})
	}
	for _, scenario := range scenarios {
		if len(scenario.Steps) == 0 {
			findings = append(findings, VerificationFinding{
				Severity:       "fail",
				Title:          "Scenario has no steps",
				Summary:        "Testing scenario без шагов бесполезен как guided demo harness.",
				Evidence:       []string{scenario.ID},
				Recommendation: "Каждый сценарий должен явно описывать демонстрируемые product capabilities.",
			})
		}
	}

	for _, path := range []string{
		filepath.Join(root, "apps", "desktop", "wailsapp", "frontend", "app.js"),
		filepath.Join(root, "apps", "desktop", "static", "app.js"),
	} {
		text := readIfExists(path)
		for _, token := range []string{"data-testing-next", "data-testing-quit", "data-testing-end"} {
			if text != "" && !strings.Contains(text, token) {
				findings = append(findings, VerificationFinding{
					Severity:       "fail",
					Title:          "Testing controls are incomplete",
					Summary:        "Product walkthrough mode должен давать `Next / Quit / End testing`.",
					Evidence:       []string{path, token},
					Recommendation: "Сохранять полный controlled-run control set в dev-only testing UI.",
				})
			}
		}
	}

	return finalizeVerificationResult(VerificationResult{
		VerifierID: "testing-verifier",
		Title:      "Testing mode",
		Domain:     "testing",
		Blocking:   false,
		Summary:    "Проверяет dev-only testing harness и его управляемость.",
	}, findings)
}

func finalizeVerificationResult(result VerificationResult, findings []VerificationFinding) VerificationResult {
	result.Findings = findings
	result.BlockedGoals = blockedGoalsForFindings(findings)
	result.Verdict = "pass"
	result.Severity = "info"
	result.RecommendedNext = "Можно переходить к следующему эпику."
	switch highestFindingSeverity(findings) {
	case "fail":
		result.Verdict = "fail"
		result.Severity = "fail"
		result.RecommendedNext = "Сначала закрыть критичные замечания этого verifier-а, потом считать эпик завершённым."
	case "warn":
		result.Verdict = "warn"
		result.Severity = "warn"
		result.RecommendedNext = "Можно продолжать, но warning points лучше закрыть до следующего product pass."
	}
	if len(findings) == 0 {
		result.Summary = firstNonEmpty(result.Summary, "Критичных замечаний нет.")
		return result
	}
	result.Summary = summarizeFindings(findings)
	return result
}

func highestFindingSeverity(findings []VerificationFinding) string {
	hasWarn := false
	for _, finding := range findings {
		switch strings.ToLower(strings.TrimSpace(finding.Severity)) {
		case "fail":
			return "fail"
		case "warn":
			hasWarn = true
		}
	}
	if hasWarn {
		return "warn"
	}
	return "info"
}

func blockedGoalsForFindings(findings []VerificationFinding) []string {
	goals := []string{}
	for _, finding := range findings {
		if strings.EqualFold(finding.Severity, "fail") {
			goals = append(goals, finding.Title)
		}
	}
	return goals
}

func summarizeFindings(findings []VerificationFinding) string {
	failCount := 0
	warnCount := 0
	for _, finding := range findings {
		switch strings.ToLower(strings.TrimSpace(finding.Severity)) {
		case "fail":
			failCount++
		case "warn":
			warnCount++
		}
	}
	switch {
	case failCount > 0 && warnCount > 0:
		return fmt.Sprintf("%d fail / %d warn", failCount, warnCount)
	case failCount > 0:
		return fmt.Sprintf("%d fail", failCount)
	case warnCount > 0:
		return fmt.Sprintf("%d warn", warnCount)
	default:
		return "informational findings only"
	}
}

func overallVerificationVerdict(results []VerificationResult) string {
	verdict := "pass"
	for _, result := range results {
		switch result.Verdict {
		case "fail":
			return "fail"
		case "warn":
			verdict = "warn"
		}
	}
	return verdict
}

func countBlockingFailures(results []VerificationResult) int {
	count := 0
	for _, result := range results {
		if result.Blocking && result.Verdict == "fail" {
			count++
		}
	}
	return count
}

func countVerificationWarnings(results []VerificationResult) int {
	count := 0
	for _, result := range results {
		if result.Verdict == "warn" {
			count++
		}
	}
	return count
}

func verificationRunPath(root string, runID string) string {
	return project.ProjectFile(root, "testing", "verification", "runs", runID, "run.json")
}

func verificationReportPath(root string, runID string, name string) string {
	return project.ProjectFile(root, "testing", "verification", "runs", runID, name)
}

func verificationSummaryPath(root string, runID string) string {
	return project.ProjectFile(root, "testing", "verification", "runs", runID, "verification_summary.json")
}

func saveVerificationRun(root string, run VerificationRun) error {
	return project.WriteJSON(verificationRunPath(root, run.ID), run)
}

func loadVerificationRun(root string, runID string) (VerificationRun, error) {
	var run VerificationRun
	if err := project.ReadJSON(verificationRunPath(root, runID), &run); err != nil {
		return VerificationRun{}, err
	}
	return run, nil
}

func newVerificationRunID() string {
	return time.Now().UTC().Format("20060102T150405Z") + "-verification"
}

func renderVerificationReport(run VerificationRun, result VerificationResult) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(result.Title)
	b.WriteString("\n\n")
	b.WriteString("- Run: `")
	b.WriteString(run.ID)
	b.WriteString("`\n")
	b.WriteString("- Verifier: `")
	b.WriteString(result.VerifierID)
	b.WriteString("`\n")
	b.WriteString("- Verdict: `")
	b.WriteString(result.Verdict)
	b.WriteString("`\n")
	b.WriteString("- Severity: `")
	b.WriteString(result.Severity)
	b.WriteString("`\n")
	b.WriteString("- Summary: ")
	b.WriteString(result.Summary)
	b.WriteString("\n\n")
	if len(result.Findings) == 0 {
		b.WriteString("No findings.\n")
		return b.String()
	}
	for i, finding := range result.Findings {
		b.WriteString(fmt.Sprintf("## %d. %s [%s]\n\n", i+1, finding.Title, strings.ToUpper(finding.Severity)))
		b.WriteString(finding.Summary)
		b.WriteString("\n\n")
		if len(finding.Evidence) > 0 {
			b.WriteString("Evidence:\n")
			for _, evidence := range finding.Evidence {
				if strings.TrimSpace(evidence) == "" {
					continue
				}
				b.WriteString("- `")
				b.WriteString(evidence)
				b.WriteString("`\n")
			}
			b.WriteString("\n")
		}
		if finding.Recommendation != "" {
			b.WriteString("Recommendation: ")
			b.WriteString(finding.Recommendation)
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func verificationSummaryDocument(run VerificationRun) map[string]any {
	results := make([]map[string]any, 0, len(run.Results))
	for _, result := range run.Results {
		findings := make([]map[string]any, 0, len(result.Findings))
		for _, finding := range result.Findings {
			findings = append(findings, map[string]any{
				"severity":        finding.Severity,
				"title":           finding.Title,
				"summary":         finding.Summary,
				"evidence":        append([]string{}, finding.Evidence...),
				"recommendation":  finding.Recommendation,
			})
		}
		results = append(results, map[string]any{
			"verifier_id":       result.VerifierID,
			"verdict":           result.Verdict,
			"severity":          result.Severity,
			"blocked_goals":     append([]string{}, result.BlockedGoals...),
			"recommended_next":  result.RecommendedNext,
			"report_path":       result.ReportPath,
			"findings":          findings,
		})
	}
	return map[string]any{
		"id":                run.ID,
		"profile_id":        run.ProfileID,
		"verifier_id":       run.VerifierID,
		"title":             run.Title,
		"status":            run.Status,
		"overall_verdict":   run.OverallVerdict,
		"blocking_failures": run.BlockingFailures,
		"warning_count":     run.WarningCount,
		"created_at":        run.CreatedAt,
		"updated_at":        run.UpdatedAt,
		"results":           results,
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func readIfExists(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func hasTODO(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed != "" && strings.Contains(strings.ToUpper(trimmed), "TODO")
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func containsAll(text string, snippets ...string) bool {
	for _, snippet := range snippets {
		if !strings.Contains(text, snippet) {
			return false
		}
	}
	return true
}

func containsAny(text string, snippets ...string) bool {
	for _, snippet := range snippets {
		if strings.Contains(text, snippet) {
			return true
		}
	}
	return false
}
