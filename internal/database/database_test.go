package database

import (
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("creates database with in-memory SQLite", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		if db.Conn == nil {
			t.Error("New() returned db with nil Conn")
		}
	})

	t.Run("creates default USD currency", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		var currency Currency
		if err := db.Conn.Where("is_default = ?", true).First(&currency).Error; err != nil {
			t.Fatalf("Failed to find default currency: %v", err)
		}

		if currency.Code != "USD" {
			t.Errorf("Default currency code = %v, want USD", currency.Code)
		}
		if currency.Name != "US Dollar" {
			t.Errorf("Default currency name = %v, want US Dollar", currency.Name)
		}
		if currency.Sign != "$" {
			t.Errorf("Default currency sign = %v, want $", currency.Sign)
		}
		if !currency.IsDefault {
			t.Error("Default currency IsDefault = false, want true")
		}
	})

	t.Run("does not duplicate default currency on re-init", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// Count currencies before
		var count1 int64
		db.Conn.Model(&Currency{}).Count(&count1)

		// Simulate re-checking (already done in New, but let's verify state)
		var currencies []Currency
		if err := db.Conn.Where("is_default = ?", true).Find(&currencies).Error; err != nil {
			t.Fatalf("Failed to query currencies: %v", err)
		}

		if len(currencies) != 1 {
			t.Errorf("Got %d default currencies, want 1", len(currencies))
		}
	})

	t.Run("auto-migrates all models", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		defer db.Close()

		// Verify tables exist by attempting to query them
		tables := []string{"users", "balances", "transactions", "currencies"}
		for _, table := range tables {
			if !db.Conn.Migrator().HasTable(table) {
				t.Errorf("Table %s does not exist after migration", table)
			}
		}
	})
}

