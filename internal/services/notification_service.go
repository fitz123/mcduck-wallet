// File: ./internal/services/notification_service.go
package services

import (
	"context"

	"github.com/fitz123/mcduck-wallet/internal/logger"
	tele "gopkg.in/telebot.v3"
)

type NotificationService interface {
	NotifyUser(ctx context.Context, telegramID int64, message string) error
}

type notificationService struct {
	bot *tele.Bot
}

func NewNotificationService(bot *tele.Bot) NotificationService {
	return &notificationService{
		bot: bot,
	}
}

func (ns *notificationService) NotifyUser(ctx context.Context, telegramID int64, message string) error {
	recipient := &tele.User{
		ID: telegramID,
	}
	if _, err := ns.bot.Send(recipient, message); err != nil {
		logger.Error("Failed to send notification", "telegramID", telegramID, "error", err)
		return err
	}
	return nil
}
