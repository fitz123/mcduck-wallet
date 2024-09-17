// File: pkg/messages/formatter.go

package messages

import (
	"fmt"

	"github.com/fitz123/mcduck-wallet/pkg/database"
)

// FormatTransactionHistory formats the transaction history for both bot and webapp responses
func FormatTransactionHistory(transactions []database.Transaction) []string {
	if len(transactions) == 0 {
		return []string{InfoNoTransactions}
	}

	formattedTransactions := make([]string, len(transactions))

	for i, t := range transactions {
		var description string
		if t.Type == "transfer" {
			if t.Amount < 0 {
				description = fmt.Sprintf(TransactionSent, -t.Amount, t.ToUsername)
			} else {
				description = fmt.Sprintf(TransactionReceived, t.Amount, t.ToUsername)
			}
		} else {
			description = fmt.Sprintf(TransactionDeposited, t.Amount)
		}

		formattedTransactions[i] = fmt.Sprintf("%s - %s", t.Timestamp.Format("2006-01-02 15:04:05"), description)
	}

	return formattedTransactions
}
