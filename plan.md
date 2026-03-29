
# ТЗ: локальная среда для работы ИИ-агентов через CLI

## 1. Рабочее название

**Agent Runtime CLI**
Короткое имя в CLI: `arc`

---

## 2. Цель продукта

Создать **local-first CLI-среду**, которая оборачивает Claude и Codex в единый управляемый runtime и позволяет использовать их в трёх режимах:

1. **Учёба** — агент помогает учиться, а не решает за пользователя.
2. **Обдуманная работа** — агент усиливает пользователя как инженера и ментора, а не заменяет мышление.
3. **Тикет-герой** — агент максимально автономно закрывает задачи с контролируемыми guardrails.

Система должна быть:

* глобально установленной на компьютере,
* независимой от конкретного проекта,
* умеющей инициализировать проектные файлы и правила,
* расширяемой под Claude/Codex/MCP,
* ориентированной на **экономию токенов и качественную работу с контекстом**.

---

## 3. Проблема, которую решает продукт

Сейчас пользователь:

* слишком сильно делегирует ИИ мышление и может деградировать как инженер/ученик;
* тратит токены неуправляемо;
* каждый раз почти с нуля восстанавливает контекст;
* не имеет единой среды для повторяемой работы с агентами;
* не может жёстко переключать модель поведения агента под разные задачи;
* не имеет надёжного механизма “не галлюцинируй, покажи доказательства”.

---

## 4. Продуктовые принципы

### P1. Local-first

Всё управление идёт локально через CLI. UI не делается в v1.

### P2. Provider-agnostic core

Внутреннее ядро не должно зависеть от конкретного провайдера. Claude и Codex подключаются как адаптеры.

### P3. Human-authored durable context

Ключевые проектные правила и карты знаний **инициализируются системой, но наполняются человеком**, не агентом.

### P4. Mode-driven behavior

Поведение агента определяется режимом. Один и тот же запрос в разных режимах должен приводить к разному поведению.

### P5. Evidence-first

Любое важное утверждение агента должно опираться на источник: код, доку, команду, тест, человека или явно помеченную гипотезу.

### P6. Progressive disclosure

Контекст, правила, карты репозитория и доки подаются агенту порционно, а не целиком.

### P7. Reproducibility

Любой запуск должен оставлять артефакты: план, найденные неизвестные, diff, результаты проверок, решения, вопросы.

---

## 5. Границы v1

## Входит в v1

* единый CLI;
* глобальная установка;
* проектная инициализация;
* режимы `study`, `work`, `hero`;
* адаптеры для Claude и Codex;
* skills/subagents как основной механизм специализации;
* локальная память;
* индекс контекста;
* поиск через `rg`, `ast-grep`, git, file maps;
* planner / implementer / verifier / reviewer / docs-agent;
* policy layer и approval gates;
* token budgeting и compaction.

## Не входит в v1

* GUI;
* автоматическая настройка MCP “под ключ”;
* автозаполнение человеком-значимых файлов;
* полноценный multi-user/team server;
* автозакрытие Jira/PR без явного подтверждения;
* глобальная vector DB как обязательный компонент;
* замена нативных возможностей Claude/Codex своим агентным loop’ом.

---

## 6. Рекомендуемый стек реализации

**Рекомендация для v1: Go.**

Почему:

* один бинарник;
* простой запуск на macOS/Linux;
* удобно оркестрировать subprocess CLI-инструментов;
* удобно работать с файлами, SQLite, JSONL логами, watch’ерами;
* низкий operational overhead.

### Разрешённые внешние зависимости

* `git`
* `ripgrep`
* `ast-grep`
* `sqlite`
* `tree-sitter` или готовые CLI/биндинги по необходимости
* установленные локально `claude` и `codex`

---

## 7. Архитектура системы

## 7.1. Верхнеуровневая схема

`arc` состоит из 7 подсистем:

1. **CLI Layer**
2. **Orchestrator**
3. **Mode Engine**
4. **Provider Adapters**
5. **Context Engine**
6. **Memory Engine**
7. **Verification & Governance Layer**

---

## 7.2. CLI Layer

Отвечает за:

* команды пользователя;
* профили;
* запуск задач;
* просмотр статуса;
* просмотр артефактов;
* переключение режима;
* инициализацию проекта.

Примеры команд:

```bash
arc doctor
arc init
arc init --provider claude,codex
arc mode set study
arc mode set work
arc mode set hero

arc task plan "реализовать changelog для auditory"
arc task run "реализовать changelog для auditory"
arc task verify
arc task review
arc docs generate

arc learn "объясни B-tree insertion, но не решай за меня"
arc learn quiz
arc learn prove "почему здесь O(log n)"

arc index build
arc index refresh
arc memory compact
arc memory status
arc questions show
arc run status
```

