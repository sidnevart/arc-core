import { callWailsBridge } from "./transport.js";
import {
  applyChatScale,
  bindNativeChatScaleEvents,
  normalizeChatScalePercent,
  CHAT_SCALE_LIMITS,
} from "./shared/density.js";
import { renderMarkdown } from "./shared/markdown.js";
import {
  RECENT_PROJECTS_KEY,
  normalizeLocale,
  parseRecentProjects,
  rememberRecentProject,
  summarizeProjectPath,
} from "./ux.js";
import {
  escapeHTML,
  excerpt,
  parseClientLogs,
  parseStoredList,
} from "./shared/ux.js";

const CLIENT_LOGS_KEY = "arc.desktop.native.clientLogs";
const ATTACHED_SESSIONS_KEY = "arc.desktop.native.attachedSessions";
const CHAT_SCALE_KEY = "arc.desktop.native.chatScalePercent";
const MAX_CLIENT_LOGS = 100;

const state = {
  locale: normalizeLocale(window.localStorage.getItem("arc.desktop.native.locale") || "ru"),
  chatScalePercent: normalizeChatScalePercent(window.localStorage.getItem(CHAT_SCALE_KEY) || 100),
  currentScreen: "chat",
  showWelcome: true,
  path: window.localStorage.getItem("arc.desktop.native.path") || "",
  draftPath: window.localStorage.getItem("arc.desktop.native.path") || "",
  recentProjects: parseRecentProjects(window.localStorage.getItem(RECENT_PROJECTS_KEY)),
  projectState: null,
  workspace: null,
  providers: [],
  agents: [],
  selectedAgent: "work",
  allowedActions: null,
  composerText: "",
  sessions: [],
  selectedSessionId: "",
  sessionDetail: null,
  currentSessionTab: "ready",
  drawerOpen: false,
  displayPanelOpen: false,
  demoPanelOpen: false,
  demoPanelTitle: "",
  demoPanelURL: "",
  demoPanelAppID: "",
  demoPanelOrigin: "",
  demoPanelKind: "demo",
  demoPanelStatus: "",
  demoPanelReason: "",
  projectAgentMenuOpen: false,
  sessionAgentMenuOpen: false,
  sessionSearch: "",
  sessionModeFilter: "",
  sessionStatusFilter: "",
  selectedMaterialId: "",
  selectedFilePath: "",
  selectedFileDetail: null,
  projectMaterials: [],
  workspaceExplorer: null,
  liveApps: [],
  composerSending: false,
  composerStatus: "",
  attachedSessionIDs: parseStoredList(window.localStorage.getItem(ATTACHED_SESSIONS_KEY)),
  developerAccess: null,
  testingScenarios: [],
  testingRun: null,
  verificationProfiles: [],
  verificationRun: null,
  desktopLogPath: "",
  clientLogs: parseClientLogs(window.localStorage.getItem(CLIENT_LOGS_KEY)),
  busy: false,
  busyLabel: "",
  liveActionState: {},
  expandedInlineOutputs: {},
  threadViewportBySession: {},
  renderedThreadSessionId: "",
};

let sessionPollTimer = null;
let testingPollTimer = null;
const preservedThreadFrames = new Map();

