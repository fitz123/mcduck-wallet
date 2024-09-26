// File: pkg/messages/formatter.go

package messages

import (
	"fmt"

	"github.com/fitz123/mcduck-wallet/internal/database"
)

// FormatTransactionHistory formats the transaction history for bot
func FormatTransactionHistory(transactions []database.Transaction) []string {
	if len(transactions) == 0 {
		return []string{InfoNoTransactions}
	}

	formattedTransactions := make([]string, len(transactions))
	for i, t := range transactions {
		var description string
		var otherParty string

		switch t.Type {
		case "transfer_out":
			description = "Sent to"
			otherParty = truncateUsername(t.ToUsername)
		case "transfer_in":
			description = "Received from"
			otherParty = truncateUsername(t.FromUsername)
		case "admin_set_balance":
			description = "Set by admin"
			otherParty = truncateUsername(t.FromUsername)
		default:
			description = "System Transaction"
			otherParty = ""
		}

		if otherParty != "" {
			description = fmt.Sprintf("%s *%s*", description, otherParty)
		}

		formattedTransactions[i] = fmt.Sprintf("%s - %s %s%.0f (Balance: %.0f)",
			t.Timestamp.Format("2006-01-02 15:04"),
			description,
			t.Balance.Currency.Sign, abs(t.Amount),
			t.BalanceAfter,
		)
	}

	return formattedTransactions
}

// abs returns the absolute value of x
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// truncateUsername shortens long usernames and adds an ellipsis
func truncateUsername(username string) string {
	maxLength := 15
	if len(username) > maxLength {
		return username[:maxLength-3] + "..."
	}
	return username
}