---

## 7.3. Orchestrator

Ядро, которое:

* принимает задачу,
* определяет режим,
* выбирает провайдера,
* собирает контекст,
* запускает роли/агентов,
* сохраняет артефакты,
* выполняет переходы между шагами.

Оркестратор должен быть реализован как **детерминированная state machine**, а не “хаотичный цикл с промптами”.

### Обязательные состояния

* `initialized`
* `context_collecting`
* `planning`
* `needs_human_context`
* `implementing`
* `verifying`
* `reviewing`
* `documenting`
* `done`
* `failed`
* `blocked`

---

## 7.4. Provider Adapters

Система должна поддерживать минимум 2 адаптера:

### A. Claude Adapter

Использует локально установленный Claude runtime.

Учитывать, что Claude Code уже поддерживает skills, subagents, hooks, CLI flags и repo guidance через `CLAUDE.md`, а также имеет Agent SDK для Python/TypeScript. ([Claude API Docs][2])

### B. Codex Adapter

Использует локально установленный Codex runtime.

Учитывать, что Codex использует `AGENTS.md` для project guidance, поддерживает skills, subagents, sandbox/approval modes и конфиг-слои, а также может использоваться в multi-agent workflow через MCP/Agents SDK. ([OpenAI Developers][3])

### Требование к адаптерам

Каждый адаптер обязан реализовать интерфейс:

* `CheckInstalled()`
* `GetCapabilities()`
* `RunTask(task, contextPack, modePolicy)`
* `ResumeSession(sessionId)`
* `ApplyProjectScaffold(projectPath)`
* `EstimateRisk(task)`
* `CollectTranscript(runId)`

---

## 7.5. Context Engine

Отвечает за:

* поиск релевантного контекста;
* упаковку контекста;
* соблюдение токен-бюджета;
* progressive disclosure.

### Источники контекста

1. task brief
2. mode policy
3. project rules
4. repo map
5. docs map
6. memory bank
7. symbol index
8. git history
9. last run artifacts
10. explicit human notes

### Порядок поиска по умолчанию

1. `REPO_MAP.md`
2. `DOCS_MAP.md`
3. symbol index
4. `rg`
5. `ast-grep`
6. tests
7. git blame/log
8. memory bank
9. open questions
10. спросить человека

---

## 7.6. Memory Engine

Память должна быть **слоистой**, а не “один огромный memory_bank.md”.

### Слои памяти

* **Global memory** — личные долговременные правила пользователя
* **Project memory** — решения и контекст проекта
* **Run memory** — временные находки текущей сессии
* **Decision log** — принятые решения
* **Unknowns** — что неясно и требует уточнения
* **Archive** — устаревшее или вытесненное

### Каждая запись памяти обязана иметь поля

* `id`
* `scope` (`global|project|run`)
* `kind` (`decision|fact|hypothesis|question|constraint|todo`)
* `source`
* `confidence`
* `created_at`
* `last_verified_at`
* `status` (`active|stale|archived`)
* `tags`

### Политика памяти

* не хранить всё подряд;
* устаревшее автоматически переводить в `stale`;
* stale не подавать в основной контекст без явной причины;
* важные решения закреплять в `decision log`;
* гипотезы не путать с фактами.

---

## 8. Структура файлов

## 8.1. Глобальная структура

```text
~/.arc/
  config.yaml
  providers/
    claude.yaml
    codex.yaml
  modes/
    study.md
    work.md
    hero.md
  skills/
  templates/
  memory/
    GLOBAL_MEMORY.md
  cache/
  sessions/
  logs/
  evals/
```

## 8.2. Проектная структура

```text
<repo>/.arc/
  project.yaml
  mode.yaml
  provider/
    CLAUDE.md
    AGENTS.md
  maps/
    REPO_MAP.md
    DOCS_MAP.md
  memory/
    MEMORY_ACTIVE.md
    MEMORY_ARCHIVE.md
    DECISIONS.md
    OPEN_QUESTIONS.md
  plans/
  specs/
  runs/
  evals/
  index/
    symbols.json
    files.json
    dependencies.json
    recent_changes.json
```

---

## 9. Критично важное правило по файлам

### Инициализируются автоматически, но не заполняются агентом по умолчанию:

* `CLAUDE.md`
* `AGENTS.md`
* `REPO_MAP.md`
* `DOCS_MAP.md`
* `DECISIONS.md`
* `OPEN_QUESTIONS.md`

