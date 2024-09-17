package bot

import (
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
)

func formatTransactionHistory(transactions []database.Transaction) string {
	formattedTransactions := messages.FormatTransactionHistory(transactions)
	return messages.InfoRecentTransactions + strings.Join(formattedTransactions, "\n")
}