const M = {
  ru: {
    "shell.native": "Desktop App",
    "shell.headline": "ARC",
    "shell.subhead": "Рабочее пространство, где разговор с агентом сразу превращается в план, демо и результат.",
    "nav.chat": "Чат",
    "nav.sessions": "Сессии",
    "nav.learn": "Обучение",
    "nav.testing": "Тестирование",
    "nav.settings": "Настройки",
    "topbar.project": "Проект",
    "topbar.projects": "Проекты",
    "topbar.reload": "Обновить",
    "topbar.defaultAgent": "Агент проекта по умолчанию",
    "hero.chat.eyebrow": "Чат",
    "hero.chat.title": "Скажи, что хочешь получить",
    "hero.chat.lead": "Выбери встроенного агента, опиши цель и смотри, как ARC связывает разговор, материалы и живой результат в одну сессию.",
    "hero.sessions.eyebrow": "Сессии",
    "hero.sessions.title": "Все важные сессии под рукой",
    "hero.sessions.lead": "Продолжай прошлые разговоры, поднимай их материалы и возвращай нужный контекст в текущую работу.",
    "hero.learn.eyebrow": "Обучение",
    "hero.learn.title": "Обучение без переключения инструментов",
    "hero.learn.lead": "Study показывает тему текстом, схемой, мини-демо и вопросами на понимание прямо внутри ARC.",
    "hero.settings.eyebrow": "Настройки",
    "hero.settings.title": "Состояние системы и текущего проекта",
    "hero.settings.lead": "Здесь видно подключённых агентов, локальный runtime, лог интерфейса и путь к проекту.",
    "hero.testing.eyebrow": "Тестирование",
    "hero.testing.title": "Автономный тур по возможностям ARC",
    "hero.testing.lead": "Этот экран доступен только developer-role и помогает прогнать весь продукт по шагам, не тыкая всё вручную.",
    "welcome.title": "Открой проект и начни работать с агентом",
    "welcome.lead": "Открой папку проекта. Дальше ARC сам сохранит разговоры, шаги работы, материалы и живые демо локально.",
    "welcome.path": "Путь к проекту",
    "welcome.choose": "Выбрать проект",
    "welcome.open": "Открыть проект",
    "welcome.setup": "Настроить ARC в папке",
    "welcome.ready": "Проект уже готов к работе.",
    "welcome.notReady": "ARC ещё не настроен в этой папке.",
    "welcome.noRecent": "Недавние проекты появятся здесь после первого открытия.",
    "welcome.recent": "Недавние проекты",
    "welcome.hint": "После открытия попадёшь сразу в чат. Отдельный менеджер задач не нужен.",
    "welcome.value1": "Говори с агентом как в обычном чате, а не управляй пайплайном вручную.",
    "welcome.value2": "Смотри планы, документы, схемы и демо прямо внутри приложения.",
    "welcome.value3": "Возвращайся к прошлым сессиям и продолжай работу с нужного места.",
    "welcome.flowTitle": "Что произойдёт дальше",
    "welcome.flow1Title": "Открой проект",
    "welcome.flow1Body": "ARC подхватит разговоры, материалы и локальные демо прямо в папке проекта.",
    "welcome.flow2Title": "Напиши, что нужно",
    "welcome.flow2Body": "Чат остаётся главным входом: просто напиши цель или прикрепи свои файлы.",
    "welcome.flow3Title": "Смотри результат рядом",
    "welcome.flow3Body": "Планы, документы, схемы и демо открываются справа, не ломая сам разговор.",
    "welcome.trustLocal": "Локально",
    "welcome.trustLocalBody": "Разговоры, материалы и демо остаются на твоей машине.",
    "welcome.trustControl": "Под контролем",
    "welcome.trustControlBody": "Запущенные демо можно открыть отдельно или остановить вручную.",
    "welcome.trustReview": "Проверяемо",
    "welcome.trustReviewBody": "История разговоров и результаты остаются привязаны к проекту.",
    "project.missing": "Сначала выбери проект.",
    "project.ready": "Проект готов",
    "project.notReady": "Нужно настроить ARC",
    "project.loading": "Загружаю состояние проекта...",
    "project.opened": "Проект открыт.",
    "project.initialized": "ARC настроен. Можно начинать работу.",
    "project.refreshed": "Состояние проекта обновлено.",
    "status.error": "Ошибка",
    "status.loading": "Загрузка...",
    "chat.agent": "Выбери встроенного агента",
    "chat.starters": "С чего начать",
    "chat.startersHint": "Быстрые сценарии помогают сразу начать разговор без ручной формулировки с нуля.",
    "chat.projectAgent": "Агент проекта",
    "chat.projectAgentHint": "Этот выбор сохраняется в проекте и станет дефолтом для новых разговоров.",
    "chat.sessionAgent": "Агент темы",
    "chat.sessionAgentHint": "Текущая тема уже закреплена за своим агентом.",
    "chat.sessionAgentLocked": "Эта тема уже ведётся выбранным агентом.",
    "chat.projectAgentUpdated": "Агент проекта обновлён.",
    "chat.agentProjectMenu": "Агент проекта",
    "chat.agentTopicMenu": "Агент этой темы",
    "chat.agentAll": "Все агенты",
    "chat.builtInAgents": "Встроенные агенты",
    "chat.installedPresets": "Установленные пресеты",
    "chat.installedPresetHint": "Пока доступны как preset layer проекта.",
    "chat.inProject": "в проекте",
    "chat.actionMenu": "Режим",
    "chat.topicUntitled": "Новый разговор",
    "chat.topicsRail": "Темы",
    "chat.openProject": "Проекты",
    "chat.newTopic": "Новый чат",
    "chat.railDivider": "Темы",
    "chat.liveLabel": "Сейчас запущено",
    "chat.readyLabel": "Материалы чата",
    "chat.detailsLabel": "Файлы",
    "chat.outputs": "Материалы чата",
    "chat.noOutputs": "Ответы, документы, схемы и демо этой темы появятся здесь после первого полезного шага.",
    "chat.detailsHint": "Здесь видны все файлы проекта и ARC-файлы, которые были добавлены через чат.",
    "chat.relatedFiles": "Связанные файлы",
    "chat.openResult": "Открыть детали чата",
    "chat.action": "Что сделать с сообщением",
    "chat.composer": "Напиши цель простыми словами",
    "chat.placeholder": "Например: объясни мне клеточное дыхание через схему и мини-симуляцию.",
    "chat.new": "Новый разговор",
    "chat.send": "Отправить",
    "chat.current": "Текущая сессия",
    "chat.live": "Что делает агент",
    "chat.liveApps": "Запущено сейчас",
    "chat.materials": "Материалы сессии",
    "chat.topics": "Темы разговоров",
    "chat.thread": "Разговор",
    "chat.currentWork": "Что делает агент",
    "chat.readyNow": "Что уже готово",
    "chat.continueLast": "Продолжить последнюю",
    "chat.fresh": "Новый чистый разговор",
    "chat.liveStatus": "Агент работает",
    "chat.projectMaterials": "Файлы ARC",
    "chat.projectMaterialsHint": "Эти файлы сохраняются внутри `.arc/materials/uploads/...` и сразу доступны агенту, не засоряя сам проект.",
    "chat.noProjectMaterials": "Пока нет загруженных ARC-файлов.",
    "chat.upload": "Загрузить материалы",
    "chat.uploadInline": "Добавить файлы",
    "chat.uploaded": "Файлы добавлены в ARC.",
    "chat.uploadBusy": "Загружаю файлы в ARC",
    "chat.materialAttachHint": "Загруженные файлы сохраняются в ARC и сразу становятся частью текущего контекста разговора.",
    "chat.projectMaterialsShort": "Недавно добавлено в ARC",
    "chat.sessions": "Последние сессии",
    "chat.tab.conversation": "Сейчас",
    "chat.tab.materials": "Готово",
    "chat.tab.demos": "Демо",
    "chat.allowedTitle": "Как сейчас будет вести себя агент",
    "chat.unlock": "Разрешить более автономный запуск",
    "chat.studyFallback": "Study не делает задачу за пользователя. Я переведу этот запрос в объяснение и план.",
    "chat.noMessages": "Начни разговор с агентом. ARC сам создаст тему после первого сообщения.",
    "chat.noMaterials": "Материалы появятся здесь после ответа, плана, запуска или демо.",
    "chat.noSession": "Сессия появится после первого сообщения или первого запуска.",
    "chat.noHistory": "История этой сессии появится после первых шагов агента.",
    "chat.idle": "Сессия готова. Можно продолжить разговор, открыть результат или запустить демо.",
    "chat.inlineResults": "Детали чата",
    "chat.inlineVisuals": "Схемы и демо",
    "chat.resultPanel": "Детали чата",
    "chat.inspectorEmpty": "Открой детали чата, чтобы смотреть материалы темы и файлы проекта.",
    "chat.liveNowHint": "Здесь видно, какие демо и симуляции этой темы сейчас реально запущены.",
    "chat.materialsHint": "Здесь лежат ответы, документы, схемы и демо, которые агент уже подготовил.",
    "chat.filesHint": "Файлы проекта и ARC-файлы можно открыть прямо из этой панели.",
    "chat.drawerClose": "Закрыть",
    "chat.waiting": "Агент думает над ответом.",
    "chat.sendingStatus": "Отправляю запрос",
    "chat.planningStatus": "Строю план",
    "chat.safeStatus": "Проверяю безопасно",
    "chat.doStatus": "Запускаю выполнение",
    "chat.failed": "Ответ не получен. Покажи ошибку и реши, что делать дальше.",
    "chat.replyMissing": "Агент завершил шаг без текстового ответа.",
    "chat.providerMissing": "Текущий агент недоступен локально. Проверь настройки провайдера и повтори попытку.",
    "chat.topicAgentFixed": "Эта тема закреплена за агентом",
    "chat.openDocument": "Открыть в деталях чата",
    "chat.openDetails": "Открыть детали чата",
    "files.project": "Файлы проекта",
    "files.arc": "Файлы ARC",
    "files.empty": "Здесь появятся файлы проекта и ARC-файлы, добавленные через чат.",
    "files.delete": "Удалить",
    "sessions.search": "Поиск по прошлым сессиям",
    "sessions.filterAgent": "Агент",
    "sessions.filterStatus": "Статус",
    "sessions.list": "История сессий",
    "sessions.empty": "Пока нет сохранённых сессий. Начни разговор в чате.",
    "sessions.detail": "Подробности сессии",
    "sessions.continue": "Продолжить разговор",
    "sessions.attach": "Подтянуть в текущий контекст",
    "sessions.detach": "Убрать из контекста",
    "sessions.materials": "Материалы",
    "sessions.runs": "Запуски внутри сессии",
    "sessions.next": "Следующий шаг",
    "sessions.updated": "Обновлено",
    "sessions.lastSignal": "Последний сигнал",
    "sessions.total": "Всего сессий",
    "sessions.activeNow": "Активны сейчас",
    "sessions.withMaterials": "С материалами",
    "sessions.summaryLead": "Найди прошлый разговор, быстро пойми чем он закончился и продолжи его без ручного поиска по логам.",
    "sessions.resultLead": "Здесь видно, о чём была сессия, что уже готово и какой следующий шаг стоит сделать.",
    "sessions.highlights": "Что уже готово",
    "sessions.liveNow": "Демо и live preview",
    "sessions.noHighlights": "Когда в сессии появятся документы, схемы или демо, они будут показаны здесь.",
    "learn.title": "Учебные сценарии",
    "learn.lead": "Study-агент умеет не только объяснять текстом, но и собирать схемы, демо и мини-симуляции прямо в приложении.",
    "learn.quickProject.title": "Объясни этот проект",
    "learn.quickProject.summary": "Получить понятное объяснение проекта и точку входа для новичка.",
    "learn.quickBio.title": "Показать тему через схему",
    "learn.quickBio.summary": "Запустить учебный сценарий по клеточному дыханию с диаграммой и мини-симуляцией.",
    "learn.quickPlan.title": "Помоги сделать план",
    "learn.quickPlan.summary": "Быстро превратить задачу в понятный следующий шаг.",
    "learn.bio.title": "Клеточное дыхание",
    "learn.bio.summary": "Попросить агента объяснить, как клетка превращает глюкозу в энергию, с диаграммой и мини-симуляцией.",
    "learn.project.title": "Объясни этот проект",
    "learn.project.summary": "Попросить объяснить проект простым языком: зачем он нужен, как устроен и с чего начать.",
    "learn.use": "Открыть в чате",
    "learn.demo": "Запустить мини-симуляцию",
    "settings.providers": "Доступные агенты",
    "settings.general": "Основное",
    "settings.advanced": "Для разработчика",
    "settings.logs": "Логи интерфейса",
    "settings.desktopLog": "Файл логов desktop",
    "settings.projectPath": "Текущий путь",
    "settings.projectAgent": "Агент проекта по умолчанию",
    "settings.language": "Язык",
    "settings.liveApps": "Локальный runtime",
    "settings.display": "Display",
    "settings.displayTitle": "Крупность интерфейса чата",
    "settings.displayLead": "Настрой масштаб рабочей области чата в процентах. Welcome, настройки и dev-экраны остаются как есть.",
    "settings.displayScale": "Масштаб чата",
    "settings.displayReset": "Сбросить до 100%",
    "settings.developer": "Роль и доступ",
    "settings.role": "Текущая роль",
    "settings.accessSource": "Источник доступа",
    "settings.devMode": "Developer mode",
    "settings.enableDevMode": "Включить dev-only функции",
    "settings.disableDevMode": "Выключить dev-only функции",
    "material.open": "Открыть материал",
    "material.discuss": "Обсудить в чате",
    "material.demo": "Открыть демо в браузере",
    "material.launch": "Запустить в ARC",
    "material.why": "Почему это здесь",
    "material.whyText": "Материалы показывают, что именно сделал агент в этой сессии: ответ, план, демо, документ, проверку или изменения.",
    "material.preview": "Предпросмотр",
    "material.files": "Связанные файлы",
    "material.source": "Источник",
    "material.sourceConversation": "Разговор",
    "material.sourceLesson": "Учебный сценарий",
    "material.sourceRun": "Запуск агента",
    "material.answerLead": "Короткий ответ, который можно сразу прочитать или обсудить дальше.",
    "material.planLead": "Пошаговый план, чтобы быстро понять, что агент предлагает делать дальше.",
    "material.documentLead": "Полезный текстовый материал, который удобно читать прямо внутри ARC.",
    "material.diagramLead": "Визуальная схема, которую можно просмотреть внутри приложения или открыть глубже.",
    "material.demoLead": "Живой результат, который агент может показать прямо внутри ARC.",
    "material.simulationLead": "Интерактивное объяснение или мини-симуляция, которую удобно запускать прямо в приложении.",
    "material.changesLead": "Сводка изменений и то, как агент менял проект.",
    "material.reviewLead": "Проверка результата и заметки о рисках, качестве и следующих шагах.",
    "material.previewEmpty": "Для этого материала пока нет встроенного предпросмотра.",
    "file.preview": "Просмотр деталей",
    "file.none": "Открой материал с файлами, чтобы увидеть детали изменений.",
    "attached.title": "Подтянутый контекст",
    "attached.empty": "Сюда можно подтянуть прошлую сессию и продолжить разговор уже с её контекстом.",
    "live.empty": "Пока ничего не запущено. Когда агент поднимет демо или мини-сайт, он появится здесь.",
    "live.open": "Открыть в ARC",
    "live.openBrowser": "Открыть в браузере",
    "live.restart": "Запустить снова",
    "live.stop": "Остановить",
    "live.logs": "Показать лог",
    "live.starting": "Поднимаю демо",
    "live.stopping": "Останавливаю приложение",
    "live.ready": "Демо готово",
    "live.failed": "Не удалось запустить демо",
    "live.stopped": "Приложение остановлено",
    "live.windowTitle": "Окно демо",
    "live.windowLead": "Здесь ARC держит живые демо и миниаппы внутри приложения, без обязательного перехода в браузер.",
    "live.windowEmpty": "Сначала запусти демо или заново открой сохранённый миниапп из чата.",
    "chat.expandInline": "Развернуть",
    "chat.collapseInline": "Свернуть",
    "status.missing": "Недоступно",
    "action.explain": "Объясни",
    "action.plan": "Сделай план",
    "action.safe": "Попробуй безопасно",
    "action.do": "Сделай за меня",
    "status.all": "Любой статус",
    "status.ready": "Готово",
    "status.running": "В работе",
    "status.failed": "С ошибкой",
    "status.archived": "В архиве",
    "result.answer": "Ответ",
    "result.plan": "План",
    "result.document": "Документ",
    "result.review": "Проверка",
    "result.changes": "Изменения",
    "result.demo": "Демо",
    "result.diagram": "Диаграмма",
    "result.simulation": "Симуляция",
    "testing.title": "Сценарии тестирования",
    "testing.lead": "AI прогоняет продукт пошагово, а ты можешь вручную вести сценарий дальше или отдать его в полностью автономный режим.",
    "testing.empty": "Пока нет запущенного тестового прогона. Выбери сценарий слева.",
    "testing.startStep": "Запустить пошагово",
    "testing.startAuto": "Прогнать до конца",
    "testing.current": "Текущий прогон",
    "testing.next": "Next",
    "testing.quit": "Quit",
    "testing.end": "End testing",
    "testing.hidden": "Экран тестирования скрыт: он доступен только developer-role.",
    "testing.stepMode": "Пошаговый режим",
    "testing.autoMode": "Автономный режим",
    "testing.status.paused": "Ждёт следующего шага",
    "testing.status.running": "Идёт сценарий",
    "testing.status.done": "Сценарий завершён",
    "testing.status.failed": "Сценарий завершился с ошибкой",
    "testing.status.ended": "Тестирование остановлено",
    "verification.title": "Verification baseline",
    "verification.lead": "Перед большим эпиком ARC может сам прогнать верификаторы по технике, UX, бизнес-логике и testing flow.",
    "verification.empty": "Пока не было verification-прогона. Запусти профиль ниже.",
    "verification.start": "Запустить профиль",
    "verification.current": "Текущий verification run",
    "verification.status.running": "Идёт проверка",
    "verification.status.done": "Проверка завершена",
    "verification.status.failed": "Проверка завершилась с ошибкой",
    "verification.overall": "Итог",
    "verification.blocking": "Блокирующие fail",
    "verification.warnings": "Warnings",
    "verification.next": "Что делать дальше",
  },
  en: {
    "shell.native": "Desktop App",
    "shell.headline": "ARC",
    "shell.subhead": "A workspace where a conversation with an agent becomes a plan, a demo, and a real result.",
    "nav.chat": "Chat",
    "nav.sessions": "Sessions",
    "nav.learn": "Learn",
    "nav.testing": "Testing",
    "nav.settings": "Settings",
    "topbar.project": "Project",
    "topbar.projects": "Projects",
    "topbar.reload": "Refresh",
    "topbar.defaultAgent": "Project default agent",
    "hero.chat.eyebrow": "Chat",
    "hero.chat.title": "Say what you want to get done",
    "hero.chat.lead": "Choose a built-in agent, describe the goal, and let ARC connect the conversation, materials, and live output in one session.",
    "hero.sessions.eyebrow": "Sessions",
    "hero.sessions.title": "Important sessions stay within reach",
    "hero.sessions.lead": "Search past conversations, reopen their materials, and pull the right context back into your current work.",
    "hero.learn.eyebrow": "Learn",
    "hero.learn.title": "Learn without leaving the workspace",
    "hero.learn.lead": "Study explains with text, diagrams, mini demos, and quick comprehension checks directly inside ARC.",
    "hero.settings.eyebrow": "Settings",
    "hero.settings.title": "System and project status",
    "hero.settings.lead": "See connected agents, the local runtime, frontend logs, and the current project path here.",
    "hero.testing.eyebrow": "Testing",
    "hero.testing.title": "Autonomous tour of ARC",
    "hero.testing.lead": "This screen is developer-only and helps run the whole product step by step without manual clicking.",
    "welcome.title": "Open a project and start working with the agent",
    "welcome.lead": "Open a project folder. ARC will keep the conversation, work steps, materials, and live demos locally from there.",
    "welcome.path": "Project path",
    "welcome.choose": "Choose project",
    "welcome.open": "Open project",
    "welcome.setup": "Set up ARC in folder",
    "welcome.ready": "Project is ready.",
    "welcome.notReady": "ARC is not set up in this folder yet.",
    "welcome.noRecent": "Recent projects will appear here after the first open.",
    "welcome.recent": "Recent projects",
    "welcome.hint": "After opening, you land directly in chat. No separate task manager is required.",
    "welcome.value1": "Talk to the agent like in a normal chat instead of driving a pipeline manually.",
    "welcome.value2": "Read plans, documents, diagrams, and demos directly inside the app.",
    "welcome.value3": "Return to past sessions and continue from the right place.",
    "welcome.flowTitle": "What happens next",
    "welcome.flow1Title": "Open the project",
    "welcome.flow1Body": "ARC keeps conversations, materials, and local demos inside the project folder.",
    "welcome.flow2Title": "Say what you need",
    "welcome.flow2Body": "Chat stays the main entry: describe the goal or attach your own files.",
    "welcome.flow3Title": "See the result beside it",
    "welcome.flow3Body": "Plans, documents, diagrams, and demos open in the side inspector without breaking the conversation.",
    "welcome.trustLocal": "Local-first",
    "welcome.trustLocalBody": "Conversations, materials, and demos stay on your machine.",
    "welcome.trustControl": "Controlled",
    "welcome.trustControlBody": "Running demos can be opened separately or stopped manually.",
    "welcome.trustReview": "Reviewable",
    "welcome.trustReviewBody": "Conversation history and outputs stay attached to the project.",
    "project.missing": "Choose a project first.",
    "project.ready": "Project ready",
    "project.notReady": "ARC setup required",
    "project.loading": "Loading project state...",
    "project.opened": "Project opened.",
    "project.initialized": "ARC is set up. You can start working.",
    "project.refreshed": "Project state refreshed.",
    "status.error": "Error",
    "status.loading": "Loading...",
    "chat.agent": "Choose a built-in agent",
    "chat.starters": "Where to start",
    "chat.startersHint": "Quick scenarios help start a useful conversation without writing everything from scratch.",
    "chat.projectAgent": "Project agent",
    "chat.projectAgentHint": "This choice is saved in the project and becomes the default for new conversations.",
    "chat.sessionAgent": "Topic agent",
    "chat.sessionAgentHint": "The current topic is already pinned to its own agent.",
    "chat.sessionAgentLocked": "This topic is already using its own agent.",
    "chat.projectAgentUpdated": "Project agent updated.",
    "chat.agentProjectMenu": "Project agent",
    "chat.agentTopicMenu": "Agent for this topic",
    "chat.agentAll": "All agents",
    "chat.builtInAgents": "Built-in agents",
    "chat.installedPresets": "Installed presets",
    "chat.installedPresetHint": "Currently available as the project preset layer.",
    "chat.inProject": "in project",
    "chat.actionMenu": "Mode",
    "chat.topicUntitled": "New conversation",
    "chat.topicsRail": "Topics",
    "chat.openProject": "Projects",
    "chat.newTopic": "New chat",
    "chat.railDivider": "Topics",
    "chat.liveLabel": "Running now",
    "chat.readyLabel": "Chat materials",
    "chat.detailsLabel": "Files",
    "chat.outputs": "Chat materials",
    "chat.noOutputs": "Answers, documents, diagrams, and demos from this topic appear here after the first useful step.",
    "chat.detailsHint": "This area shows all project files plus ARC-managed files added through chat.",
    "chat.relatedFiles": "Related files",
    "chat.openResult": "Open chat details",
    "chat.action": "What should happen with this message",
    "chat.composer": "Describe the goal in plain language",
    "chat.placeholder": "For example: explain cellular respiration with a diagram and a mini simulation.",
    "chat.new": "New conversation",
    "chat.send": "Send",
    "chat.current": "Current session",
    "chat.live": "What the agent is doing",
    "chat.liveApps": "Running now",
    "chat.materials": "Session materials",
    "chat.topics": "Conversation topics",
    "chat.thread": "Conversation",
    "chat.currentWork": "Current work",
    "chat.readyNow": "Ready now",
    "chat.continueLast": "Continue last session",
    "chat.fresh": "Start a fresh conversation",
    "chat.liveStatus": "Agent is working",
    "chat.projectMaterials": "ARC files",
    "chat.projectMaterialsHint": "These files are saved inside `.arc/materials/uploads/...` and become visible to the agent right away without cluttering the project itself.",
    "chat.noProjectMaterials": "No ARC files yet.",
    "chat.upload": "Upload materials",
    "chat.uploadInline": "Add files",
    "chat.uploaded": "Files were added into ARC.",
    "chat.uploadBusy": "Uploading files into ARC",
    "chat.materialAttachHint": "Uploaded files are stored by ARC and immediately become part of the current chat context.",
    "chat.projectMaterialsShort": "Recently added to ARC",
    "chat.sessions": "Recent sessions",
    "chat.tab.conversation": "Now",
    "chat.tab.materials": "Ready",
    "chat.tab.demos": "Demo",
    "chat.allowedTitle": "How the agent will behave right now",
    "chat.unlock": "Allow a more autonomous run",
    "chat.studyFallback": "Study will not complete the task for the user. The request will be turned into explanation and planning.",
    "chat.noMessages": "Start the conversation. ARC creates a topic after the first message.",
    "chat.noMaterials": "Materials appear here after a reply, plan, run, or demo.",
    "chat.noSession": "A session will appear after the first message or the first run.",
    "chat.noHistory": "Session history appears after the first agent steps.",
    "chat.idle": "This session is ready. Continue the conversation, open a result, or launch a demo.",
    "chat.inlineResults": "Chat details",
    "chat.inlineVisuals": "Diagrams and demos",
    "chat.resultPanel": "Chat details",
    "chat.inspectorEmpty": "Open chat details to browse topic materials and project files.",
    "chat.liveNowHint": "This section shows which demos and simulations for the current topic are actually running right now.",
    "chat.materialsHint": "Answers, docs, diagrams, and demos produced in this topic stay here.",
    "chat.filesHint": "Project files and ARC-managed files can be opened from this panel.",
    "chat.drawerClose": "Close",
    "chat.waiting": "The agent is thinking about the reply.",
    "chat.sendingStatus": "Sending request",
    "chat.planningStatus": "Building plan",
    "chat.safeStatus": "Running safely",
    "chat.doStatus": "Starting execution",
    "chat.failed": "No reply arrived. Show the error and decide what to do next.",
    "chat.replyMissing": "The agent finished the step without a text reply.",
    "chat.providerMissing": "The current agent is unavailable locally. Check provider setup and try again.",
    "chat.topicAgentFixed": "This topic is pinned to agent",
    "chat.openDocument": "Open in chat details",
    "chat.openDetails": "Open chat details",
    "files.project": "Project files",
    "files.arc": "ARC files",
    "files.empty": "Project files and ARC-managed chat files will appear here.",
    "files.delete": "Delete",
    "sessions.search": "Search past sessions",
    "sessions.filterAgent": "Agent",
    "sessions.filterStatus": "Status",
    "sessions.list": "Session history",
    "sessions.empty": "No saved sessions yet. Start a conversation in chat.",
    "sessions.detail": "Session details",
    "sessions.continue": "Continue conversation",
    "sessions.attach": "Pull into current context",
    "sessions.detach": "Remove from context",
    "sessions.materials": "Materials",
    "sessions.runs": "Runs inside the session",
    "sessions.next": "Next step",
    "sessions.updated": "Updated",
    "sessions.lastSignal": "Latest signal",
    "sessions.total": "Total sessions",
    "sessions.activeNow": "Active now",
    "sessions.withMaterials": "With materials",
    "sessions.summaryLead": "Find a past conversation, understand how it ended, and continue it without digging through logs.",
    "sessions.resultLead": "This view shows what the session was about, what is already ready, and the best next step.",
    "sessions.highlights": "Ready to use",
    "sessions.liveNow": "Live demos and previews",
    "sessions.noHighlights": "When this session produces documents, diagrams, or demos, they will appear here.",
    "learn.title": "Learning scenarios",
    "learn.lead": "The Study agent can explain with text, diagrams, demos, and mini-simulations inside the app.",
    "learn.quickProject.title": "Explain this project",
    "learn.quickProject.summary": "Get a plain-language explanation of the project and a good starting point.",
    "learn.quickBio.title": "Show a topic with a diagram",
    "learn.quickBio.summary": "Launch the cellular respiration lesson with a diagram and mini simulation.",
    "learn.quickPlan.title": "Help me make a plan",
    "learn.quickPlan.summary": "Turn a goal into a clear next step quickly.",
    "learn.bio.title": "Cellular respiration",
    "learn.bio.summary": "Ask the agent to explain how a cell turns glucose into energy with a diagram and a mini simulation.",
    "learn.project.title": "Explain this project",
    "learn.project.summary": "Ask for a plain-language walkthrough of what the project does and where to start.",
    "learn.use": "Open in chat",
    "learn.demo": "Launch mini simulation",
    "settings.providers": "Available agents",
    "settings.general": "General",
    "settings.advanced": "Developer",
    "settings.logs": "Frontend logs",
    "settings.desktopLog": "Desktop log file",
    "settings.projectPath": "Current path",
    "settings.projectAgent": "Project default agent",
    "settings.language": "Language",
    "settings.liveApps": "Local runtime",
    "settings.display": "Display",
    "settings.displayTitle": "Chat workspace scale",
    "settings.displayLead": "Tune the chat workspace scale in percent. Welcome, settings, and dev screens stay unchanged.",
    "settings.displayScale": "Chat scale",
    "settings.displayReset": "Reset to 100%",
    "settings.developer": "Role and access",
    "settings.role": "Current role",
    "settings.accessSource": "Access source",
    "settings.devMode": "Developer mode",
    "settings.enableDevMode": "Enable dev-only features",
    "settings.disableDevMode": "Disable dev-only features",
    "material.open": "Open material",
    "material.discuss": "Discuss in chat",
    "material.demo": "Open demo in browser",
    "material.launch": "Launch in ARC",
    "material.why": "Why this is here",
    "material.whyText": "Materials show what the agent produced in this session: an answer, plan, demo, document, review, or change summary.",
    "material.preview": "Preview",
    "material.files": "Related files",
    "material.source": "Source",
    "material.sourceConversation": "Conversation",
    "material.sourceLesson": "Learning flow",
    "material.sourceRun": "Agent run",
    "material.answerLead": "A quick answer you can read immediately or discuss further.",
    "material.planLead": "A step-by-step plan so you can quickly see what the agent suggests next.",
    "material.documentLead": "A useful written document that is comfortable to read inside ARC.",
    "material.diagramLead": "A visual diagram you can inspect inside the app or open in more detail.",
    "material.demoLead": "A live result the agent can show directly inside ARC.",
    "material.simulationLead": "An interactive explanation or mini-simulation that works best inside the app.",
    "material.changesLead": "A summary of what changed and how the agent updated the project.",
    "material.reviewLead": "A result check with quality, risk, and follow-up notes.",
    "material.previewEmpty": "There is no embedded preview for this material yet.",
    "file.preview": "Details preview",
    "file.none": "Open a material with files to inspect change details.",
    "attached.title": "Attached context",
    "attached.empty": "You can pull a past session here and continue the conversation with its context.",
    "live.empty": "Nothing is running yet. When the agent launches a demo or mini-site, it will appear here.",
    "live.open": "Open in ARC",
    "live.openBrowser": "Open in browser",
    "live.restart": "Restart",
    "live.stop": "Stop",
    "live.logs": "Show log",
    "live.starting": "Starting demo",
    "live.stopping": "Stopping app",
    "live.ready": "Demo ready",
    "live.failed": "Demo failed",
    "live.stopped": "App stopped",
    "live.windowTitle": "Demo window",
    "live.windowLead": "ARC keeps live demos and miniapps inside the app, without forcing a browser jump.",
    "live.windowEmpty": "Start a demo first or reopen a saved miniapp from chat.",
    "chat.expandInline": "Expand",
    "chat.collapseInline": "Collapse",
    "status.missing": "Missing",
    "action.explain": "Explain",
    "action.plan": "Make a plan",
    "action.safe": "Try safely",
    "action.do": "Do it for me",
    "status.all": "Any status",
    "status.ready": "Ready",
    "status.running": "Running",
    "status.failed": "Failed",
    "status.archived": "Archived",
    "result.answer": "Answer",
    "result.plan": "Plan",
    "result.document": "Document",
    "result.review": "Review",
    "result.changes": "Changes",
    "result.demo": "Demo",
    "result.diagram": "Diagram",
    "result.simulation": "Simulation",
    "testing.title": "Testing scenarios",
    "testing.lead": "AI runs the product step by step while you can move to the next step, hand the run over to autonomy, or end it.",
    "testing.empty": "No testing run yet. Pick a scenario on the left.",
    "testing.startStep": "Start step by step",
    "testing.startAuto": "Run to completion",
    "testing.current": "Current run",
    "testing.next": "Next",
    "testing.quit": "Quit",
    "testing.end": "End testing",
    "testing.hidden": "Testing is hidden. It is available only for the developer role.",
    "testing.stepMode": "Step mode",
    "testing.autoMode": "Autonomous mode",
    "testing.status.paused": "Waiting for the next step",
    "testing.status.running": "Scenario is running",
    "testing.status.done": "Scenario finished",
    "testing.status.failed": "Scenario failed",
    "testing.status.ended": "Testing stopped",
    "verification.title": "Verification baseline",
    "verification.lead": "Before a large implementation pass, ARC can run focused verifiers for technical health, UX, business logic, and testing flow.",
    "verification.empty": "No verification run yet. Start a profile below.",
    "verification.start": "Run profile",
    "verification.current": "Current verification run",
    "verification.status.running": "Verification is running",
    "verification.status.done": "Verification finished",
    "verification.status.failed": "Verification failed",
    "verification.overall": "Overall verdict",
    "verification.blocking": "Blocking fails",
    "verification.warnings": "Warnings",
    "verification.next": "What to do next",
  },
};

