package messages

import (
	"strings"
	"testing"
	"time"

	"github.com/fitz123/mcduck-wallet/internal/database"
)

func TestFormatTransactionHistory(t *testing.T) {
	t.Run("returns info message for empty transactions", func(t *testing.T) {
		result := FormatTransactionHistory([]database.Transaction{})

		if len(result) != 1 {
			t.Errorf("FormatTransactionHistory() count = %v, want 1", len(result))
		}
		if result[0] != InfoNoTransactions {
			t.Errorf("FormatTransactionHistory() = %v, want %v", result[0], InfoNoTransactions)
		}
	})

	t.Run("formats transfer_out transaction", func(t *testing.T) {
		tx := database.Transaction{
			Type:         "transfer_out",
			Amount:       -50.00,
			ToUsername:   "recipient",
			FromUsername: "sender",
			Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			BalanceAfter: 100.00,
			Balance: database.Balance{
				Currency: database.Currency{Sign: "$"},
			},
		}

		result := FormatTransactionHistory([]database.Transaction{tx})

		if len(result) != 1 {
			t.Fatalf("FormatTransactionHistory() count = %v, want 1", len(result))
		}
		if !strings.Contains(result[0], "Sent to") {
			t.Errorf("Result should contain 'Sent to': %v", result[0])
		}
		if !strings.Contains(result[0], "*recipient*") {
			t.Errorf("Result should contain '*recipient*': %v", result[0])
		}
		if !strings.Contains(result[0], "$50.00") {
			t.Errorf("Result should contain '$50.00': %v", result[0])
		}
		if !strings.Contains(result[0], "Balance: 100.00") {
			t.Errorf("Result should contain 'Balance: 100.00': %v", result[0])
		}
		if !strings.Contains(result[0], "2024-01-15 10:30") {
			t.Errorf("Result should contain timestamp: %v", result[0])
		}
	})

	t.Run("formats transfer_in transaction", func(t *testing.T) {
		tx := database.Transaction{
			Type:         "transfer_in",
			Amount:       75.50,
			FromUsername: "sender",
			ToUsername:   "recipient",
			Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			BalanceAfter: 175.50,
			Balance: database.Balance{
				Currency: database.Currency{Sign: "€"},
			},
		}

		result := FormatTransactionHistory([]database.Transaction{tx})

		if !strings.Contains(result[0], "Received from") {
			t.Errorf("Result should contain 'Received from': %v", result[0])
		}
		if !strings.Contains(result[0], "*sender*") {
			t.Errorf("Result should contain '*sender*': %v", result[0])
		}
		if !strings.Contains(result[0], "€75.50") {
			t.Errorf("Result should contain '€75.50': %v", result[0])
		}
	})

	t.Run("formats admin_set_balance transaction", func(t *testing.T) {
		tx := database.Transaction{
			Type:         "admin_set_balance",
			Amount:       500.00,
			FromUsername: "admin",
			Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			BalanceAfter: 500.00,
			Balance: database.Balance{
				Currency: database.Currency{Sign: "$"},
			},
		}

		result := FormatTransactionHistory([]database.Transaction{tx})

		if !strings.Contains(result[0], "Set by admin") {
			t.Errorf("Result should contain 'Set by admin': %v", result[0])
		}
		if !strings.Contains(result[0], "*admin*") {
			t.Errorf("Result should contain '*admin*': %v", result[0])
		}
	})

	t.Run("formats exchange_out transaction", func(t *testing.T) {
		tx := database.Transaction{
			Type:         "exchange_out",
			Amount:       -100.00,
			FromUsername: "user",
			ToUsername:   "user",
			Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			BalanceAfter: 0.00,
			Balance: database.Balance{
				Currency: database.Currency{Sign: "S", Code: "SHL"},
			},
		}

		result := FormatTransactionHistory([]database.Transaction{tx})

		if !strings.Contains(result[0], "Exchanged") {
			t.Errorf("Result should contain 'Exchanged': %v", result[0])
		}
		if !strings.Contains(result[0], "S100.00") {
			t.Errorf("Result should contain 'S100.00': %v", result[0])
		}
	})

	t.Run("formats exchange_in transaction", func(t *testing.T) {
		tx := database.Transaction{
			Type:         "exchange_in",
			Amount:       780.00,
			FromUsername: "user",
			ToUsername:   "user",
			Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			BalanceAfter: 780.00,
			Balance: database.Balance{
				Currency: database.Currency{Sign: "₽", Code: "RUB"},
			},
		}

		result := FormatTransactionHistory([]database.Transaction{tx})

		if !strings.Contains(result[0], "Received from exchange") {
			t.Errorf("Result should contain 'Received from exchange': %v", result[0])
		}
		if !strings.Contains(result[0], "₽780.00") {
			t.Errorf("Result should contain '₽780.00': %v", result[0])
		}
	})

	t.Run("handles unknown transaction type", func(t *testing.T) {
		tx := database.Transaction{
			Type:         "unknown_type",
			Amount:       10.00,
			Timestamp:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			BalanceAfter: 10.00,
			Balance: database.Balance{
				Currency: database.Currency{Sign: "$"},
			},
		}

		result := FormatTransactionHistory([]database.Transaction{tx})

		if !strings.Contains(result[0], "System Transaction") {
			t.Errorf("Result should contain 'System Transaction': %v", result[0])
		}
	})

	t.Run("formats multiple transactions", func(t *testing.T) {
		txs := []database.Transaction{
			{
				Type:         "transfer_in",
				Amount:       100,
				FromUsername: "alice",
				Timestamp:    time.Now(),
				BalanceAfter: 100,
				Balance:      database.Balance{Currency: database.Currency{Sign: "$"}},
			},
			{
				Type:         "transfer_out",
				Amount:       -50,
				ToUsername:   "bob",
				Timestamp:    time.Now(),
				BalanceAfter: 50,
				Balance:      database.Balance{Currency: database.Currency{Sign: "$"}},
			},
		}

		result := FormatTransactionHistory(txs)

		if len(result) != 2 {
			t.Errorf("FormatTransactionHistory() count = %v, want 2", len(result))
		}
	})

	t.Run("uses absolute value for amount", func(t *testing.T) {
		tx := database.Transaction{
			Type:         "transfer_out",
			Amount:       -99.99,
			ToUsername:   "recipient",
			Timestamp:    time.Now(),
			BalanceAfter: 0.01,
			Balance: database.Balance{
				Currency: database.Currency{Sign: "$"},
			},
		}

		result := FormatTransactionHistory([]database.Transaction{tx})

		if strings.Contains(result[0], "-99.99") {
			t.Errorf("Result should use absolute value, not negative: %v", result[0])
		}
		if !strings.Contains(result[0], "$99.99") {
			t.Errorf("Result should contain '$99.99': %v", result[0])
		}
	})
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"positive number", 42.5, 42.5},
		{"negative number", -42.5, 42.5},
		{"zero", 0, 0},
		{"small negative", -0.01, 0.01},
		{"large negative", -1000000.99, 1000000.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abs(tt.input)
			if result != tt.expected {
				t.Errorf("abs(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short username", "alice", "alice"},
		{"exactly 15 chars", "exactly15chars!", "exactly15chars!"},
		{"16 chars", "sixteencharname!", "sixteencharn..."},
		{"long username", "verylongusernamethatshouldbetru", "verylonguser..."},
		{"empty username", "", ""},
		{"14 chars", "fourteenchars!", "fourteenchars!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateUsername(tt.input)
			if result != tt.expected {
				t.Errorf("truncateUsername(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}

	t.Run("truncated length is max 15", func(t *testing.T) {
		result := truncateUsername("averylongusernamethatshouldbetruncated")
		if len(result) > 15 {
			t.Errorf("truncateUsername() len = %v, want <= 15", len(result))
		}
	})

	t.Run("truncated ends with ...", func(t *testing.T) {
		result := truncateUsername("averylongusernamethatshouldbetruncated")
		if !strings.HasSuffix(result, "...") {
			t.Errorf("truncateUsername() should end with '...': %v", result)
		}
	})
}

func TestMessageConstants(t *testing.T) {
	t.Run("InfoWelcome contains placeholder", func(t *testing.T) {
		if !strings.Contains(InfoWelcome, "%s") {
			t.Error("InfoWelcome should contain string placeholder")
		}
	})

	t.Run("InfoTransferSuccessful contains placeholders", func(t *testing.T) {
		if !strings.Contains(InfoTransferSuccessful, "%.2f") {
			t.Error("InfoTransferSuccessful should contain float placeholder")
		}
		if strings.Count(InfoTransferSuccessful, "%") < 3 {
			t.Error("InfoTransferSuccessful should have at least 3 placeholders")
		}
	})

	t.Run("UsageTransfer contains command format", func(t *testing.T) {
		if !strings.Contains(UsageTransfer, "/transfer") {
			t.Error("UsageTransfer should contain /transfer command")
		}
		if !strings.Contains(UsageTransfer, "@username") {
			t.Error("UsageTransfer should mention @username")
		}
	})
}
