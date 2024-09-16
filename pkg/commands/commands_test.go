package commands

import (
	"testing"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
)

func TestBuildTransactionHistory(t *testing.T) {
	transactions := []database.Transaction{
		{
			UserID:    1,
			Amount:    100,
			Type:      "deposit",
			Timestamp: time.Date(2024, 9, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			UserID:     1,
			Amount:     -50,
			Type:       "transfer",
			ToUsername: "user2",
			Timestamp:  time.Date(2024, 9, 15, 11, 0, 0, 0, time.UTC),
		},
		{
			UserID:     1,
			Amount:     30,
			Type:       "transfer",
			ToUsername: "user3",
			Timestamp:  time.Date(2024, 9, 15, 12, 0, 0, 0, time.UTC),
		},
	}

	history := BuildTransactionHistory(transactions)

	expected := `Your recent transactions:

2024-09-15 10:00:00 - Deposited ¤100.00
2024-09-15 11:00:00 - Sent ¤50.00 to user2
2024-09-15 12:00:00 - Received ¤30.00 from user3
`

	if history != expected {
		t.Errorf("BuildTransactionHistory returned unexpected result.\nExpected:\n%s\nGot:\n%s", expected, history)
	}
}

func TestBuildTransactionHistoryEmpty(t *testing.T) {
	history := BuildTransactionHistory([]database.Transaction{})

	if history != "No transactions found." {
		t.Errorf("BuildTransactionHistory with empty slice should return 'No transactions found.', got: %s", history)
	}
}
