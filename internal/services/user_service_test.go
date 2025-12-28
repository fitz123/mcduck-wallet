package services

import (
	"context"
	"testing"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/testutil"
)

func TestUserService_GetUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	svc := NewUserService(db)
	ctx := context.Background()

	t.Run("returns user by telegram ID", func(t *testing.T) {
		testutil.CreateTestUser(t, db, 12345, "testuser", false)

		user, err := svc.GetUser(ctx, 12345)
		if err != nil {
			t.Fatalf("GetUser() error = %v", err)
		}
		if user.Username != "testuser" {
			t.Errorf("GetUser() username = %v, want testuser", user.Username)
		}
		if user.TelegramID != 12345 {
			t.Errorf("GetUser() telegramID = %v, want 12345", user.TelegramID)
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		_, err := svc.GetUser(ctx, 99999)
		if err == nil {
			t.Error("GetUser() expected error for non-existent user")
		}
	})

	t.Run("preloads accounts with currency", func(t *testing.T) {
		user := testutil.CreateTestUser(t, db, 11111, "withbalance", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, user.ID, currency.ID, 100.0)

		retrieved, err := svc.GetUser(ctx, 11111)
		if err != nil {
			t.Fatalf("GetUser() error = %v", err)
		}
		if len(retrieved.Accounts) != 1 {
			t.Errorf("GetUser() accounts count = %v, want 1", len(retrieved.Accounts))
		}
		if retrieved.Accounts[0].Currency.Code != "USD" {
			t.Errorf("GetUser() account currency = %v, want USD", retrieved.Accounts[0].Currency.Code)
		}
	})
}

func TestUserService_GetUserByUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	svc := NewUserService(db)

	t.Run("returns user by username", func(t *testing.T) {
		testutil.CreateTestUser(t, db, 22222, "findme", false)

		user, err := svc.GetUserByUsername("findme")
		if err != nil {
			t.Fatalf("GetUserByUsername() error = %v", err)
		}
		if user.TelegramID != 22222 {
			t.Errorf("GetUserByUsername() telegramID = %v, want 22222", user.TelegramID)
		}
	})

	t.Run("returns error for non-existent username", func(t *testing.T) {
		_, err := svc.GetUserByUsername("doesnotexist")
		if err == nil {
			t.Error("GetUserByUsername() expected error for non-existent user")
		}
		if err.Error() != "user not found" {
			t.Errorf("GetUserByUsername() error = %v, want 'user not found'", err)
		}
	})

	t.Run("preloads accounts with currency", func(t *testing.T) {
		user := testutil.CreateTestUser(t, db, 33333, "withaccount", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, user.ID, currency.ID, 50.0)

		retrieved, err := svc.GetUserByUsername("withaccount")
		if err != nil {
			t.Fatalf("GetUserByUsername() error = %v", err)
		}
		if len(retrieved.Accounts) != 1 {
			t.Errorf("GetUserByUsername() accounts count = %v, want 1", len(retrieved.Accounts))
		}
	})
}

func TestUserService_CreateUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	svc := NewUserService(db)
	ctx := context.Background()

	t.Run("creates new user", func(t *testing.T) {
		user := &database.User{
			TelegramID: 44444,
			Username:   "newuser",
		}
		err := svc.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("CreateUser() error = %v", err)
		}
		if user.ID == 0 {
			t.Error("CreateUser() should set user ID")
		}

		// Verify in database
		var found database.User
		db.Conn.First(&found, user.ID)
		if found.Username != "newuser" {
			t.Errorf("CreateUser() stored username = %v, want newuser", found.Username)
		}
	})

	t.Run("returns error for duplicate telegram ID", func(t *testing.T) {
		user1 := &database.User{TelegramID: 55555, Username: "first"}
		user2 := &database.User{TelegramID: 55555, Username: "second"}

		svc.CreateUser(ctx, user1)
		err := svc.CreateUser(ctx, user2)
		if err == nil {
			t.Error("CreateUser() expected error for duplicate telegram ID")
		}
	})
}

func TestUserService_UpdateUsername(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	svc := NewUserService(db)
	ctx := context.Background()

	t.Run("updates username successfully", func(t *testing.T) {
		testutil.CreateTestUser(t, db, 66666, "oldname", false)

		err := svc.UpdateUsername(ctx, 66666, "newname")
		if err != nil {
			t.Fatalf("UpdateUsername() error = %v", err)
		}

		// Verify update
		user, _ := svc.GetUser(ctx, 66666)
		if user.Username != "newname" {
			t.Errorf("UpdateUsername() username = %v, want newname", user.Username)
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		err := svc.UpdateUsername(ctx, 99999, "anyname")
		if err == nil {
			t.Error("UpdateUsername() expected error for non-existent user")
		}
		if err.Error() != "user not found" {
			t.Errorf("UpdateUsername() error = %v, want 'user not found'", err)
		}
	})
}

func TestUserService_IsAdmin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	svc := NewUserService(db)
	ctx := context.Background()

	t.Run("returns true for admin user", func(t *testing.T) {
		testutil.CreateTestUser(t, db, 77777, "adminuser", true)

		isAdmin := svc.IsAdmin(ctx, 77777)
		if !isAdmin {
			t.Error("IsAdmin() = false for admin user, want true")
		}
	})

	t.Run("returns false for non-admin user", func(t *testing.T) {
		testutil.CreateTestUser(t, db, 88888, "regularuser", false)

		isAdmin := svc.IsAdmin(ctx, 88888)
		if isAdmin {
			t.Error("IsAdmin() = true for non-admin user, want false")
		}
	})

	t.Run("returns false for non-existent user", func(t *testing.T) {
		isAdmin := svc.IsAdmin(ctx, 99999)
		if isAdmin {
			t.Error("IsAdmin() = true for non-existent user, want false")
		}
	})
}

func TestUserService_UpdateLastUsedCurrency(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()
	svc := NewUserService(db)
	ctx := context.Background()

	t.Run("updates last used currency", func(t *testing.T) {
		testutil.CreateTestUser(t, db, 10101, "currencyuser", false)
		currency := testutil.CreateTestCurrency(t, db, "EUR", "Euro", "€", false)

		err := svc.UpdateLastUsedCurrency(ctx, 10101, currency.ID)
		if err != nil {
			t.Fatalf("UpdateLastUsedCurrency() error = %v", err)
		}

		// Verify update
		user, _ := svc.GetUser(ctx, 10101)
		if user.LastUsedCurrencyID != currency.ID {
			t.Errorf("UpdateLastUsedCurrency() currencyID = %v, want %v", user.LastUsedCurrencyID, currency.ID)
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		err := svc.UpdateLastUsedCurrency(ctx, 99999, 1)
		if err == nil {
			t.Error("UpdateLastUsedCurrency() expected error for non-existent user")
		}
		if err.Error() != "user not found" {
			t.Errorf("UpdateLastUsedCurrency() error = %v, want 'user not found'", err)
		}
	})
}
