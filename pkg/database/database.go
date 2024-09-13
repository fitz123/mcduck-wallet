package database

import (
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("mcduck_wallet.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto Migrate the schema
	err = DB.AutoMigrate(&User{}, &Transaction{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}

type User struct {
	gorm.Model
	TelegramID   int64 `gorm:"uniqueIndex"`
	Username     string
	Balance      float64
	IsAdmin      bool `gorm:"default:false"`
	Transactions []Transaction
}

type Transaction struct {
	gorm.Model
	UserID    uint
	Amount    float64
	Type      string // "deposit", "transfer", or "admin_deposit"
	ToUserID  *uint  // Pointer to allow null for deposits
	Timestamp time.Time
}
