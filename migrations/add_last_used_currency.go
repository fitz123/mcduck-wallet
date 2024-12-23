// File: migrations/add_last_used_currency.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	fmt.Printf("Current working directory: %s\n", cwd)

	// Try different possible database paths
	possiblePaths := []string{
		"mcduck_wallet.db",
		"./mcduck_wallet.db",
		"../mcduck_wallet.db",
	}

	var dbPath string
	for _, path := range possiblePaths {
		absPath := filepath.Join(cwd, path)
		fmt.Printf("Checking database path: %s\n", absPath)
		if _, err := os.Stat(absPath); err == nil {
			dbPath = absPath
			fmt.Printf("Found database at: %s\n", dbPath)
			break
		}
	}

	if dbPath == "" {
		log.Fatal("Could not find database file in any of the checked locations")
	}

	fmt.Printf("Opening database at: %s\n", dbPath)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test database connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Successfully connected to database")

	// Create a backup of the database
	backupPath := dbPath + ".backup"
	fmt.Printf("Creating backup at: %s\n", backupPath)
	err = copyFile(dbPath, backupPath)
	if err != nil {
		log.Fatalf("Failed to create backup: %v", err)
	}
	fmt.Println("Successfully created backup")

	// Start transaction
	fmt.Println("Starting transaction...")
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}

	// Check if users table exists
	var tableExists bool
	err = tx.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM sqlite_master 
		WHERE type='table' AND name='users'
	`).Scan(&tableExists)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Failed to check if users table exists: %v", err)
	}
	fmt.Printf("Users table exists: %v\n", tableExists)

	if !tableExists {
		tx.Rollback()
		log.Fatal("Users table does not exist")
	}

	// Check if the column already exists
	var columnExists bool
	err = tx.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('users') 
		WHERE name = 'last_used_currency_id'
	`).Scan(&columnExists)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Failed to check column existence: %v", err)
	}
	fmt.Printf("LastUsedCurrencyID column exists: %v\n", columnExists)

	if !columnExists {
		fmt.Println("Adding last_used_currency_id column...")
		// Add the new column
		_, err = tx.Exec(`ALTER TABLE users ADD COLUMN last_used_currency_id INTEGER DEFAULT NULL`)
		if err != nil {
			tx.Rollback()
			log.Fatalf("Failed to add column: %v", err)
		}
		fmt.Println("Successfully added column")

		// Optional: Set default values based on users' latest transactions
		fmt.Println("Setting default values from latest transactions...")
		_, err = tx.Exec(`
			WITH LastTransactions AS (
				SELECT 
					u.id as user_id,
					b.currency_id,
					ROW_NUMBER() OVER (PARTITION BY u.id ORDER BY t.created_at DESC) as rn
				FROM users u
				JOIN transactions t ON t.user_id = u.id
				JOIN balances b ON t.balance_id = b.id
				WHERE t.deleted_at IS NULL
			)
			UPDATE users
			SET last_used_currency_id = (
				SELECT currency_id
				FROM LastTransactions
				WHERE rn = 1
				AND user_id = users.id
			)
			WHERE EXISTS (
				SELECT 1
				FROM LastTransactions
				WHERE rn = 1
				AND user_id = users.id
			)
		`)
		if err != nil {
			tx.Rollback()
			log.Fatalf("Failed to set default values: %v", err)
		}
		fmt.Println("Successfully set default values")
	} else {
		fmt.Println("Column 'last_used_currency_id' already exists, skipping migration")
	}

	// Commit transaction
	fmt.Println("Committing transaction...")
	err = tx.Commit()
	if err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println("Migration completed successfully!")

	// Print final table structure
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		log.Printf("Failed to get table structure: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nCurrent users table structure:")
	for rows.Next() {
		var cid, notnull, pk int
		var name, type_ string
		var dflt_value interface{}
		err = rows.Scan(&cid, &name, &type_, &notnull, &dflt_value, &pk)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("Column: %s, Type: %s, NotNull: %d, DefaultValue: %v\n",
			name, type_, notnull, dflt_value)
	}
}

func copyFile(src, dst string) error {
	fmt.Printf("Copying %s to %s\n", src, dst)
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %v", err)
	}

	err = os.WriteFile(dst, input, 0644)
	if err != nil {
		return fmt.Errorf("failed to write backup file: %v", err)
	}

	return nil
}
