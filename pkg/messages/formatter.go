// File: pkg/messages/formatter.go

package messages

import (
	"fmt"

	"github.com/fitz123/mcduck-wallet/pkg/models"
)

// FormatTransactionHistory formats the transaction history for bot
func FormatTransactionHistory(transactions []models.TransactionJSON) []string {
	if len(transactions) == 0 {
		return []string{InfoNoTransactions}
	}

	formattedTransactions := make([]string, len(transactions))

	for i, t := range transactions {
		var description string

		if t.Type == "transfer" {
			if t.Amount < 0 {
				description = fmt.Sprintf("Send to *%s*", truncateUsername(t.ToUsername))
			} else {
				description = fmt.Sprintf("Received from *%s*", truncateUsername(t.ToUsername))
			}
		} else {
			description = "System Transaction"
		}

		formattedTransactions[i] = fmt.Sprintf("%s - %s %s%.2f",
			t.Timestamp.Format("2006-01-02 15:04:05"),
			description,
			t.CurrencySign, abs(t.Amount),
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
	maxLength := 18
	if len(username) > maxLength {
		return username[:maxLength-3] + "..."
	}
	return username
}
