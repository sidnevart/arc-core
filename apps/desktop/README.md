# ARC Desktop

`apps/desktop` — продуктовая зона desktop-интерфейсов ARC.

Сейчас в ней разделены два слоя:

- `static/` — browser-based preview shell для быстрого локального цикла разработки
- `wailsapp/` — канонический native Wails desktop package, который использует тот же `internal/desktop` bridge напрямую

Текущий operating policy:

- основная проверка продукта идёт через native `.app`
- `static/` поднимается только временно, когда нужен debug fallback
- preview не должен жить пачкой на нескольких портах

## Что является каноническим

- Канонический runtime и orchestration layer остаются в `cmd/arc` и `internal/*`
- Канонический desktop contract живёт в `internal/desktop/bridge.go`
- Канонический native desktop package живёт в `apps/desktop/wailsapp`
- Preview и native shell обязаны читать один и тот же semantic contract
- Базовый shared frontend слой теперь живёт в `apps/desktop/wailsapp/frontend/shared/` и может потребляться preview через `/shared/*`

## Что теперь лежит в `cmd/`

`cmd/arc-desktop-wails/` теперь только thin entrypoint для сборки native shell.
В нём больше не живёт сам frontend bundle и не живёт desktop logic.

То есть `apps/desktop/` теперь не просто логическая, а и фактическая продуктовая зона desktop UX.

Следующий ожидаемый шаг — дотащить общий слой дальше: после выноса shared helpers/messages foundation убрать оставшееся дублирование state/render/copy между preview и Wails UI.

## Основные UX поверхности

- `Welcome / Open Project` — вход в проект по модели recent/open chooser
- `Chat` — главный рабочий экран с topic rail, широким thread и drawer по требованию
- `Settings` — состояние приложения, агентов и логов
- `Testing` — dev-only сценарии и verification profiles

## Built-in agents

`Study / Work / Hero` теперь подаются не как абстрактные режимы, а как встроенные first-party agents:

- `Study` — объяснения, схемы, мини-симуляции и короткая проверка понимания
- `Work` — совместная работа, где агент показывает ход и промежуточные материалы
- `Hero` — более автономное выполнение с акцентом на итоговый результат

Дальше marketplace должен расширять этот же picker installed agents, а не вводить второй параллельный способ выбора.

## Chat-first shell

Desktop теперь собран вокруг одной основной модели:

- слева всегда rail тем/разговоров вместо отдельного top-level `Sessions`
- по центру живёт широкий thread с fixed composer, inline live-status и inline visual outputs
- справа больше нет постоянной колонки; `Детали чата` открываются только в on-demand drawer с живой секцией `Сейчас запущено`, затем `Материалы чата` и `Файлы`
- единственный видимый chooser агента в beginner UX живёт в левой rail-секции `Агент проекта`, рядом с проектом и кнопкой `Новый чат`
- текущая тема больше не дублирует chooser агента в теле чата
- normal chat UX теперь вообще не показывает `Explain / Plan / Safe / Do`; агент должен понимать намерение из самого сообщения
- в левой колонке остались только поиск и список тем; отдельные mode/status filters убраны как лишний шум
- на экране `Chat` скрывается основной app sidebar, чтобы у пользователя оставалась одна рабочая боковая панель — проект, агент, новый чат и список тем
- настройки языка, display и переход в `Настройки` больше не должны занимать рабочий viewport чата; для native desktop они вынесены в application menu
- длинный page scroll на desktop запрещён; скролл допускается только внутри rail, thread и drawer
- assistant replies в thread теперь рендерятся как markdown, а не как plain escaped text
- диаграммы, демо и симуляции по умолчанию показываются прямо в thread как structured outputs, а документы и детали открываются в drawer
- если в обычном чате пользователь явно просит мини-симуляцию или демо, ARC materialize-ит output, пытается автоматически поднять live preview и показывает pending/failed lifecycle прямо в сообщении
- крупность чата теперь настраивается через отдельную `Display` surface: в native она открывается из меню `View -> Display...`, а quick actions в меню остаются только вспомогательными
- scale теперь задаётся в процентах и влияет на bubbles, composer, topbar, picker агента, drawer и список тем; legacy `compact/default/comfortable` больше не являются продуктовой моделью

