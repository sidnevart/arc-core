# ARC Desktop Wails App

Это канонический native desktop package для ARC.

Здесь находятся:

- `app.go` — native bridge-facing application object
- `assets.go` — embedded asset bundle для Wails
- `frontend/` — текущий native frontend bundle

## Роль в структуре проекта

- `apps/desktop/static/` — preview shell
- `apps/desktop/wailsapp/` — native shell package
- `cmd/arc-desktop-wails/` — только launcher для сборки native desktop app

## Почему это важно

Такой layout отделяет:

- продуктовую desktop-зону
- native packaging entrypoint
- runtime/backend contract

И позволяет развивать desktop как отдельный продуктовый слой, не смешивая его с CLI entrypoints.
