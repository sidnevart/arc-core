package app

import (
	"strings"

	"agent-os/internal/chat"
)

func chatFailureNextAction(session chat.Session) string {
	lastError := strings.ToLower(strings.TrimSpace(session.Metadata["last_error"]))
	lastPrompt := strings.ToLower(strings.TrimSpace(lastUserPrompt(session)))
	retryExhausted := strings.EqualFold(strings.TrimSpace(session.Metadata["chat_retry_status"]), "exhausted")

	switch {
	case strings.Contains(lastError, "failed to refresh available models"):
		if promptRequestsVisualOutput(lastPrompt) {
			if retryExhausted {
				return "ARC уже сам попробовал повторить этот миниапп-запрос, но Codex снова завис на обновлении моделей. Повтори запрос позже, переключи модель или сначала попроси короткое объяснение, а потом отдельным сообщением миниапп."
			}
			return "Повтори запрос ещё раз. Если Codex снова зависнет на обновлении моделей, переключи модель или сначала попроси короткое объяснение, а потом отдельным сообщением миниапп."
		}
		if retryExhausted {
			return "ARC уже сам попробовал повторить запрос, но Codex снова завис на обновлении моделей. Повтори запрос позже, переключи модель или перезапусти локальный Codex runtime."
		}
		return "Повтори запрос ещё раз. Если ошибка повторится, переключи модель или перезапусти локальный Codex runtime."
	case strings.Contains(lastError, "timed out after"):
		if promptRequestsVisualOutput(lastPrompt) {
			return "Для миниаппов и симуляций Codex иногда упирается в timeout. Повтори запрос ещё раз; если снова сорвётся, сначала запроси короткий текстовый ответ, а затем отдельным сообщением миниапп."
		}
		return "Повтори запрос ещё раз или разбей задачу на более короткий шаг."
	default:
		return "Открой последний ответ и попроси агента объяснить, что пошло не так."
	}
}

func lastUserPrompt(session chat.Session) string {
	for i := len(session.Messages) - 1; i >= 0; i-- {
		if session.Messages[i].Role != "user" {
			continue
		}
		return chat.SanitizeVisibleChatText(session.Messages[i].Content)
	}
	return ""
}
