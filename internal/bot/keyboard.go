package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"screenmate-bot/internal/service"
)

func roomsKeyboard(rooms []service.RoomShort) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(rooms))

	for _, room := range rooms {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(room.Name, roomCallback(room.Key)),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func roomKeyboard(room service.RoomView) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	for _, ac := range room.Conditioners {
		if !ac.Found {
			continue
		}

		icon := "🔴"
		action := "включить"

		if ac.Power {
			icon = "🟢"
			action = "выключить"
		}

		name := fmt.Sprintf("Кондиционер %d", ac.Number)
		if ac.Comment != "" {
			name = ac.Comment
		}

		powerLabel := fmt.Sprintf("%s %s · %s", icon, name, action)

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				powerLabel,
				toggleCallback(room.Key, ac.Number, ac.Power),
			),
		))

		if ac.HasSetpoint {
			tempLabel := fmt.Sprintf("🌡 %s%s", ac.Setpoint, ac.SetpointUnit)

			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"🥶",
					temperatureCallback(room.Key, ac.Number, "d", ac.Setpoint),
				),
				tgbotapi.NewInlineKeyboardButtonData(
					tempLabel,
					callbackNoop,
				),
				tgbotapi.NewInlineKeyboardButtonData(
					"🔥",
					temperatureCallback(room.Key, ac.Number, "u", ac.Setpoint),
				),
			))
		}
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить комнату", refreshRoomCallback(room.Key)),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⬅️ К комнатам", callbackRoomsList),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func loadingKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏳ Выполняется...", callbackWait),
		),
	)
}
