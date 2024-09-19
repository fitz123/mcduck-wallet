// File: pkg/core/operations.go

package core

import (
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
	"github.com/fitz123/mcduck-wallet/pkg/models"
)

// OperationContext represents the context in which an operation is executed
type OperationContext interface {
	GetUserID() int64
}

// GetBalance retrieves the balance for a user
func GetBalance(ctx OperationContext) (float64, database.Currency, error) {
	user, err := GetUser(ctx.GetUserID())
	if err != nil {
		logger.Error("Failed to get user", "error", err, "userID", ctx.GetUserID())
		return 0, database.Currency{}, err
	}

	var currency database.Currency
	if err := database.DB.First(&currency, user.CurrencyID).Error; err != nil {
		logger.Error("Failed to get currency", "error", err, "currencyID", user.CurrencyID)
		return 0, database.Currency{}, err
	}

	logger.Info("Retrieved balance", "userID", ctx.GetUserID(), "balance", user.Balance, "currency", currency.Code)
	return user.Balance, currency, nil
}

// TransferMoney handles money transfer between users
func TransferMoney(ctx OperationContext, toUsername string, amount float64) error {
	fromUser, err := GetUser(ctx.GetUserID())
	if err != nil {
		logger.Error("Failed to get or create sender", "error", err, "userID", ctx.GetUserID())
		return err
	}

	toUsername = strings.TrimPrefix(toUsername, "@")
	var toUser database.User
	if err := database.DB.Where("username = ?", toUsername).First(&toUser).Error; err != nil {
		logger.Error(messages.ErrUserNotFound, "error", err, "toUsername", toUsername)
		return ErrUserNotFound
	}

	// Ensure both users have the same currency
	if fromUser.CurrencyID != toUser.CurrencyID {
		logger.Error(messages.ErrCurrencyMismatch, "fromUserID", fromUser.TelegramID, "toUserID", toUser.TelegramID)
		return ErrCurrencyMismatch
	}

	var currency database.Currency
	if err := database.DB.First(&currency, fromUser.CurrencyID).Error; err != nil {
		logger.Error("Failed to get currency", "error", err, "currencyID", fromUser.CurrencyID)
		return err
	}

	logger.Debug("Initiating transfer", "fromUserID", fromUser.TelegramID, "toUserID", toUser.TelegramID, "amount", amount)
	err = transferMoney(fromUser.TelegramID, toUser.TelegramID, amount)
	if err != nil {
		logger.Error("Transfer failed", "error", err, "fromUserID", fromUser.TelegramID, "toUserID", toUser.TelegramID, "amount", amount)
		return err
	}
	logger.Info("Transfer successful",
		"fromUserID", fromUser.TelegramID,
		"toUserID", toUser.TelegramID,
		"amount", amount,
		"currency", currency.Code)
	return nil
}

func GetTransactionHistory(ctx OperationContext) ([]models.TransactionJSON, error) {
	transactions, err := getTransactionHistory(ctx.GetUserID())
	if err != nil {
		logger.Error("Failed to get transaction history", "error", err, "userID", ctx.GetUserID())
		return nil, err
	}

	transactionsJSON := make([]models.TransactionJSON, len(transactions))
	for i, t := range transactions {
		var currency database.Currency
		if err := database.DB.First(&currency, t.CurrencyID).Error; err != nil {
			logger.Error("Failed to get currency for transaction", "error", err, "transactionID", t.ID, "currencyID", t.CurrencyID)
			return nil, err
		}

		transactionsJSON[i] = models.TransactionJSON{
			ID:           t.ID,
			Amount:       t.Amount,
			Type:         t.Type,
			ToUserID:     t.ToUserID,
			ToUsername:   t.ToUsername,
			Timestamp:    t.Timestamp,
			CurrencyID:   currency.ID,
			CurrencySign: currency.Sign,
			CurrencyName: currency.Name,
		}
	}

	logger.Info("Retrieved transaction history", "userID", ctx.GetUserID(), "transactionCount", len(transactionsJSON))
	return transactionsJSON, nil
}
