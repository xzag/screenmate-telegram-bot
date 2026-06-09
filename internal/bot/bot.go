package bot

import (
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"screenmate-bot/internal/config"
	"screenmate-bot/internal/service"
	"screenmate-bot/internal/timeutil"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	cfg     config.Config
	service *service.Service
	clock   *timeutil.Clock
}

func New(cfg config.Config, svc *service.Service) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, err
	}

	clock, err := timeutil.NewNovosibirskClock()
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:     api,
		cfg:     cfg,
		service: svc,
		clock:   clock,
	}, nil
}

func (b *Bot) Run() {
	log.Printf("telegram bot authorized as @%s", b.api.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := b.api.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(update.Message)
			continue
		}

		if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
			continue
		}
	}
}

func (b *Bot) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("send text: %v", err)
	}
}

func (b *Bot) answerCallback(callbackID string, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)

	if _, err := b.api.Request(callback); err != nil {
		log.Printf("answer callback: %v", err)
	}
}

func (b *Bot) isAllowed(_ int64) bool {
	return true
}

func contextTimeout() time.Duration {
	return 60 * time.Second
}
