package handlers

import (
	"errors"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"gorm.io/gorm"
)

func RegisterUser(telegramID int64, username string) error {
	user := database.User{
		TelegramID: telegramID,
		Username:   username,
		Balance:    0, // Initial balance
	}

	result := database.DB.Create(&user)
	return result.Error
}

func GetOrCreateUser(telegramID int64, username string) (*database.User, error) {
	var user database.User
	result := database.DB.Where("telegram_id = ?", telegramID).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// User not found, create a new one
			err := RegisterUser(telegramID, username)
			if err != nil {
				return nil, err
			}
			// Fetch the newly created user
			result = database.DB.Where("telegram_id = ?", telegramID).First(&user)
			if result.Error != nil {
				return nil, result.Error
			}
		} else {
			return nil, result.Error
		}
	}

	return &user, nil
}

func TransferMoney(fromUserID, toUserID int64, amount float64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var fromUser, toUser database.User

		// Get the sender
		if err := tx.Where("telegram_id = ?", fromUserID).First(&fromUser).Error; err != nil {
			return err
		}

		// Check if sender has enough balance
		if fromUser.Balance < amount {
			return errors.New("insufficient balance")
		}

		// Get the receiver
		if err := tx.Where("telegram_id = ?", toUserID).First(&toUser).Error; err != nil {
			return err
		}

		// Update balances
		fromUser.Balance -= amount
		toUser.Balance += amount

		// Create transactions
		now := time.Now()
		fromTransaction := database.Transaction{
			UserID:    fromUser.ID,
			Amount:    -amount,
			Type:      "transfer",
			ToUserID:  &toUser.ID,
			Timestamp: now,
		}
		toTransaction := database.Transaction{
			UserID:    toUser.ID,
			Amount:    amount,
			Type:      "transfer",
			ToUserID:  &fromUser.ID,
			Timestamp: now,
		}

		// Save everything
		if err := tx.Save(&fromUser).Error; err != nil {
			return err
		}
		if err := tx.Save(&toUser).Error; err != nil {
			return err
		}
		if err := tx.Create(&fromTransaction).Error; err != nil {
			return err
		}
		if err := tx.Create(&toTransaction).Error; err != nil {
			return err
		}

		return nil
	})
}

func GetTransactionHistory(userID int64) ([]database.Transaction, error) {
	var user database.User
	if err := database.DB.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	var transactions []database.Transaction
	if err := database.DB.Where("user_id = ?", user.ID).Order("timestamp desc").Limit(10).Find(&transactions).Error; err != nil {
		return nil, err
	}

	return transactions, nil
}

func AdminAddMoney(adminID, targetUserID int64, amount float64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var admin, targetUser database.User

		// Verify admin
		if err := tx.Where("telegram_id = ?", adminID).First(&admin).Error; err != nil {
			return errors.New("admin not found")
		}
		if !admin.IsAdmin {
			return errors.New("unauthorized: user is not an admin")
		}

		// Get target user
		if err := tx.Where("telegram_id = ?", targetUserID).First(&targetUser).Error; err != nil {
			return errors.New("target user not found")
		}

		// Update balance
		targetUser.Balance = amount // Set the balance to the specified amount

		// Create transaction
		transaction := database.Transaction{
			UserID:    targetUser.ID,
			Amount:    amount - targetUser.Balance, // The difference is the amount added
			Type:      "admin_set_balance",
			Timestamp: time.Now(),
		}

		// Save everything
		if err := tx.Save(&targetUser).Error; err != nil {
			return err
		}
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}

		return nil
	})
}

func SetAdminStatus(telegramID int64, isAdmin bool) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var user database.User
		if err := tx.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
			return err
		}
		user.IsAdmin = isAdmin
		return tx.Save(&user).Error
	})
}
