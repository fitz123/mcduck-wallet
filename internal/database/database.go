// File: ./internal/database/database.go
package database

import (
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
