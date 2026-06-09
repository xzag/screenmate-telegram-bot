// internal/bot/handlers.go

package bot

import (
	"context"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	msg := tgbotapi.NewMessage(chatID, formatAllRooms(rooms))
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

	if cb.Data == callbackWait {
		b.answerCallback(cb.ID, "Команда уже выполняется...")
		return
	}

	if cb.Data == callbackRoomsList {
		b.answerCallback(cb.ID, "Комнаты")
		b.editRoomsMenu(cb)
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

		text := "⚠️ <b>Не удалось открыть комнату</b>\n\n<code>" + escapeHTML(err.Error()) + "</code>"

		edit := tgbotapi.NewEditMessageTextAndMarkup(
			cb.Message.Chat.ID,
			cb.Message.MessageID,
			text,
			tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("⬅️ К комнатам", callbackRoomsList),
				),
			),
		)
		edit.ParseMode = tgbotapi.ModeHTML

		if _, sendErr := b.api.Send(edit); sendErr != nil {
			log.Printf("edit room error: %v", sendErr)
		}

		return
	}

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		formatRoom(room),
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

	result, err := b.service.TogglePowerIfState(ctx, roomKey, acNumber, expectedPower)
	if err != nil {
		log.Printf("toggle power: %v", err)

		if cb.Message != nil {
			edit := tgbotapi.NewEditMessageTextAndMarkup(
				cb.Message.Chat.ID,
				cb.Message.MessageID,
				"⚠️ <b>Не удалось переключить кондиционер</b>\n\n<code>"+escapeHTML(err.Error())+"</code>",
				tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить комнату", refreshRoomCallback(roomKey)),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("⬅️ К комнатам", callbackRoomsList),
					),
				),
			)
			edit.ParseMode = tgbotapi.ModeHTML

			if _, sendErr := b.api.Send(edit); sendErr != nil {
				log.Printf("edit toggle error: %v", sendErr)
			}
		}

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

	time.Sleep(1 * time.Second)

	b.editRoom(cb, roomKey)
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
