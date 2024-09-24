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
	ErrCurrencyMismatch     = errors.New(messages.ErrCurrencyMismatch)
	ErrCurrencyNotExist     = errors.New(messages.ErrCurrencyNotExist)
)

func GetUser(telegramID int64) (*database.User, error) {
	var user database.User
	result := database.DB.Preload("Accounts.Currency").Where("telegram_id = ?", telegramID).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		logger.Error("Failed to get user", "error", result.Error, "telegramID", telegramID)
		return nil, result.Error
	}

	logger.Debug(
		"User fetched",
		"telegramID", user.TelegramID,
		"username", user.Username,
		"Accounts", user.Accounts,
	)

	return &user, nil
}

func CreateUser(telegramID int64, username string) (*database.User, error) {
	defaultCurrency, err := database.GetDefaultCurrency()
	if err != nil {
		return nil, err
	}

	user := database.User{
		TelegramID: telegramID,
		Username:   username,
		Accounts: []database.Balance{
			{
				Amount:     0,
				CurrencyID: defaultCurrency.ID,
				Currency:   defaultCurrency,
			},
		},
	}

	if err := database.DB.Create(&user).Error; err != nil {
		logger.Error("Failed to create new user", "error", err, "telegramID", telegramID)
		return nil, err
	}

	logger.Info("New user created", "telegramID", telegramID, "username", username)
	return &user, nil
}

func UpdateUsername(telegramID int64, username string) error {
	result := database.DB.Model(&database.User{}).Where("telegram_id = ?", telegramID).Update("username", username)
	if result.Error != nil {
		logger.Error("Failed to update username", "error", result.Error, "telegramID", telegramID)
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	logger.Info("Username updated", "telegramID", telegramID, "newUsername", username)
	return nil
}

func transferMoney(fromUser, toUser *database.User, amount float64, currencyCode string) error {
	var fromBalance, toBalance *database.Balance

	// Find the correct balance for each user
	for i := range fromUser.Accounts {
		if fromUser.Accounts[i].Currency.Code == currencyCode {
			fromBalance = &fromUser.Accounts[i]
			break
		}
	}
	for i := range toUser.Accounts {
		if toUser.Accounts[i].Currency.Code == currencyCode {
			toBalance = &toUser.Accounts[i]
			break
		}
	}

	// Check for errors
	if fromUser.ID == toUser.ID {
		return ErrCannotTransferToSelf
	}
	if fromBalance == nil || toBalance == nil {
		return ErrCurrencyNotExist
	}
	if fromBalance.Amount < amount {
		return ErrInsufficientBalance
	}
	if amount < 0 {
		return ErrNegativeAmount
	}
	if fromBalance.CurrencyID != toBalance.CurrencyID {
		return ErrCurrencyMismatch
	}

	// Perform the transfer
	fromBalance.Amount -= amount
	toBalance.Amount += amount

	// Create transactions
	now := time.Now()
	fromTransaction := database.Transaction{
		UserID:       fromUser.ID,
		BalanceID:    fromBalance.ID,
		Balance:      *fromBalance,
		Amount:       -amount,
		Type:         "transfer_out",
		FromUserID:   fromUser.ID,
		FromUsername: fromUser.Username,
		ToUserID:     toUser.ID,
		ToUsername:   toUser.Username,
		Timestamp:    now,
		BalanceAfter: fromBalance.Amount,
	}
	toTransaction := database.Transaction{
		UserID:       toUser.ID,
		BalanceID:    toBalance.ID,
		Balance:      *toBalance,
		Amount:       amount,
		Type:         "transfer_in",
		FromUserID:   fromUser.ID,
		FromUsername: fromUser.Username,
		ToUserID:     toUser.ID,
		ToUsername:   toUser.Username,
		Timestamp:    now,
		BalanceAfter: toBalance.Amount,
	}

	// Save everything to the database
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(fromBalance).Error; err != nil {
			return err
		}
		if err := tx.Save(toBalance).Error; err != nil {
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
	if err := database.DB.Where("user_id = ?", user.ID).
		Preload("Balance.Currency"). // Preload the Currency information
		Order("timestamp desc").
		Limit(10).
		Find(&transactions).Error; err != nil {
		logger.Error("Failed to get transaction history", "error", err, "userID", userID)
		return nil, err
	}

	logger.Info("Retrieved transaction history", "userID", userID, "transactionCount", len(transactions))
	return transactions, nil
}

func GetUserByUsername(username string) (*database.User, error) {
	var user database.User
	result := database.DB.Preload("Accounts.Currency").Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, result.Error
	}
	return &user, nil
}

func GetUsernameByTelegramID(telegramID int64) (string, error) {
	var user database.User
	result := database.DB.Where("telegram_id = ?", telegramID).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", ErrUserNotFound
		}
		return "", result.Error
	}
	return user.Username, nil
}