const byId = (id) => document.getElementById(id);

function t(key) {
  return M[state.locale]?.[key] || M.en[key] || key;
}

function formatBytes(value) {
  const size = Number(value || 0);
  if (!size) return "0 B";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDateLabel(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  return new Intl.DateTimeFormat(state.locale === "ru" ? "ru-RU" : "en-US", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function formatRailDateLabel(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  const now = new Date();
  const sameDay = now.getFullYear() === date.getFullYear()
    && now.getMonth() === date.getMonth()
    && now.getDate() === date.getDate();
  return new Intl.DateTimeFormat(state.locale === "ru" ? "ru-RU" : "en-US", sameDay
    ? { hour: "2-digit", minute: "2-digit" }
    : { day: "2-digit", month: "2-digit" }).format(date);
}

function renderStructuredPreview(text) {
  return renderMarkdown(text);
}

function readFileAsBase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const raw = String(reader.result || "");
      const base64 = raw.includes(",") ? raw.split(",").pop() : raw;
      resolve({
        name: file.name,
        mime_type: file.type || "application/octet-stream",
        size: file.size || 0,
        content_base64: base64 || "",
      });
    };
    reader.onerror = () => reject(new Error(`failed to read ${file.name}`));
    reader.readAsDataURL(file);
  });
}

async function encodeUploadedFiles(fileList) {
  return Promise.all(Array.from(fileList || []).map((file) => readFileAsBase64(file)));
}

function persistClientLogs() {
  window.localStorage.setItem(CLIENT_LOGS_KEY, JSON.stringify(state.clientLogs.slice(-MAX_CLIENT_LOGS)));
}

function pushClientLog(level, message) {
  state.clientLogs = [
    ...state.clientLogs.slice(-(MAX_CLIENT_LOGS - 1)),
    { level: level || "info", message: String(message || ""), created_at: new Date().toISOString() },
  ];
  persistClientLogs();
  callWailsBridge(window, "LogFrontend", level || "info", String(message || "")).catch(() => {});
}

function showBanner(message, kind = "") {
  byId("status-banner").innerHTML = message
    ? `<div class="banner ${kind}">${escapeHTML(message)}</div>`
    : "";
}

function card(title, body, className = "") {
  const classes = ["card", className].filter(Boolean).join(" ");
  return `<article class="${classes}"><h3>${escapeHTML(title)}</h3>${body}</article>`;
}

function emptyBlock(message) {
  return `<div class="empty">${escapeHTML(message)}</div>`;
}

const transport = {
  chooseProject: (startPath) => callWailsBridge(window, "ChooseWorkspaceDirectory", startPath),
  projectState: (path) => callWailsBridge(window, "ProjectState", path),
  initWorkspace: (req) => callWailsBridge(window, "InitWorkspace", req),
  setWorkspaceMode: (req) => callWailsBridge(window, "SetWorkspaceMode", req),
  developerAccess: (path) => callWailsBridge(window, "DeveloperAccess", path),
  setDeveloperRole: (req) => callWailsBridge(window, "SetDeveloperRole", req),
  providers: () => callWailsBridge(window, "Providers"),
  agents: () => callWailsBridge(window, "Agents"),
  allowedActions: (path, mode, sessionID) => callWailsBridge(window, "AllowedActions", path, mode || "", sessionID || ""),
  sessions: (path, query, mode, status) => callWailsBridge(window, "Sessions", path, 100, query, mode, status),
  session: (path, sessionID) => callWailsBridge(window, "Session", path, sessionID),
  testingScenarios: (path) => callWailsBridge(window, "TestingScenarios", path),
  startTesting: (req) => callWailsBridge(window, "StartTesting", req),
  testingStatus: (path, runID) => callWailsBridge(window, "TestingStatus", path, runID),
  testingControl: (req) => callWailsBridge(window, "TestingControl", req),
  verificationProfiles: (path) => callWailsBridge(window, "VerificationProfiles", path),
  startVerification: (req) => callWailsBridge(window, "StartVerification", req),
  verificationStatus: (path, runID) => callWailsBridge(window, "VerificationStatus", path, runID),
  chatStart: (req) => callWailsBridge(window, "ChatStart", req),
  chatSend: (req) => callWailsBridge(window, "ChatSend", req),
  taskPlan: (req) => callWailsBridge(window, "TaskPlan", req),
  taskRun: (req) => callWailsBridge(window, "TaskRun", req),
  workspaceFile: (path, file) => callWailsBridge(window, "WorkspaceFile", path, file),
  workspaceFiles: (path, limit = 400) => callWailsBridge(window, "WorkspaceFiles", path, limit),
  projectMaterials: (path) => callWailsBridge(window, "ProjectMaterials", path),
  uploadProjectMaterials: (req) => callWailsBridge(window, "UploadProjectMaterials", req),
  deleteProjectMaterial: (req) => callWailsBridge(window, "DeleteProjectMaterial", req),
  desktopLogPath: () => callWailsBridge(window, "DesktopLogPath"),
  openExternalURL: (url) => callWailsBridge(window, "OpenExternalURL", url),
  chatScalePercent: () => callWailsBridge(window, "ChatScalePercent"),
  setChatScalePercent: (value) => callWailsBridge(window, "SetChatScalePercent", Number(value || 100)),
  liveApps: (path, sessionID) => callWailsBridge(window, "LiveApps", path, sessionID || ""),
  startMaterialLiveApp: (req) => callWailsBridge(window, "StartMaterialLiveApp", req),
  startLessonDemo: (req) => callWailsBridge(window, "StartLessonDemo", req),
  ensureLiveApp: (req) => callWailsBridge(window, "EnsureLiveApp", req),
  stopLiveApp: (req) => callWailsBridge(window, "StopLiveApp", req),
};

function setBusy(nextValue, label = "") {
  state.busy = !!nextValue;
  state.busyLabel = state.busy ? label : "";
  document.body.classList.toggle("app-busy", state.busy);
  document.querySelectorAll("button").forEach((button) => {
    if (button.dataset.noBusy === "true") return;
    button.disabled = state.busy || button.classList.contains("disabled");
    button.classList.toggle("is-loading", state.busy && label !== "");
  });
}

async function withBusy(label, action, successMessage = "") {
  setBusy(true, label);
  if (label) showBanner(label);
  try {
    const result = await action();
    if (successMessage) showBanner(successMessage);
    return result;
  } catch (error) {
    pushClientLog("error", error?.message || String(error));
    showBanner(error?.message || String(error), "error");
    throw error;
  } finally {
    setBusy(false, "");
  }
}

function isReady() {
  return state.projectState?.state === "ready";
}

function currentProvider() {
  return state.workspace?.default_provider || state.projectState?.workspace?.default_provider || "codex";
}

function projectDefaultMode() {
  return state.workspace?.mode || state.projectState?.workspace?.mode || "work";
}

function agentNameFromMode(mode) {
  switch (mode) {
    case "study":
      return "Study";
    case "hero":
      return "Hero";
    default:
      return "Work";
  }
}

function agentNameByMode(mode) {
  const agent = (state.agents || []).find((item) => item.mode === mode);
  return agent?.name || agentNameFromMode(mode);
}

function persistAttachedSessions() {
  window.localStorage.setItem(ATTACHED_SESSIONS_KEY, JSON.stringify(state.attachedSessionIDs));
}

function persistChatScale() {
  window.localStorage.setItem(CHAT_SCALE_KEY, String(state.chatScalePercent));
}

function applyChatScaleState() {
  state.chatScalePercent = applyChatScale(window, state.chatScalePercent);
  persistChatScale();
}

function renderStaticCopy() {
  applyChatScaleState();
  document.title = t("shell.headline");
  document.querySelectorAll("[data-i18n]").forEach((node) => {
    node.textContent = t(node.dataset.i18n);
  });
  if (byId("locale-select")) {
    byId("locale-select").value = state.locale;
  }
}

function updateProjectChrome() {
  const projectChip = byId("project-chip");
  const subtitle = byId("project-subtitle");
  const inline = byId("project-default-agent-inline");
  if (!projectChip || !subtitle || !inline) {
    return;
  }
  if (!state.projectState || state.projectState.state === "no_project_selected") {
    projectChip.textContent = "ARC не открыт";
    subtitle.textContent = t("project.missing");
    inline.innerHTML = "";
    return;
  }
  const label = state.projectState.name || summarizeProjectPath(state.projectState.path || state.path);
  projectChip.textContent = label;
  subtitle.textContent = state.projectState.message || t("project.ready");
  inline.innerHTML = renderAgentMenu("project", projectDefaultMode(), { topbar: true });
}

function renderNavState() {
  const testingVisible = !!state.developerAccess?.can_use_testing;
  byId("nav-testing")?.classList.toggle("hidden", !testingVisible);
  byId("topbar-testing")?.classList.toggle("hidden", !testingVisible);
  if (!testingVisible && state.currentScreen === "testing") {
    state.currentScreen = "chat";
  }
  if (state.currentScreen === "learn" || state.currentScreen === "sessions") {
    state.currentScreen = "chat";
  }
  document.querySelector(".shell")?.classList.toggle("chat-focus", state.currentScreen === "chat");
  document.querySelectorAll(".nav-item, .topbar-nav-item").forEach((button) => {
    if ((button.id === "nav-testing" || button.id === "topbar-testing") && !testingVisible) {
      return;
    }
    const requires = button.dataset.requiresProject === "true";
    const active = button.dataset.screen === state.currentScreen;
    button.classList.toggle("active", active);
    button.disabled = requires && !isReady();
    button.classList.toggle("disabled", requires && !isReady());
  });
}

function renderWelcomeOverlay() {
  const overlay = byId("welcome-overlay");
  const shouldShow = state.showWelcome || !isReady();
  overlay.className = shouldShow ? "welcome-overlay visible" : "welcome-overlay";
  if (!shouldShow) {
    overlay.innerHTML = "";
    return;
  }
  const recent = state.recentProjects.length
    ? `<div class="recent-projects">${state.recentProjects
        .map((path) => `<button class="recent-project-button" data-recent-project="${escapeHTML(path)}">${escapeHTML(summarizeProjectPath(path))}</button>`)
        .join("")}</div>`
    : `<div class="empty">${escapeHTML(t("welcome.noRecent"))}</div>`;
  const stateBlock = state.projectState && state.path
    ? `<div class="welcome-state ${escapeHTML(state.projectState.state)}">
        <strong>${escapeHTML(state.projectState.state === "ready" ? t("welcome.ready") : t("welcome.notReady"))}</strong>
        <div class="muted">${escapeHTML(state.projectState.message || "")}</div>
      </div>`
    : "";
  overlay.innerHTML = `
    <div class="welcome-dialog">
      <div class="eyebrow">ARC</div>
      <h2>${escapeHTML(t("welcome.title"))}</h2>
      <p class="muted">${escapeHTML(t("welcome.lead"))}</p>
      <div class="welcome-value-grid">
        <div class="welcome-value-card">${escapeHTML(t("welcome.value1"))}</div>
        <div class="welcome-value-card">${escapeHTML(t("welcome.value2"))}</div>
        <div class="welcome-value-card">${escapeHTML(t("welcome.value3"))}</div>
      </div>
      <div class="welcome-flow-block">
        <strong>${escapeHTML(t("welcome.flowTitle"))}</strong>
        <div class="welcome-flow-grid">
          <div class="welcome-flow-card"><span>1</span><div><strong>${escapeHTML(t("welcome.flow1Title"))}</strong><p>${escapeHTML(t("welcome.flow1Body"))}</p></div></div>
          <div class="welcome-flow-card"><span>2</span><div><strong>${escapeHTML(t("welcome.flow2Title"))}</strong><p>${escapeHTML(t("welcome.flow2Body"))}</p></div></div>
          <div class="welcome-flow-card"><span>3</span><div><strong>${escapeHTML(t("welcome.flow3Title"))}</strong><p>${escapeHTML(t("welcome.flow3Body"))}</p></div></div>
        </div>
      </div>
      <label class="metric-label" for="welcome-path-input">${escapeHTML(t("welcome.path"))}</label>
      <div class="path-row">
        <input id="welcome-path-input" class="text-input" value="${escapeHTML(state.draftPath || state.path || "")}" />
        <button id="welcome-choose" class="secondary-button">${escapeHTML(t("welcome.choose"))}</button>
        <button id="welcome-open" class="primary-button">${escapeHTML(t("welcome.open"))}</button>
      </div>
      ${stateBlock}
      <div class="button-row">
        ${state.projectState?.state === "selected_not_initialized" ? `<button id="welcome-init" class="primary-button">${escapeHTML(t("welcome.setup"))}</button>` : ""}
      </div>
      <div class="section-gap">
        <strong>${escapeHTML(t("welcome.recent"))}</strong>
        ${recent}
      </div>
      <div class="welcome-trust-grid">
        <div class="welcome-trust-card"><strong>${escapeHTML(t("welcome.trustLocal"))}</strong><p>${escapeHTML(t("welcome.trustLocalBody"))}</p></div>
        <div class="welcome-trust-card"><strong>${escapeHTML(t("welcome.trustControl"))}</strong><p>${escapeHTML(t("welcome.trustControlBody"))}</p></div>
        <div class="welcome-trust-card"><strong>${escapeHTML(t("welcome.trustReview"))}</strong><p>${escapeHTML(t("welcome.trustReviewBody"))}</p></div>
      </div>
      <p class="muted">${escapeHTML(t("welcome.hint"))}</p>
    </div>
  `;

  byId("welcome-path-input").addEventListener("input", (event) => {
    state.draftPath = event.target.value;
  });
  byId("welcome-choose").onclick = chooseProject;
  byId("welcome-open").onclick = () => openProject(state.draftPath);
  byId("welcome-init")?.addEventListener("click", () => initWorkspace());
  overlay.querySelectorAll("[data-recent-project]").forEach((button) => {
    button.addEventListener("click", () => {
      state.draftPath = button.dataset.recentProject;
      openProject(state.draftPath);
    });
  });
}

function attachedSessionPills() {
  if (!state.attachedSessionIDs.length) {
    return `<div class="empty">${escapeHTML(t("attached.empty"))}</div>`;
  }
  return `<div class="pill-row">${state.attachedSessionIDs.map((sessionID) => {
    const session = state.sessions.find((item) => item.id === sessionID);
    const label = session ? session.title : sessionID;
    return `<button class="pill removable-pill" data-detach-session="${escapeHTML(sessionID)}">${escapeHTML(label)} ×</button>`;
  }).join("")}</div>`;
}

function renderProjectMaterialsList() {
  const items = state.projectMaterials || [];
  if (!items.length) {
    return emptyBlock(t("chat.noProjectMaterials"));
  }
  return `<div class="list">${items.slice(0, 6).map((item) => `
    <div class="list-button static">
      <div class="list-title"><span>${escapeHTML(item.name)}</span><span>${escapeHTML(formatBytes(item.size))}</span></div>
      <div class="body-copy">${escapeHTML(item.path)}</div>
      <div class="muted">${escapeHTML(item.uploaded_at || item.uploadedAt || "")}</div>
    </div>
  `).join("")}</div>`;
}

function renderProjectMaterialsCompact() {
  const items = state.projectMaterials || [];
  if (!items.length) {
    return "";
  }
  return `
    <div class="composer-support">
      <div class="eyebrow">${escapeHTML(t("chat.projectMaterialsShort"))}</div>
      <div class="pill-row">
        ${items.slice(0, 4).map((item) => `<span class="pill attachment-pill">${escapeHTML(item.name)} <span class="attachment-pill-status">${escapeHTML(t("chat.inProject"))}</span></span>`).join("")}
      </div>
    </div>
  `;
}

function providerHealth(providerName) {
  return (state.providers || []).find((item) => item.name === providerName) || null;
}

function sessionError(detail) {
  return detail?.metadata?.last_error || detail?.metadata?.lastError || "";
}

function isVisualMaterial(material) {
  return ["diagram", "demo", "simulation"].includes(material?.type);
}

function isDocumentMaterial(material) {
  return !!material && !isVisualMaterial(material);
}

function currentDocumentMaterial(detail) {
  const materials = detail?.materials || [];
  const selected = materials.find((item) => item.id === state.selectedMaterialId);
  if (selected && isDocumentMaterial(selected)) return selected;
  return materials.find((item) => isDocumentMaterial(item)) || null;
}

function currentVisualMaterial(detail) {
  const materials = detail?.materials || [];
  const selected = materials.find((item) => item.id === state.selectedMaterialId);
  if (selected && isVisualMaterial(selected)) return selected;
  return materials.find((item) => isVisualMaterial(item)) || null;
}

function currentThreadViewportKey() {
  return state.selectedSessionId || "__draft__";
}

