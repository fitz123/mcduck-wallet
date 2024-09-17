// File: pkg/core/core.go

package core

import (
	"errors"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
	"gorm.io/gorm"
)

var (
	ErrInsufficientBalance  = errors.New(messages.ErrInsufficientBalance)
	ErrUserNotFound         = errors.New(messages.ErrUserNotFound)
	ErrUnauthorized         = errors.New(messages.ErrUnauthorized)
	ErrNegativeAmount       = errors.New(messages.ErrNegativeAmount)
	ErrCannotTransferToSelf = errors.New(messages.ErrCannotTransferToSelf)
)

func GetOrCreateUser(telegramID int64, username string) (*database.User, error) {
	var user database.User

	// Try to find the user, including soft-deleted ones
	result := database.DB.Unscoped().Where("telegram_id = ?", telegramID).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// User doesn't exist, create a new one
			user = database.User{
				TelegramID: telegramID,
				Username:   username,
				Balance:    0,
			}
			if err := database.DB.Create(&user).Error; err != nil {
				logger.Error("Failed to create new user", "error", err, "telegramID", telegramID)
				return nil, err
			}
			logger.Info("New user created", "telegramID", telegramID, "username", username)
		} else {
			// Some other error occurred
			logger.Error("Error while fetching user", "error", result.Error, "telegramID", telegramID)
			return nil, result.Error
		}
	} else {
		// User found (might be soft-deleted)
		if user.DeletedAt.Valid {
			// User was soft-deleted, restore it
			if err := database.DB.Unscoped().Model(&user).Update("deleted_at", nil).Error; err != nil {
				logger.Error("Failed to restore soft-deleted user", "error", err, "telegramID", telegramID)
				return nil, err
			}
			logger.Info("Soft-deleted user restored", "telegramID", telegramID, "username", username)
		}

		// Update username if it has changed
		if user.Username != username && username != "" {
			if err := database.DB.Model(&user).Update("username", username).Error; err != nil {
				logger.Error("Failed to update username", "error", err, "telegramID", telegramID)
				return nil, err
			}
			logger.Info("Username updated", "telegramID", telegramID, "oldUsername", user.Username, "newUsername", username)
		}
	}

	return &user, nil
}

func transferMoney(fromUserID, toUserID int64, amount float64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}

	// Prevent self-transfer
	if fromUserID == toUserID {
		return ErrCannotTransferToSelf
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		var fromUser, toUser database.User

		if err := tx.Where("telegram_id = ?", fromUserID).First(&fromUser).Error; err != nil {
			return ErrUserNotFound
		}

		if fromUser.Balance < amount {
			return ErrInsufficientBalance
		}

		if err := tx.Where("telegram_id = ?", toUserID).First(&toUser).Error; err != nil {
			return ErrUserNotFound
		}

		fromUser.Balance -= amount
		toUser.Balance += amount

		now := time.Now()
		fromTransaction := database.Transaction{
			UserID:     fromUser.ID,
			Amount:     -amount,
			Type:       "transfer",
			ToUserID:   &toUser.ID,
			ToUsername: toUser.Username,
			Timestamp:  now,
		}
		toTransaction := database.Transaction{
			UserID:     toUser.ID,
			Amount:     amount,
			Type:       "transfer",
			ToUserID:   &fromUser.ID,
			ToUsername: fromUser.Username,
			Timestamp:  now,
		}

		if err := tx.Save(&fromUser).Error; err != nil {
			return err
		}
		if err := tx.Save(&toUser).Error; err != nil {
			return err
		}
		if err := tx.Create(&fromTransaction).Error; err != nil {
			return err
		}
		if err := tx.Create(&toTransaction).Error; err != nil {
			return err
		}

		return nil
	})
}

func getTransactionHistory(userID int64) ([]database.Transaction, error) {
	var user database.User
	if err := database.DB.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
		return nil, ErrUserNotFound
	}

	var transactions []database.Transaction
	if err := database.DB.Where("user_id = ?", user.ID).Order("timestamp desc").Limit(10).Find(&transactions).Error; err != nil {
		return nil, err
	}

	return transactions, nil
}
