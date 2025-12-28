// File: ./internal/database/models.go
package database

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramID         int64 `gorm:"uniqueIndex"`
	Username           string
	Accounts           []Balance
	IsAdmin            bool `gorm:"default:false"`
	Transactions       []Transaction
	LastUsedCurrencyID uint // New field
}

type Balance struct {
	gorm.Model
	UserID     uint
	Amount     float64
	CurrencyID uint
	Currency   Currency
}

type Transaction struct {
	gorm.Model
	UserID       uint
	BalanceID    uint
	Balance      Balance
	Amount       float64
	Type         string
	FromUserID   uint
	FromUsername string
	ToUserID     uint
	ToUsername   string
	Timestamp    time.Time
	BalanceAfter float64
	ExchangeRef  string  // UUID linking exchange_out and exchange_in transactions
	ExchangeRate float64 // Exchange rate used (for audit trail)
}

type Currency struct {
	gorm.Model
	Code      string  `gorm:"uniqueIndex"`
	Name      string
	Sign      string
	IsDefault bool    `gorm:"default:false"`
	IsReal    bool    `gorm:"default:false"` // true for real currencies (USD, EUR), false for made-up (SHL)
	FixedRate float64 `gorm:"default:0"`     // For made-up currencies: units per 1 USD
}

// UserWithBalance is a DTO for listing users with their balances
type UserWithBalance struct {
	TelegramID int64
	Username   string
	Balances   map[string]float64
}