function captureThreadViewportState() {
  const scroller = document.querySelector(".message-thread-scroll");
  if (!scroller) return;
  const scrollerRect = scroller.getBoundingClientRect();
  const anchor = Array.from(scroller.querySelectorAll("[data-message-key]")).find((row) => row.getBoundingClientRect().bottom > scrollerRect.top + 4);
  const maxOffset = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
  const fromBottom = Math.max(0, maxOffset - scroller.scrollTop);
  state.threadViewportBySession[currentThreadViewportKey()] = {
    top: scroller.scrollTop,
    fromBottom,
    stickToBottom: fromBottom < 48,
    anchorKey: anchor?.dataset.messageKey || "",
    anchorOffset: anchor ? anchor.getBoundingClientRect().top - scrollerRect.top : 0,
  };
}

function restoreThreadViewportState() {
  const scroller = document.querySelector(".message-thread-scroll");
  if (!scroller) return;
  const maxOffset = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
  const saved = state.threadViewportBySession[currentThreadViewportKey()];
  if (!saved) {
    scroller.scrollTop = maxOffset;
    return;
  }
  if (saved.stickToBottom) {
    scroller.scrollTop = maxOffset;
    return;
  }
  if (saved.anchorKey) {
    const anchor = Array.from(scroller.querySelectorAll("[data-message-key]")).find((row) => row.dataset.messageKey === saved.anchorKey);
    if (anchor) {
      const scrollerRect = scroller.getBoundingClientRect();
      const anchorRect = anchor.getBoundingClientRect();
      scroller.scrollTop += anchorRect.top - scrollerRect.top - (saved.anchorOffset || 0);
      return;
    }
  }
  scroller.scrollTop = Math.min(saved.top, maxOffset);
}

function restoreThreadViewportStateDeferred() {
  restoreThreadViewportState();
  window.requestAnimationFrame(() => {
    restoreThreadViewportState();
    window.requestAnimationFrame(() => {
      restoreThreadViewportState();
    });
  });
}

function bindThreadViewportTracking() {
  const scroller = document.querySelector(".message-thread-scroll");
  if (!scroller || scroller.dataset.viewportBound === "true") return;
  scroller.addEventListener("scroll", () => captureThreadViewportState(), { passive: true });
  scroller.dataset.viewportBound = "true";
}

function renderInlineFrameSlot(frameKey, className, src, title) {
  if (!src) return "";
  return `
    <div
      class="preserved-frame-slot"
      data-preserve-frame-key="${escapeHTML(frameKey)}"
      data-frame-class="${escapeHTML(className)}"
      data-frame-src="${escapeHTML(src)}"
      data-frame-title="${escapeHTML(title || "")}"
    ></div>
  `;
}

function capturePreservedThreadFrames() {
  document.querySelectorAll(".message-thread-scroll [data-preserve-frame-key]").forEach((slot) => {
    const key = slot.dataset.preserveFrameKey;
    const frame = slot.querySelector("iframe");
    if (key && frame) {
      preservedThreadFrames.set(key, frame);
    }
  });
}

function hydratePreservedThreadFrames() {
  document.querySelectorAll(".message-thread-scroll [data-preserve-frame-key]").forEach((slot) => {
    const key = slot.dataset.preserveFrameKey;
    const src = slot.dataset.frameSrc || "";
    if (!key || !src) return;
    const title = slot.dataset.frameTitle || "";
    const className = slot.dataset.frameClass || "inline-output-frame";
    let frame = preservedThreadFrames.get(key);
    if (!frame || frame.getAttribute("src") !== src) {
      frame = document.createElement("iframe");
      frame.setAttribute("src", src);
      frame.addEventListener("load", () => restoreThreadViewportStateDeferred(), { once: true });
    }
    frame.className = className;
    frame.setAttribute("title", title);
    slot.replaceChildren(frame);
    preservedThreadFrames.set(key, frame);
  });
}

function chatDetailSignature(detail) {
  if (!detail?.session?.id) return "empty";
  const messages = detail.messages || [];
  const lastMessage = messages[messages.length - 1] || {};
  const outputs = (lastMessage.outputs || []).map((item) => [
    item.id || "",
    item.status || "",
    item.live_app_id || item.liveAppID || "",
    item.preview_url || item.previewURL || "",
  ].join(":")).join("|");
  const liveApps = (detail.live_apps || detail.liveApps || []).map((item) => [
    item.id || "",
    item.status || "",
    item.preview_url || item.previewURL || "",
  ].join(":")).join("|");
  return [
    detail.session.id || "",
    detail.session.status || "",
    detail.session.updated_at || detail.session.updatedAt || "",
    String(messages.length),
    lastMessage.role || "",
    lastMessage.content || "",
    lastMessage.failure || "",
    outputs,
    liveApps,
    detail.metadata?.last_error || detail.metadata?.lastError || "",
  ].join("§");
}

async function openExternalURL(url) {
  const target = String(url || "").trim();
  if (!target) return;
  try {
    await transport.openExternalURL(target);
  } catch (error) {
    pushClientLog("warn", `open external failed: ${error?.message || String(error)}`);
    const fallback = typeof window.open === "function" ? window.open(target, "_blank", "noopener,noreferrer") : null;
    if (!fallback) {
      showBanner(error?.message || String(error), "error");
      throw error;
    }
  }
}

function syncDemoPanelFromSession() {
  if (!state.demoPanelOpen) return;
  const runtime = liveAppRuntime(state.sessionDetail, state.demoPanelAppID, state.demoPanelOrigin, state.demoPanelURL);
  state.demoPanelAppID = runtime.liveAppID || state.demoPanelAppID;
  state.demoPanelURL = runtime.url || "";
  state.demoPanelStatus = runtime.status || state.demoPanelStatus;
  state.demoPanelReason = runtime.reason || "";
}

function closeDemoPanel() {
  state.demoPanelOpen = false;
  state.demoPanelTitle = "";
  state.demoPanelURL = "";
  state.demoPanelAppID = "";
  state.demoPanelOrigin = "";
  state.demoPanelKind = "demo";
  state.demoPanelStatus = "";
  state.demoPanelReason = "";
}

async function resolveLivePreview(appID = "", materialID = "") {
  let detail = null;
  let lastError = null;
  if (String(appID || "").trim()) {
    try {
      detail = await transport.ensureLiveApp({ path: state.path, app_id: appID });
    } catch (error) {
      lastError = error;
    }
  }
  if (!detail && state.selectedSessionId && String(materialID || "").trim()) {
    try {
      detail = await transport.startMaterialLiveApp({
        path: state.path,
        session_id: state.selectedSessionId,
        material_id: materialID,
      });
    } catch (error) {
      lastError = error;
    }
  }
  if (!detail) {
    throw lastError || new Error(state.locale === "en" ? "Miniapp preview is unavailable." : "Миниапп сейчас недоступен.");
  }
  const target = detail?.preview_url || detail?.previewURL || "";
  if (!target) {
    throw new Error(state.locale === "en" ? "Miniapp preview is unavailable." : "Миниапп сейчас недоступен.");
  }
  await refreshLiveApps(state.selectedSessionId || "");
  if (state.selectedSessionId) {
    await loadSessionDetail(state.selectedSessionId);
  }
  return detail;
}

function renderLiveWorkStatus(detail) {
  const live = detail?.live || null;
  const transcript = live?.artifact_previews?.transcript || live?.artifactPreviews?.transcript || "";
  const stderr = live?.artifact_previews?.stderr || live?.artifactPreviews?.stderr || "";
  const error = sessionError(detail);
  const phase = livePhase(detail, transcript, stderr);
  const phaseDetail = livePhaseDetail(detail, transcript, stderr);
  if (!detail?.session) {
    return "";
  }
  if (detail.session.status === "failed") {
    return `
      <div class="live-status-card inline-work-card failed">
        <div class="live-status-row">
          <div>
            <div class="eyebrow">${escapeHTML(t("chat.liveStatus"))}</div>
            <div class="body-copy"><strong>${escapeHTML(t("chat.failed"))}</strong></div>
          </div>
          <span class="pill status-pill failed">${escapeHTML(sessionStateLabel(detail.session.status || "failed"))}</span>
        </div>
        <div class="body-copy">${escapeHTML(error || t("chat.replyMissing"))}</div>
        ${stderr ? `<pre class="code-block compact">${escapeHTML(stderr)}</pre>` : ""}
      </div>
    `;
  }
  if (detail.session.status !== "running" && !transcript && !stderr) {
    return "";
  }
  return `
    <div class="live-status-card inline-work-card">
      <div class="live-status-row">
        <div>
          <div class="eyebrow">${escapeHTML(t("chat.liveStatus"))}</div>
          <div class="body-copy"><strong>${escapeHTML(phase)}</strong></div>
        </div>
        <span class="pill status-pill">${escapeHTML(sessionStateLabel(detail.session.status || "running"))}</span>
      </div>
      <div class="muted">${escapeHTML(phaseDetail || detail.next_action || t("chat.waiting"))}</div>
      ${transcript ? `<pre class="code-block compact">${escapeHTML(transcript)}</pre>` : ""}
      ${stderr ? `<pre class="code-block compact">${escapeHTML(stderr)}</pre>` : ""}
    </div>
  `;
}

function sessionLiveApps(detail = state.sessionDetail) {
  return detail?.live_apps || detail?.liveApps || state.liveApps || [];
}

function findLiveApp(detail, appID = "", origin = "") {
  const apps = sessionLiveApps(detail);
  if (appID) {
    const byID = apps.find((item) => item.id === appID);
    if (byID) return byID;
  }
  if (origin) {
    return apps.find((item) => (item.origin || "") === origin) || null;
  }
  return null;
}

function liveAppRuntime(detail, appID = "", origin = "", fallbackURL = "") {
  const liveApp = findLiveApp(detail, appID, origin);
  return {
    liveApp,
    liveAppID: liveApp?.id || appID || "",
    status: liveApp?.status || "",
    url: liveApp?.preview_url || liveApp?.previewURL || fallbackURL || "",
    reason: liveApp?.stop_reason || liveApp?.stopReason || "",
  };
}

function liveActionKey(appID = "", origin = "") {
  return String(origin || appID || "").trim();
}

function setLiveActionState(appID = "", origin = "", next = null) {
  const key = liveActionKey(appID, origin);
  if (!key) return;
  if (next) {
    state.liveActionState[key] = next;
    return;
  }
  delete state.liveActionState[key];
}

function currentLiveAction(appID = "", origin = "") {
  const originKey = liveActionKey("", origin);
  if (originKey && state.liveActionState[originKey]) {
    return state.liveActionState[originKey];
  }
  const appKey = liveActionKey(appID, "");
  if (appKey && state.liveActionState[appKey]) {
    return state.liveActionState[appKey];
  }
  return null;
}

function livePhase(detail, transcript, stderr) {
  const runtime = sessionLiveApps(detail);
  const lower = String(transcript || "").toLowerCase();
  if (stderr) return state.locale === "en" ? "Needs attention" : "Нужна проверка";
  if (runtime.some((item) => item.status === "starting")) {
    return state.locale === "en" ? "Launching miniapp" : "Поднимает миниапп";
  }
  if (lower.includes("mermaid") || lower.includes("<svg") || lower.includes("diagram")) {
    return state.locale === "en" ? "Preparing diagram" : "Готовит схему";
  }
  if (lower.includes("<html") || lower.includes("simulation") || lower.includes("demo")) {
    return state.locale === "en" ? "Building miniapp" : "Собирает миниапп";
  }
  if (transcript) return state.locale === "en" ? "Preparing answer" : "Готовит ответ";
  return state.locale === "en" ? "Thinking" : "Думает";
}

function livePhaseDetail(detail, transcript, stderr) {
  if (stderr) return excerpt(stderr, 180);
  if (transcript) return excerpt(transcript, 180);
  const starting = sessionLiveApps(detail).find((item) => item.status === "starting");
  if (starting) {
    return state.locale === "en"
      ? `Preparing ${starting.title || "miniapp"} for preview.`
      : `Готовит ${starting.title || "миниапп"} и проверяет preview.`;
  }
  return state.locale === "en"
    ? "Checks context and assembles the next useful result."
    : "Проверяет контекст и собирает следующий полезный результат.";
}

function renderChatRail() {
  const sessions = state.sessions || [];
  const projectLabel = state.projectState?.name || summarizeProjectPath(state.projectState?.path || state.path || "");
  const projectSubtitle = state.projectState?.message || t("project.missing");
  return `
    <div class="chat-rail-project">
      <div class="eyebrow">${escapeHTML(t("topbar.project"))}</div>
      <button id="chat-rail-projects" class="project-chip rail-project-chip">${escapeHTML(projectLabel || "ARC")}</button>
      <div class="muted rail-project-subtitle">${escapeHTML(projectSubtitle)}</div>
    </div>
    <div class="chat-rail-agent">
      ${renderAgentMenu("project", projectDefaultMode(), { topbar: false })}
    </div>
    <div class="chat-rail-actions">
      <button id="chat-rail-projects-secondary" class="secondary-button">${escapeHTML(t("chat.openProject"))}</button>
      <button id="chat-rail-new" class="primary-button">${escapeHTML(t("chat.newTopic"))}</button>
    </div>
    <div class="chat-rail-divider"><span>${escapeHTML(t("chat.railDivider"))}</span></div>
    <div class="chat-rail-header">
      <div class="eyebrow">${escapeHTML(t("chat.topicsRail"))}</div>
    </div>
    <input id="chat-rail-search" class="text-input compact chat-rail-search-input" placeholder="${escapeHTML(t("sessions.search"))}" value="${escapeHTML(state.sessionSearch)}" />
    <div class="chat-rail-list">
      ${sessions.length ? sessions.map((session) => renderSessionListCard(session, session.id === state.selectedSessionId)).join("") : emptyBlock(t("sessions.empty"))}
    </div>
  `;
}

function agentMenuOpen(scope) {
  return scope === "project" ? state.projectAgentMenuOpen : state.sessionAgentMenuOpen;
}

function installedPresets() {
  const workspace = state.workspace || state.projectState?.workspace || null;
  const items = workspace?.installed_presets || workspace?.installedPresets || [];
  return Array.isArray(items) ? items : [];
}

function agentDetails(mode) {
  return (state.agents || []).find((agent) => agent.mode === mode) || null;
}

function renderAgentMenu(scope, currentMode, options = {}) {
  const open = agentMenuOpen(scope);
  const currentAgent = agentDetails(currentMode);
  const title = scope === "project" ? t("chat.agentProjectMenu") : t("chat.agentTopicMenu");
  const presets = installedPresets();
  const currentName = currentAgent?.name || agentNameByMode(currentMode);
  const helperText = currentAgent?.tagline || currentAgent?.description || "";
  const topbar = !!options.topbar;
  return `
    <div class="header-agent-control ${topbar ? "topbar-agent-control" : ""}">
      <div class="compact-menu">
        <button class="compact-menu-button agent-menu-button ${open ? "active" : ""}" data-toggle-agent-menu="${escapeHTML(scope)}">
          <span class="compact-menu-label">${escapeHTML(title)}</span>
          <strong>${escapeHTML(currentName)}</strong>
          ${helperText ? `<span class="compact-menu-meta">${escapeHTML(helperText)}</span>` : ""}
        </button>
        ${open ? `
          <div class="compact-menu-popover agent-picker-popover">
            <div class="compact-menu-section">
              <div class="eyebrow">${escapeHTML(t("chat.builtInAgents"))}</div>
              <div class="compact-menu-list">
                ${(state.agents || []).map((agent) => `
                  <button class="compact-menu-item ${agent.mode === currentMode ? "active" : ""}" data-select-agent="${escapeHTML(scope)}:${escapeHTML(agent.mode)}">
                    <strong>${escapeHTML(agent.name)}</strong>
                    <span>${escapeHTML(agent.tagline || agent.description || "")}</span>
                  </button>
                `).join("")}
              </div>
            </div>
            ${presets.length ? `
              <div class="compact-menu-section">
                <div class="eyebrow">${escapeHTML(t("chat.installedPresets"))}</div>
                <div class="compact-menu-list">
                  ${presets.map((preset) => `
                    <div class="compact-menu-item static">
                      <strong>${escapeHTML(preset.preset_id || preset.presetID || preset.install_id || preset.installID || "preset")}</strong>
                      <span>${escapeHTML(`${preset.version || ""} · ${preset.status || ""}`.trim())}</span>
                    </div>
                  `).join("")}
                </div>
                <div class="muted compact-menu-footnote">${escapeHTML(t("chat.installedPresetHint"))}</div>
              </div>
            ` : ""}
          </div>
        ` : ""}
      </div>
    </div>
  `;
}

function renderThreadOutputStrip(detail) {
  const materials = detail?.materials || [];
  if (!materials.length && !(state.workspaceExplorer?.files || []).length && !(state.projectMaterials || []).length) return "";
  return `
    <div class="thread-output-strip">
      <div class="thread-output-label">${escapeHTML(t("chat.inlineResults"))}</div>
      <div class="thread-output-chips">
        <button class="thread-output-chip" data-open-chat-panel="details"><span>${escapeHTML(t("chat.resultPanel"))}</span><small>${escapeHTML(t("chat.openDetails"))}</small></button>
      </div>
    </div>
  `;
}

function isVisualOutput(output) {
  return ["diagram", "demo", "simulation"].includes(output?.kind || output?.type);
}

function outputKind(output) {
  return output?.kind || output?.type || "";
}

