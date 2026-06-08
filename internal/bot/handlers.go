package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

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

// internal/bot/handlers.go

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
	log.Printf("toggle callback data=%q from=%d", cb.Data, cb.From.ID)
	b.answerCallback(cb.ID, "Переключаю...")

	roomKey, acNumber, err := parseToggleCallback(cb.Data)
	if err != nil {
		log.Printf("parse toggle callback: %v", err)
		return
	}

	log.Printf("toggle parsed roomKey=%q acNumber=%d", roomKey, acNumber)

	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout())
	defer cancel()

	if err := b.service.TogglePower(ctx, roomKey, acNumber); err != nil {
		log.Printf("toggle power: %v", err)

		if cb.Message != nil {
			msg := tgbotapi.NewMessage(cb.Message.Chat.ID, "⚠️ Не удалось переключить кондиционер: "+err.Error())
			if _, sendErr := b.api.Send(msg); sendErr != nil {
				log.Printf("send toggle error: %v", sendErr)
			}
		}

		return
	}

	b.editStatus(cb)
}

func parseToggleCallback(data string) (string, int, error) {
	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		return "", 0, fmt.Errorf("invalid callback data: %q", data)
	}

	if parts[0] != callbackTogglePrefix {
		return "", 0, fmt.Errorf("invalid callback prefix: %q", parts[0])
	}

	acNumber, err := strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid ac number: %w", err)
	}

	return parts[1], acNumber, nil
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
