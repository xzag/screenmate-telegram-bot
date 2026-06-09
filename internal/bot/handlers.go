package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
	case "start", "status":
		b.sendStatus(msg.Chat.ID)
	default:
		b.sendText(msg.Chat.ID, "Команды: /status")
	}
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	if cb.From == nil || !b.isAllowed(cb.From.ID) {
		b.answerCallback(cb.ID, "Нет доступа.")
		return
	}

	if cb.Data == callbackRefreshAll {
		b.editStatus(cb)
		return
	}

	if strings.HasPrefix(cb.Data, callbackTogglePrefix+":") {
		b.handleToggleCallback(cb)
		return
	}

	b.answerCallback(cb.ID, "Неизвестная команда")
}

func (b *Bot) handleToggleCallback(cb *tgbotapi.CallbackQuery) {
	roomKey, acNumber, expectedPower, err := parseToggleCallback(cb.Data)
	if err != nil {
		log.Printf("parse toggle callback: %v", err)
		b.answerCallback(cb.ID, "Некорректная кнопка")
		return
	}

	b.answerCallback(cb.ID, "Проверяю состояние...")

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	result, err := b.service.TogglePowerIfState(ctx, roomKey, acNumber, expectedPower)
	if err != nil {
		log.Printf("toggle power: %v", err)

		if cb.Message != nil {
			msg := tgbotapi.NewMessage(
				cb.Message.Chat.ID,
				"⚠️ Не удалось переключить кондиционер: "+err.Error(),
			)

			if _, sendErr := b.api.Send(msg); sendErr != nil {
				log.Printf("send toggle error: %v", sendErr)
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

	b.editStatus(cb)
}

func parseToggleCallback(data string) (roomKey string, acNumber int, expectedPower bool, err error) {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return "", 0, false, fmt.Errorf("invalid callback data: %q", data)
	}

	if parts[0] != callbackTogglePrefix {
		return "", 0, false, fmt.Errorf("invalid callback prefix: %q", parts[0])
	}

	acNumber, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, false, fmt.Errorf("invalid ac number: %w", err)
	}

	switch parts[3] {
	case "1":
		expectedPower = true
	case "0":
		expectedPower = false
	default:
		return "", 0, false, fmt.Errorf("invalid expected power: %q", parts[3])
	}

	return parts[1], acNumber, expectedPower, nil
}

func (b *Bot) sendStatus(chatID int64) {
	text, keyboard := b.loadStatus()

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("send status: %v", err)
	}
}

func (b *Bot) editStatus(cb *tgbotapi.CallbackQuery) {
	if cb.Message == nil {
		b.answerCallback(cb.ID, "Не удалось обновить сообщение")
		return
	}

	text, keyboard := b.loadStatus()

	edit := tgbotapi.NewEditMessageTextAndMarkup(
		cb.Message.Chat.ID,
		cb.Message.MessageID,
		text,
		keyboard,
	)
	edit.ParseMode = tgbotapi.ModeHTML

	if _, err := b.api.Send(edit); err != nil {
		log.Printf("edit status: %v", err)
		b.answerCallback(cb.ID, "Не удалось обновить")
		return
	}

	b.answerCallback(cb.ID, "Обновлено")
}

func (b *Bot) loadStatus() (string, tgbotapi.InlineKeyboardMarkup) {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	rooms := b.service.AllRoomsStatus(ctx)

	return formatAllRooms(rooms), statusKeyboard(rooms)
}
