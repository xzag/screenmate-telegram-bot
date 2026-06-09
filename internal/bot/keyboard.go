package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"screenmate-bot/internal/service"
)

const (
	callbackRefreshAll   = "refresh_all"
	callbackTogglePrefix = "t"
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

			currentState := 0
			icon := "🔴"
			action := "включить"

			if ac.Power {
				currentState = 1
				icon = "🟢"
				action = "выключить"
			}

			name := fmt.Sprintf("Кондиционер %d", ac.Number)
			if ac.Comment != "" {
				name = ac.Comment
			}

			label := fmt.Sprintf("%s %s · %s", icon, name, action)

			callbackData := fmt.Sprintf(
				"%s:%s:%d:%d",
				callbackTogglePrefix,
				room.Key,
				ac.Number,
				currentState,
			)

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
