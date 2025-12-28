// Package testutil provides test utilities and mocks for testing
package testutil

import (
	"context"

	"github.com/fitz123/mcduck-wallet/internal/database"
)

// MockUserService is a mock implementation of services.UserService
type MockUserService struct {
	GetUserFunc               func(ctx context.Context, telegramID int64) (*database.User, error)
	GetUserByUsernameFunc     func(username string) (*database.User, error)
	CreateUserFunc            func(ctx context.Context, user *database.User) error
	UpdateUsernameFunc        func(ctx context.Context, telegramID int64, username string) error
	IsAdminFunc               func(ctx context.Context, telegramID int64) bool
	UpdateLastUsedCurrencyFunc func(ctx context.Context, telegramID int64, currencyID uint) error
}

func (m *MockUserService) GetUser(ctx context.Context, telegramID int64) (*database.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, telegramID)
	}
	return nil, nil
}

func (m *MockUserService) GetUserByUsername(username string) (*database.User, error) {
	if m.GetUserByUsernameFunc != nil {
		return m.GetUserByUsernameFunc(username)
	}
	return nil, nil
}

func (m *MockUserService) CreateUser(ctx context.Context, user *database.User) error {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, user)
	}
	return nil
}

func (m *MockUserService) UpdateUsername(ctx context.Context, telegramID int64, username string) error {
	if m.UpdateUsernameFunc != nil {
		return m.UpdateUsernameFunc(ctx, telegramID, username)
	}
	return nil
}

func (m *MockUserService) IsAdmin(ctx context.Context, telegramID int64) bool {
	if m.IsAdminFunc != nil {
		return m.IsAdminFunc(ctx, telegramID)
	}
	return false
}

func (m *MockUserService) UpdateLastUsedCurrency(ctx context.Context, telegramID int64, currencyID uint) error {
	if m.UpdateLastUsedCurrencyFunc != nil {
		return m.UpdateLastUsedCurrencyFunc(ctx, telegramID, currencyID)
	}
	return nil
}

// MockCoreService is a mock implementation of services.CoreService
type MockCoreService struct {
	GetBalancesFunc           func(ctx context.Context, telegramID int64) ([]database.Balance, error)
	GetDefaultCurrencyFunc    func(ctx context.Context) (*database.Currency, error)
	TransferMoneyFunc         func(ctx context.Context, fromTelegramID int64, toUsername string, amount float64, currencyCode string) error
	GetTransactionHistoryFunc func(ctx context.Context, telegramID int64, offset, limit int) ([]database.Transaction, int64, error)
	SetAdminStatusFunc        func(ctx context.Context, targetUsername string, isAdmin bool) error
	AdminSetBalanceFunc       func(ctx context.Context, adminTelegramID int64, targetUsername string, amount float64, currencyCode string) error
	GetCurrencyByCodeFunc     func(ctx context.Context, code string) (*database.Currency, error)
	ListUsersWithBalancesFunc func(ctx context.Context) ([]database.UserWithBalance, error)
	DisableUserFunc           func(ctx context.Context, username string) error
	AddUserFunc               func(ctx context.Context, telegramID int64, username string) error
	DestroyUserFunc           func(ctx context.Context, username string) error
	AddCurrencyFunc           func(ctx context.Context, code, name, sign string) error
	SetDefaultCurrencyFunc    func(ctx context.Context, code string) error
	GetPreviousRecipientsFunc func(ctx context.Context, userID uint) ([]string, error)
}

func (m *MockCoreService) GetBalances(ctx context.Context, telegramID int64) ([]database.Balance, error) {
	if m.GetBalancesFunc != nil {
		return m.GetBalancesFunc(ctx, telegramID)
	}
	return nil, nil
}

func (m *MockCoreService) GetDefaultCurrency(ctx context.Context) (*database.Currency, error) {
	if m.GetDefaultCurrencyFunc != nil {
		return m.GetDefaultCurrencyFunc(ctx)
	}
	return nil, nil
}

func (m *MockCoreService) TransferMoney(ctx context.Context, fromTelegramID int64, toUsername string, amount float64, currencyCode string) error {
	if m.TransferMoneyFunc != nil {
		return m.TransferMoneyFunc(ctx, fromTelegramID, toUsername, amount, currencyCode)
	}
	return nil
}

func (m *MockCoreService) GetTransactionHistory(ctx context.Context, telegramID int64, offset, limit int) ([]database.Transaction, int64, error) {
	if m.GetTransactionHistoryFunc != nil {
		return m.GetTransactionHistoryFunc(ctx, telegramID, offset, limit)
	}
	return nil, 0, nil
}

func (m *MockCoreService) SetAdminStatus(ctx context.Context, targetUsername string, isAdmin bool) error {
	if m.SetAdminStatusFunc != nil {
		return m.SetAdminStatusFunc(ctx, targetUsername, isAdmin)
	}
	return nil
}

func (m *MockCoreService) AdminSetBalance(ctx context.Context, adminTelegramID int64, targetUsername string, amount float64, currencyCode string) error {
	if m.AdminSetBalanceFunc != nil {
		return m.AdminSetBalanceFunc(ctx, adminTelegramID, targetUsername, amount, currencyCode)
	}
	return nil
}

func (m *MockCoreService) GetCurrencyByCode(ctx context.Context, code string) (*database.Currency, error) {
	if m.GetCurrencyByCodeFunc != nil {
		return m.GetCurrencyByCodeFunc(ctx, code)
	}
	return nil, nil
}

func (m *MockCoreService) ListUsersWithBalances(ctx context.Context) ([]database.UserWithBalance, error) {
	if m.ListUsersWithBalancesFunc != nil {
		return m.ListUsersWithBalancesFunc(ctx)
	}
	return nil, nil
}

func (m *MockCoreService) DisableUser(ctx context.Context, username string) error {
	if m.DisableUserFunc != nil {
		return m.DisableUserFunc(ctx, username)
	}
	return nil
}

func (m *MockCoreService) AddUser(ctx context.Context, telegramID int64, username string) error {
	if m.AddUserFunc != nil {
		return m.AddUserFunc(ctx, telegramID, username)
	}
	return nil
}

func (m *MockCoreService) DestroyUser(ctx context.Context, username string) error {
	if m.DestroyUserFunc != nil {
		return m.DestroyUserFunc(ctx, username)
	}
	return nil
}

func (m *MockCoreService) AddCurrency(ctx context.Context, code, name, sign string) error {
	if m.AddCurrencyFunc != nil {
		return m.AddCurrencyFunc(ctx, code, name, sign)
	}
	return nil
}

func (m *MockCoreService) SetDefaultCurrency(ctx context.Context, code string) error {
	if m.SetDefaultCurrencyFunc != nil {
		return m.SetDefaultCurrencyFunc(ctx, code)
	}
	return nil
}

func (m *MockCoreService) GetPreviousRecipients(ctx context.Context, userID uint) ([]string, error) {
	if m.GetPreviousRecipientsFunc != nil {
		return m.GetPreviousRecipientsFunc(ctx, userID)
	}
	return nil, nil
}

// MockNotificationService is a mock implementation of services.NotificationService
type MockNotificationService struct {
	NotifyUserFunc func(ctx context.Context, telegramID int64, message string) error
}

func (m *MockNotificationService) NotifyUser(ctx context.Context, telegramID int64, message string) error {
	if m.NotifyUserFunc != nil {
		return m.NotifyUserFunc(ctx, telegramID, message)
	}
	return nil
}
