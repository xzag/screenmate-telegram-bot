package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"screenmate-bot/internal/screenmate"
	"screenmate-bot/internal/service"
)

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if msg.From == nil || !b.isAllowed(msg.From.ID) {
		b.sendText(msg.Chat.ID, "Нет доступа.")
		return
	}

	switch msg.Command() {
	case "start":
		b.sendRoomsMenu(msg.Chat.ID)

	case "status":
		b.sendAllRoomsStatus(msg.Chat.ID)

	default:
		b.sendText(msg.Chat.ID, "Команды: /start, /status")
	}
}

func (b *Bot) sendAllRoomsStatus(chatID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	rooms := b.service.AllRoomsStatus(ctx)

	msg := tgbotapi.NewMessage(chatID, b.formatAllRooms(rooms))
	msg.ParseMode = tgbotapi.ModeHTML

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("send all rooms status: %v", err)
	}
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	if cb.From == nil || !b.isAllowed(cb.From.ID) {
		b.answerCallback(cb.ID, "Нет доступа.")
		return
	}

	if cb.Data == callbackNoop {
		b.answerCallback(cb.ID, "")
		return
	}

	if cb.Data == callbackWait {
		b.answerCallback(cb.ID, "Команда уже выполняется...")
		return
	}

	if cb.Data == callbackRoomsList {
		b.answerCallback(cb.ID, "Комнаты")
		b.editRoomsMenu(cb)
		return
	}

	if strings.HasPrefix(cb.Data, callbackTempPrefix+":") {
		b.handleTemperatureCallback(cb)
		return
	}

	if strings.HasPrefix(cb.Data, callbackOpenRoom+":") {
		b.handleOpenRoomCallback(cb)
		return
	}

	if strings.HasPrefix(cb.Data, callbackRefreshRoom+":") {
		b.handleRefreshRoomCallback(cb)
		return
	}

	if strings.HasPrefix(cb.Data, callbackTogglePrefix+":") {
		b.handleToggleCallback(cb)
		return
	}

	b.answerCallback(cb.ID, "Неизвестная команда")
}

func (b *Bot) sendRoomsMenu(chatID int64) {
	rooms := b.service.Rooms()

	msg := tgbotapi.NewMessage(chatID, formatRoomsMenu(rooms))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = roomsKeyboard(rooms)

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("send rooms menu: %v", err)
	}
}

func (b *Bot) editRoomsMenu(cb *tgbotapi.CallbackQuery) {
	if cb.Message == nil {
		return
	}

	rooms := b.service.Rooms()

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		formatRoomsMenu(rooms),
		roomsKeyboard(rooms),
	)
	edit.ParseMode = tgbotapi.ModeHTML

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("edit rooms menu: %v", err)
		return
	}
}

func (b *Bot) handleOpenRoomCallback(cb *tgbotapi.CallbackQuery) {
	roomKey, err := parseRoomCallback(cb.Data, callbackOpenRoom)
	if err != nil {
		log.Printf("parse open room callback: %v", err)
		b.answerCallback(cb.ID, "Некорректная кнопка")
		return
	}

	b.answerCallback(cb.ID, "Открываю...")
	b.editLoading(cb, "Загружаю комнату")

	b.editRoom(cb, roomKey)
}

func (b *Bot) handleRefreshRoomCallback(cb *tgbotapi.CallbackQuery) {
	roomKey, err := parseRoomCallback(cb.Data, callbackRefreshRoom)
	if err != nil {
		log.Printf("parse refresh room callback: %v", err)
		b.answerCallback(cb.ID, "Некорректная кнопка")
		return
	}

	b.answerCallback(cb.ID, "Обновляю...")
	b.editLoading(cb, "Обновляю комнату")

	b.editRoom(cb, roomKey)
}