Система должна:

* создать шаблон;
* вставить секции и инструкции;
* пометить незаполненные места;
* предложить человеку открыть файл и заполнить вручную.

### Машинно-производные данные должны храниться отдельно:

* symbol index
* file graph
* dependency graph
* recent changes digest
* AST findings

То есть:
**человеческая карта репы ≠ машинный индекс**.

Это очень важно.

---

## 10. Режимы работы

# 10.1. Режим `study`

### Цель

Максимально улучшить обучение пользователя и минимизировать тупую делегацию мышления.

### Политика режима

Агент:

* не должен сразу давать готовое решение;
* обязан сначала выяснить текущий уровень понимания;
* обязан задавать встречные вопросы;
* обязан просить пользователя объяснить ход мысли;
* обязан спорить с плохими аргументами;
* обязан требовать доказательства, если пользователь утверждает что-то техническое;
* может давать подсказки лесенкой, а не “сразу всё”;
* может строить визуализации, песочницы, мини-демки и обучающие локальные инструменты;
* обязан фиксировать пробелы в знаниях.

### Лестница помощи

1. уточняющий вопрос
2. намёк
3. разбор одного шага
4. аналогичный пример
5. частичное решение (только если все плохо и не идет)
6. полное решение только по явному unlock пользователя ()

### Запрещено по умолчанию

* писать курсовую/ДЗ/проект полностью без попытки пользователя;
* решать задачу до того, как пользователь показал своё понимание;
* скрывать, почему решение верное.

### Обязательные артефакты

* `learning_goal.md`
* `current_understanding.md`
* `knowledge_gaps.md`
* `challenge_log.md`
* `practice_tasks.md`

### Подроли

* tutor
* challenger
* evaluator
* resource-scout
* visualizer

---

# 10.2. Режим `work`

### Цель

Повышать инженерную зрелость пользователя в реальном проекте, а не просто ускорять output.

### Политика режима

Агент:

* сначала помогает понять систему;
* предлагает, что именно читать в коде;
* строит flow задачи;
* просит пользователя сформулировать решение;
* валидирует и челленджит решение;
* подмечает слабые места;
* только потом помогает с реализацией каких-то подзадач (каждая подзадача может реализоваться только с разрешения пользователя, иначе же если разрешения нет, то агент не выполняет за пользователя);
* тупую рутину может выполнять сам.

### Обязательная последовательность

1. понять задачу
2. найти затронутые части системы
3. составить flow/sequence
4. сформулировать неизвестные
5. запросить предложение решения у пользователя
6. раскритиковать/улучшить
7. только затем кодить

### Примеры рутинных задач, которые можно делегировать

* генерация boilerplate тестов;
* обновление моков;
* приведение кода к стилю проекта;
* массовые rename/refactor без изменения логики;
* составление PR description;
* черновик release notes;
* log triage по шаблону;
* сборка changelog по коммитам;
* поиск затронутых файлов по фиче;
* draft документации и диаграмм;
* сборка списка мест для observability;
* подготовка чеклиста проверки;
* вытаскивание build/test команд из проекта;
* фиксация неизвестных и вопросов к команде.

### Обязательные артефакты

* `task_map.md`
* `system_flow.md`
* `solution_options.md`
* `unknowns.md`
* `validation_checklist.md`

### Подроли

* mapper
* mentor
* critic
* implementer-lite
* verifier
* docs-agent

---

# 10.3. Режим `hero`

### Цель

Максимально автономно закрывать задачи по тикетам с минимальным участием пользователя.

### Подроли

* planner
* spec-reviewer
* context-requester
* implementer
* verifier
* reviewer
* help-requester
* docs-agent
* orchestrator

### Пайплайн

1. intake задачи
2. planner строит бизнес+тех spec
3. spec-reviewer ищет дыры
4. context-requester добирает недостающее
5. implementer пишет код и тесты
6. verifier гоняет проверки
7. reviewer критикует решение
8. docs-agent обновляет доку и диаграммы
9. orchestrator решает: done / blocked / need human
10. refactoring-agent

### Правила hero-режима

* неизвестное не домысливать;
* сначала искать доказательства;
* если контекста не хватает — формировать bundle вопросов;
* любой risky action должен проходить approval gate;
* все изменения должны быть воспроизводимы;
* финальный пакет должен содержать spec, diff summary, test results, docs summary.

### Обязательные артефакты

* `ticket_spec.md`
* `business_spec.md`
* `tech_spec.md`
* `unknowns.md`
* `question_bundle.md`
* `implementation_log.md`
* `verification_report.md`
* `review_report.md`
* `docs_delta.md`

