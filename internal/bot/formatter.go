package bot

import (
	"fmt"
	"strings"
	"time"

	"screenmate-bot/internal/service"
)

func formatAllRooms(rooms []service.RoomView) string {
	var b strings.Builder

	b.WriteString("❄️ <b>Кондиционирование</b>\n\n")

	if len(rooms) == 0 {
		b.WriteString("Комнаты не настроены.\n")
		b.WriteString(fmt.Sprintf("\nОбновлено: <code>%s</code>", time.Now().Format("15:04:05")))
		return b.String()
	}

	for _, room := range rooms {
		b.WriteString(fmt.Sprintf("🏢 <b>%s</b>\n\n", escapeHTML(room.Name)))

		if room.Err != nil {
			b.WriteString(fmt.Sprintf(
				"⚠️ ошибка: <code>%s</code>\n\n",
				escapeHTML(room.Err.Error()),
			))
			continue
		}

		if len(room.Conditioners) == 0 {
			b.WriteString("Кондиционеры не настроены.\n\n")
			continue
		}

		for _, ac := range room.Conditioners {
			icon := "🔴"

			if !ac.Found {
				icon = "⚪"
			} else if ac.Power {
				icon = "🟢"
			}

			label := fmt.Sprintf("Кондиционер %d", ac.Number)
			if ac.Comment != "" {
				label += " — " + ac.Comment
			}

			b.WriteString(fmt.Sprintf(
				"%s %s\n",
				icon,
				escapeHTML(label),
			))
		}

		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Обновлено: <code>%s</code>", time.Now().Format("15:04:05")))

	return b.String()
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")

	return s
}
