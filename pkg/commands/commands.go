// File: pkg/commands/commands.go

package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
)

// CommandContext represents the context in which a command is executed
type CommandContext interface {
	GetUserID() int64
	GetUsername() string
	Reply(message string) error
}

// Balance handles the balance command
func Balance(ctx CommandContext) error {
	user, err := core.GetOrCreateUser(ctx.GetUserID(), ctx.GetUsername())
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	return ctx.Reply(fmt.Sprintf(messages.InfoCurrentBalance, user.Balance))
}

// Transfer handles the transfer command
func Transfer(ctx CommandContext, args ...string) error {
	if len(args) != 2 {
		return ctx.Reply(messages.UsageTransfer)
	}

	toUsername := strings.TrimPrefix(args[0], "@")
	var toUser database.User
	if err := database.DB.Where("username = ?", toUsername).First(&toUser).Error; err != nil {
		return ctx.Reply(messages.ErrRecipientNotFound)
	}

	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return ctx.Reply(messages.ErrInvalidAmount)
	}

	err = core.TransferMoney(ctx.GetUserID(), toUser.TelegramID, amount)
	if err != nil {
		return ctx.Reply("Transfer failed: " + err.Error())
	}

	return ctx.Reply(fmt.Sprintf(messages.InfoTransferSuccessful, amount, toUsername))
}

// History handles the transaction history command
func History(ctx CommandContext) error {
	transactions, err := core.GetTransactionHistory(ctx.GetUserID())
	if err != nil {
		return ctx.Reply("Error fetching transaction history: " + err.Error())
	}

	return ctx.Reply(BuildTransactionHistory(transactions))
}

// BuildTransactionHistory creates a string representation of the transaction history
func BuildTransactionHistory(transactions []database.Transaction) string {
	if len(transactions) == 0 {
		return messages.InfoNoTransactions
	}

	var response strings.Builder
	response.WriteString(messages.InfoRecentTransactions)
	for _, t := range transactions {
		var description string
		if t.Type == "transfer" {
			if t.Amount < 0 {
				description = fmt.Sprintf(messages.TransactionSent, -t.Amount, t.ToUsername)
			} else {
				description = fmt.Sprintf(messages.TransactionReceived, t.Amount, t.ToUsername)
			}
		} else {
			description = fmt.Sprintf(messages.TransactionDeposited, t.Amount)
		}
		response.WriteString(fmt.Sprintf("%s - %s\n", t.Timestamp.Format("2006-01-02 15:04:05"), description))
	}

	return response.String()
}