function renderLaunchState(runtime, kind, actionState = null) {
  if (actionState?.phase === "starting") {
    return `<div class="inline-output-state"><strong>${escapeHTML(state.locale === "en" ? "Starting miniapp" : "Запускаю миниапп")}</strong><span>${escapeHTML(actionState.message || (state.locale === "en" ? "ARC is preparing the preview inside the chat." : "ARC поднимает preview прямо в чате."))}</span></div>`;
  }
  if (actionState?.phase === "opening") {
    return `<div class="inline-output-state"><strong>${escapeHTML(state.locale === "en" ? "Opening in ARC" : "Открываю в ARC")}</strong><span>${escapeHTML(actionState.message || (state.locale === "en" ? "ARC is preparing a fresh preview inside the app." : "ARC готовит свежий preview прямо внутри приложения."))}</span></div>`;
  }
  if (actionState?.phase === "stopping") {
    return `<div class="inline-output-state"><strong>${escapeHTML(state.locale === "en" ? "Stopping miniapp" : "Останавливаю миниапп")}</strong><span>${escapeHTML(actionState.message || (state.locale === "en" ? "The current preview instance is shutting down." : "Текущий preview-экземпляр завершается."))}</span></div>`;
  }
  if (actionState?.phase === "failed") {
    return `<div class="inline-output-state error"><strong>${escapeHTML(t("live.failed"))}</strong><span>${escapeHTML(actionState.message || (state.locale === "en" ? "Miniapp action failed." : "Не удалось выполнить действие с миниаппом."))}</span></div>`;
  }
  const status = runtime.status || "";
  if (!status) {
    return `<div class="inline-output-state"><strong>${escapeHTML(state.locale === "en" ? "Saved in chat" : "Сохранено в чате")}</strong><span>${escapeHTML(materialLeadText({ type: kind }))}</span></div>`;
  }
  if (status === "starting") {
    return `<div class="inline-output-state"><strong>${escapeHTML(t("live.starting"))}</strong><span>${escapeHTML(state.locale === "en" ? "ARC is launching a fresh preview instance." : "ARC поднимает новый preview-экземпляр.")}</span></div>`;
  }
  if (status === "stopped") {
    return `<div class="inline-output-state"><strong>${escapeHTML(t("live.stopped"))}</strong><span>${escapeHTML(state.locale === "en" ? "The files are still saved in chat. You can restart this miniapp." : "Файлы сохранены в чате. Миниапп можно запустить снова.")}</span></div>`;
  }
  if (status === "failed") {
    return `<div class="inline-output-state error"><strong>${escapeHTML(t("live.failed"))}</strong><span>${escapeHTML(runtime.reason || (state.locale === "en" ? "Preview instance failed to start." : "Не удалось поднять preview-экземпляр."))}</span></div>`;
  }
  return `<div class="inline-output-state"><strong>${escapeHTML(status)}</strong></div>`;
}

function liveStatusValue(runtime, actionState = null) {
  if (actionState?.phase === "starting") return "starting";
  if (actionState?.phase === "opening") return "ready";
  if (actionState?.phase === "stopping") return "starting";
  if (actionState?.phase === "failed") return "failed";
  return runtime.status || "";
}

function renderMiniappActionButtons({ runtime, actionState = null, originID = "", launchable = false, detailButton = "" }) {
  const buttons = [];
  if (detailButton) buttons.push(detailButton);
  if (!launchable) return buttons.join("");
  const status = liveStatusValue(runtime, actionState);
  const transient = actionState?.phase === "starting" || actionState?.phase === "opening" || actionState?.phase === "stopping";
  if ((status === "ready" || status === "starting") && runtime.liveAppID) {
    buttons.push(`<button class="link-button" data-open-live-app="${escapeHTML(runtime.liveAppID)}" data-open-live-origin="${escapeHTML(originID)}">${escapeHTML(t("live.open"))}</button>`);
    if (!transient) {
      buttons.push(`<button class="secondary-button" data-stop-live-app="${escapeHTML(runtime.liveAppID)}">${escapeHTML(t("live.stop"))}</button>`);
    }
    return buttons.join("");
  }
  if (originID && !transient) {
    buttons.push(`<button class="secondary-button" data-launch-material="${escapeHTML(originID)}">${escapeHTML(t(status === "stopped" || status === "failed" ? "live.restart" : "material.launch"))}</button>`);
  }
  return buttons.join("");
}

function outputInlinePreview(output, detail) {
  const kind = outputKind(output);
  const runtime = liveAppRuntime(detail, output.live_app_id || output.liveAppID || "", output.id, output.url || output.preview_url || output.previewURL || "");
  const actionState = currentLiveAction(runtime.liveAppID, output.id);
  if (kind === "diagram" && String(output.preview || "").includes("<svg")) {
    return `<div class="inline-visual-canvas">${output.preview}</div>`;
  }
  if (kind === "demo" || kind === "simulation") {
    if ((runtime.status === "ready" || runtime.status === "starting") && runtime.url) {
      const frameKey = runtime.liveAppID || output.id || runtime.url;
      return renderInlineFrameSlot(`thread-output:${frameKey}`, "inline-output-frame", runtime.url, output.title);
    }
    return renderLaunchState(runtime, kind, actionState);
  }
  if (kind === "document" && output.preview) {
    return `<div class="document-preview compact">${renderStructuredPreview(output.preview)}</div>`;
  }
  return `<div class="empty">${escapeHTML(output.error || output.preview || t("material.previewEmpty"))}</div>`;
}

function renderInlineVisualMaterial(material, detail) {
  const runtime = liveAppRuntime(detail, material.live_app_id || material.liveAppID || "", material.id, material.url || "");
  const actionState = currentLiveAction(runtime.liveAppID, material.id);
  const preview = material.type === "diagram" && String(material.preview || "").includes("<svg")
    ? `<div class="inline-visual-canvas">${material.preview}</div>`
    : (runtime.status === "ready" || runtime.status === "starting") && runtime.url
      ? renderInlineFrameSlot(`thread-material:${runtime.liveAppID || material.id || runtime.url}`, "inline-output-frame", runtime.url, material.title)
      : renderLaunchState(runtime, material.type, actionState);
  return `
    <article class="inline-visual-card ${escapeHTML(material.type || "")}">
      <div class="list-title"><span>${escapeHTML(material.title)}</span><span class="pill">${escapeHTML(materialTypeLabel(material.type))}</span></div>
      <div class="body-copy">${escapeHTML(material.summary || materialLeadText(material))}</div>
      ${preview}
      ${(runtime.status === "ready" || runtime.status === "starting") && actionState ? renderLaunchState(runtime, material.type, actionState) : ""}
      <div class="button-row inline-visual-actions">
        ${renderMiniappActionButtons({
          runtime,
          actionState,
          originID: material.id,
          launchable: !!material.launchable,
          detailButton: `<button class="secondary-button" data-open-output-inline="${escapeHTML(material.id)}:ready">${escapeHTML(t("chat.openDetails"))}</button>`,
        })}
      </div>
    </article>
  `;
}

function renderInlineLiveApp(item) {
  const actionState = currentLiveAction(item.id, item.origin || "");
  const runtime = {
    liveAppID: item.id || "",
    status: item.status || "",
    url: item.preview_url || item.previewURL || "",
    reason: item.stop_reason || item.stopReason || "",
  };
  return `
    <article class="inline-visual-card demo">
      <div class="list-title"><span>${escapeHTML(item.title)}</span><span class="pill">${escapeHTML(liveStatusLabel(item.status))}</span></div>
      <div class="body-copy">${escapeHTML(item.origin || t("material.demoLead"))}</div>
      ${(item.status === "ready" || item.status === "starting") && (item.preview_url || item.previewURL)
        ? renderInlineFrameSlot(`thread-live-app:${item.id || item.preview_url || item.previewURL}`, "inline-output-frame", item.preview_url || item.previewURL, item.title)
        : renderLaunchState(runtime, item.type || "demo", actionState)}
      ${(item.status === "ready" || item.status === "starting") && actionState ? renderLaunchState(runtime, item.type || "demo", actionState) : ""}
      <div class="button-row inline-visual-actions">
        ${renderMiniappActionButtons({
          runtime,
          actionState,
          originID: item.origin || "",
          launchable: true,
          detailButton: `<button class="secondary-button" data-open-demo-tab="${escapeHTML(item.id)}">${escapeHTML(t("chat.openDetails"))}</button>`,
        })}
      </div>
    </article>
  `;
}

function renderMessageOutputs(message, detail) {
  const outputs = (message?.outputs || []).filter((item) => item.inline || isVisualOutput(item) || outputKind(item) === "document");
  if (!outputs.length) return "";
  return `
    <div class="message-output-stack">
      ${outputs.map((output) => renderMessageOutput(output, detail)).join("")}
    </div>
  `;
}

function renderMessageOutput(output, detail) {
  const kind = outputKind(output);
  const isVisual = kind === "diagram" || kind === "demo" || kind === "simulation";
  const expanded = !!state.expandedInlineOutputs[output.id];
  const runtime = liveAppRuntime(detail, output.live_app_id || output.liveAppID || "", output.id, output.url || output.preview_url || output.previewURL || "");
  const liveAppID = runtime.liveAppID;
  const actionState = currentLiveAction(liveAppID, output.id);
  const selectedMaterial = (detail?.materials || []).find((item) => item.id === output.id) || null;
  return `
    <article class="inline-visual-card ${escapeHTML(kind)} ${expanded ? "expanded" : ""}">
      <div class="list-title"><span>${escapeHTML(output.title)}</span><span class="pill">${escapeHTML(materialTypeLabel(kind))}</span></div>
      <div class="body-copy">${escapeHTML(output.error || selectedMaterial?.summary || materialLeadText({ type: kind }))}</div>
      ${outputInlinePreview(output, detail)}
      ${actionState?.phase === "failed" && (runtime.status === "ready" || runtime.status === "starting") ? renderLaunchState(runtime, kind, actionState) : ""}
      <div class="button-row inline-visual-actions">
        ${isVisual ? `<button class="secondary-button" data-toggle-inline-output="${escapeHTML(output.id)}">${escapeHTML(expanded ? t("chat.collapseInline") : t("chat.expandInline"))}</button>` : ""}
        ${renderMiniappActionButtons({
          runtime,
          actionState,
          originID: output.id,
          launchable: !!output.launchable,
          detailButton: selectedMaterial ? `<button class="secondary-button" data-open-output-inline="${escapeHTML(selectedMaterial.id)}:ready">${escapeHTML(t("chat.openDetails"))}</button>` : "",
        })}
      </div>
    </article>
  `;
}

function materialTypeLabel(type) {
  return t(`result.${type}`) || type || "";
}

function liveStatusLabel(status) {
  switch (status) {
    case "ready":
      return t("live.ready");
    case "starting":
      return t("live.starting");
    case "failed":
      return t("live.failed");
    case "stopped":
      return t("live.stopped");
    default:
      return status || "";
  }
}

function sessionStateLabel(status) {
  return t(`status.${status}`) || status || "";
}

function sessionSignal(session) {
  return excerpt(session.last_assistant_message || session.lastAssistantMessage || session.last_user_message || session.lastUserMessage || session.summary || "", 140);
}

function materialSourceLabel(material) {
  if (!material?.source) return "—";
  if (material.source === "conversation") return t("material.sourceConversation");
  if (String(material.source).includes("lesson")) return t("material.sourceLesson");
  return `${t("material.sourceRun")} · ${excerpt(material.source, 12)}`;
}

function materialLeadText(material) {
  switch (material?.type) {
    case "answer":
      return t("material.answerLead");
    case "plan":
      return t("material.planLead");
    case "diagram":
      return t("material.diagramLead");
    case "demo":
      return t("material.demoLead");
    case "simulation":
      return t("material.simulationLead");
    case "changes":
      return t("material.changesLead");
    case "review":
      return t("material.reviewLead");
    default:
      return t("material.documentLead");
  }
}

function renderMaterialBody(material) {
  if (!material) {
    return emptyBlock(t("chat.noMaterials"));
  }
  const runtime = liveAppRuntime(state.sessionDetail, material.live_app_id || material.liveAppID || "", material.id, material.url || "");
  const actionState = currentLiveAction(runtime.liveAppID, material.id);
  if (material.type === "diagram" && String(material.preview || "").includes("<svg")) {
    return `
      <div class="diagram-preview">
        <div class="diagram-preview-head">
          <strong>${escapeHTML(material.title || materialTypeLabel(material.type))}</strong>
          <span>${escapeHTML(materialLeadText(material))}</span>
        </div>
        <div class="diagram-preview-canvas">${material.preview}</div>
      </div>
    `;
  }
  if (["answer", "plan", "document", "review"].includes(material.type) && material.preview) {
    return `
      <div class="document-preview">
        <div class="document-preview-lead">${escapeHTML(materialLeadText(material))}</div>
        ${renderStructuredPreview(material.preview)}
      </div>
    `;
  }
  if (material.type === "changes" && material.preview) {
    return `<pre class="code-block">${escapeHTML(material.preview)}</pre>`;
  }
  if (material.type === "demo" || material.type === "simulation") {
    if ((runtime.status === "ready" || runtime.status === "starting") && runtime.url) {
      return `${actionState ? renderLaunchState(runtime, material.type, actionState) : ""}<iframe class="preview-frame" src="${escapeHTML(runtime.url)}" title="${escapeHTML(material.title)}"></iframe>`;
    }
    return renderLaunchState(runtime, material.type, actionState);
  }
  if (material.preview) {
    return `<pre class="code-block">${escapeHTML(material.preview)}</pre>`;
  }
  return emptyBlock(t("material.previewEmpty"));
}

function renderSessionListCard(session, selected = false) {
  const title = String(session.title || t("chat.topicUntitled")).trim() || t("chat.topicUntitled");
  return `
    <button class="chat-thread-item ${selected ? "active" : ""}" data-open-session="${escapeHTML(session.id)}">
      <div class="chat-thread-row stacked">
        <span class="chat-thread-title">${escapeHTML(title)}</span>
        <span class="session-status-inline compact">
          <span class="status-dot ${escapeHTML(session.status || "")}"></span>
          ${escapeHTML(formatRailDateLabel(session.updated_at || session.updatedAt || ""))}
        </span>
      </div>
    </button>
  `;
}

function sessionTabs() {
  return [
    { id: "ready", label: t("chat.readyLabel") },
    { id: "details", label: t("chat.detailsLabel") },
  ];
}

function currentInspectorTab() {
  return ["ready", "details"].includes(state.currentSessionTab) ? state.currentSessionTab : "ready";
}

function renderLiveAppsList(apps) {
  if (!apps?.length) {
    return emptyBlock(t("live.empty"));
  }
  return `<div class="material-grid">${apps.map((item) => `
    <div class="live-app-card ${escapeHTML(item.status || "")}">
      <div class="list-title">
        <span>${escapeHTML(item.title)}</span>
        <span class="pill">${escapeHTML(liveStatusLabel(item.status))}</span>
      </div>
      <div class="body-copy">${escapeHTML(item.origin || "")}</div>
      <div class="muted">${escapeHTML(item.preview_url || item.previewURL || "")}</div>
      ${(item.status === "ready" || item.status === "starting") && (item.preview_url || item.previewURL)
        ? `<iframe class="preview-frame compact" src="${escapeHTML(item.preview_url || item.previewURL)}" title="${escapeHTML(item.title)}"></iframe>`
        : renderLaunchState({ status: item.status, reason: item.stop_reason || item.stopReason || "" }, item.type || "demo", currentLiveAction(item.id, item.origin || ""))}
      <div class="button-row">${renderMiniappActionButtons({
        runtime: {
          liveAppID: item.id || "",
          status: item.status || "",
          url: item.preview_url || item.previewURL || "",
          reason: item.stop_reason || item.stopReason || "",
        },
        actionState: currentLiveAction(item.id, item.origin || ""),
        originID: item.origin || "",
        launchable: true,
        detailButton: item.origin ? `<button class="secondary-button" data-open-output-inline="${escapeHTML(item.origin)}:details">${escapeHTML(t("chat.openDetails"))}</button>` : "",
      })}</div>
    </div>
  `).join("")}</div>`;
}

function renderMaterialPreview(material) {
  if (!material) {
    return emptyBlock(t("chat.inspectorEmpty"));
  }
  const metaBits = [
    materialSourceLabel(material),
    material.summary ? excerpt(material.summary, 72) : "",
    material.files?.length ? `${material.files.length} file${material.files.length > 1 ? "s" : ""}` : "",
  ].filter(Boolean);
  const fileButtons = material.files?.length
    ? `
      <div class="section-gap">
        <strong>${escapeHTML(t("material.files"))}</strong>
        <div class="pill-row">${material.files.filter(Boolean).map((file) => `<button class="pill removable-pill" data-open-file="${escapeHTML(file)}">${escapeHTML(file)}</button>`).join("")}</div>
      </div>
    `
    : "";
  const filePreview = state.selectedFileDetail
    ? `<div class="section-gap"><strong>${escapeHTML(t("file.preview"))}</strong><pre class="code-block">${escapeHTML(state.selectedFileDetail.content || "")}</pre></div>`
    : "";
  return `
    <div class="material-preview-shell">
      <div class="material-preview-header">
        <div>
          <div class="eyebrow">${escapeHTML(t("material.source"))}: ${escapeHTML(materialSourceLabel(material))}</div>
          <div class="list-title"><span>${escapeHTML(material.title)}</span><span>${escapeHTML(materialTypeLabel(material.type))}</span></div>
        </div>
        <div class="pill-row">
          <span class="pill">${escapeHTML(materialTypeLabel(material.type))}</span>
        </div>
      </div>
      <div class="material-meta-strip">${metaBits.map((bit) => `<span class="material-meta-pill">${escapeHTML(bit)}</span>`).join("")}</div>
      <div class="material-intro">
        <div class="body-copy">${escapeHTML(material.summary || "")}</div>
        <div class="muted">${escapeHTML(materialLeadText(material))}</div>
      </div>
      <div class="section-gap"><strong>${escapeHTML(t("material.preview"))}</strong></div>
      <div class="button-row">${renderMiniappActionButtons({
        runtime: liveAppRuntime(state.sessionDetail, material.live_app_id || material.liveAppID || "", material.id, material.url || ""),
        actionState: currentLiveAction(material.live_app_id || material.liveAppID || "", material.id),
        originID: material.id,
        launchable: !!material.launchable,
      })}</div>
      ${renderMaterialBody(material)}
      ${fileButtons}
      ${filePreview}
    </div>
  `;
}

