// File: pkg/core/operations.go

package core

import (
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
)

// OperationContext represents the context in which an operation is executed
type OperationContext interface {
	GetUserID() int64
}

type BalanceWithCurrency struct {
	Amount   float64
	Currency database.Currency
}

func GetBalance(ctx OperationContext) ([]database.Balance, error) {
	user, err := GetUser(ctx.GetUserID())
	if err != nil {
		logger.Error("Failed to get user", "error", err, "userID", ctx.GetUserID())
		return nil, err
	}

	return user.Accounts, nil
}

// TransferMoney is the public API for transferring money between users
func TransferMoney(ctx OperationContext, toUsername string, amount float64, currencyCode string) error {
	fromUser, err := GetUser(ctx.GetUserID())
	if err != nil {
		return err
	}

	toUser, err := GetUserByUsername(toUsername)
	if err != nil {
		return err
	}

	return transferMoney(fromUser, toUser, amount, currencyCode)
}

func GetTransactionHistory(ctx OperationContext) ([]database.Transaction, error) {
	transactions, err := getTransactionHistory(ctx.GetUserID())
	if err != nil {
		logger.Error("Failed to get transaction history", "error", err, "userID", ctx.GetUserID())
		return nil, err
	}

	logger.Info("Retrieved transaction history", "userID", ctx.GetUserID(), "transactionCount", len(transactions))
	return transactions, nil
}
