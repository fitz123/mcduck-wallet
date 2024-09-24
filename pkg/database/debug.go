package database

import (
	"fmt"

	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"gorm.io/gorm"
)

func DebugCurrencyAndUserData() {
	// Update user and transaction currencies
	if err := ResetDefaultCurrency(); err != nil {
		logger.Error("Failed to reset default currency", "error", err)
	}

	if err := UpdateUserCurrencies(); err != nil {
		logger.Error("Failed to update user currencies", "error", err)
	}

	if err := UpdateTransactionCurrencies(); err != nil {
		logger.Error("Failed to update transaction currencies", "error", err)
	}

	GetDefaultCurrency()
}

func GetDebugCurrencyAndUserData() {
	var currencies []Currency
	if err := DB.Find(&currencies).Error; err != nil {
		logger.Error("Failed to fetch currencies", "error", err)
		return
	}
	logger.Debug("Currencies in database", "count", len(currencies))
	for _, c := range currencies {
		logger.Debug("Currency details", "id", c.ID, "code", c.Code, "name", c.Name, "sign", c.Sign, "isDefault", c.IsDefault)
	}

	var users []User
	if err := DB.Find(&users).Error; err != nil {
		logger.Error("Failed to fetch users", "error", err)
		return
	}
	logger.Debug("Users in database", "count", len(users))
	for _, u := range users {
		logger.Debug("User details", "id", u.ID, "telegramID", u.TelegramID, "username", u.Username)
	}
}

func UpdateUserCurrencies() error {
	defaultCurrency, err := GetDefaultCurrency()
	if err != nil {
		logger.Error("Failed to get default currency", "error", err)
		return err
	}

	result := DB.Model(&User{}).Where("currency_id = ? OR currency_id IS NULL", 0).Update("currency_id", defaultCurrency.ID)
	if result.Error != nil {
		logger.Error("Failed to update user currencies", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.Warn("Updated user currencies", "usersUpdated", result.RowsAffected)
	} else {
		logger.Debug("All user currencies are set")
	}

	return nil
}

func UpdateTransactionCurrencies() error {
	defaultCurrency, err := GetDefaultCurrency()
	if err != nil {
		logger.Error("Failed to get default currency", "error", err)
		return err
	}

	result := DB.Model(&Transaction{}).Where("currency_id = ? OR currency_id IS NULL", 0).Update("currency_id", defaultCurrency.ID)
	if result.Error != nil {
		logger.Error("Failed to update transaction currencies", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.Warn("Updated transaction currencies", "transactionsUpdated", result.RowsAffected)
	} else {
		logger.Debug("All transaction currencies are set")
	}

	return nil
}

func ResetDefaultCurrency() error {
	var defaultCurrency Currency
	if err := DB.Where("is_default = ?", true).First(&defaultCurrency).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// No default currency found, set the first currency as default
			if err := DB.First(&defaultCurrency).Error; err != nil {
				logger.Error("No currencies found in database")
				return err
			}
			defaultCurrency.IsDefault = true
			if err := DB.Save(&defaultCurrency).Error; err != nil {
				logger.Error("Failed to set default currency", "error", err)
				return err
			}
			logger.Info("Set default currency", "id", defaultCurrency.ID)
		} else {
			logger.Error("Error checking for default currency:", err)
			return err
		}
	}
	return nil
}

func CleanupDatabase() error {
	// Drop any temporary tables that might have been created
	tempTables := []string{"transactions__temp", "users__temp", "accounts__temp"}
	for _, table := range tempTables {
		DB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	}
	return nil
}

func FinalMigrationFix() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Step 1: Ensure the transactions table has the correct structure
		if err := tx.Exec(`
            CREATE TABLE IF NOT EXISTS transactions_new (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                account_id INTEGER NOT NULL,
                amount REAL NOT NULL,
                type TEXT NOT NULL,
                to_user_id INTEGER,
                to_username TEXT,
                timestamp TIMESTAMP,
                created_at TIMESTAMP,
                updated_at TIMESTAMP,
                FOREIGN KEY (account_id) REFERENCES accounts(id)
            );

            INSERT OR IGNORE INTO transactions_new (id, account_id, amount, type, to_user_id, to_username, timestamp, created_at, updated_at)
            SELECT id, account_id, amount, type, to_user_id, to_username, timestamp, created_at, updated_at
            FROM transactions;

            DROP TABLE IF EXISTS transactions;
            ALTER TABLE transactions_new RENAME TO transactions;
        `).Error; err != nil {
			return fmt.Errorf("failed to recreate transactions table: %w", err)
		}

		// Step 2: Ensure all users have an account with the default currency
		var defaultCurrency struct {
			ID int
		}
		if err := tx.Table("currencies").Where("is_default = ?", true).First(&defaultCurrency).Error; err != nil {
			return fmt.Errorf("failed to get default currency: %w", err)
		}

		if err := tx.Exec(`
            INSERT OR IGNORE INTO accounts (user_id, currency_id, amount)
            SELECT id, ?, 0
            FROM users
            WHERE NOT EXISTS (
                SELECT 1 FROM accounts WHERE accounts.user_id = users.id
            )
        `, defaultCurrency.ID).Error; err != nil {
			return fmt.Errorf("failed to create default accounts: %w", err)
		}

		// Step 3: Update transactions to use valid account_id
		if err := tx.Exec(`
            UPDATE transactions
            SET account_id = (
                SELECT accounts.id
                FROM accounts
                JOIN users ON users.id = accounts.user_id
                WHERE users.telegram_id = transactions.to_user_id
                LIMIT 1
            )
            WHERE account_id IS NULL OR account_id NOT IN (SELECT id FROM accounts);
        `).Error; err != nil {
			return fmt.Errorf("failed to update transactions with account_id: %w", err)
		}

		// Step 4: Remove any transactions that still don't have a valid account_id
		if err := tx.Exec(`
            DELETE FROM transactions
            WHERE account_id IS NULL OR account_id NOT IN (SELECT id FROM accounts);
        `).Error; err != nil {
			return fmt.Errorf("failed to remove invalid transactions: %w", err)
		}

		return nil
	})
}

func CheckDatabaseState() {
	db := DB
	var userColumns, transactionColumns, accountColumns []struct {
		Name string
		Type string
	}

	db.Raw("PRAGMA table_info(users)").Scan(&userColumns)
	db.Raw("PRAGMA table_info(transactions)").Scan(&transactionColumns)
	db.Raw("PRAGMA table_info(accounts)").Scan(&accountColumns)

	fmt.Println("Users table structure:")
	for _, col := range userColumns {
		fmt.Printf("%s: %s\n", col.Name, col.Type)
	}

	fmt.Println("\nTransactions table structure:")
	for _, col := range transactionColumns {
		fmt.Printf("%s: %s\n", col.Name, col.Type)
	}

	fmt.Println("\nAccounts table structure:")
	for _, col := range accountColumns {
		fmt.Printf("%s: %s\n", col.Name, col.Type)
	}

	var userCount, transactionCount, accountCount int64
	db.Table("users").Count(&userCount)
	db.Table("transactions").Count(&transactionCount)
	db.Table("accounts").Count(&accountCount)

	fmt.Printf("\nData counts:\nUsers: %d\nTransactions: %d\nAccounts: %d\n", userCount, transactionCount, accountCount)
}
