// File: pkg/webapp/helpers.go

package webapp

import (
	"fmt"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
)

// TransactionResponse represents a single transaction in a format suitable for JSON response
type TransactionResponse struct {
	Timestamp   string  `json:"timestamp"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	ToUsername  string  `json:"to_username,omitempty"`
}

// FormatTransactionHistory formats the transaction history for webapp responses
func formatTransactionHistory(transactions []database.Transaction) []TransactionResponse {
	if len(transactions) == 0 {
		return []TransactionResponse{}
	}

	formattedTransactions := make([]TransactionResponse, len(transactions))

	for i, t := range transactions {
		var description string
		if t.Type == "transfer" {
			if t.Amount < 0 {
				description = fmt.Sprintf(messages.TransactionSent, -t.Amount, t.ToUsername)
			} else {
				description = fmt.Sprintf(messages.TransactionReceived, t.Amount, t.ToUsername)
			}
		} else {
			description = messages.TransactionDeposited
		}

		formattedTransactions[i] = TransactionResponse{
			Timestamp:   t.Timestamp.Format(time.RFC3339),
			Amount:      t.Amount,
			Description: description,
			ToUsername:  t.ToUsername,
		}
	}

	return formattedTransactions
}