func (b *Bot) editRoom(cb *tgbotapi.CallbackQuery, roomKey string) {
	if cb.Message == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	room, err := b.service.RoomStatus(ctx, roomKey)
	if err != nil {
		log.Printf("room status roomKey=%q: %v", roomKey, err)

		b.editRoomError(cb, roomKey, "Не удалось открыть комнату", err)
		return
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		b.formatRoom(room),
		roomKeyboard(room),
	)
	edit.ParseMode = tgbotapi.ModeHTML

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("edit room: %v", err)
		return
	}
}

func (b *Bot) handleToggleCallback(cb *tgbotapi.CallbackQuery) {
	roomKey, acNumber, expectedPower, err := parseToggleCallback(cb.Data)
	if err != nil {
		log.Printf("parse toggle callback: %v", err)
		b.answerCallback(cb.ID, "Некорректная кнопка")
		return
	}

	b.answerCallback(cb.ID, "Переключаю...")
	b.editLoading(cb, "Переключаю кондиционер")

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	result, room, err := b.service.TogglePowerIfState(ctx, roomKey, acNumber, expectedPower)
	if err != nil {
		log.Printf("toggle power: %v", err)
		b.editRoomError(cb, roomKey, "Не удалось переключить кондиционер", err)
		return
	}

	if result.StateChanged {
		log.Printf(
			"skip toggle: room=%s ac=%d expected=%v current=%v",
			roomKey,
			acNumber,
			expectedPower,
			result.CurrentPower,
		)
	}

	b.editRoomView(cb, room)
}

func (b *Bot) editRoomError(
	cb *tgbotapi.CallbackQuery,
	roomKey string,
	title string,
	err error,
) {
	if cb.Message == nil {
		return
	}

	text := fmt.Sprintf(
		"⚠️ <b>%s</b>\n\n<code>%s</code>",
		escapeHTML(title),
		escapeHTML(err.Error()),
	)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🔄 Обновить комнату",
				refreshRoomCallback(roomKey),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"⬅️ К комнатам",
				callbackRoomsList,
			),
		),
	)

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		text,
		keyboard,
	)
	edit.ParseMode = tgbotapi.ModeHTML

	if _, sendErr := b.api.Send(edit); sendErr != nil {
		log.Printf("edit room error: %v", sendErr)
	}
}

func (b *Bot) editRoomView(cb *tgbotapi.CallbackQuery, room service.RoomView) {
	if cb.Message == nil {
		return
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		b.formatRoom(room),
		roomKeyboard(room),
	)
	edit.ParseMode = tgbotapi.ModeHTML

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("edit room view: %v", err)
	}
}

func (b *Bot) editLoading(cb *tgbotapi.CallbackQuery, action string) {
	if cb.Message == nil {
		return
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		formatRoomLoading(action),
		loadingKeyboard(),
	)
	edit.ParseMode = tgbotapi.ModeHTML

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("edit loading: %v", err)
	}
}

func (b *Bot) handleTemperatureCallback(cb *tgbotapi.CallbackQuery) {
	roomKey, acNumber, directionRaw, expectedSetpoint, err := parseTemperatureCallback(cb.Data)
	if err != nil {
		log.Printf("parse temperature callback: %v", err)
		b.answerCallback(cb.ID, "Некорректная кнопка")
		return
	}

	b.answerCallback(cb.ID, "Меняю температуру...")
	b.editLoading(cb, "Меняю температуру")

	var direction screenmate.TemperatureDirection
	switch directionRaw {
	case "u":
		direction = screenmate.TemperatureUp
	case "d":
		direction = screenmate.TemperatureDown
	default:
		b.answerCallback(cb.ID, "Некорректное направление")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	result, room, err := b.service.AdjustTemperatureIfState(ctx, roomKey, acNumber, direction, expectedSetpoint)
	if err != nil {
		log.Printf("adjust temperature: %v", err)
		b.editRoomError(cb, roomKey, "Не удалось изменить температуру", err)
		return
	}

	if result.StateChanged {
		log.Printf(
			"skip temperature change: room=%s ac=%d expected=%s current=%s",
			roomKey,
			acNumber,
			expectedSetpoint,
			result.CurrentSetpoint,
		)
	}

	b.editRoomView(cb, room)
}
