# ARC Desktop Wails Shell

Этот каталог содержит первый native shell для ARC Desktop.

Текущий подход:

- `cmd/arc-desktop-wails` теперь только thin launcher
- канонический native desktop package вынесен в `apps/desktop/wailsapp`
- Wails shell ходит в `internal/desktop` bridge напрямую
- `cmd/arc-desktop` остаётся отдельной dev/fallback поверхностью для быстрого локального цикла

Важно:

- файлы в этом каталоге собираются только с build tag `wails`
- обычный `go build ./...` их не трогает
- для реальной сборки нужен установленный `Wails v2`

Следующий логичный шаг:

- расширить native shell до полного parity с preview desktop surface
- вынести общий desktop message catalog между preview и Wails UI
- перейти от текущего asset bundle к полноценному desktop source/toolchain внутри `apps/desktop/`