## Managed local runtime

Desktop теперь умеет поднимать простые локальные demo/site/simulation preview как управляемые процессы без обязательного Docker:

- preview запускается локально на свободном порту
- ARC показывает его прямо внутри сессии и в `Settings`
- пользователь всегда может открыть миниапп в отдельном окне ARC, затем при желании увести его в браузер, либо остановить
- временные preview apps автоостанавливаются по policy

Это v1 runtime на managed local processes. Более тяжёлая изоляция остаётся следующим этапом.

## Контекстная модель

Desktop intentionally не editor-first.
Основные сущности:

- project
- built-in agent
- topic/session как одна backend-сущность с chat-first semantics
- materials
- runs as internal execution layer
- attachments

Файлы остаются частью продукта, но только как secondary inspect/detail surface из материалов сессии.

## Что уже важно для обычного пользователя

- основной путь начинается с `Welcome / Open Project`
- проект можно открыть через recent chooser или native кнопку `Выбрать проект`
- приложение запоминает последние открытые проекты
- в `Чате` можно нажать `Загрузить материалы`, чтобы положить свои файлы в `materials/uploads/...` и сразу сделать их доступными агенту
- новый разговор по умолчанию открывается пустым, а прошлые темы выбираются только слева
- базовый path теперь такой: `Открыть проект -> Чат -> написать цель -> Send`
- после отправки сообщения thread сразу показывает pending/live/failure lifecycle, а не «молчащий» чат
- если агент вернул схему, документ или интерактивный HTML output, ARC связывает его с сообщением как structured output и показывает inline card или live miniapp block вместо сырых артефактов
- source files для demo/simulation теперь считаются частью истории чата и сохраняются в `.arc/chats/.../turn-XXX-outputs/`; live preview process может auto-stop’иться, но миниапп остаётся restartable прямо из темы
- `Открыть в ARC` теперь открывает полноразмерный in-app demo surface; оттуда же можно уйти в браузер
- browser-open больше не зависит от старого `preview_url`: ARC сначала поднимает или оживляет live preview из сохранённого source artifact, а уже потом открывает внешний браузер
- `View -> Display...` теперь позволяет уменьшать chat workspace до `60%`, а topics rail и верхний chrome стали компактнее по умолчанию

## Как тестировать desktop

- `make test-desktop` — Go-тесты + frontend unit tests
- `go run ./cmd/arc workspace repair --path .` — дотянуть старый `.arc` scaffold до актуального состояния
- `go build -tags wails ./cmd/arc-desktop-wails` — проверить native launcher
- `make desktop-wails-package` — собрать `.app` через repo-level packaging script
- `make desktop-preview` — запускать только по требованию, а не держать постоянно в фоне
- в `Testing` есть профиль `chat-ui-minimalism`, который проверяет thread-first shell, один visible project agent picker, drawer по требованию и отсутствие starter clutter
- `chat-ui-minimalism` теперь также проверяет отсутствие duplicate agent chooser, отсутствие visible action modes в normal chat, markdown-first thread rendering, percentage display scale surface и наличие inline visual outputs в thread
- `chat-reliability` теперь проверяет, что normal chat path не теряет pending/failure states и что inline output actions materialized для demo/simulation lifecycle
- `runtime-demo-health` теперь стоит прогонять после любых правок в demo/simulation flow: он показывает, сломался ли relaunch из сохранённого artifact или остались только внешние provider warnings
- для ручного frontend-аудита добавлен project-local skill `.arc/skills/chat-frontend-auditor`
- для стабилизации desktop chat/runtime добавлен repo-local validator skill `.codex/skills/desktop-chat-validator`, который проверяет `Запустить снова`, sync между сообщением и `Детали чата`, и компактность рабочего chat shell

Если хочется проверить именно Wails CLI path отдельно:

- `cd cmd/arc-desktop-wails && GOCACHE=/tmp/agent-os-gocache CGO_ENABLED=1 GOOS=darwin GOARCH=$(uname -m) wails build -platform darwin/$(uname -m) -tags wails -clean -nopackage`
