// File: pkg/models/transaction.go

package models

import (
	"time"
)

// TransactionJSON represents a transaction in JSON format
type TransactionJSON struct {
	ID           uint      `json:"id"`
	Amount       float64   `json:"amount"`
	Type         string    `json:"type"`
	ToUserID     *uint     `json:"to_user_id,omitempty"`
	ToUsername   string    `json:"to_username,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	CurrencyID   uint      `json:"currency_id"`
	CurrencySign string    `json:"currency_sign"`
	CurrencyName string    `json:"currency_name"`
}
