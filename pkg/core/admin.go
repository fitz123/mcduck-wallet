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

func IsAdmin(telegramID int64) bool {
	username, err := GetUsernameByTelegramID(telegramID)
	if err != nil {
		return false
	}
	return username == ADMIN_USERNAME
}

func SetAdminStatus(targetUserID int64, isAdmin bool) error {
	return database.DB.Model(&database.User{}).Where("telegram_id = ?", targetUserID).Update("is_admin", isAdmin).Error
}

func AdminSetBalance(adminID, targetUserID int64, amount float64, currencyCode string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var admin, targetUser database.User

		// Check if admin exists and has admin rights
		if err := tx.Where("telegram_id = ?", adminID).First(&admin).Error; err != nil {
			return ErrUserNotFound
		}
		if !admin.IsAdmin {
			return ErrUnauthorized
		}

		// Get target user with their accounts
		if err := tx.Preload("Accounts.Currency").Where("telegram_id = ?", targetUserID).First(&targetUser).Error; err != nil {
			return ErrUserNotFound
		}

		// Find the account for the specified currency
		var targetAccount *database.Balance
		for i := range targetUser.Accounts {
			if targetUser.Accounts[i].Currency.Code == currencyCode {
				targetAccount = &targetUser.Accounts[i]
				break
			}
		}

		// If the account doesn't exist, create a new one
		if targetAccount == nil {
			currency, err := database.GetCurrencyByCode(currencyCode)
			if err != nil {
				return err
			}
			targetAccount = &database.Balance{
				UserID:     targetUser.ID,
				CurrencyID: currency.ID,
				Currency:   currency,
			}
			targetUser.Accounts = append(targetUser.Accounts, *targetAccount)
		}

		oldBalance := targetAccount.Amount
		targetAccount.Amount = amount // Update the account balance

		// Create a transaction record
		transaction := database.Transaction{
			UserID:       targetUser.ID,
			BalanceID:    targetAccount.ID,
			Balance:      *targetAccount,
			Amount:       amount - oldBalance,
			Type:         "admin_set_balance",
			FromUserID:   admin.ID,
			FromUsername: admin.Username,
			ToUserID:     targetUser.ID,
			ToUsername:   targetUser.Username,
			Timestamp:    time.Now(),
			BalanceAfter: amount,
		}

		// Save changes
		if err := tx.Save(targetAccount).Error; err != nil {
			return err
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		return nil
	})
}

type UserWithBalance struct {
	TelegramID int64
	Username   string
	Balances   map[string]float64 // key is currency code, value is balance
}

func ListUsersWithBalances() ([]UserWithBalance, error) {
	var users []database.User
	err := database.DB.Preload("Accounts.Currency").Find(&users).Error
	if err != nil {
		logger.Error("Failed to fetch users", "error", err)
		return nil, err
	}

	usersWithBalances := make([]UserWithBalance, 0, len(users))
	for _, user := range users {
		userWithBalance := UserWithBalance{
			TelegramID: user.TelegramID,
			Username:   user.Username,
			Balances:   make(map[string]float64),
		}

		for _, account := range user.Accounts {
			userWithBalance.Balances[account.Currency.Code] = account.Amount
		}

		usersWithBalances = append(usersWithBalances, userWithBalance)
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

func AddUser(adminID int64, telegramID int64, username string) error {
	if !IsAdmin(adminID) {
		return ErrUnauthorized
	}

	_, err := GetUser(telegramID)
	if err == nil {
		return errors.New("user already exists")
	}

	newUser, err := CreateUser(telegramID, username)
	if err != nil {
		logger.Error("Failed to create new user", "error", err)
		return err
	}

	logger.Info("New user added by admin", "adminID", adminID, "newUserID", newUser.ID)
	return nil
}

func AddCurrency(adminID int64, code string, name string, sign string) error {
	if !IsAdmin(adminID) {
		return ErrUnauthorized
	}

	_, err := database.GetCurrencyByCode(code)
	if err == nil {
		return errors.New("currency already exists")
	}

	newCurrency := database.Currency{
		Code: code,
		Name: name,
		Sign: sign,
	}

	if err := database.DB.Create(&newCurrency).Error; err != nil {
		logger.Error("Failed to create new currency", "error", err)
		return err
	}

	logger.Info("New currency added by admin", "adminID", adminID, "currencyCode", code)
	return nil
}

func SetDefaultCurrency(adminID int64, code string) error {
	if !IsAdmin(adminID) {
		return ErrUnauthorized
	}

	currency, err := database.GetCurrencyByCode(code)
	if err != nil {
		return err
	}

	// Unset the current default currency
	if err := database.DB.Model(&database.Currency{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		logger.Error("Failed to unset current default currency", "error", err)
		return err
	}

	// Set the new default currency
	currency.IsDefault = true
	if err := database.DB.Save(&currency).Error; err != nil {
		logger.Error("Failed to set new default currency", "error", err)
		return err
	}

	logger.Info("Default currency changed by admin", "adminID", adminID, "newDefaultCurrency", code)
	return nil
}