function renderDetailsInspector(detail) {
  const attached = attachedSessionPills();
  const hasAttached = state.attachedSessionIDs.length > 0;
  const selectedMaterial = (detail.materials || []).find((item) => item.id === state.selectedMaterialId) || currentVisualMaterial(detail) || currentDocumentMaterial(detail) || null;
  const relatedFiles = selectedMaterial?.files || [];
  const relatedFilesBlock = relatedFiles.length ? `
    <div class="section-gap">
      <strong>${escapeHTML(t("chat.relatedFiles"))}</strong>
      <div class="pill-row">${relatedFiles.map((file) => `<button class="pill removable-pill" data-open-file="${escapeHTML(file)}">${escapeHTML(file)}</button>`).join("")}</div>
    </div>
  ` : "";
  const filePreview = state.selectedFileDetail
    ? `<div class="section-gap"><strong>${escapeHTML(t("file.preview"))}</strong><pre class="code-block">${escapeHTML(state.selectedFileDetail.content || "")}</pre></div>`
    : "";
  return `
    <div class="inspector-summary-grid">
      <div class="session-overview-card">
        <div class="eyebrow">${escapeHTML(t("sessions.next"))}</div>
        <strong>${escapeHTML(excerpt(detail.next_action || t("chat.idle"), 88))}</strong>
      </div>
      <div class="session-overview-card">
        <div class="eyebrow">${escapeHTML(t("chat.projectAgent"))}</div>
        <strong>${escapeHTML(detail.session.agent_name || agentNameByMode(projectDefaultMode()))}</strong>
      </div>
    </div>
    <div class="live-status-card">
      <div class="eyebrow">${escapeHTML(t("chat.detailsLabel"))}</div>
      <div class="body-copy">${escapeHTML(t("chat.detailsHint"))}</div>
    </div>
    <div class="section-gap">
      <strong>${escapeHTML(t("attached.title"))}</strong>
      ${hasAttached ? attached : emptyBlock(t("attached.empty"))}
    </div>
    ${relatedFilesBlock}
    ${filePreview}
  `;
}

function renderFilesInspector() {
  const explorerFiles = (state.workspaceExplorer?.files || []).filter((item) => !String(item.path || "").startsWith(".arc/"));
  const arcFiles = state.projectMaterials || [];
  const selectedPath = state.selectedFilePath || "";
  const selectedDetails = state.selectedFileDetail
    ? `<div class="section-gap"><strong>${escapeHTML(t("file.preview"))}</strong><pre class="code-block">${escapeHTML(state.selectedFileDetail.content || "")}</pre></div>`
    : emptyBlock(t("file.none"));
  const renderFileButton = (path, title, meta, extraAction = "") => `
    <div class="list-button static ${selectedPath === path ? "active" : ""}">
      <button class="list-button file-list-button ${selectedPath === path ? "active" : ""}" data-open-file="${escapeHTML(path)}">
        <div class="list-title"><span>${escapeHTML(title)}</span><span>${escapeHTML(meta)}</span></div>
        <div class="body-copy">${escapeHTML(path)}</div>
      </button>
      ${extraAction}
    </div>
  `;
  return `
    <div class="section-gap">
      <strong>${escapeHTML(t("files.arc"))}</strong>
      <div class="material-list">
        ${arcFiles.length ? arcFiles.map((item) => renderFileButton(
          item.path,
          item.name,
          formatRailDateLabel(item.uploaded_at || item.uploadedAt || ""),
          `<div class="button-row"><button class="secondary-button" data-delete-project-material="${escapeHTML(item.id)}">${escapeHTML(t("files.delete"))}</button></div>`
        )).join("") : emptyBlock(t("chat.noProjectMaterials"))}
      </div>
    </div>
    <div class="section-gap">
      <strong>${escapeHTML(t("files.project"))}</strong>
      <div class="material-list">
        ${explorerFiles.length ? explorerFiles.slice(0, 200).map((item) => renderFileButton(
          item.path,
          item.path.split("/").pop() || item.path,
          formatRailDateLabel(item.mod_time || item.modTime || "")
        )).join("") : emptyBlock(t("files.empty"))}
      </div>
    </div>
    ${selectedDetails}
  `;
}

function renderSessionPanel(detail) {
  if (!detail) {
    return card(t("chat.resultPanel"), `<div class="inspector-empty-state"><div class="eyebrow">${escapeHTML(t("chat.resultPanel"))}</div><strong>${escapeHTML(t("chat.inspectorEmpty"))}</strong></div>`, "card--subtle inspector-card");
  }
  const panelTitle = t("chat.resultPanel");
  const materials = detail.materials || [];
  const material = (detail.materials || []).find((item) => item.id === state.selectedMaterialId) || currentVisualMaterial(detail) || currentDocumentMaterial(detail) || null;
  const liveApps = detail.live_apps || detail.liveApps || [];
  const materialsList = `
    <div class="material-list">${materials.length ? materials.map((item) => `
      <button class="list-button material-card ${state.selectedMaterialId === item.id ? "active" : ""}" data-material-id="${escapeHTML(item.id)}">
        <div class="list-title"><span>${escapeHTML(item.title)}</span><span class="pill">${escapeHTML(materialTypeLabel(item.type))}</span></div>
        <div class="body-copy">${escapeHTML(item.summary)}</div>
        <div class="session-meta-row">
          <span>${escapeHTML(materialSourceLabel(item))}</span>
          <span>${escapeHTML(materialLeadText(item))}</span>
        </div>
      </button>
    `).join("") : emptyBlock(t("chat.noOutputs"))}</div>
  `;
  return card(panelTitle, `
    <div class="muted">${escapeHTML(detail.session.summary || detail.next_action || "")}</div>
    <div class="session-panel details-chat-panel">
      <section class="details-chat-section">
        <div class="section-title-row">
          <strong>${escapeHTML(t("chat.liveLabel"))}</strong>
          <span class="muted">${escapeHTML(t("chat.liveNowHint"))}</span>
        </div>
        ${renderLiveAppsList(liveApps)}
      </section>
      <section class="details-chat-section">
        <div class="section-title-row">
          <strong>${escapeHTML(t("chat.readyLabel"))}</strong>
          <span class="muted">${escapeHTML(t("chat.materialsHint"))}</span>
        </div>
        ${materialsList}
        ${renderMaterialPreview(material)}
      </section>
      <section class="details-chat-section">
        <div class="section-title-row">
          <strong>${escapeHTML(t("chat.detailsLabel"))}</strong>
          <span class="muted">${escapeHTML(t("chat.filesHint"))}</span>
        </div>
        ${renderFilesInspector()}
      </section>
    </div>
  `, "card--command inspector-card");
}

function renderSessionDrawer(detail) {
  if (!state.drawerOpen || !detail) {
    return "";
  }
  return `
    <div class="chat-drawer-backdrop" data-close-drawer="true"></div>
    <div class="chat-drawer-panel">
      <button class="chat-drawer-close" data-close-drawer="true" aria-label="${escapeHTML(t("chat.drawerClose"))}">×</button>
      ${renderSessionPanel(detail)}
    </div>
  `;
}

function messageKey(message, index) {
  return `${message?.role || "message"}:${message?.turn || 0}:${index}`;
}

function renderMessageThread(detail) {
  const messages = detail?.messages || [];
  if (!messages.length) {
    return `<div class="thread-empty-state"><div class="eyebrow">${escapeHTML(t("chat.thread"))}</div><strong>${escapeHTML(t("chat.noMessages"))}</strong></div>`;
  }
  return `
    <div class="message-thread">
      ${messages
        .filter((message) => !(message.role === "assistant" && !String(message.content || "").trim() && !(message.outputs || []).length))
        .map((message, index) => `
        <div class="message-row ${message.role}" data-message-key="${escapeHTML(messageKey(message, index))}">
          <div class="message-bubble ${message.role}">
            <div class="eyebrow">${escapeHTML(message.role === "user" ? "You" : detail?.session?.agent_name || "Agent")}</div>
            ${message.content ? `<div class="markdown-body">${renderMarkdown(message.content || "")}</div>` : ""}
            ${message.failure ? `<div class="banner error inline-banner">${escapeHTML(message.failure)}</div>` : ""}
            ${renderMessageOutputs(message, detail)}
          </div>
        </div>
      `).join("")}
    </div>
  `;
}

function ensureChatWorkspaceShell() {
  const root = byId("chat-main");
  if (!root?.querySelector(".chat-workspace")) {
    root.innerHTML = `
      <div class="chat-workspace">
        <aside id="chat-rail-region" class="chat-rail card card--subtle"></aside>
        <section class="chat-thread-shell">
          <article class="card card--command thread-shell-card">
            <div class="chat-thread-header">
              <div class="chat-thread-heading">
                <h3 id="chat-thread-title"></h3>
              </div>
              <div class="chat-thread-actions">
                <button id="chat-details-button" class="secondary-button">${escapeHTML(t("chat.resultPanel"))}</button>
              </div>
            </div>
            <div class="thread-shell-body">
              <div class="message-thread-scroll">
                <div id="chat-thread-region"></div>
              </div>
            </div>
            <div id="chat-composer-region" class="chat-composer-shell"></div>
          </article>
        </section>
      </div>
    `;
  }
}

function renderChatThreadRegion(detail) {
  if ((state.renderedThreadSessionId || "") === (state.selectedSessionId || "")) {
    captureThreadViewportState();
  }
  capturePreservedThreadFrames();
  byId("chat-thread-title").textContent = detail?.session?.title || t("chat.topicUntitled");
  byId("chat-thread-region").innerHTML = `
    ${renderMessageThread(detail)}
    ${renderLiveWorkStatus(detail)}
    ${renderThreadOutputStrip(detail)}
  `;
  state.renderedThreadSessionId = state.selectedSessionId || "";
  hydratePreservedThreadFrames();
  restoreThreadViewportStateDeferred();
  bindThreadViewportTracking();
}

function renderChatComposerRegion() {
  byId("chat-composer-region").innerHTML = `
    <textarea id="composer-input" rows="3" placeholder="${escapeHTML(t("chat.placeholder"))}">${escapeHTML(state.composerText)}</textarea>
    <div class="button-row composer-row">
      <button id="chat-upload-button" class="secondary-button">${escapeHTML(t("chat.uploadInline"))}</button>
      <input id="project-materials-input" type="file" multiple hidden />
      <button id="chat-send-button" class="primary-button ${state.composerSending ? "is-loading" : ""}">${escapeHTML(state.composerSending ? state.composerStatus || t("chat.send") : t("chat.send"))}</button>
    </div>
    ${renderProjectMaterialsCompact()}
  `;
}

function renderChatLiveRefresh() {
  if (state.currentScreen !== "chat" || !isReady()) {
    return;
  }
  ensureChatWorkspaceShell();
  renderChatThreadRegion(state.sessionDetail);
  byId("chat-side").className = `chat-side-drawer ${state.drawerOpen && state.sessionDetail ? "open" : ""}`;
  byId("chat-side").innerHTML = renderSessionDrawer(state.sessionDetail);
  bindChatSurfaceActions();
}

function bindChatSurfaceActions() {
  if (!isReady()) {
    return;
  }
  document.querySelectorAll("[data-open-session]").forEach((button) => {
    button.onclick = async () => {
      state.projectAgentMenuOpen = false;
      state.sessionAgentMenuOpen = false;
      state.drawerOpen = false;
      await loadSessionDetail(button.dataset.openSession);
      renderAll();
    };
  });
  if (byId("chat-rail-new")) {
    byId("chat-rail-new").onclick = () => {
      state.selectedSessionId = "";
      state.sessionDetail = null;
      state.selectedMaterialId = "";
      state.selectedFilePath = "";
      state.selectedFileDetail = null;
      state.currentSessionTab = "ready";
      state.selectedAgent = projectDefaultMode();
      state.projectAgentMenuOpen = false;
      state.sessionAgentMenuOpen = false;
      state.drawerOpen = false;
      state.composerText = "";
      refreshAllowedActions().then(() => renderAll());
    };
  }
  document.querySelectorAll("#chat-rail-projects, #chat-rail-projects-secondary").forEach((button) => {
    button.onclick = () => {
      state.showWelcome = true;
      renderWelcomeOverlay();
    };
  });
  if (byId("chat-rail-search")) {
    byId("chat-rail-search").oninput = async (event) => {
      state.sessionSearch = event.target.value;
      await refreshSessions(false);
      renderAll();
    };
  }
  if (byId("composer-input")) {
    byId("composer-input").oninput = (event) => {
      state.composerText = event.target.value;
    };
    byId("composer-input").onkeydown = (event) => {
      if (event.key !== "Enter") return;
      if (event.shiftKey || event.isComposing) return;
      event.preventDefault();
      if (state.composerSending) return;
      submitComposer();
    };
  }
  if (byId("chat-send-button")) {
    byId("chat-send-button").onclick = () => submitComposer();
  }
  byId("chat-details-button")?.addEventListener("click", () => {
    state.drawerOpen = true;
    renderAll();
  });
  if (byId("chat-upload-button")) {
    byId("chat-upload-button").onclick = () => {
      byId("project-materials-input")?.click();
    };
  }
  if (byId("project-materials-input")) {
    byId("project-materials-input").onchange = async (event) => {
      const files = event.target.files;
      if (!files?.length) return;
      await uploadProjectMaterials(files);
      event.target.value = "";
    };
  }
  document.querySelectorAll("[data-session-tab]").forEach((button) => {
    button.onclick = () => {
      state.drawerOpen = true;
      renderAll();
    };
  });
  document.querySelectorAll("[data-open-output]").forEach((button) => {
    button.onclick = () => {
      state.selectedMaterialId = button.dataset.openOutput;
      state.drawerOpen = true;
      renderAll();
    };
  });
  document.querySelectorAll("[data-open-output-inline]").forEach((button) => {
    button.onclick = () => {
      const [materialID] = String(button.dataset.openOutputInline || "").split(":");
      state.selectedMaterialId = materialID || "";
      state.drawerOpen = true;
      renderAll();
    };
  });
  document.querySelectorAll("[data-open-demo-tab]").forEach((button) => {
    button.onclick = () => {
      state.drawerOpen = true;
      renderAll();
    };
  });
  document.querySelectorAll("[data-open-chat-panel]").forEach((button) => {
    button.onclick = () => {
      state.drawerOpen = true;
      renderAll();
    };
  });
  document.querySelectorAll("[data-toggle-inline-output]").forEach((button) => {
    button.onclick = () => {
      const id = button.dataset.toggleInlineOutput;
      state.expandedInlineOutputs[id] = !state.expandedInlineOutputs[id];
      renderChatLiveRefresh();
    };
  });
  document.querySelectorAll("[data-material-id]").forEach((button) => {
    button.onclick = () => {
      state.selectedMaterialId = button.dataset.materialId;
      state.currentSessionTab = "ready";
      state.drawerOpen = true;
      renderAll();
    };
  });
  document.querySelectorAll("[data-open-file]").forEach((button) => {
    button.onclick = async () => {
      state.selectedFilePath = button.dataset.openFile;
      state.selectedFileDetail = await transport.workspaceFile(state.path, state.selectedFilePath);
      state.drawerOpen = true;
      renderAll();
    };
  });
  document.querySelectorAll("[data-delete-project-material]").forEach((button) => {
    button.onclick = async () => {
      await deleteProjectMaterial(button.dataset.deleteProjectMaterial);
    };
  });
  document.querySelectorAll("[data-launch-material]").forEach((button) => {
    button.onclick = async () => {
      await launchMaterial(button.dataset.launchMaterial);
    };
  });
  document.querySelectorAll("[data-stop-live-app]").forEach((button) => {
    button.onclick = async () => {
      await stopLiveApp(button.dataset.stopLiveApp);
    };
  });
  document.querySelectorAll("[data-open-live-app]").forEach((button) => {
    button.onclick = async () => {
      await openLiveApp(button.dataset.openLiveApp, button.dataset.openLiveOrigin);
    };
  });
  document.querySelectorAll("[data-detach-session]").forEach((button) => {
    button.onclick = () => {
      state.attachedSessionIDs = state.attachedSessionIDs.filter((id) => id !== button.dataset.detachSession);
      persistAttachedSessions();
      renderChat();
    };
  });
  document.querySelectorAll("[data-close-drawer]").forEach((button) => {
    button.onclick = () => {
      state.drawerOpen = false;
      renderAll();
    };
  });
}

function renderChat() {
  if (!isReady()) {
    byId("chat-main").innerHTML = card(t("nav.chat"), emptyBlock(t("project.missing")));
    byId("chat-side").className = "chat-side-drawer";
    byId("chat-side").innerHTML = "";
    state.renderedThreadSessionId = "";
    return;
  }
  ensureChatWorkspaceShell();
  byId("chat-rail-region").innerHTML = renderChatRail();
  renderChatThreadRegion(state.sessionDetail);
  renderChatComposerRegion();
  byId("chat-side").className = `chat-side-drawer ${state.drawerOpen && state.sessionDetail ? "open" : ""}`;
  byId("chat-side").innerHTML = renderSessionDrawer(state.sessionDetail);
  bindChatSurfaceActions();
}

function testingStatusLabel(status) {
  return t(`testing.status.${status}`) || status || "";
}

function verificationStatusLabel(status) {
  return t(`verification.status.${status}`) || status || "";
}

