package services

import (
	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/logger"
	"gorm.io/gorm"
)

func (s *coreService) createInitialBalance(tx *gorm.DB, userID uint) error {
	var defaultCurrency database.Currency
	if err := tx.Where("is_default = ?", true).First(&defaultCurrency).Error; err != nil {
		logger.Error("Failed to get default currency", "error", err)
		return err
	}
	logger.Debug("Default currency found", "currencyID", defaultCurrency.ID, "code", defaultCurrency.Code)

	balance := &database.Balance{
		UserID:     userID,
		Amount:     0,
		CurrencyID: defaultCurrency.ID,
	}
	if err := tx.Create(balance).Error; err != nil {
		logger.Error("Failed to create initial balance", "error", err)
		return err
	}
	logger.Debug("Initial balance created", "balanceID", balance.ID, "userID", userID, "amount", balance.Amount)
	return nil
}
