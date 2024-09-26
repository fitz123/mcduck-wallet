// File: ./internal/services/core_service.go
package services

import (
	"context"
	"errors"
	"time"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"gorm.io/gorm"
)

type CoreService interface {
	GetBalances(ctx context.Context, telegramID int64) ([]database.Balance, error)
	GetDefaultCurrency(ctx context.Context) (*database.Currency, error)
	TransferMoney(ctx context.Context, fromTelegramID int64, toUsername string, amount float64, currencyCode string) error
	GetTransactionHistory(ctx context.Context, telegramID int64) ([]database.Transaction, error)
	SetAdminStatus(ctx context.Context, targetUsername string, isAdmin bool) error
	AdminSetBalance(ctx context.Context, adminTelegramID int64, targetUsername string, amount float64, currencyCode string) error
	GetCurrencyByCode(ctx context.Context, code string) (*database.Currency, error)
	ListUsersWithBalances(ctx context.Context) ([]UserWithBalance, error)
	RemoveUser(ctx context.Context, username string) error
	AddUser(ctx context.Context, telegramID int64, username string) error
	AddCurrency(ctx context.Context, code, name, sign string) error
	SetDefaultCurrency(ctx context.Context, code string) error
}

type coreService struct {
	db          *database.DB
	userService UserService
}

func NewCoreService(db *database.DB, userService UserService) CoreService {
	return &coreService{
		db:          db,
		userService: userService,
	}
}

func (s *coreService) GetBalances(ctx context.Context, telegramID int64) ([]database.Balance, error) {
	user, err := s.userService.GetUser(ctx, telegramID)
	if err != nil {
		return nil, err
	}
	return user.Accounts, nil
}

func (s *coreService) GetDefaultCurrency(ctx context.Context) (*database.Currency, error) {
	var currency database.Currency
	err := s.db.Conn.WithContext(ctx).
		Where("is_default = ?", true).
		First(&currency).Error
	if err != nil {
		return nil, err
	}
	return &currency, nil
}

