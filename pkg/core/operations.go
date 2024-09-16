// File: pkg/core/operations.go

package core

import (
	"github.com/fitz123/mcduck-wallet/pkg/database"
)

// OperationContext represents the context in which an operation is executed
type OperationContext interface {
	GetUserID() int64
	GetUsername() string
}

// GetBalance retrieves the balance for a user
func GetBalance(ctx OperationContext) (float64, error) {
	user, err := GetOrCreateUser(ctx.GetUserID(), ctx.GetUsername())
	if err != nil {
		return 0, err
	}
	return user.Balance, nil
}

// TransferMoney handles money transfer between users
func TransferMoney(ctx OperationContext, toUsername string, amount float64) error {
	fromUser, err := GetOrCreateUser(ctx.GetUserID(), ctx.GetUsername())
	if err != nil {
		return err
	}

	var toUser database.User
	if err := database.DB.Where("username = ?", toUsername).First(&toUser).Error; err != nil {
		return ErrUserNotFound
	}

	return transferMoney(fromUser.TelegramID, toUser.TelegramID, amount)
}

// GetTransactionHistory retrieves the transaction history for a user
func GetTransactionHistory(ctx OperationContext) ([]database.Transaction, error) {
	return getTransactionHistory(ctx.GetUserID())
}
