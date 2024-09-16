// File: pkg/webapp/handlers_test.go

package webapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

func TestGetBalance(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Create a test user
	user := database.User{TelegramID: 123456, Username: "testuser", Balance: 100}
	db.Create(&user)

	// Create a request with a mock context
	req, err := http.NewRequest("GET", "/balance", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create a handler function that adds the user ID to the context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", int64(123456))
		getBalance(w, r.WithContext(ctx))
	})

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := "Your current balance is ¤100.00"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestTransferMoney(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Create test users
	user1 := database.User{TelegramID: 123456, Username: "sender", Balance: 100}
	user2 := database.User{TelegramID: 789012, Username: "receiver", Balance: 50}
	db.Create(&user1)
	db.Create(&user2)

	// Create a request with transfer data
	form := url.Values{}
	form.Add("to_username", "receiver")
	form.Add("amount", "25")
	req, err := http.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create a handler function that adds the user ID to the context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", int64(123456))
		transferMoney(w, r.WithContext(ctx))
	})

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := "Successfully transferred ¤25.00 to receiver"
	if !strings.Contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}

	// Check if the balances were updated correctly
	var updatedUser1, updatedUser2 database.User
	db.First(&updatedUser1, user1.ID)
	db.First(&updatedUser2, user2.ID)

	if updatedUser1.Balance != 75 || updatedUser2.Balance != 75 {
		t.Errorf("Transfer not applied correctly. Sender balance: %v, Receiver balance: %v", updatedUser1.Balance, updatedUser2.Balance)
	}
}

func TestGetTransactionHistory(t *testing.T) {
	db := setupTestDB()
	defer db.Migrator().DropTable(&database.User{}, &database.Transaction{})

	// Create a test user
	user := database.User{TelegramID: 123456, Username: "testuser", Balance: 100}
	db.Create(&user)

	// Create some test transactions
	transactions := []database.Transaction{
		{UserID: user.ID, Amount: 50, Type: "deposit"},
		{UserID: user.ID, Amount: -20, Type: "transfer"},
	}
	db.Create(&transactions)

	// Create a request
	req, err := http.NewRequest("GET", "/history", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create a handler function that adds the user ID to the context
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", int64(123456))
		getTransactionHistory(w, r.WithContext(ctx))
	})

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check if the response contains transaction information
	if !strings.Contains(rr.Body.String(), "Deposited ¤50.00") || !strings.Contains(rr.Body.String(), "Sent ¤20.00") {
		t.Errorf("handler returned unexpected body: %v", rr.Body.String())
	}
}
