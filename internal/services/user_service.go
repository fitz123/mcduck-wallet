// File: ./internal/services/user_service.go
package services

import (
	"context"
	"errors"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"gorm.io/gorm"
)

type UserService interface {
	GetUser(ctx context.Context, telegramID int64) (*database.User, error)
	GetUserByUsername(username string) (*database.User, error)
	CreateUser(ctx context.Context, user *database.User) error
	UpdateUsername(ctx context.Context, telegramID int64, username string) error
	IsAdmin(ctx context.Context, telegramID int64) bool
}

type userService struct {
	db *database.DB
}

func NewUserService(db *database.DB) UserService {
	return &userService{db: db}
}

func (s *userService) GetUser(ctx context.Context, telegramID int64) (*database.User, error) {
	var user database.User
	if err := s.db.Conn.WithContext(ctx).
		Preload("Accounts.Currency").
		Where("telegram_id = ?", telegramID).
		First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *userService) GetUserByUsername(username string) (*database.User, error) {
	var user database.User
	result := s.db.Conn.Preload("Accounts.Currency").Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, result.Error
	}
	return &user, nil
}

func (s *userService) CreateUser(ctx context.Context, user *database.User) error {
	return s.db.Conn.WithContext(ctx).Create(user).Error
}

func (s *userService) UpdateUsername(ctx context.Context, telegramID int64, username string) error {
	result := s.db.Conn.WithContext(ctx).
		Model(&database.User{}).
		Where("telegram_id = ?", telegramID).
		Update("username", username)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}

func (s *userService) IsAdmin(ctx context.Context, telegramID int64) bool {
	user, err := s.GetUser(ctx, telegramID)
	if err != nil {
		return false
	}
	return user.IsAdmin
}
