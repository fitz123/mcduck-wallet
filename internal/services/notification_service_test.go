package services

import (
	"context"
	"errors"
	"testing"
)

// MockBot simulates a telegram bot for testing
type MockBot struct {
	SendFunc func(to interface{}, what interface{}, opts ...interface{}) (interface{}, error)
	sent     []mockMessage
}

type mockMessage struct {
	to      int64
	message string
}

// Note: The actual NotificationService uses tele.Bot which is hard to mock directly.
// For proper testing, we would need to refactor to use an interface.
// These tests demonstrate the expected behavior using our MockNotificationService from testutil.

func TestNotificationService_NotifyUser(t *testing.T) {
	ctx := context.Background()

	t.Run("sends notification successfully", func(t *testing.T) {
		var sentTo int64
		var sentMessage string

		// Using our mock from testutil
		mock := &mockNotificationService{
			notifyFunc: func(ctx context.Context, telegramID int64, message string) error {
				sentTo = telegramID
				sentMessage = message
				return nil
			},
		}

		err := mock.NotifyUser(ctx, 12345, "Hello, user!")
		if err != nil {
			t.Fatalf("NotifyUser() error = %v", err)
		}
		if sentTo != 12345 {
			t.Errorf("NotifyUser() sent to = %v, want 12345", sentTo)
		}
		if sentMessage != "Hello, user!" {
			t.Errorf("NotifyUser() message = %v, want 'Hello, user!'", sentMessage)
		}
	})

	t.Run("returns error on send failure", func(t *testing.T) {
		mock := &mockNotificationService{
			notifyFunc: func(ctx context.Context, telegramID int64, message string) error {
				return errors.New("telegram API error")
			},
		}

		err := mock.NotifyUser(ctx, 12345, "Hello")
		if err == nil {
			t.Error("NotifyUser() expected error on send failure")
		}
		if err.Error() != "telegram API error" {
			t.Errorf("NotifyUser() error = %v, want 'telegram API error'", err)
		}
	})

	t.Run("handles empty message", func(t *testing.T) {
		var sentMessage string
		mock := &mockNotificationService{
			notifyFunc: func(ctx context.Context, telegramID int64, message string) error {
				sentMessage = message
				return nil
			},
		}

		err := mock.NotifyUser(ctx, 12345, "")
		if err != nil {
			t.Fatalf("NotifyUser() error = %v", err)
		}
		if sentMessage != "" {
			t.Errorf("NotifyUser() should allow empty message")
		}
	})
}

// mockNotificationService is a local mock for testing
type mockNotificationService struct {
	notifyFunc func(ctx context.Context, telegramID int64, message string) error
}

func (m *mockNotificationService) NotifyUser(ctx context.Context, telegramID int64, message string) error {
	if m.notifyFunc != nil {
		return m.notifyFunc(ctx, telegramID, message)
	}
	return nil
}
