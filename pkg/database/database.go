package database

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"gorm.io/driver/sqlite"
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
	BalanceAfter float64 // Balance after the transaction for the user
}

type Currency struct {
	gorm.Model
	Code      string `gorm:"uniqueIndex"`
	Name      string
	Sign      string
	IsDefault bool `gorm:"default:false"`
}

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("mcduck_wallet.db"), &gorm.Config{})
	if err != nil {
		logger.Error("Failed to connect to database:", err)
	}

	// Auto Migrate the schema
	err = DB.AutoMigrate(&Currency{}, &User{}, &Transaction{}, &Balance{})
	if err != nil {
		logger.Error("Failed to migrate database:", err)
	}

	// Create default currency if it doesn't exist
	if err := CreateDefaultCurrencyIfNotExists(); err != nil {
		logger.Error("Failed to create default currency", "error", err)
		log.Fatal(err)
	}
}

func GetDefaultCurrency() (Currency, error) {
	var currency Currency
	err := DB.Where("is_default = ?", true).First(&currency).Error
	return currency, err
}

func GetCurrencyByCode(code string) (Currency, error) {
	var currency Currency
	result := DB.Where("code = ?", code).First(&currency)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return Currency{}, fmt.Errorf("currency not found: %s", code)
		}
		return Currency{}, result.Error
	}
	return currency, nil
}

func CreateDefaultCurrencyIfNotExists() error {
	var count int64
	DB.Model(&Currency{}).Count(&count)
	if count > 0 {
		logger.Info("Default currency already exists, skipping creation")
		return nil
	}

	defaultCurrency := Currency{
		Code:      "SHL",
		Name:      "ϣƛöƞȡρƐρ(øʋ)",
		Sign:      "¤",
		IsDefault: true,
	}

	if err := DB.Create(&defaultCurrency).Error; err != nil {
		logger.Error("Failed to create default currency", "error", err)
		return err
	}

	logger.Info("Default currency created", "code", defaultCurrency.Code, "name", defaultCurrency.Name)
	return nil
}
