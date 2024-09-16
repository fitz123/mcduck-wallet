// File: pkg/core/core_test.go

package core

import (
	"testing"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	err = db.AutoMigrate(&database.User{}, &database.Transaction{})
	if err != nil {
		panic("failed to migrate database")
	}

	database.DB = db
	return db
}

func TestGetOrCreateUser(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Test creating a new user
	user, err := GetOrCreateUser(123456, "testuser")
	if err != nil {
		t.Errorf("Failed to create user: %v", err)
	}
	if user.TelegramID != 123456 || user.Username != "testuser" || user.Balance != 0 {
		t.Errorf("User data incorrect. Got: %v", user)
	}

	// Test getting an existing user
	user, err = GetOrCreateUser(123456, "testuser")
	if err != nil {
		t.Errorf("Failed to get existing user: %v", err)
	}
	if user.TelegramID != 123456 || user.Username != "testuser" {
		t.Errorf("Existing user data incorrect. Got: %v", user)
	}
}

func TestTransferMoney(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Create two users
	user1, _ := GetOrCreateUser(111111, "user1")
	user2, _ := GetOrCreateUser(222222, "user2")

	// Set initial balance for user1
	db.Model(&user1).Update("balance", 100)

	// Test successful transfer
	err := TransferMoney(111111, 222222, 50)
	if err != nil {
		t.Errorf("Failed to transfer money: %v", err)
	}

	// Check balances
	db.First(&user1, user1.ID)
	db.First(&user2, user2.ID)
	if user1.Balance != 50 || user2.Balance != 50 {
		t.Errorf("Transfer failed. User1 balance: %v, User2 balance: %v", user1.Balance, user2.Balance)
	}

	// Test insufficient balance
	err = TransferMoney(111111, 222222, 100)
	if err != ErrInsufficientBalance {
		t.Errorf("Expected insufficient balance error, got: %v", err)
	}

	// Test non-existent user
	err = TransferMoney(333333, 222222, 50)
	if err != ErrUserNotFound {
		t.Errorf("Expected user not found error, got: %v", err)
	}
}

func TestGetTransactionHistory(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	user, _ := GetOrCreateUser(111111, "user1")

	// Create some transactions
	transactions := []database.Transaction{
		{UserID: user.ID, Amount: 100, Type: "deposit", Timestamp: time.Now().Add(-2 * time.Hour)},
		{UserID: user.ID, Amount: -30, Type: "transfer", Timestamp: time.Now().Add(-1 * time.Hour)},
		{UserID: user.ID, Amount: 50, Type: "deposit", Timestamp: time.Now()},
	}
	db.Create(&transactions)

	// Test getting transaction history
	history, err := GetTransactionHistory(111111)
	if err != nil {
		t.Errorf("Failed to get transaction history: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("Expected 3 transactions, got %d", len(history))
	}

	// Check if transactions are in descending order by timestamp
	for i := 1; i < len(history); i++ {
		if history[i-1].Timestamp.Before(history[i].Timestamp) {
			t.Errorf("Transactions not in descending order by timestamp")
		}
	}
}

func TestAdminAddMoney(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	admin, _ := GetOrCreateUser(999999, "admin")
	db.Model(&admin).Update("is_admin", true)
	user, _ := GetOrCreateUser(111111, "user1")

	// Test admin adding money
	err := AdminAddMoney(999999, 111111, 100)
	if err != nil {
		t.Errorf("Failed to add money as admin: %v", err)
	}

	// Check user's balance
	db.First(&user, user.ID)
	if user.Balance != 100 {
		t.Errorf("Expected balance 100, got %v", user.Balance)
	}

	// Test non-admin trying to add money
	nonAdmin, _ := GetOrCreateUser(888888, "nonAdmin")
	err = AdminAddMoney(nonAdmin.TelegramID, 111111, 100)
	if err != ErrUnauthorized {
		t.Errorf("Expected unauthorized error, got: %v", err)
	}
}

func TestSetAdminStatus(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	user, _ := GetOrCreateUser(111111, "user1")

	// Test setting admin status
	err := SetAdminStatus(111111, true)
	if err != nil {
		t.Errorf("Failed to set admin status: %v", err)
	}

	// Check admin status
	db.First(&user, user.ID)
	if !user.IsAdmin {
		t.Errorf("Expected user to be admin, but IsAdmin is false")
	}

	// Test unsetting admin status
	err = SetAdminStatus(111111, false)
	if err != nil {
		t.Errorf("Failed to unset admin status: %v", err)
	}

	// Check admin status again
	db.First(&user, user.ID)
	if user.IsAdmin {
		t.Errorf("Expected user to not be admin, but IsAdmin is true")
	}

	// Test setting admin status for non-existent user
	err = SetAdminStatus(999999, true)
	if err != ErrUserNotFound {
		t.Errorf("Expected user not found error, got: %v", err)
	}
}

func TestGetOrCreateUserWithExistingUser(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Create an initial user
	initialUser := database.User{TelegramID: 123456, Username: "existinguser", Balance: 100}
	db.Create(&initialUser)

	// Try to get or create the same user
	user, err := GetOrCreateUser(123456, "existinguser")
	if err != nil {
		t.Errorf("Failed to get existing user: %v", err)
	}

	if user.TelegramID != 123456 || user.Username != "existinguser" || user.Balance != 100 {
		t.Errorf("Unexpected user data. Got: %+v", user)
	}
}

func TestGetOrCreateUserWithNewUser(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Try to get or create a new user
	user, err := GetOrCreateUser(789012, "newuser")
	if err != nil {
		t.Errorf("Failed to create new user: %v", err)
	}

	if user.TelegramID != 789012 || user.Username != "newuser" || user.Balance != 0 {
		t.Errorf("Unexpected user data. Got: %+v", user)
	}
}

func TestTransferMoneyInsufficientBalance(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Create users
	sender := database.User{TelegramID: 123456, Username: "sender", Balance: 50}
	receiver := database.User{TelegramID: 789012, Username: "receiver", Balance: 0}
	db.Create(&sender)
	db.Create(&receiver)

	// Attempt to transfer more money than the sender has
	err := TransferMoney(sender.TelegramID, receiver.TelegramID, 100)
	if err != ErrInsufficientBalance {
		t.Errorf("Expected insufficient balance error, got: %v", err)
	}

	// Check that balances remain unchanged
	var updatedSender, updatedReceiver database.User
	db.First(&updatedSender, sender.ID)
	db.First(&updatedReceiver, receiver.ID)

	if updatedSender.Balance != 50 || updatedReceiver.Balance != 0 {
		t.Errorf("Balances should not have changed. Sender: %v, Receiver: %v", updatedSender.Balance, updatedReceiver.Balance)
	}
}

func TestAdminAddMoneyUnauthorized(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Create a non-admin user
	nonAdmin := database.User{TelegramID: 123456, Username: "nonadmin", IsAdmin: false}
	targetUser := database.User{TelegramID: 789012, Username: "targetuser", Balance: 0}
	db.Create(&nonAdmin)
	db.Create(&targetUser)

	// Attempt to add money as a non-admin
	err := AdminAddMoney(nonAdmin.TelegramID, targetUser.TelegramID, 100)
	if err != ErrUnauthorized {
		t.Errorf("Expected unauthorized error, got: %v", err)
	}

	// Check that target user's balance remains unchanged
	var updatedTargetUser database.User
	db.First(&updatedTargetUser, targetUser.ID)

	if updatedTargetUser.Balance != 0 {
		t.Errorf("Target user's balance should not have changed. Got: %v", updatedTargetUser.Balance)
	}
}
