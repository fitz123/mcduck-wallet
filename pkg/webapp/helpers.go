// File: pkg/webapp/helpers.go

package webapp

import (
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
)

// FormatTransactionHistory formats the transaction history for webapp responses
func formatTransactionHistory(transactions []database.Transaction) []string {
	return messages.FormatTransactionHistory(transactions)
}
