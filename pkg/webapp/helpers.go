// File: pkg/webapp/helpers.go

package webapp

import (
	"fmt"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
)

// FormatTransactionHistory formats the transaction history for webapp responses
func formatTransactionHistory(transactions []database.Transaction) []string {
	if len(transactions) == 0 {
		return []string{messages.InfoNoTransactions}
	}

	formattedTransactions := make([]string, len(transactions))

	for i, t := range transactions {
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

		formattedTransactions[i] = fmt.Sprintf("%s - %s", t.Timestamp.Format("2006-01-02 15:04:05"), description)
	}

	return formattedTransactions
}
