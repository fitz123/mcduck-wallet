package bot

import (
	"fmt"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
)

func formatTransactionHistory(transactions []database.Transaction) string {
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
