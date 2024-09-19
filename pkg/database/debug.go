package database

import (
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"gorm.io/gorm"
)

func DebugCurrencyAndUserData() {
	// Update user and transaction currencies
	if err := ResetDefaultCurrency(); err != nil {
		logger.Error("Failed to reset default currency", "error", err)
	}

	if err := UpdateUserCurrencies(); err != nil {
		logger.Error("Failed to update user currencies", "error", err)
	}

	if err := UpdateTransactionCurrencies(); err != nil {
		logger.Error("Failed to update transaction currencies", "error", err)
	}

	GetDefaultCurrency()
}

func GetDebugCurrencyAndUserData() {
	var currencies []Currency
	if err := DB.Find(&currencies).Error; err != nil {
		logger.Error("Failed to fetch currencies", "error", err)
		return
	}
	logger.Debug("Currencies in database", "count", len(currencies))
	for _, c := range currencies {
		logger.Debug("Currency details", "id", c.ID, "code", c.Code, "name", c.Name, "sign", c.Sign, "isDefault", c.IsDefault)
	}

	var users []User
	if err := DB.Find(&users).Error; err != nil {
		logger.Error("Failed to fetch users", "error", err)
		return
	}
	logger.Debug("Users in database", "count", len(users))
	for _, u := range users {
		logger.Debug("User details", "id", u.ID, "telegramID", u.TelegramID, "username", u.Username, "currencyID", u.CurrencyID)
	}
}

func UpdateUserCurrencies() error {
	defaultCurrency, err := GetDefaultCurrency()
	if err != nil {
		logger.Error("Failed to get default currency", "error", err)
		return err
	}

	result := DB.Model(&User{}).Where("currency_id = ? OR currency_id IS NULL", 0).Update("currency_id", defaultCurrency.ID)
	if result.Error != nil {
		logger.Error("Failed to update user currencies", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.Warn("Updated user currencies", "usersUpdated", result.RowsAffected)
	} else {
		logger.Debug("All user currencies are set")
	}

	return nil
}

func UpdateTransactionCurrencies() error {
	defaultCurrency, err := GetDefaultCurrency()
	if err != nil {
		logger.Error("Failed to get default currency", "error", err)
		return err
	}

	result := DB.Model(&Transaction{}).Where("currency_id = ? OR currency_id IS NULL", 0).Update("currency_id", defaultCurrency.ID)
	if result.Error != nil {
		logger.Error("Failed to update transaction currencies", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		logger.Warn("Updated transaction currencies", "transactionsUpdated", result.RowsAffected)
	} else {
		logger.Debug("All transaction currencies are set")
	}

	return nil
}

func ResetDefaultCurrency() error {
	var defaultCurrency Currency
	if err := DB.Where("is_default = ?", true).First(&defaultCurrency).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// No default currency found, set the first currency as default
			if err := DB.First(&defaultCurrency).Error; err != nil {
				logger.Error("No currencies found in database")
				return err
			}
			defaultCurrency.IsDefault = true
			if err := DB.Save(&defaultCurrency).Error; err != nil {
				logger.Error("Failed to set default currency", "error", err)
				return err
			}
			logger.Info("Set default currency", "id", defaultCurrency.ID)
		} else {
			logger.Error("Error checking for default currency:", err)
			return err
		}
	}
	return nil
}
