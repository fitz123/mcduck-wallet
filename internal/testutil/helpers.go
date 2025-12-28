package testutil

import (
	"testing"

	"github.com/fitz123/mcduck-wallet/internal/database"
)

// SetupTestDB creates a temporary SQLite database for testing
// The database is automatically cleaned up when the test finishes
func SetupTestDB(t *testing.T) *database.DB {
	t.Helper()
	testDB, err := database.NewTest()
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		testDB.Cleanup()
	})
	return testDB.DB
}

// CreateTestUser creates a user in the test database
func CreateTestUser(t *testing.T, db *database.DB, telegramID int64, username string, isAdmin bool) *database.User {
	t.Helper()
	user := &database.User{
		TelegramID: telegramID,
		Username:   username,
		IsAdmin:    isAdmin,
	}
	if err := db.Conn.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

// CreateTestCurrency creates a currency in the test database
func CreateTestCurrency(t *testing.T, db *database.DB, code, name, sign string, isDefault bool) *database.Currency {
	t.Helper()
	currency := &database.Currency{
		Code:      code,
		Name:      name,
		Sign:      sign,
		IsDefault: isDefault,
	}
	if err := db.Conn.Create(currency).Error; err != nil {
		t.Fatalf("Failed to create test currency: %v", err)
	}
	return currency
}

// CreateTestBalance creates a balance for a user in the test database
func CreateTestBalance(t *testing.T, db *database.DB, userID uint, currencyID uint, amount float64) *database.Balance {
	t.Helper()
	balance := &database.Balance{
		UserID:     userID,
		CurrencyID: currencyID,
		Amount:     amount,
	}
	if err := db.Conn.Create(balance).Error; err != nil {
		t.Fatalf("Failed to create test balance: %v", err)
	}
	return balance
}

// CreateTestTransaction creates a transaction in the test database
func CreateTestTransaction(t *testing.T, db *database.DB, tx *database.Transaction) *database.Transaction {
	t.Helper()
	if err := db.Conn.Create(tx).Error; err != nil {
		t.Fatalf("Failed to create test transaction: %v", err)
	}
	return tx
}

// GetDefaultCurrency returns the default currency from the test database
func GetDefaultCurrency(t *testing.T, db *database.DB) *database.Currency {
	t.Helper()
	var currency database.Currency
	if err := db.Conn.Where("is_default = ?", true).First(&currency).Error; err != nil {
		t.Fatalf("Failed to get default currency: %v", err)
	}
	return &currency
}

// CreateTestCurrencyWithRate creates a currency with IsReal and FixedRate fields
func CreateTestCurrencyWithRate(t *testing.T, db *database.DB, code, name, sign string, isReal bool, fixedRate float64) *database.Currency {
	t.Helper()
	currency := &database.Currency{
		Code:      code,
		Name:      name,
		Sign:      sign,
		IsReal:    isReal,
		FixedRate: fixedRate,
	}
	if err := db.Conn.Create(currency).Error; err != nil {
		t.Fatalf("Failed to create test currency: %v", err)
	}
	return currency
}