---

## 11. Anti-hallucination protocol

Каждый агент обязан маркировать утверждения как:

* `CODE_VERIFIED`
* `DOC_VERIFIED`
* `COMMAND_VERIFIED`
* `HUMAN_PROVIDED`
* `INFERRED`
* `UNKNOWN`

### Жёсткие правила

* `INFERRED` нельзя выдавать как факт;
* если нет доказательств — писать `UNKNOWN`;
* перед вопросом человеку агент обязан показать, что уже проверил;
* planner не имеет права описывать архитектурные детали, которые не нашёл в коде/доке/контексте;
* reviewer обязан отдельно проверять, не придумал ли implementer лишнего.

---

## 12. Token economy и продвинутая работа с контекстом

Это один из ключевых блоков продукта.

## 12.1. Принципы

* не лить всю репу в контекст;
* сначала дешёвый поиск, потом дорогой контекст;
* держать краткие maps + машинные индексы;
* сжимать run-memory после каждого крупного этапа;
* подавать правила и навыки по требованию.

Codex skills уже используют progressive disclosure: сначала грузится метаинформация о skill, а полная `SKILL.md` подгружается только когда skill реально нужен. Это надо взять как один из базовых принципов всей системы. ([OpenAI Developers][4])

## 12.2. Слои контекста

* L0: глобальные системные правила
* L1: mode policy
* L2: project guidance
* L3: активный task brief
* L4: repo/docs excerpts
* L5: run memory
* L6: raw findings

### Правило

Чем выше слой, тем стабильнее и короче он должен быть.

## 12.3. Бюджеты по умолчанию на один запуск

* mode policy: до 800 токенов
* project guidance: до 1200
* task brief: до 1500
* repo map excerpts: до 2500
* docs excerpts: до 2500
* active memory: до 1200
* unknowns + decisions: до 800

Если лимит превышен:

1. сначала сжать run-memory,
2. потом сократить excerpts,
3. потом выкинуть stale,
4. и только потом просить доп. контекст.

## 12.4. Compaction policy

После каждого этапа:

* сохранить сырой лог;
* построить короткий summary;
* оставить только факты, решения и открытые вопросы;
* переместить частные детали в archive.

Для Claude-совместимого memory-режима полезно держать очень компактную “верхушку” памяти, потому что в subagents flow есть явная опора на верхнюю часть `MEMORY.md`. ([Claude API Docs][5])

## 12.5. Свежесть информации

Каждый memory item обязан иметь дату последней валидации.

Статусы:

* `fresh`
* `warm`
* `stale`
* `archived`

Stale по умолчанию не попадает в основной prompt pack.

## 12.6. Индексация

Обязательно реализовать:

* файловый индекс
* symbol index
* dependency index
* git recent changes digest
* AST search adapters
* grep adapter
* docs references index

---

## 13. Skills и subagents

Здесь нужно сделать **канонический skill-слой** внутри `arc`, а под провайдеров уже делать адаптацию.

### Каноническая структура skill

```text
skills/
  planner/
    SKILL.md
    rules.md
    checklist.md
    scripts/
  verifier/
    SKILL.md
  reviewer/
    SKILL.md
```

Такой подход хорошо ложится на текущий рынок: и Codex, и Claude уже поддерживают skills на базе `SKILL.md` и open Agent Skills standard. ([OpenAI Developers][4])

### Базовые skills для v1

* `plan-task`
* `review-spec`
* `request-context`
* `implement-feature`
* `verify-changes`
* `review-code`
* `write-docs`
* `teach-challenge`
* `build-visualizer`
* `compact-memory`

---

## 14. Approval gates

Даже в hero-режиме нельзя всё пускать в полный автопилот.

### Обязательные approval gates

* удаление файлов;
* изменение секретов и конфигов;
* сетевые действия;
* миграции БД;
* git push / PR creation;
* package publish;
* изменение CI/CD;
* destructive shell commands;
* внешние интеграции и MCP actions.

### Уровни автономии

* `low` — почти всё через approve
* `medium` — кодит и тестит сам, risky actions спрашивает
* `high` — сам делает всё кроме destructive / secret / external side effects

---

## 15. Инициализация проекта

Команда:

```bash
arc init
```

Что делает:

1. определяет root проекта;
2. создаёт `.arc/`;
3. создаёт шаблоны `CLAUDE.md`, `AGENTS.md`, `REPO_MAP.md`, `DOCS_MAP.md`;
4. создаёт `project.yaml`;
5. создаёт `mode.yaml`;
6. проверяет наличие `claude`, `codex`, `rg`, `ast-grep`, `git`;
7. предлагает выбрать default provider и default mode;
8. создаёт базовые skills;
9. создаёт индексы.

