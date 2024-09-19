// File: pkg/core/operations.go

package core

import (
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"github.com/fitz123/mcduck-wallet/pkg/models"
)

// OperationContext represents the context in which an operation is executed
type OperationContext interface {
	GetUserID() int64
}

// GetBalance retrieves the balance for a user
func GetBalance(ctx OperationContext) (float64, error) {
	user, err := GetUser(ctx.GetUserID())
	if err != nil {
		logger.Error("Failed to get or create user", "error", err, "userID", ctx.GetUserID())
		return 0, err
	}
	logger.Info("Retrieved balance", "userID", ctx.GetUserID(), "balance", user.Balance)
	return user.Balance, nil
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
		logger.Error("Recipient not found", "error", err, "toUsername", toUsername)
		return ErrUserNotFound
	}

	logger.Debug("Initiating transfer", "fromUserID", fromUser.TelegramID, "toUserID", toUser.TelegramID, "amount", amount)
	err = transferMoney(fromUser.TelegramID, toUser.TelegramID, amount)
	if err != nil {
		logger.Error("Transfer failed", "error", err, "fromUserID", fromUser.TelegramID, "toUserID", toUser.TelegramID, "amount", amount)
		return err
	}
	logger.Info("Transfer successful", "fromUserID", fromUser.TelegramID, "toUserID", toUser.TelegramID, "amount", amount)
	return nil
}

// GetTransactionHistory retrieves the transaction history for a user
func GetTransactionHistory(ctx OperationContext) ([]models.TransactionJSON, error) {
	transactions, err := getTransactionHistory(ctx.GetUserID())
	if err != nil {
		logger.Error("Failed to get transaction history", "error", err, "userID", ctx.GetUserID())
		return nil, err
	}

	currency := database.GetDefaultCurrency()
	transactionsJSON := make([]models.TransactionJSON, len(transactions))
	for i, t := range transactions {
		transactionsJSON[i] = models.TransactionJSON{
			ID:         t.ID,
			Amount:     t.Amount,
			Type:       t.Type,
			ToUserID:   t.ToUserID,
			ToUsername: t.ToUsername,
			Timestamp:  t.Timestamp,
			Currency:   currency,
		}
	}

	logger.Info("Retrieved transaction history", "userID", ctx.GetUserID(), "transactionCount", len(transactionsJSON))
	return transactionsJSON, nil
}
