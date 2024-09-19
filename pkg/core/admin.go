// File: pkg/core/admin.go

package core

import (
	"errors"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"gorm.io/gorm"
)

const ADMIN_USERNAME = "notbuddy"

func IsAdmin(username string) bool {
	return username == ADMIN_USERNAME
}

func SetAdminStatus(targetUserID int64, isAdmin bool) error {
	return database.DB.Model(&database.User{}).Where("telegram_id = ?", targetUserID).Update("is_admin", isAdmin).Error
}

func AdminSetBalance(adminID, targetUserID int64, amount float64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var admin, targetUser database.User

		if err := tx.Where("telegram_id = ?", adminID).First(&admin).Error; err != nil {
			return ErrUserNotFound
		}
		if !admin.IsAdmin {
			return ErrUnauthorized
		}

		if err := tx.Where("telegram_id = ?", targetUserID).First(&targetUser).Error; err != nil {
			return ErrUserNotFound
		}

		oldBalance := targetUser.Balance
		targetUser.Balance = amount

		transaction := database.Transaction{
			UserID:    targetUser.ID,
			Amount:    amount - oldBalance,
			Type:      "admin_set_balance",
			Timestamp: time.Now(),
		}

		if err := tx.Save(&targetUser).Error; err != nil {
			return err
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		return nil
	})
}

type UserWithBalance struct {
	TelegramID   int64
	Username     string
	Balance      float64
	CurrencySign string
	CurrencyName string
}

func ListUsersWithBalances() ([]UserWithBalance, error) {
	var users []database.User
	err := database.DB.Find(&users).Error
	if err != nil {
		logger.Error("Failed to fetch users", "error", err)
		return nil, err
	}

	usersWithBalances := make([]UserWithBalance, 0, len(users))
	for _, user := range users {
		// Use GetUser to ensure Currency is loaded
		fullUser, err := GetUser(user.TelegramID)
		if err != nil {
			logger.Error("Failed to get full user data", "error", err, "telegramID", user.TelegramID)
			continue
		}

		usersWithBalances = append(usersWithBalances, UserWithBalance{
			TelegramID:   fullUser.TelegramID,
			Username:     fullUser.Username,
			Balance:      fullUser.Balance,
			CurrencySign: fullUser.Currency.Sign,
			CurrencyName: fullUser.Currency.Name,
		})

		logger.Debug("User currency details",
			"userID", fullUser.ID,
			"telegramID", fullUser.TelegramID,
			"currencyID", fullUser.CurrencyID,
			"currencySign", fullUser.Currency.Sign,
			"currencyName", fullUser.Currency.Name)
	}

	logger.Info("Listed users with balances", "userCount", len(usersWithBalances))
	return usersWithBalances, nil
}

func RemoveUser(username string) error {
	if username == "" {
		return errors.New("username cannot be empty")
	}

	result := database.DB.Where("username = ?", username).Delete(&database.User{})
	if result.Error != nil {
		logger.Error("Failed to remove user", "error", result.Error, "username", username)
		return result.Error
	}

	if result.RowsAffected == 0 {
		logger.Warn("No user found to remove", "username", username)
		return errors.New("user not found")
	}

	logger.Info("User removed", "username", username)
	return nil
}