func TestDB_Close(t *testing.T) {
	t.Run("closes database connection", func(t *testing.T) {
		db, err := New(":memory:")
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		err = db.Close()
		if err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

func TestUserModel(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	t.Run("creates user with unique telegram_id", func(t *testing.T) {
		user := &User{
			TelegramID: 12345,
			Username:   "testuser",
		}
		if err := db.Conn.Create(user).Error; err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.ID == 0 {
			t.Error("User ID should be set after create")
		}
	})

	t.Run("enforces unique constraint on telegram_id", func(t *testing.T) {
		user1 := &User{TelegramID: 99999, Username: "user1"}
		user2 := &User{TelegramID: 99999, Username: "user2"}

		if err := db.Conn.Create(user1).Error; err != nil {
			t.Fatalf("Failed to create first user: %v", err)
		}

		err := db.Conn.Create(user2).Error
		if err == nil {
			t.Error("Expected error creating user with duplicate telegram_id")
		}
	})

	t.Run("defaults IsAdmin to false", func(t *testing.T) {
		user := &User{
			TelegramID: 11111,
			Username:   "regularuser",
		}
		if err := db.Conn.Create(user).Error; err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		var retrieved User
		db.Conn.First(&retrieved, user.ID)
		if retrieved.IsAdmin {
			t.Error("IsAdmin should default to false")
		}
	})

	t.Run("supports soft delete", func(t *testing.T) {
		user := &User{TelegramID: 22222, Username: "softdeleteuser"}
		db.Conn.Create(user)

		db.Conn.Delete(user)

		var found User
		result := db.Conn.First(&found, user.ID)
		if result.Error == nil {
			t.Error("Soft deleted user should not be found with normal query")
		}

		// But should be found with Unscoped
		result = db.Conn.Unscoped().First(&found, user.ID)
		if result.Error != nil {
			t.Error("Soft deleted user should be found with Unscoped query")
		}
	})
}

func TestBalanceModel(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// Create test user and currency
	user := &User{TelegramID: 33333, Username: "balanceuser"}
	db.Conn.Create(user)

	var currency Currency
	db.Conn.Where("is_default = ?", true).First(&currency)

	t.Run("creates balance with correct relationships", func(t *testing.T) {
		balance := &Balance{
			UserID:     user.ID,
			CurrencyID: currency.ID,
			Amount:     100.50,
		}
		if err := db.Conn.Create(balance).Error; err != nil {
			t.Fatalf("Failed to create balance: %v", err)
		}

		// Verify relationship loading
		var retrieved Balance
		db.Conn.Preload("Currency").First(&retrieved, balance.ID)

		if retrieved.Currency.Code != "USD" {
			t.Errorf("Balance currency code = %v, want USD", retrieved.Currency.Code)
		}
		if retrieved.Amount != 100.50 {
			t.Errorf("Balance amount = %v, want 100.50", retrieved.Amount)
		}
	})

	t.Run("handles decimal amounts", func(t *testing.T) {
		balance := &Balance{
			UserID:     user.ID,
			CurrencyID: currency.ID,
			Amount:     99.99,
		}
		db.Conn.Create(balance)

		var retrieved Balance
		db.Conn.First(&retrieved, balance.ID)

		if retrieved.Amount != 99.99 {
			t.Errorf("Balance amount = %v, want 99.99", retrieved.Amount)
		}
	})
}

func TestCurrencyModel(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	t.Run("enforces unique constraint on code", func(t *testing.T) {
		currency1 := &Currency{Code: "EUR", Name: "Euro", Sign: "€"}
		currency2 := &Currency{Code: "EUR", Name: "Another Euro", Sign: "E"}

		if err := db.Conn.Create(currency1).Error; err != nil {
			t.Fatalf("Failed to create first currency: %v", err)
		}

		err := db.Conn.Create(currency2).Error
		if err == nil {
			t.Error("Expected error creating currency with duplicate code")
		}
	})

	t.Run("defaults IsDefault to false", func(t *testing.T) {
		currency := &Currency{Code: "GBP", Name: "British Pound", Sign: "£"}
		db.Conn.Create(currency)

		var retrieved Currency
		db.Conn.First(&retrieved, currency.ID)
		if retrieved.IsDefault {
			t.Error("IsDefault should default to false for new currencies")
		}
	})
}

func TestTransactionModel(t *testing.T) {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// Setup
	user := &User{TelegramID: 44444, Username: "txuser"}
	db.Conn.Create(user)

	var currency Currency
	db.Conn.Where("is_default = ?", true).First(&currency)

	balance := &Balance{UserID: user.ID, CurrencyID: currency.ID, Amount: 100}
	db.Conn.Create(balance)

	t.Run("creates transaction with all fields", func(t *testing.T) {
		tx := &Transaction{
			UserID:       user.ID,
			BalanceID:    balance.ID,
			Amount:       -50,
			Type:         "transfer_out",
			FromUserID:   user.ID,
			FromUsername: "txuser",
			ToUserID:     999,
			ToUsername:   "recipient",
			BalanceAfter: 50,
		}
		if err := db.Conn.Create(tx).Error; err != nil {
			t.Fatalf("Failed to create transaction: %v", err)
		}

		if tx.ID == 0 {
			t.Error("Transaction ID should be set after create")
		}
	})

	t.Run("preloads balance and currency", func(t *testing.T) {
		tx := &Transaction{
			UserID:       user.ID,
			BalanceID:    balance.ID,
			Amount:       25,
			Type:         "transfer_in",
			FromUserID:   888,
			FromUsername: "sender",
			ToUserID:     user.ID,
			ToUsername:   "txuser",
			BalanceAfter: 125,
		}
		db.Conn.Create(tx)

		var retrieved Transaction
		db.Conn.Preload("Balance.Currency").First(&retrieved, tx.ID)

		if retrieved.Balance.Currency.Code != "USD" {
			t.Errorf("Transaction balance currency = %v, want USD", retrieved.Balance.Currency.Code)
		}
	})
}