### Важно

`arc init` **не должен** сам заполнять проектные правила смыслом. Он создаёт каркас, секции и TODO.

---

## 16. Логи, трассировка, артефакты

Каждый run должен сохранять:

* входную задачу;
* выбранный режим;
* выбранного провайдера;
* какие skill’ы были активированы;
* какой контекст был подан;
* какие команды запускались;
* какие файлы менялись;
* что найдено;
* что осталось неизвестным;
* итоговую верификацию.

Форматы:

* `JSONL` для машинных логов
* `Markdown` для человеческих артефактов

---

## 17. Evals и критерии качества

Система должна уметь оценивать себя не по “мне понравилось”, а по метрикам.

### Метрики для study

* доля ответов без немедленного полного решения;
* число уточняющих вопросов до выдачи ответа;
* число зафиксированных knowledge gaps;
* число случаев, когда пользователь сам сформулировал решение.

### Метрики для work

* количество найденных unknowns до кодинга;
* доля задач, где был построен flow;
* доля решений, где агент сначала раскритиковал подход;
* доля задач с обновлённой документацией.

### Метрики для hero

* pass rate тестов;
* количество циклов rework;
* число blocked задач;
* число выдуманных фактов;
* количество human handoff’ов;
* качество spec → code consistency.

---

## 18. Нефункциональные требования

### NFR-1

macOS и Linux first-class support.

### NFR-2

Один бинарник для `arc`.

### NFR-3

Работа без UI.

### NFR-4

Детерминированная файловая структура.

### NFR-5

Система должна переживать перезапуск и уметь резюмировать run.

### NFR-6

Любой provider failure не должен ломать локальные артефакты.

### NFR-7

Индексация должна работать инкрементально.

### NFR-8

Все risky actions должны быть auditable.

---

## 19. Что считать готовым в v1

## DoD v1

Система готова, если:

1. можно установить `arc` глобально;
2. можно инициализировать новый или существующий проект;
3. есть переключение между `study`, `work`, `hero`;
4. есть рабочие адаптеры Claude и Codex;
5. есть project scaffolding;
6. есть context engine с `rg` + `ast-grep` + file maps;
7. есть memory engine с active/stale/archive;
8. есть минимум 6 skills;
9. hero pipeline проходит путь `plan -> implement -> verify -> review -> docs`;
10. risky actions проходят approval gates;
11. каждая сессия сохраняет артефакты;
12. можно восстановить run и понять, почему агент сделал то, что сделал.

---

## 20. План реализации по фазам

## Фаза 1 — Основа

* CLI
* config
* init
* project scaffolding
* provider detection
* basic logging
* mode switch

## Фаза 2 — Контекст

* индексы
* grep/ast search
* memory engine
* context pack assembler
* compaction

## Фаза 3 — Режимы

* study policy
* work policy
* hero policy
* базовые skills
* approval gates

## Фаза 4 — Оркестрация

* role pipeline
* run state machine
* resume/retry
* artifact generator

## Фаза 5 — Качество

* verifier
* reviewer
* docs-agent
* evals
* anti-hallucination checks

---

## 21. Самое важное архитектурное решение

**Не делать provider-specific brain. Делать provider-neutral orchestration core.**

То есть:

* внутри `arc` — канонические режимы, роли, память, индексы, политики;
* снаружи — Claude/Codex adapters;
* проектные правила — в файлах;
* реальная сила продукта — в guardrails, context packaging и workflow discipline.

Именно это и даст “прорыв”, а не просто очередной промпт.

---

## 22. Отдельные прямые указания для code-agent

### Обязательные решения

* не использовать vector DB как обязательную зависимость в v1;
* не строить UI;
* не делать “магическое” автозаполнение человеческих файлов;
* не делать всё через один гигантский prompt;
* не тащить всю репу в контекст;
* не позволять hero-режиму придумывать неизвестные детали;
* не смешивать машинный индекс и human-authored maps.

### Предпочтения реализации

* Go
* SQLite для метаданных
* JSON/YAML config
* JSONL logs
* Markdown artifacts
* subprocess adapters для `claude` и `codex`

---

## 23. Минимальный стартовый backlog

1. `arc doctor`
2. `arc init`
3. `arc mode set`
4. `arc index build`
5. `arc task plan`
6. `arc task run`
7. `arc task verify`
8. `arc docs generate`
9. `arc memory compact`
10. `arc run resume`