function renderTesting() {
  if (!state.developerAccess?.can_use_testing) {
    byId("testing-main").innerHTML = card(t("nav.testing"), emptyBlock(t("testing.hidden")));
    byId("testing-side").innerHTML = card(t("testing.current"), emptyBlock(t("testing.hidden")));
    return;
  }

  const scenarios = state.testingScenarios || [];
  const run = state.testingRun;
  const verificationProfiles = state.verificationProfiles || [];
  const verificationRun = state.verificationRun;
  byId("testing-main").innerHTML = `
    ${card(t("testing.title"), `
      <div class="body-copy">${escapeHTML(t("testing.lead"))}</div>
      <div class="material-grid">${scenarios.length ? scenarios.map((scenario) => `
        <div class="agent-card static">
          <div class="list-title"><span>${escapeHTML(scenario.title)}</span><span class="pill">${escapeHTML(agentNameByMode(scenario.agent_id || scenario.agentID || "work"))}</span></div>
          <div class="body-copy">${escapeHTML(scenario.summary)}</div>
          <div class="muted">${escapeHTML(String(scenario.steps || 0))} steps</div>
          <div class="button-row">
            <button class="secondary-button" data-testing-start-step="${escapeHTML(scenario.id)}">${escapeHTML(t("testing.startStep"))}</button>
            <button class="primary-button" data-testing-start-auto="${escapeHTML(scenario.id)}">${escapeHTML(t("testing.startAuto"))}</button>
          </div>
        </div>
      `).join("") : emptyBlock(t("testing.hidden"))}</div>
    `, "card--command")}
    ${card(t("verification.title"), `
      <div class="body-copy">${escapeHTML(t("verification.lead"))}</div>
      <div class="material-grid">${verificationProfiles.length ? verificationProfiles.map((profile) => `
        <div class="agent-card static">
          <div class="list-title"><span>${escapeHTML(profile.title)}</span><span class="pill">${escapeHTML(String((profile.verifier_ids || profile.verifierIDs || []).length))}</span></div>
          <div class="body-copy">${escapeHTML(profile.summary)}</div>
          <div class="muted">${escapeHTML((profile.verifier_ids || profile.verifierIDs || []).join(", "))}</div>
          <div class="button-row">
            <button class="primary-button" data-verification-start="${escapeHTML(profile.id)}">${escapeHTML(t("verification.start"))}</button>
          </div>
        </div>
      `).join("") : emptyBlock(t("verification.empty"))}</div>
    `, "card--subtle")}
  `;

  const currentStep = run?.steps?.[Math.max(run?.current_step || 0, 0)] || null;
  const testingCard = !run ? card(t("testing.current"), emptyBlock(t("testing.empty")), "card--subtle") : card(run.title, `
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("testing.current"))}</div><div>${escapeHTML(testingStatusLabel(run.status))}</div></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(run.step_mode ? t("testing.stepMode") : t("testing.autoMode"))}</div><div>${escapeHTML(String(Math.max(0, run.current_step + 1)))} / ${escapeHTML(String(run.steps?.length || 0))}</div></div>
      ${run.last_error ? `<div class="banner error">${escapeHTML(run.last_error)}</div>` : ""}
      <div class="list">${(run.steps || []).map((step, index) => `
        <div class="list-button static ${index === run.current_step ? "active" : ""}">
          <div class="list-title"><span>${escapeHTML(step.title)}</span><span>${escapeHTML(step.status)}</span></div>
          <div class="body-copy">${escapeHTML(step.summary)}</div>
          ${step.details ? `<div class="muted">${escapeHTML(step.details)}</div>` : ""}
        </div>
      `).join("")}</div>
      <div class="button-row">
        <button class="secondary-button" data-testing-next="true">${escapeHTML(t("testing.next"))}</button>
        <button class="secondary-button" data-testing-quit="true">${escapeHTML(t("testing.quit"))}</button>
        <button class="primary-button" data-testing-end="true">${escapeHTML(t("testing.end"))}</button>
      </div>
      ${currentStep?.preview_url || currentStep?.previewURL ? `<iframe class="preview-frame" src="${escapeHTML(currentStep.preview_url || currentStep.previewURL)}" title="${escapeHTML(currentStep.title)}"></iframe>` : ""}
    `, "card--command");
  const verificationCard = !verificationRun ? card(t("verification.current"), emptyBlock(t("verification.empty")), "card--subtle") : card(verificationRun.title, `
    <div class="metric-row"><div class="metric-label">${escapeHTML(t("verification.current"))}</div><div>${escapeHTML(verificationStatusLabel(verificationRun.status))}</div></div>
    <div class="metric-row"><div class="metric-label">${escapeHTML(t("verification.overall"))}</div><div>${escapeHTML(verificationRun.overall_verdict || "-")}</div></div>
    <div class="metric-row"><div class="metric-label">${escapeHTML(t("verification.blocking"))}</div><div>${escapeHTML(String(verificationRun.blocking_failures || 0))}</div></div>
    <div class="metric-row"><div class="metric-label">${escapeHTML(t("verification.warnings"))}</div><div>${escapeHTML(String(verificationRun.warning_count || 0))}</div></div>
    ${verificationRun.last_error ? `<div class="banner error">${escapeHTML(verificationRun.last_error)}</div>` : ""}
    <div class="list">${(verificationRun.results || []).map((result) => `
      <div class="list-button static">
        <div class="list-title"><span>${escapeHTML(result.title)}</span><span>${escapeHTML(result.verdict || "")}</span></div>
        <div class="body-copy">${escapeHTML(result.summary || "")}</div>
        ${(result.blocked_goals || result.blockedGoals || []).length ? `<div class="muted">${escapeHTML((result.blocked_goals || result.blockedGoals || []).join("; "))}</div>` : ""}
        ${(result.recommended_next || result.recommendedNext) ? `<div class="muted"><strong>${escapeHTML(t("verification.next"))}:</strong> ${escapeHTML(result.recommended_next || result.recommendedNext)}</div>` : ""}
      </div>
    `).join("")}</div>
  `, "card--command");
  byId("testing-side").innerHTML = `${testingCard}${verificationCard}`;

  document.querySelectorAll("[data-testing-start-step]").forEach((button) => {
    button.addEventListener("click", async () => {
      await startTesting(button.dataset.testingStartStep, true);
    });
  });
  document.querySelectorAll("[data-testing-start-auto]").forEach((button) => {
    button.addEventListener("click", async () => {
      await startTesting(button.dataset.testingStartAuto, false);
    });
  });
  byId("testing-side")?.querySelector("[data-testing-next]")?.addEventListener("click", async () => {
    await controlTesting("next");
  });
  byId("testing-side")?.querySelector("[data-testing-quit]")?.addEventListener("click", async () => {
    await controlTesting("quit");
  });
  byId("testing-side")?.querySelector("[data-testing-end]")?.addEventListener("click", async () => {
    await controlTesting("end");
  });
  document.querySelectorAll("[data-verification-start]").forEach((button) => {
    button.addEventListener("click", async () => {
      await startVerification(button.dataset.verificationStart);
    });
  });
}

function renderSettings() {
  const providerRows = (state.providers || []).map((provider) => `
    <div class="metric-row"><div class="metric-label">${escapeHTML(provider.name)}</div><div>${escapeHTML(provider.installed ? t("status.ready") : t("status.missing"))}</div></div>
  `).join("");
  const logLines = state.clientLogs.length
    ? `<pre class="code-block">${escapeHTML(state.clientLogs.map((item) => `[${item.level}] ${item.created_at} ${item.message}`).join("\n"))}</pre>`
    : emptyBlock("Логи интерфейса появятся после действий в приложении.");
  byId("settings-panel").innerHTML = `
    ${card(t("settings.general"), `
      <div class="section-gap"><strong>${escapeHTML(t("settings.projectPath"))}</strong></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("topbar.project"))}</div><div>${escapeHTML(state.projectState?.path || state.path || "-")}</div></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("settings.projectAgent"))}</div><div>${escapeHTML(agentNameByMode(projectDefaultMode()))}</div></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("settings.displayScale"))}</div><div>${escapeHTML(`${state.chatScalePercent}%`)}</div></div>
      <div class="button-row"><button id="display-settings-button" class="secondary-button">${escapeHTML(t("settings.display"))}</button></div>
      <div class="section-gap"><strong>${escapeHTML(t("settings.providers"))}</strong></div>
      ${providerRows || emptyBlock("Нет данных.")}
      <div class="section-gap"><strong>${escapeHTML(t("settings.liveApps"))}</strong></div>
      ${renderLiveAppsList(state.liveApps)}
    `, "card--command")}
    ${card(t("settings.advanced"), `
      <div class="section-gap"><strong>${escapeHTML(t("settings.developer"))}</strong></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("settings.role"))}</div><div>${escapeHTML(state.developerAccess?.role || "-")}</div></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("settings.accessSource"))}</div><div>${escapeHTML(state.developerAccess?.source || "-")}</div></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("nav.testing"))}</div><div>${escapeHTML(state.developerAccess?.can_use_testing ? t("status.ready") : t("status.missing"))}</div></div>
      ${state.developerAccess?.can_manage_role ? `<div class="button-row"><button id="developer-role-toggle" class="secondary-button">${escapeHTML(state.developerAccess?.can_use_testing ? t("settings.disableDevMode") : t("settings.enableDevMode"))}</button></div>` : ""}
      <div class="section-gap"><strong>${escapeHTML(t("settings.desktopLog"))}</strong></div>
      <div class="metric-row"><div class="metric-label">${escapeHTML(t("settings.desktopLog"))}</div><div>${escapeHTML(state.desktopLogPath || "-")}</div></div>
      <div class="section-gap"><strong>${escapeHTML(t("settings.logs"))}</strong></div>
      ${logLines}
    `, "card--subtle")}
  `;
  document.querySelectorAll("[data-stop-live-app]").forEach((button) => {
    button.addEventListener("click", async () => {
      await stopLiveApp(button.dataset.stopLiveApp);
    });
  });
  byId("developer-role-toggle")?.addEventListener("click", async () => {
    await setDeveloperRole(state.developerAccess?.can_use_testing ? "user" : "developer");
  });
  byId("display-settings-button")?.addEventListener("click", () => {
    state.displayPanelOpen = true;
    renderAll();
  });
}

function renderDisplayOverlay() {
  const host = byId("display-overlay");
  if (!host) return;
  host.className = `welcome-overlay ${state.displayPanelOpen ? "visible" : ""}`;
  host.innerHTML = state.displayPanelOpen ? `
    <div class="welcome-dialog display-dialog">
      <div class="section-intro">
        <div>
          <div class="eyebrow">${escapeHTML(t("settings.display"))}</div>
          <h2>${escapeHTML(t("settings.displayTitle"))}</h2>
        </div>
        <p>${escapeHTML(t("settings.displayLead"))}</p>
      </div>
      <div class="section-gap"><strong>${escapeHTML(t("settings.displayScale"))}</strong></div>
      <div class="display-control-row">
        <input id="display-scale-range" type="range" min="${CHAT_SCALE_LIMITS.min}" max="${CHAT_SCALE_LIMITS.max}" step="${CHAT_SCALE_LIMITS.step}" value="${escapeHTML(String(state.chatScalePercent))}" />
        <input id="display-scale-input" class="text-input compact" type="number" min="${CHAT_SCALE_LIMITS.min}" max="${CHAT_SCALE_LIMITS.max}" step="${CHAT_SCALE_LIMITS.step}" value="${escapeHTML(String(state.chatScalePercent))}" />
        <span class="pill">${escapeHTML(`${state.chatScalePercent}%`)}</span>
      </div>
      <div class="button-row">
        <button id="display-scale-reset" class="secondary-button">${escapeHTML(t("settings.displayReset"))}</button>
        <button id="display-overlay-close" class="primary-button">${escapeHTML(t("chat.drawerClose"))}</button>
      </div>
    </div>
  ` : "";
  host.onclick = () => {
    state.displayPanelOpen = false;
    renderAll();
  };
  host.querySelector(".display-dialog")?.addEventListener("click", (event) => event.stopPropagation());
  byId("display-overlay-close")?.addEventListener("click", () => {
    state.displayPanelOpen = false;
    renderAll();
  });
  byId("display-scale-range")?.addEventListener("input", async (event) => {
    await setChatScalePercent(event.target.value);
  });
  byId("display-scale-input")?.addEventListener("change", async (event) => {
    await setChatScalePercent(event.target.value);
  });
  byId("display-scale-reset")?.addEventListener("click", async () => {
    await setChatScalePercent(100);
  });
}

function renderDemoOverlay() {
  const host = byId("demo-overlay");
  if (!host) return;
  host.className = `welcome-overlay ${state.demoPanelOpen ? "visible" : ""}`;
  if (!state.demoPanelOpen) {
    host.innerHTML = "";
    return;
  }
  const runtime = {
    status: state.demoPanelStatus,
    reason: state.demoPanelReason,
    url: state.demoPanelURL,
    liveAppID: state.demoPanelAppID,
  };
  const actionState = currentLiveAction(state.demoPanelAppID, state.demoPanelOrigin);
  const canRestart = !!state.demoPanelOrigin;
  const canStop = !!state.demoPanelAppID && ["ready", "starting"].includes(runtime.status || "");
  host.innerHTML = `
    <div class="welcome-dialog demo-overlay-dialog">
      <div class="section-intro">
        <div>
          <div class="eyebrow">${escapeHTML(t("live.windowTitle"))}</div>
          <h2>${escapeHTML(state.demoPanelTitle || t("live.windowTitle"))}</h2>
        </div>
        <p>${escapeHTML(t("live.windowLead"))}</p>
      </div>
      <div class="button-row">
        ${canRestart ? `<button id="demo-overlay-restart" class="secondary-button">${escapeHTML(t("live.restart"))}</button>` : ""}
        ${canStop ? `<button id="demo-overlay-stop" class="secondary-button">${escapeHTML(t("live.stop"))}</button>` : ""}
        <button id="demo-overlay-close" class="primary-button">${escapeHTML(t("chat.drawerClose"))}</button>
      </div>
      <div class="demo-overlay-body">
        ${(runtime.status === "ready" || runtime.status === "starting") && runtime.url
          ? `${actionState ? renderLaunchState(runtime, state.demoPanelKind, actionState) : ""}<iframe class="demo-overlay-frame" src="${escapeHTML(runtime.url)}" title="${escapeHTML(state.demoPanelTitle || t("live.windowTitle"))}"></iframe>`
          : renderLaunchState(runtime, state.demoPanelKind, actionState) || `<div class="empty">${escapeHTML(t("live.windowEmpty"))}</div>`}
      </div>
    </div>
  `;
  host.onclick = () => {
    closeDemoPanel();
    renderAll();
  };
  host.querySelector(".demo-overlay-dialog")?.addEventListener("click", (event) => event.stopPropagation());
  byId("demo-overlay-close")?.addEventListener("click", () => {
    closeDemoPanel();
    renderAll();
  });
  byId("demo-overlay-restart")?.addEventListener("click", async () => {
    await launchMaterial(state.demoPanelOrigin);
  });
  byId("demo-overlay-stop")?.addEventListener("click", async () => {
    await stopLiveApp(state.demoPanelAppID);
  });
}

function bindAgentCards() {
  document.querySelectorAll("[data-agent]").forEach((button) => {
    button.addEventListener("click", async () => {
      state.selectedAgent = button.dataset.agent;
      await refreshAllowedActions();
      renderChat();
    });
  });
}

async function chooseProject() {
  try {
    const selected = await transport.chooseProject(state.draftPath || state.path || ".");
    if (!selected) return;
    state.draftPath = selected;
    await openProject(selected);
  } catch (error) {
    showBanner(error.message || String(error), "error");
  }
}

async function openProject(pathValue) {
  state.path = String(pathValue || "").trim();
  state.draftPath = state.path;
  window.localStorage.setItem("arc.desktop.native.path", state.path);
  showBanner(t("project.loading"));
  const projectState = await withBusy(t("project.loading"), () => transport.projectState(state.path));
  state.projectState = projectState;
  state.workspace = projectState.workspace || null;
  updateProjectChrome();
  if (projectState.state === "ready") {
    state.showWelcome = false;
    state.recentProjects = rememberRecentProject(state.recentProjects, state.path);
    window.localStorage.setItem(RECENT_PROJECTS_KEY, JSON.stringify(state.recentProjects));
    state.selectedSessionId = "";
    state.sessionDetail = null;
    state.selectedMaterialId = "";
    state.selectedFilePath = "";
    state.selectedFileDetail = null;
    state.currentSessionTab = "ready";
    await loadProjectData();
    showBanner(t("project.opened"));
  } else {
    state.sessions = [];
    state.sessionDetail = null;
    state.selectedSessionId = "";
    state.allowedActions = null;
    state.projectMaterials = [];
    renderAll();
  }
}

async function initWorkspace() {
  const summary = await withBusy("Настраиваю ARC", () => transport.initWorkspace({ path: state.path, provider: "codex", mode: state.selectedAgent || "work" }), t("project.initialized"));
  state.workspace = summary;
  state.projectState = {
    path: summary.root,
    name: summary.name,
    state: "ready",
    message: t("project.ready"),
    workspace: summary,
  };
  state.showWelcome = false;
  state.recentProjects = rememberRecentProject(state.recentProjects, summary.root);
  window.localStorage.setItem(RECENT_PROJECTS_KEY, JSON.stringify(state.recentProjects));
  state.selectedSessionId = "";
  state.sessionDetail = null;
  state.selectedMaterialId = "";
  state.selectedFilePath = "";
  state.selectedFileDetail = null;
  state.currentSessionTab = "ready";
  await loadProjectData();
}

async function loadProjectData() {
  if (!isReady()) return;
  const [providers, agents, sessions, liveApps, projectMaterials, workspaceExplorer] = await Promise.all([
    transport.providers(),
    transport.agents(),
    refreshSessions(false),
    transport.liveApps(state.path, ""),
    transport.projectMaterials(state.path),
    transport.workspaceFiles(state.path, 400),
  ]);
  state.providers = providers;
  state.agents = agents;
  state.liveApps = liveApps || [];
  state.projectMaterials = projectMaterials || [];
  state.workspaceExplorer = workspaceExplorer || null;
  state.selectedAgent = projectDefaultMode();
  await refreshDeveloperAccess();
  if (state.selectedSessionId && sessions && sessions.some((item) => item.id === state.selectedSessionId)) {
    await loadSessionDetail(state.selectedSessionId);
  }
  await refreshAllowedActions();
  renderAll();
}

async function refreshSessions(renderAfter = true) {
  if (!isReady()) {
    state.sessions = [];
    if (renderAfter) renderAll();
    return [];
  }
  const sessions = await transport.sessions(state.path, state.sessionSearch, state.sessionModeFilter, state.sessionStatusFilter);
  state.sessions = sessions || [];
  if (state.selectedSessionId && !state.sessions.some((item) => item.id === state.selectedSessionId)) {
    state.selectedSessionId = "";
    state.sessionDetail = null;
  }
  if (renderAfter) renderAll();
  return state.sessions;
}

async function loadSessionDetail(sessionID) {
  if (!sessionID || !isReady()) return;
  const switchingSession = state.selectedSessionId !== sessionID;
  state.selectedSessionId = sessionID;
  if (switchingSession) {
    state.selectedMaterialId = "";
    state.selectedFilePath = "";
    state.selectedFileDetail = null;
  }
  state.sessionDetail = await transport.session(state.path, sessionID);
  state.selectedAgent = state.sessionDetail?.session?.mode || projectDefaultMode();
  state.liveApps = state.sessionDetail?.live_apps || state.sessionDetail?.liveApps || [];
  state.allowedActions = state.sessionDetail?.allowed_actions || state.sessionDetail?.allowedActions || null;
  if (!state.selectedMaterialId && state.sessionDetail.materials?.length) {
    state.selectedMaterialId = currentDocumentMaterial(state.sessionDetail)?.id || currentVisualMaterial(state.sessionDetail)?.id || state.sessionDetail.materials[0].id;
  }
  scheduleSessionPoll();
}

