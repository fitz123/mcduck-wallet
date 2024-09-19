// File: pkg/models/transaction.go

package models

import (
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
)

// TransactionJSON represents a transaction in JSON format
type TransactionJSON struct {
	ID         uint              `json:"id"`
	Amount     float64           `json:"amount"`
	Type       string            `json:"type"`
	ToUserID   *uint             `json:"to_user_id,omitempty"`
	ToUsername string            `json:"to_username,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Currency   database.Currency `json:"currency"`
}
