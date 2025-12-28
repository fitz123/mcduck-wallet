// File: ./internal/database/database.go
package database

import (
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	Conn *gorm.DB
}

func New(dsn string) (*DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate your models here
	err = db.AutoMigrate(&User{}, &Balance{}, &Transaction{}, &Currency{})
	if err != nil {
		return nil, err
	}

	// Check if default currency exists, if not create it
	var defaultCurrency Currency
	if err := db.Where("is_default = ?", true).First(&defaultCurrency).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			defaultCurrency = Currency{
				Code:      "USD",
				Name:      "US Dollar",
				Sign:      "$",
				IsDefault: true,
			}
			if err := db.Create(&defaultCurrency).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &DB{Conn: db}, nil
}

func (db *DB) Close() error {
	sqlDB, err := db.Conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// TestDB wraps DB with the temp file path for cleanup
type TestDB struct {
	*DB
	FilePath string
}

// NewTest creates a temporary SQLite database for testing
func NewTest() (*TestDB, error) {
	tmpFile, err := os.CreateTemp("", "mcduck-test-*.db")
	if err != nil {
		return nil, err
	}
	filePath := tmpFile.Name()
	tmpFile.Close()

	db, err := gorm.Open(sqlite.Open(filePath), &gorm.Config{})
	if err != nil {
		os.Remove(filePath)
		return nil, err
	}

	err = db.AutoMigrate(&User{}, &Balance{}, &Transaction{}, &Currency{})
	if err != nil {
		os.Remove(filePath)
		return nil, err
	}

	var defaultCurrency Currency
	if err := db.Where("is_default = ?", true).First(&defaultCurrency).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			defaultCurrency = Currency{
				Code:      "USD",
				Name:      "US Dollar",
				Sign:      "$",
				IsDefault: true,
			}
			if err := db.Create(&defaultCurrency).Error; err != nil {
				os.Remove(filePath)
				return nil, err
			}
		} else {
			os.Remove(filePath)
			return nil, err
		}
	}

	return &TestDB{DB: &DB{Conn: db}, FilePath: filePath}, nil
}

// Cleanup closes and removes the test database file
func (tdb *TestDB) Cleanup() error {
	if err := tdb.Close(); err != nil {
		return err
	}
	return os.Remove(tdb.FilePath)
}
