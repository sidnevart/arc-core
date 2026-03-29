<p align="center">
  <img src="./assets/arc-logo.svg" alt="ARC logo" width="220" />
</p>

# ARC

Проектная консоль для локальной работы с AI над реальным кодом.

Русский: этот файл  
English: [README.md](./README.md)

## Что такое ARC

ARC помогает работать с AI над проектом так, чтобы не терять контроль:

- начинать не с пустого чата, а с состояния проекта
- хранить в одном месте задачи, чаты, память, карты и артефакты
- переключаться между CLI и desktop без потери контекста
- смотреть, что изменилось, до того как доверять результату

ARC local-first. Проект остаётся на диске. CLI — это основной runtime. Desktop — более простой визуальный слой сверху.

## Что лежит в репозитории

- [cmd/arc](./cmd/arc) — основная CLI-утилита
- [cmd/arc-desktop](./cmd/arc-desktop) — локальный browser preview
- [cmd/arc-desktop-wails](./cmd/arc-desktop-wails) — native desktop launcher
- [apps/desktop](./apps/desktop) — продуктовая зона desktop
- [apps/docs](./apps/docs) — сайт документации
- [presets/official](./presets/official) — встроенные наборы
- [content/editorial](./content/editorial) — посты, статьи и кампании

## Быстрый старт

### 1. Собери CLI

```bash
make build
```

### 2. Проверь окружение

```bash
./bin/arc doctor
```

### 3. Инициализируй ARC в проекте

```bash
./bin/arc init --path .
```

### 4. Сделай первый план

```bash
./bin/arc task plan --path . --mode work --provider codex "Посмотри проект и предложи следующее безопасное изменение"
```

### 5. Собери native desktop app

```bash
go build -tags wails ./cmd/arc-desktop-wails
```

Канонический способ собрать `.app` bundle:

```bash
make desktop-wails-package
```

Если хочешь отдельно проверить Wails CLI path:

```bash
cd cmd/arc-desktop-wails
GOCACHE=/tmp/agent-os-gocache CGO_ENABLED=1 GOOS=darwin GOARCH=$(uname -m) wails build -platform darwin/$(uname -m) -tags wails -clean -nopackage
```

### 6. Запускай desktop preview только когда он реально нужен

```bash
make build-desktop
make desktop-preview
```

Browser preview теперь считается временным debug-слоем. Основной путь для проверки продукта — native desktop app.

### 7. Прогони desktop-тесты

```bash
make test-desktop
```

## Как пользоваться

### CLI

CLI удобен, если ты любишь терминал и хочешь полный контроль:

- `arc doctor`
- `arc init --path .`
- `arc mode set work --path .`
- `arc task plan --path . ...`
- `arc task run --path . ...`
- `arc run list --path .`
- `arc chat start ...`
- `arc preset list`

Пошаговый гайд: [apps/docs/docs/cli-guide.md](./apps/docs/docs/cli-guide.md)

### Desktop

Desktop удобен, если нужен простой визуальный поток работы:

- `Welcome / Open Project` — вход в проект через recent/open chooser
- `Чат` — главный рабочий экран
- `Study / Work / Hero` — встроенные агенты-пресеты под разные стили работы
- `Сессии` — история, поиск и продолжение прошлых разговоров
- материалы, демо и проверки открываются внутри сессии, а не в отдельном task manager
- простые локальные демо и мини-сайты ARC поднимает сам как managed local runtime и показывает прямо внутри приложения

Пошаговый гайд: [apps/docs/docs/desktop-guide.md](./apps/docs/docs/desktop-guide.md)

Текущий рабочий режим:

- основной способ тестирования — native `.app`
- browser preview запускается только как временный debug fallback
- docs локально поднимаются только когда ты реально проверяешь документацию

## Документация

Главные страницы:

- [Обзор](./apps/docs/docs/intro.md)
- [Установка](./apps/docs/docs/install.md)
- [Быстрый старт](./apps/docs/docs/quickstart.md)
- [CLI для новичков](./apps/docs/docs/cli-guide.md)
- [Desktop для новичков](./apps/docs/docs/desktop-guide.md)
- [Первый сценарий от начала до конца](./apps/docs/docs/first-task.md)
- [Команды](./apps/docs/docs/commands.md)
- [FAQ](./apps/docs/docs/faq.md)
- [Скриншоты и подсказки](./apps/docs/docs/screenshots.md)

Локальный запуск docs только по необходимости:

```bash
make docs-install
make docs-dev
```

Сборка docs:

```bash
make docs-build
```

## Что уже готово

Сейчас в репозитории уже есть:

- рабочий CLI runtime
- локальные run, память, карты docs и артефакты
- desktop preview
- native Wails desktop shell
- preview, install и rollback для наборов
- локальные chat sessions с transcript

Текущие ограничения:

- live `codex` зависит от стабильности локального бинаря `codex`
- verify сейчас сильнее всего работает в Go-проектах
- desktop уже usable, но UX ещё упрощается под обычного пользователя

## Для контрибьюторов

Если меняется поведение продукта, обновляй durable context в [memory_bank](./memory_bank).

Branding contract: исходное изображение для UI и иконки desktop должно жить в [assets/branding](./assets/branding).