async function refreshLiveApps(sessionID = "") {
  if (!isReady()) {
    state.liveApps = [];
    return [];
  }
  const apps = await transport.liveApps(state.path, sessionID);
  state.liveApps = apps || [];
  return state.liveApps;
}

async function refreshWorkspaceExplorer(renderAfter = true) {
  if (!isReady()) {
    state.workspaceExplorer = null;
    if (renderAfter) renderAll();
    return null;
  }
  state.workspaceExplorer = await transport.workspaceFiles(state.path, 400);
  if (state.selectedFilePath) {
    const allPaths = new Set([
      ...((state.workspaceExplorer?.files || []).map((item) => item.path)),
      ...((state.projectMaterials || []).map((item) => item.path)),
    ]);
    if (!allPaths.has(state.selectedFilePath)) {
      state.selectedFilePath = "";
      state.selectedFileDetail = null;
    }
  }
  if (renderAfter) renderAll();
  return state.workspaceExplorer;
}

async function uploadProjectMaterials(fileList) {
  if (!isReady()) return;
  const files = await encodeUploadedFiles(fileList);
  if (!files.length) return;
  await withBusy(t("chat.uploadBusy"), async () => {
    await transport.uploadProjectMaterials({
      path: state.path,
      session_id: state.selectedSessionId || "",
      files,
    });
    state.projectMaterials = await transport.projectMaterials(state.path);
    await refreshWorkspaceExplorer(false);
    if (state.selectedSessionId) {
      await loadSessionDetail(state.selectedSessionId);
    }
    renderAll();
  }, t("chat.uploaded"));
}

async function deleteProjectMaterial(materialID) {
  if (!isReady() || !materialID) return;
  const material = (state.projectMaterials || []).find((item) => item.id === materialID) || null;
  await withBusy(state.locale === "en" ? "Removing ARC file" : "Удаляю файл ARC", async () => {
    state.projectMaterials = await transport.deleteProjectMaterial({
      path: state.path,
      material_id: materialID,
    });
    if (material?.path && state.selectedFilePath === material.path) {
      state.selectedFilePath = "";
      state.selectedFileDetail = null;
    }
    await refreshWorkspaceExplorer(false);
    renderAll();
  }, state.locale === "en" ? "ARC file removed" : "Файл ARC удалён");
}

async function launchMaterial(materialID) {
  if (!state.selectedSessionId) return;
  setLiveActionState("", materialID, {
    phase: "starting",
    message: state.locale === "en" ? "ARC is launching this saved miniapp in the current chat." : "ARC запускает сохранённый миниапп в текущем чате.",
  });
  renderAll();
  try {
    await withBusy(t("live.starting"), async () => {
      await transport.startMaterialLiveApp({
        path: state.path,
        session_id: state.selectedSessionId,
        material_id: materialID,
      });
      await loadSessionDetail(state.selectedSessionId);
      await refreshLiveApps(state.selectedSessionId);
      setLiveActionState("", materialID, null);
      renderAll();
    }, t("live.ready"));
  } catch (error) {
    setLiveActionState("", materialID, {
      phase: "failed",
      message: error?.message || String(error),
    });
    renderAll();
    throw error;
  }
}

async function openLiveApp(appID, materialID = "") {
  if (!isReady()) return;
  setLiveActionState(appID, materialID, {
    phase: "opening",
    message: state.locale === "en" ? "ARC is preparing a fresh preview inside the app." : "ARC готовит свежий preview прямо внутри приложения.",
  });
  renderAll();
  try {
    await withBusy(t("live.starting"), async () => {
      const detail = await resolveLivePreview(appID, materialID);
      state.demoPanelOpen = true;
      state.demoPanelTitle = detail?.title || state.demoPanelTitle || t("live.windowTitle");
      state.demoPanelURL = detail?.preview_url || detail?.previewURL || "";
      state.demoPanelAppID = detail?.id || appID || "";
      state.demoPanelOrigin = materialID || detail?.origin || state.demoPanelOrigin || "";
      state.demoPanelKind = detail?.type || state.demoPanelKind || "demo";
      state.demoPanelStatus = detail?.status || "ready";
      state.demoPanelReason = detail?.stop_reason || detail?.stopReason || "";
      setLiveActionState(appID, materialID, null);
      renderAll();
    });
  } catch (error) {
    setLiveActionState(appID, materialID, {
      phase: "failed",
      message: error?.message || String(error),
    });
    renderAll();
    throw error;
  }
}

async function openLiveAppInBrowser(appID, materialID = "") {
  if (!isReady()) return;
  setLiveActionState(appID, materialID, {
    phase: "opening",
    message: state.locale === "en" ? "ARC is preparing a fresh preview before opening the browser." : "ARC готовит свежий preview перед открытием браузера.",
  });
  renderAll();
  try {
    await withBusy(t("live.starting"), async () => {
      const detail = await resolveLivePreview(appID, materialID);
      const target = detail?.preview_url || detail?.previewURL || "";
      if (!target) {
        throw new Error(state.locale === "en" ? "Miniapp preview is unavailable." : "Миниапп сейчас недоступен.");
      }
      state.demoPanelTitle = detail?.title || state.demoPanelTitle || t("live.windowTitle");
      state.demoPanelURL = target;
      state.demoPanelAppID = detail?.id || appID || "";
      state.demoPanelOrigin = materialID || detail?.origin || state.demoPanelOrigin || "";
      state.demoPanelKind = detail?.type || state.demoPanelKind || "demo";
      state.demoPanelStatus = detail?.status || "ready";
      state.demoPanelReason = detail?.stop_reason || detail?.stopReason || "";
      await openExternalURL(target);
      setLiveActionState(appID, materialID, null);
      renderAll();
    });
  } catch (error) {
    setLiveActionState(appID, materialID, {
      phase: "failed",
      message: error?.message || String(error),
    });
    renderAll();
    throw error;
  }
}

async function launchLessonDemo(lessonID) {
  await withBusy(t("live.starting"), async () => {
    await transport.startLessonDemo({
      path: state.path,
      session_id: state.selectedSessionId || "",
      lesson_id: lessonID,
    });
    await refreshLiveApps(state.selectedSessionId || "");
    if (state.selectedSessionId) {
      await loadSessionDetail(state.selectedSessionId);
    }
    renderAll();
  }, t("live.ready"));
}

async function stopLiveApp(appID) {
  setLiveActionState(appID, "", {
    phase: "stopping",
    message: state.locale === "en" ? "ARC is stopping the current preview instance." : "ARC останавливает текущий preview-экземпляр.",
  });
  renderAll();
  try {
    await withBusy(t("live.stopping"), async () => {
      await transport.stopLiveApp({ path: state.path, app_id: appID });
      await refreshLiveApps(state.selectedSessionId || "");
      if (state.selectedSessionId) {
        await loadSessionDetail(state.selectedSessionId);
      }
      setLiveActionState(appID, "", null);
      renderAll();
    }, t("live.stopped"));
  } catch (error) {
    setLiveActionState(appID, "", {
      phase: "failed",
      message: error?.message || String(error),
    });
    renderAll();
    throw error;
  }
}

async function setChatScalePercent(value) {
  const next = normalizeChatScalePercent(value);
  state.chatScalePercent = next;
  applyChatScaleState();
  renderAll();
  try {
    state.chatScalePercent = normalizeChatScalePercent(await transport.setChatScalePercent(next));
  } catch (error) {
    pushClientLog("warn", `chat scale unavailable: ${error.message || String(error)}`);
  }
  applyChatScaleState();
  renderAll();
}

async function refreshAllowedActions() {
  if (!isReady()) {
    state.allowedActions = null;
    return null;
  }
  const mode = state.sessionDetail?.session?.mode || state.selectedAgent || projectDefaultMode();
  state.allowedActions = state.sessionDetail?.allowed_actions || state.sessionDetail?.allowedActions || await transport.allowedActions(state.path, mode, state.selectedSessionId || "");
  return state.allowedActions;
}

async function refreshDeveloperAccess() {
  state.developerAccess = await transport.developerAccess(state.path || "");
  if (state.developerAccess?.can_use_testing) {
    state.testingScenarios = await transport.testingScenarios(state.path || "");
    state.verificationProfiles = await transport.verificationProfiles(state.path || "");
  } else {
    state.testingScenarios = [];
    state.testingRun = null;
    state.verificationProfiles = [];
    state.verificationRun = null;
  }
}

async function setDeveloperRole(role) {
  await withBusy(t("project.loading"), async () => {
    state.developerAccess = await transport.setDeveloperRole({ path: state.path, role });
    await refreshDeveloperAccess();
    if (!state.developerAccess?.can_use_testing && state.currentScreen === "testing") {
      state.currentScreen = "chat";
    }
    renderAll();
  }, role === "developer" ? t("settings.enableDevMode") : t("settings.disableDevMode"));
}

async function startTesting(scenarioID, stepMode) {
  if (!state.developerAccess?.can_use_testing) return;
  const run = await withBusy(stepMode ? "Запускаю пошаговый тест" : "Запускаю автономный тест", () => transport.startTesting({
    path: state.path,
    scenario: scenarioID,
    agent_id: state.selectedAgent || projectDefaultMode(),
    step_mode: stepMode,
  }));
  state.testingRun = run;
  state.currentScreen = "testing";
  if (!stepMode) {
    await controlTesting("quit", false);
    renderAll();
    return;
  }
  state.currentScreen = "testing";
  renderAll();
}

async function controlTesting(action, rerender = true) {
  if (!state.testingRun?.id) return;
  const run = await withBusy("Выполняю шаг тестирования", () => transport.testingControl({
    path: state.path,
    run_id: state.testingRun.id,
    action,
  }));
  state.testingRun = run;
  scheduleTestingPoll();
  if (rerender) {
    renderAll();
  }
}

async function startVerification(profileID) {
  if (!state.developerAccess?.can_use_testing) return;
  const run = await withBusy("Запускаю verification baseline", () => transport.startVerification({
    path: state.path,
    profile_id: profileID,
  }));
  state.verificationRun = run;
  state.currentScreen = "testing";
  renderAll();
}

function scheduleTestingPoll() {
  if (testingPollTimer) {
    window.clearTimeout(testingPollTimer);
    testingPollTimer = null;
  }
  if (!state.testingRun?.id) return;
  if (!["running"].includes(state.testingRun.status)) return;
  testingPollTimer = window.setTimeout(async () => {
    state.testingRun = await transport.testingStatus(state.path, state.testingRun.id);
    renderAll();
    scheduleTestingPoll();
  }, 1200);
}

async function setProjectAgent(mode) {
  if (!isReady() || !mode) return;
  await withBusy(t("project.loading"), async () => {
    const summary = await transport.setWorkspaceMode({ path: state.path, mode });
    state.workspace = summary;
    if (state.projectState) {
      state.projectState = {
        ...state.projectState,
        workspace: summary,
        message: t("project.ready"),
      };
    }
    if (!state.selectedSessionId) {
      state.selectedAgent = mode;
    }
    await refreshAllowedActions();
    renderAll();
  }, t("chat.projectAgentUpdated"));
}

function scheduleSessionPoll() {
  if (sessionPollTimer) {
    window.clearTimeout(sessionPollTimer);
    sessionPollTimer = null;
  }
  if (!state.selectedSessionId || !state.sessionDetail) return;
  if (state.sessionDetail.session.status !== "running") return;
  sessionPollTimer = window.setTimeout(async () => {
    const before = chatDetailSignature(state.sessionDetail);
    await loadSessionDetail(state.selectedSessionId);
    if (before !== chatDetailSignature(state.sessionDetail)) {
      renderChatLiveRefresh();
    }
  }, 1800);
}

async function submitComposer() {
  if (!isReady()) {
    showBanner(t("project.missing"), "error");
    return;
  }
  const prompt = state.composerText.trim();
  if (!prompt) return;
  const provider = currentProvider();
  const mode = state.selectedAgent || "work";
  const providerState = providerHealth(provider);
  if (providerState && providerState.installed === false) {
    showBanner(providerState.notes?.[0] || t("chat.providerMissing"), "error");
    return;
  }
  await refreshAllowedActions();
  state.composerSending = true;
  state.composerStatus = t("chat.sendingStatus");
  renderAll();
  try {
    let detail = null;
    if (state.selectedSessionId) {
      detail = await transport.chatSend({
        path: state.path,
        session_id: state.selectedSessionId,
        model: "",
        prompt,
        dry_run: false,
        async: true,
        action: "",
        allow_autonomy: false,
        attach_session_ids: state.attachedSessionIDs,
      });
    } else {
      detail = await transport.chatStart({
        path: state.path,
        provider,
        mode,
        model: "",
        prompt,
        dry_run: false,
        async: true,
        action: "",
        allow_autonomy: false,
        attach_session_ids: state.attachedSessionIDs,
      });
    }
    if (detail?.session?.id) {
      state.selectedSessionId = detail.session.id;
      state.sessionDetail = detail;
      state.allowedActions = detail.allowed_actions || detail.allowedActions || state.allowedActions;
      state.liveApps = detail.live_apps || detail.liveApps || state.liveApps;
    }
    state.composerText = "";
    renderAll();
    await refreshSessions(false);
    if (state.selectedSessionId) {
      await loadSessionDetail(state.selectedSessionId);
    }
    await refreshLiveApps(state.selectedSessionId || "");
  } catch (error) {
    pushClientLog("error", error?.message || String(error));
    showBanner(error?.message || String(error), "error");
  } finally {
    state.composerSending = false;
    state.composerStatus = "";
    renderAll();
  }
}

function renderAll() {
  renderStaticCopy();
  updateProjectChrome();
  renderNavState();
  renderWelcomeOverlay();
  syncDemoPanelFromSession();
  renderDemoOverlay();
  renderDisplayOverlay();
  renderChat();
  renderTesting();
  renderSettings();
  document.querySelectorAll(".screen").forEach((screen) => {
    screen.classList.toggle("active", screen.id === `screen-${state.currentScreen}`);
  });
  bindAgentMenus();
}

function bindAgentMenus() {
  document.querySelectorAll("[data-toggle-agent-menu]").forEach((button) => {
    button.addEventListener("click", () => {
      const scope = button.dataset.toggleAgentMenu;
      if (scope === "project") {
        state.projectAgentMenuOpen = !state.projectAgentMenuOpen;
      }
      renderAll();
    });
  });
  document.querySelectorAll("[data-select-agent]").forEach((button) => {
    button.addEventListener("click", async () => {
      const [scope, mode] = String(button.dataset.selectAgent || "").split(":");
      state.projectAgentMenuOpen = false;
      if (!mode) return;
      if (scope === "project") {
        await setProjectAgent(mode);
      }
    });
  });
}

function bindShellEvents() {
  document.querySelectorAll(".nav-item, .topbar-nav-item").forEach((button) => {
    button.addEventListener("click", () => {
      if (button.disabled) return;
      state.currentScreen = button.dataset.screen;
      renderAll();
    });
  });
  byId("show-projects-button")?.addEventListener("click", () => {
    state.showWelcome = true;
    renderWelcomeOverlay();
  });
  byId("reload-button")?.addEventListener("click", async () => {
    if (!state.path) return;
    await openProject(state.path);
    showBanner(t("project.refreshed"));
  });
  byId("locale-select")?.addEventListener("change", (event) => {
    state.locale = normalizeLocale(event.target.value);
    window.localStorage.setItem("arc.desktop.native.locale", state.locale);
    renderAll();
  });
}

function bindNativeMenuEvents(root = window) {
  const runtime = root?.runtime;
  if (!runtime || typeof runtime.EventsOn !== "function") {
    return () => {};
  }
  const unsubscribeProjects = runtime.EventsOn("arc:open-projects", () => {
    state.showWelcome = true;
    renderAll();
  });
  const unsubscribeSettings = runtime.EventsOn("arc:open-settings", () => {
    state.currentScreen = "settings";
    renderAll();
  });
  const unsubscribeLocale = runtime.EventsOn("arc:set-locale", (...data) => {
    state.locale = normalizeLocale(data[0] || state.locale);
    window.localStorage.setItem("arc.desktop.native.locale", state.locale);
    renderAll();
  });
  return () => {
    if (typeof unsubscribeProjects === "function") unsubscribeProjects();
    if (typeof unsubscribeSettings === "function") unsubscribeSettings();
    if (typeof unsubscribeLocale === "function") unsubscribeLocale();
  };
}

async function bootstrap() {
  try {
    bindNativeMenuEvents(window);
    bindNativeChatScaleEvents(window, (next) => {
      state.chatScalePercent = next;
      persistChatScale();
      renderAll();
    }, () => {
      state.displayPanelOpen = true;
      renderAll();
    });
    applyChatScaleState();
    renderAll();
    bindShellEvents();
    try {
      state.chatScalePercent = normalizeChatScalePercent(await transport.chatScalePercent());
      applyChatScaleState();
    } catch (error) {
      pushClientLog("warn", `chat scale unavailable: ${error.message || String(error)}`);
    }
    state.desktopLogPath = await transport.desktopLogPath();
    const projectState = await transport.projectState(state.path || "");
    state.projectState = projectState;
    state.workspace = projectState.workspace || null;
    await refreshDeveloperAccess();
    state.providers = await transport.providers();
    state.agents = await transport.agents();
    if (isReady()) {
      state.showWelcome = false;
      state.recentProjects = rememberRecentProject(state.recentProjects, state.path);
      window.localStorage.setItem(RECENT_PROJECTS_KEY, JSON.stringify(state.recentProjects));
      state.selectedSessionId = "";
      state.sessionDetail = null;
      await loadProjectData();
    }
    renderAll();
    pushClientLog("info", "frontend bootstrap started");
  } catch (error) {
    showBanner(error.message || String(error), "error");
    pushClientLog("error", error.message || String(error));
  }
}

bootstrap();
