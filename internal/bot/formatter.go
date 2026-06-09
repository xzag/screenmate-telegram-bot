package bot

import (
	"fmt"
	"strings"

	"screenmate-bot/internal/service"
)

func (b *Bot) formatAllRooms(rooms []service.RoomView) string {
	var sb strings.Builder

	sb.WriteString("❄️ <b>Кондиционирование</b>\n\n")

	if len(rooms) == 0 {
		sb.WriteString("Комнаты не настроены.\n")
		sb.WriteString(fmt.Sprintf("\nОбновлено: <code>%s</code>", b.clock.FormatTime(b.clock.Now())))
		return sb.String()
	}

	for _, room := range rooms {
		sb.WriteString(fmt.Sprintf("🏢 <b>%s</b>\n\n", escapeHTML(room.Name)))

		if room.Err != nil {
			sb.WriteString(fmt.Sprintf(
				"⚠️ ошибка: <code>%s</code>\n\n",
				escapeHTML(room.Err.Error()),
			))
			continue
		}

		if len(room.Conditioners) == 0 {
			sb.WriteString("Кондиционеры не настроены.\n\n")
			continue
		}

		for _, ac := range room.Conditioners {
			icon := "🔴"
			state := "выключен"

			if !ac.Found {
				icon = "⚪"
				state = "не найден"
			} else if ac.Power {
				icon = "🟢"
				state = "включен"
			}

			label := fmt.Sprintf("Кондиционер %d", ac.Number)
			if ac.Comment != "" {
				label += " — " + ac.Comment
			}

			setpoint := ""
			if ac.HasSetpoint {
				setpoint = fmt.Sprintf(", %s%s", ac.Setpoint, ac.SetpointUnit)
			}

			sb.WriteString(fmt.Sprintf(
				"%s %s - %s%s\n",
				icon,
				escapeHTML(label),
				state,
				escapeHTML(setpoint),
			))
		}

		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Обновлено: <code>%s</code>", b.clock.FormatTime(b.clock.Now())))

	return sb.String()
}

func formatRoomsMenu(rooms []service.RoomShort) string {
	var b strings.Builder

	b.WriteString("❄️ <b>Кондиционирование</b>\n\n")

	if len(rooms) == 0 {
		b.WriteString("Комнаты не настроены.")
		return b.String()
	}

	b.WriteString("Выбери комнату:")

	return b.String()
}

func (b *Bot) formatRoom(room service.RoomView) string {
	var sb strings.Builder

	sb.WriteString("❄️ <b>Кондиционирование</b>\n\n")
	sb.WriteString(fmt.Sprintf("🏢 <b>%s</b>\n\n", escapeHTML(room.Name)))

	if room.Err != nil {
		sb.WriteString(fmt.Sprintf("⚠️ ошибка: <code>%s</code>\n", escapeHTML(room.Err.Error())))
		return sb.String()
	}

	if len(room.Conditioners) == 0 {
		sb.WriteString("Кондиционеры не настроены.\n")
	} else {
		for _, ac := range room.Conditioners {
			icon := "🔴"
			state := "выключен"

			if !ac.Found {
				icon = "⚪"
				state = "не найден"
			} else if ac.Power {
				icon = "🟢"
				state = "включен"
			}

			name := fmt.Sprintf("Кондиционер %d", ac.Number)
			if ac.Comment != "" {
				name = ac.Comment
			}

			setpoint := ""
			if ac.HasSetpoint {
				setpoint = fmt.Sprintf(", %s%s", ac.Setpoint, ac.SetpointUnit)
			}

			sb.WriteString(fmt.Sprintf(
				"%s %s — %s%s\n",
				icon,
				escapeHTML(name),
				state,
				escapeHTML(setpoint),
			))
		}
	}

	sb.WriteString(fmt.Sprintf("\nОбновлено: <code>%s</code>", b.clock.FormatTime(b.clock.Now())))

	return sb.String()
}

func formatRoomLoading(action string) string {
	return fmt.Sprintf(
		"❄️ <b>Кондиционирование</b>\n\n⏳ %s...",
		escapeHTML(action),
	)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")

	return s
}
