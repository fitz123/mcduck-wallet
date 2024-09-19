package database

import (
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramID   int64 `gorm:"uniqueIndex"`
	Username     string
	Balance      float64
	CurrencyID   uint
	Currency     Currency
	IsAdmin      bool `gorm:"default:false"`
	Transactions []Transaction
}

type Transaction struct {
	gorm.Model
	UserID     uint
	Amount     float64
	CurrencyID uint
	Currency   Currency
	Type       string
	ToUserID   *uint
	ToUsername string
	Timestamp  time.Time
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
	err = DB.AutoMigrate(&Currency{}, &User{}, &Transaction{})
	if err != nil {
		logger.Error("Failed to migrate database:", err)
	}

	EnsureDefaultCurrency()
}

func EnsureDefaultCurrency() {
	var defaultCurrency Currency
	if err := DB.Where("is_default = ?", true).First(&defaultCurrency).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			defaultCurrency = Currency{
				Code:      "SHL",
				Name:      "ϣƛöƞȡρƐρ(øʋ)",
				Sign:      "¤",
				IsDefault: true,
			}
			if err := DB.Create(&defaultCurrency).Error; err != nil {
				logger.Error("Failed to create default currency:", err)
			} else {
				logger.Info("Created default currency", "id", defaultCurrency.ID)
			}
		} else {
			logger.Error("Error checking for default currency:", err)
		}
	} else {
		logger.Info("Default currency exists", "id", defaultCurrency.ID)
	}
}

func GetDefaultCurrency() (Currency, error) {
	var currency Currency
	err := DB.Where("is_default = ?", true).First(&currency).Error
	return currency, err
}

func (c *Currency) BeforeCreate(tx *gorm.DB) error {
	if c.IsDefault {
		// Set all other currencies to non-default
		tx.Model(&Currency{}).Where("is_default = ?", true).Update("is_default", false)
	}
	return nil
}
