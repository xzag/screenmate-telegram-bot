// internal/bot/keyboard.go

package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"screenmate-bot/internal/service"
)

const (
	callbackRefreshAll   = "refresh_all"
	callbackTogglePrefix = "toggle"
)

func statusKeyboard(rooms []service.RoomView) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, room := range rooms {
		if room.Err != nil {
			continue
		}

		for _, ac := range room.Conditioners {
			if !ac.Found {
				continue
			}

			icon := "🔴"
			if ac.Power {
				icon = "🟢"
			}

			label := fmt.Sprintf("%s %s: %d", icon, room.Name, ac.Number)
			if ac.Comment != "" {
				label = fmt.Sprintf("%s %s: %s", icon, room.Name, ac.Comment)
			}

			callbackData := fmt.Sprintf("%s:%s:%d", callbackTogglePrefix, room.Key, ac.Number)

			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(label, callbackData),
			))
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", callbackRefreshAll),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