func (s *coreService) TransferMoney(ctx context.Context, fromTelegramID int64, toUsername string, amount float64, currencyCode string) error {
	fromUser, err := s.userService.GetUser(ctx, fromTelegramID)
	if err != nil {
		return err
	}

	toUser, err := s.userService.GetUserByUsername(toUsername)
	if err != nil {
		return err
	}

	if fromUser.ID == toUser.ID {
		return errors.New("cannot transfer to self")
	}
	if amount <= 0 {
		return errors.New("transfer amount must be positive")
	}

	var fromBalance, toBalance *database.Balance
	for i := range fromUser.Accounts {
		if fromUser.Accounts[i].Currency.Code == currencyCode {
			fromBalance = &fromUser.Accounts[i]
			break
		}
	}
	for i := range toUser.Accounts {
		if toUser.Accounts[i].Currency.Code == currencyCode {
			toBalance = &toUser.Accounts[i]
			break
		}
	}

	if fromBalance == nil || toBalance == nil {
		return errors.New("currency not supported")
	}
	if fromBalance.Amount < amount {
		return errors.New("insufficient balance")
	}

	fromBalance.Amount -= amount
	toBalance.Amount += amount

	// Create transactions
	now := time.Now()
	fromTransaction := database.Transaction{
		UserID:       fromUser.ID,
		BalanceID:    fromBalance.ID,
		Amount:       -amount,
		Type:         "transfer_out",
		FromUserID:   fromUser.ID,
		FromUsername: fromUser.Username,
		ToUserID:     toUser.ID,
		ToUsername:   toUser.Username,
		Timestamp:    now,
		BalanceAfter: fromBalance.Amount,
	}
	toTransaction := database.Transaction{
		UserID:       toUser.ID,
		BalanceID:    toBalance.ID,
		Amount:       amount,
		Type:         "transfer_in",
		FromUserID:   fromUser.ID,
		FromUsername: fromUser.Username,
		ToUserID:     toUser.ID,
		ToUsername:   toUser.Username,
		Timestamp:    now,
		BalanceAfter: toBalance.Amount,
	}

	// Save to database
	return s.db.Conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(fromBalance).Error; err != nil {
			return err
		}
		if err := tx.Save(toBalance).Error; err != nil {
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

func (s *coreService) GetTransactionHistory(ctx context.Context, telegramID int64) ([]database.Transaction, error) {
	user, err := s.userService.GetUser(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	var transactions []database.Transaction
	err = s.db.Conn.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Preload("Balance.Currency").
		Order("timestamp desc").
		Limit(10).
		Find(&transactions).Error
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

// Admin functions

func (s *coreService) SetAdminStatus(ctx context.Context, targetUsername string, isAdmin bool) error {
	var user database.User
	if err := s.db.Conn.WithContext(ctx).
		Where("username = ?", targetUsername).
		First(&user).Error; err != nil {
		return err
	}

	return s.db.Conn.WithContext(ctx).
		Model(&user).
		Update("is_admin", isAdmin).Error
}

func (s *coreService) AdminSetBalance(ctx context.Context, adminTelegramID int64, targetUsername string, amount float64, currencyCode string) error {
	if !s.userService.IsAdmin(ctx, adminTelegramID) {
		return errors.New("unauthorized")
	}

	var targetUser database.User
	if err := s.db.Conn.WithContext(ctx).
		Preload("Accounts.Currency").
		Where("username = ?", targetUsername).
		First(&targetUser).Error; err != nil {
		return err
	}

	var targetBalance *database.Balance
	for i := range targetUser.Accounts {
		if targetUser.Accounts[i].Currency.Code == currencyCode {
			targetBalance = &targetUser.Accounts[i]
			break
		}
	}

	if targetBalance == nil {
		// Create new balance
		currency, err := s.GetCurrencyByCode(ctx, currencyCode)
		if err != nil {
			return err
		}
		targetBalance = &database.Balance{
			UserID:     targetUser.ID,
			Amount:     amount,
			CurrencyID: currency.ID,
		}
		if err := s.db.Conn.WithContext(ctx).Create(targetBalance).Error; err != nil {
			return err
		}
	} else {
		targetBalance.Amount = amount
		if err := s.db.Conn.WithContext(ctx).Save(targetBalance).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *coreService) GetCurrencyByCode(ctx context.Context, code string) (*database.Currency, error) {
	var currency database.Currency
	if err := s.db.Conn.WithContext(ctx).
		Where("code = ?", code).
		First(&currency).Error; err != nil {
		return nil, err
	}
	return &currency, nil
}

type UserWithBalance struct {
	TelegramID int64
	Username   string
	Balances   map[string]float64
}

func (s *coreService) ListUsersWithBalances(ctx context.Context) ([]UserWithBalance, error) {
	var users []database.User
	err := s.db.Conn.WithContext(ctx).
		Preload("Accounts.Currency").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	var result []UserWithBalance
	for _, user := range users {
		ub := UserWithBalance{
			TelegramID: user.TelegramID,
			Username:   user.Username,
			Balances:   make(map[string]float64),
		}
		for _, acc := range user.Accounts {
			ub.Balances[acc.Currency.Code] = acc.Amount
		}
		result = append(result, ub)
	}
	return result, nil
}

func (s *coreService) RemoveUser(ctx context.Context, username string) error {
	result := s.db.Conn.WithContext(ctx).
		Where("username = ?", username).
		Delete(&database.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *coreService) AddUser(ctx context.Context, telegramID int64, username string) error {
	user := &database.User{
		TelegramID: telegramID,
		Username:   username,
	}
	return s.userService.CreateUser(ctx, user)
}

func (s *coreService) AddCurrency(ctx context.Context, code, name, sign string) error {
	currency := &database.Currency{
		Code: code,
		Name: name,
		Sign: sign,
	}
	return s.db.Conn.WithContext(ctx).Create(currency).Error
}

func (s *coreService) SetDefaultCurrency(ctx context.Context, code string) error {
	return s.db.Conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unset previous default
		if err := tx.Model(&database.Currency{}).
			Where("is_default = ?", true).
			Update("is_default", false).Error; err != nil {
			return err
		}
		// Set new default
		if err := tx.Model(&database.Currency{}).
			Where("code = ?", code).
			Update("is_default", true).Error; err != nil {
			return err
		}
		return nil
	})
}
