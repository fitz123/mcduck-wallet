// File: ./internal/database/models.go
package database

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramID   int64 `gorm:"uniqueIndex"`
	Username     string
	Accounts     []Balance
	IsAdmin      bool `gorm:"default:false"`
	Transactions []Transaction
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
}

type Currency struct {
	gorm.Model
	Code      string `gorm:"uniqueIndex"`
	Name      string
	Sign      string
	IsDefault bool `gorm:"default:false"`
}
