package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"screenmate-bot/internal/screenmate"
	"screenmate-bot/internal/service"
)

func (b *Bot) handleMyChatMember(update *tgbotapi.ChatMemberUpdated) {
	chat := update.Chat
	from := update.From

	log.Printf(
		"bot chat member updated: chat_id=%d chat_type=%s chat_title=%q chat_username=%q from_user_id=%d from_username=%q from_name=%q old_status=%s new_status=%s",
		chat.ID,
		chat.Type,
		chat.Title,
		chat.UserName,
		from.ID,
		from.UserName,
		strings.TrimSpace(from.FirstName+" "+from.LastName),
		update.OldChatMember.Status,
		update.NewChatMember.Status,
	)

	// Можно отдельно красиво подсветить именно добавление/выдачу доступа.
	switch update.NewChatMember.Status {
	case "member", "administrator":
		log.Printf(
			"bot added/enabled: chat_id=%d chat_type=%s chat_title=%q added_by_user_id=%d added_by_username=%q",
			chat.ID,
			chat.Type,
			chat.Title,
			from.ID,
			from.UserName,
		)

	case "left", "kicked":
		log.Printf(
			"bot removed/disabled: chat_id=%d chat_type=%s chat_title=%q removed_by_user_id=%d removed_by_username=%q",
			chat.ID,
			chat.Type,
			chat.Title,
			from.ID,
			from.UserName,
		)
	}
}

func isPrivateChat(chat *tgbotapi.Chat) bool {
	return chat != nil && chat.Type == "private"
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	if msg.From == nil {
		return
	}

	// В группах/каналах молчим полностью.
	// Никаких "Нет доступа", никаких /chatid, ничего.
	if !isPrivateChat(msg.Chat) {
		log.Printf(
			"ignore non-private message: chat_id=%d chat_type=%s chat_title=%q from_user_id=%d command=%q",
			msg.Chat.ID,
			msg.Chat.Type,
			msg.Chat.Title,
			msg.From.ID,
			msg.Command(),
		)
		return
	}

	if !b.isAllowed(msg.From.ID) {
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
	if cb.From == nil {
		return
	}

	if cb.Message != nil && !isPrivateChat(cb.Message.Chat) {
		log.Printf(
			"ignore non-private callback: chat_id=%d chat_type=%s from_user_id=%d data=%q",
			cb.Message.Chat.ID,
			cb.Message.Chat.Type,
			cb.From.ID,
			cb.Data,
		)

		// Это не сообщение в группу, а маленький popup у нажавшего.
		b.answerCallback(cb.ID, "Бот работает только в личке")
		return
	}

	if !b.isAllowed(cb.From.ID) {
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

	b.editRefreshedRoom(cb, roomKey)
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

	b.editRefreshedRoom(cb, roomKey)
}

func (b *Bot) editRefreshedRoom(cb *tgbotapi.CallbackQuery, roomKey string) {
	if cb.Message == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	room, err := b.service.RefreshRoomStatus(ctx, roomKey)
	if err != nil {
		log.Printf("refresh room status roomKey=%q: %v", roomKey, err)
		b.editRoomError(cb, roomKey, "Не удалось обновить комнату", err)
		return
	}

	b.editRoomView(cb, room)
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

	started := time.Now()

	audit := newAuditEvent(cb.From, auditActionTogglePower, roomKey, acNumber)
	audit.Expected = boolString(expectedPower)

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	result, room, err := b.service.TogglePowerIfState(ctx, roomKey, acNumber, expectedPower)
	if err != nil {
		audit.Result = auditResultError
		audit.Error = err.Error()
		audit.Duration = time.Since(started)
		logAudit(audit)

		log.Printf("toggle power: %v", err)
		b.editRoomError(cb, roomKey, "Не удалось переключить кондиционер", err)
		return
	}

	audit.RoomName = room.Name
	audit.Actual = boolString(result.CurrentPower)
	audit.Duration = time.Since(started)

	if result.StateChanged {
		audit.Result = auditResultSkipped
	} else {
		audit.Result = auditResultSuccess
	}

	logAudit(audit)

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
	action := auditActionTempUp

	switch directionRaw {
	case "u":
		direction = screenmate.TemperatureUp
		action = auditActionTempUp

	case "d":
		direction = screenmate.TemperatureDown
		action = auditActionTempDown

	default:
		b.editRoomError(
			cb,
			roomKey,
			"Не удалось изменить температуру",
			fmt.Errorf("unknown temperature direction %q", directionRaw),
		)
		return
	}

	started := time.Now()

	audit := newAuditEvent(cb.From, action, roomKey, acNumber)
	audit.Expected = expectedSetpoint

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	result, room, err := b.service.AdjustTemperatureIfState(
		ctx,
		roomKey,
		acNumber,
		direction,
		expectedSetpoint,
	)
	if err != nil {
		audit.Result = auditResultError
		audit.Error = err.Error()
		audit.Duration = time.Since(started)
		logAudit(audit)

		log.Printf("adjust temperature: %v", err)
		b.editRoomError(cb, roomKey, "Не удалось изменить температуру", err)
		return
	}

	audit.RoomName = room.Name
	audit.Actual = result.CurrentSetpoint
	audit.Duration = time.Since(started)

	if result.StateChanged {
		audit.Result = auditResultSkipped
	} else {
		audit.Result = auditResultSuccess
	}

	logAudit(audit)

	b.editRoomView(cb, room)
}
