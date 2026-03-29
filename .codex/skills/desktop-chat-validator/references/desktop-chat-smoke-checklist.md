# Desktop Chat Smoke Checklist

## Demo/runtime

- Open an old chat that contains a saved demo or simulation.
- Click `Запустить снова` on the message card.
- Confirm the clicked card shows `starting`, then `ready`, then inline preview.
- Open `Детали чата` and confirm the same runtime appears under running items.
- Stop the runtime from either surface and confirm both surfaces show `stopped`.
- Open the same demo in ARC fullscreen and confirm it reuses the same runtime truth.

## Drawer/details

- `Детали чата` must show live runtimes first when they exist.
- Materials must include the current topic outputs.
- Files must include ARC-managed uploads and project files.
- The panel must never look empty when the chat already has outputs.

## Chat UX

- The main chat viewport should fit multiple short messages without feeling oversized.
- Project/agent/new chat controls should live in the left rail, not in a heavy top chrome.
- Language/settings should not occupy the main chat workspace.
- Composer should keep `Enter` to send and `Shift+Enter` for a new line.

## Fail conditions

- `Запустить снова` visibly does nothing.
- The message card still says `Приложение остановлено` after a successful launch.
- `Детали чата` and the message card disagree about runtime state.
- Demo fullscreen inside ARC opens a stale or broken preview.
- Chat shell chrome pushes the thread so low that only a couple of small messages fit on screen.
