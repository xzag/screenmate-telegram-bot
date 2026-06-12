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

	membershipCache map[int64]membershipCacheItem
}

type membershipCacheItem struct {
	allowed   bool
	expiresAt time.Time
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
		api:             api,
		cfg:             cfg,
		service:         svc,
		clock:           clock,
		membershipCache: make(map[int64]membershipCacheItem),
	}, nil
}

func (b *Bot) Run() {
	log.Printf("telegram bot authorized as @%s", b.api.Self.UserName)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30
	updateConfig.AllowedUpdates = []string{
		"message",
		"callback_query",
		"my_chat_member",
	}

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

		if update.MyChatMember != nil {
			b.handleMyChatMember(update.MyChatMember)
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

func (b *Bot) isAdmin(userID int64) bool {
	for _, adminID := range b.cfg.Telegram.Admins {
		if adminID == userID {
			return true
		}
	}

	return false
}

func (b *Bot) isAllowed(userID int64) bool {
	return true

	// for now we skip that
	//if b.isAdmin(userID) {
	//	return true
	//}
	//
	//return b.isMemberOfAnyAccessGroup(userID)
}

func (b *Bot) isMemberOfAnyAccessGroup(userID int64) bool {
	if len(b.cfg.Telegram.AccessGroupIDs) == 0 {
		return false
	}

	if cached, ok := b.membershipCache[userID]; ok && time.Now().Before(cached.expiresAt) {
		return cached.allowed
	}

	allowed := false

	for _, groupID := range b.cfg.Telegram.AccessGroupIDs {
		if b.isMemberOfGroup(userID, groupID) {
			allowed = true
			break
		}
	}

	b.membershipCache[userID] = membershipCacheItem{
		allowed:   allowed,
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	return allowed
}

func (b *Bot) isMemberOfGroup(userID int64, groupID int64) bool {
	member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: groupID,
			UserID: userID,
		},
	})
	if err != nil {
		log.Printf(
			"get chat member failed: user_id=%d group_id=%d err=%v",
			userID,
			groupID,
			err,
		)
		return false
	}

	switch member.Status {
	case "creator", "administrator", "member":
		return true

	case "restricted":
		// Спорно. Я бы сначала не пускала.
		return false

	case "left", "kicked":
		return false

	default:
		log.Printf(
			"unknown chat member status: user_id=%d group_id=%d status=%q",
			userID,
			groupID,
			member.Status,
		)
		return false
	}
}

func contextTimeout() time.Duration {
	return 60 * time.Second
}
