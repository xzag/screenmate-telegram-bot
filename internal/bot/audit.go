package bot

import (
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	auditResultSuccess = "success"
	auditResultSkipped = "skipped"
	auditResultError   = "error"

	auditActionTogglePower = "toggle_power"
	auditActionTempUp      = "temp_up"
	auditActionTempDown    = "temp_down"
)

type auditEvent struct {
	Action string
	Result string

	UserID       int64
	UserUsername string
	UserName     string

	RoomKey  string
	RoomName string
	ACNumber int

	Expected string
	Actual   string
	Error    string

	Duration time.Duration
}

func newAuditEvent(user *tgbotapi.User, action string, roomKey string, acNumber int) auditEvent {
	event := auditEvent{
		Action:   action,
		RoomKey:  roomKey,
		ACNumber: acNumber,
	}

	if user != nil {
		event.UserID = user.ID
		event.UserUsername = user.UserName
		event.UserName = strings.TrimSpace(user.FirstName + " " + user.LastName)
	}

	return event
}

func logAudit(event auditEvent) {
	log.Printf(
		"audit action=%q result=%q user_id=%d username=%q name=%q room=%q room_name=%q ac=%d expected=%q actual=%q duration_ms=%d error=%q",
		event.Action,
		event.Result,
		event.UserID,
		event.UserUsername,
		event.UserName,
		event.RoomKey,
		event.RoomName,
		event.ACNumber,
		event.Expected,
		event.Actual,
		event.Duration.Milliseconds(),
		event.Error,
	)
}

func boolString(v bool) string {
	if v {
		return "on"
	}

	return "off"
}
