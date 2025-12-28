package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/logger"
	"github.com/fitz123/mcduck-wallet/internal/testutil"
)

func init() {
	logger.Init("debug")
}

func setupCoreServiceTest(t *testing.T) (*database.DB, CoreService, *testutil.MockNotificationService) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	userService := NewUserService(db)
	mockNotifier := &testutil.MockNotificationService{}
	coreService := NewCoreService(db, userService, mockNotifier)
	return db, coreService, mockNotifier
}

func TestCoreService_GetBalances(t *testing.T) {
	db, svc, _ := setupCoreServiceTest(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("returns user balances", func(t *testing.T) {
		user := testutil.CreateTestUser(t, db, 12345, "balanceuser", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, user.ID, currency.ID, 100.50)

		balances, err := svc.GetBalances(ctx, 12345)
		if err != nil {
			t.Fatalf("GetBalances() error = %v", err)
		}
		if len(balances) != 1 {
			t.Errorf("GetBalances() count = %v, want 1", len(balances))
		}
		if balances[0].Amount != 100.50 {
			t.Errorf("GetBalances() amount = %v, want 100.50", balances[0].Amount)
		}
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		_, err := svc.GetBalances(ctx, 99999)
		if err == nil {
			t.Error("GetBalances() expected error for non-existent user")
		}
	})

	t.Run("returns multiple currency balances", func(t *testing.T) {
		user := testutil.CreateTestUser(t, db, 11111, "multicurrency", false)
		usd := testutil.GetDefaultCurrency(t, db)
		eur := testutil.CreateTestCurrency(t, db, "EUR", "Euro", "€", false)
		testutil.CreateTestBalance(t, db, user.ID, usd.ID, 100)
		testutil.CreateTestBalance(t, db, user.ID, eur.ID, 50)

		balances, err := svc.GetBalances(ctx, 11111)
		if err != nil {
			t.Fatalf("GetBalances() error = %v", err)
		}
		if len(balances) != 2 {
			t.Errorf("GetBalances() count = %v, want 2", len(balances))
		}
	})
}

func TestCoreService_GetDefaultCurrency(t *testing.T) {
	db, svc, _ := setupCoreServiceTest(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("returns default currency", func(t *testing.T) {
		currency, err := svc.GetDefaultCurrency(ctx)
		if err != nil {
			t.Fatalf("GetDefaultCurrency() error = %v", err)
		}
		if currency.Code != "USD" {
			t.Errorf("GetDefaultCurrency() code = %v, want USD", currency.Code)
		}
		if !currency.IsDefault {
			t.Error("GetDefaultCurrency() IsDefault = false, want true")
		}
	})
}

func TestCoreService_TransferMoney(t *testing.T) {
	ctx := context.Background()

	t.Run("transfers money successfully", func(t *testing.T) {
		db, svc, mockNotifier := setupCoreServiceTest(t)
		defer db.Close()

		notifications := make(map[int64]string)
		mockNotifier.NotifyUserFunc = func(ctx context.Context, telegramID int64, message string) error {
			notifications[telegramID] = message
			return nil
		}

		sender := testutil.CreateTestUser(t, db, 10001, "sender", false)
		receiver := testutil.CreateTestUser(t, db, 10002, "receiver", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, sender.ID, currency.ID, 100)

		err := svc.TransferMoney(ctx, 10001, "receiver", 50, "USD")
		if err != nil {
			t.Fatalf("TransferMoney() error = %v", err)
		}

		// Verify sender balance
		var senderBalance database.Balance
		db.Conn.Where("user_id = ?", sender.ID).First(&senderBalance)
		if senderBalance.Amount != 50 {
			t.Errorf("Sender balance = %v, want 50", senderBalance.Amount)
		}

		// Verify receiver balance
		var receiverBalance database.Balance
		db.Conn.Where("user_id = ?", receiver.ID).First(&receiverBalance)
		if receiverBalance.Amount != 50 {
			t.Errorf("Receiver balance = %v, want 50", receiverBalance.Amount)
		}

		// Verify both sender and receiver got notifications
		if len(notifications) != 2 {
			t.Errorf("Expected 2 notifications, got %d", len(notifications))
		}
		if _, ok := notifications[10001]; !ok {
			t.Error("Sender should have received notification")
		}
		if _, ok := notifications[10002]; !ok {
			t.Error("Receiver should have received notification")
		}
	})

	t.Run("rejects self-transfer", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		user := testutil.CreateTestUser(t, db, 20001, "selfuser", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, user.ID, currency.ID, 100)

		err := svc.TransferMoney(ctx, 20001, "selfuser", 50, "USD")
		if err == nil {
			t.Error("TransferMoney() should reject self-transfer")
		}
		if err.Error() != "cannot transfer to self" {
			t.Errorf("TransferMoney() error = %v, want 'cannot transfer to self'", err)
		}
	})

	t.Run("rejects amount less than 0.01", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		testutil.CreateTestUser(t, db, 30001, "sender2", false)
		testutil.CreateTestUser(t, db, 30002, "receiver2", false)

		err := svc.TransferMoney(ctx, 30001, "receiver2", 0.001, "USD")
		if err == nil {
			t.Error("TransferMoney() should reject amount < 0.01")
		}
		if err.Error() != "transfer amount must be at least 0.01" {
			t.Errorf("TransferMoney() error = %v", err)
		}
	})

	t.Run("rejects insufficient balance", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		sender := testutil.CreateTestUser(t, db, 40001, "poorsender", false)
		testutil.CreateTestUser(t, db, 40002, "richreceiver", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, sender.ID, currency.ID, 10)

		err := svc.TransferMoney(ctx, 40001, "richreceiver", 100, "USD")
		if err == nil {
			t.Error("TransferMoney() should reject insufficient balance")
		}
		if err.Error() != "insufficient balance" {
			t.Errorf("TransferMoney() error = %v, want 'insufficient balance'", err)
		}
	})

	t.Run("rejects transfer when sender lacks currency", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		testutil.CreateTestUser(t, db, 50001, "nocurrency", false)
		testutil.CreateTestUser(t, db, 50002, "receiver3", false)

		err := svc.TransferMoney(ctx, 50001, "receiver3", 10, "USD")
		if err == nil {
			t.Error("TransferMoney() should reject when sender has no balance")
		}
	})

	t.Run("creates receiver balance if not exists", func(t *testing.T) {
		db, svc, mockNotifier := setupCoreServiceTest(t)
		defer db.Close()

		mockNotifier.NotifyUserFunc = func(ctx context.Context, telegramID int64, message string) error {
			return nil
		}

		sender := testutil.CreateTestUser(t, db, 60001, "sender3", false)
		testutil.CreateTestUser(t, db, 60002, "newreceiver", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, sender.ID, currency.ID, 100)

		err := svc.TransferMoney(ctx, 60001, "newreceiver", 25, "USD")
		if err != nil {
			t.Fatalf("TransferMoney() error = %v", err)
		}

		// Verify receiver balance was created
		var receiverBalance database.Balance
		result := db.Conn.Joins("JOIN users ON balances.user_id = users.id").
			Where("users.username = ?", "newreceiver").
			First(&receiverBalance)
		if result.Error != nil {
			t.Fatalf("Failed to find receiver balance: %v", result.Error)
		}
		if receiverBalance.Amount != 25 {
			t.Errorf("Receiver balance = %v, want 25", receiverBalance.Amount)
		}
	})

	t.Run("creates transaction records", func(t *testing.T) {
		db, svc, mockNotifier := setupCoreServiceTest(t)
		defer db.Close()

		mockNotifier.NotifyUserFunc = func(ctx context.Context, telegramID int64, message string) error {
			return nil
		}

		sender := testutil.CreateTestUser(t, db, 70001, "txsender", false)
		receiver := testutil.CreateTestUser(t, db, 70002, "txreceiver", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, sender.ID, currency.ID, 100)

		err := svc.TransferMoney(ctx, 70001, "txreceiver", 30, "USD")
		if err != nil {
			t.Fatalf("TransferMoney() error = %v", err)
		}

		// Verify sender transaction
		var senderTx database.Transaction
		db.Conn.Where("user_id = ? AND type = ?", sender.ID, "transfer_out").First(&senderTx)
		if senderTx.Amount != -30 {
			t.Errorf("Sender transaction amount = %v, want -30", senderTx.Amount)
		}
		if senderTx.ToUsername != "txreceiver" {
			t.Errorf("Sender transaction ToUsername = %v, want txreceiver", senderTx.ToUsername)
		}

		// Verify receiver transaction
		var receiverTx database.Transaction
		db.Conn.Where("user_id = ? AND type = ?", receiver.ID, "transfer_in").First(&receiverTx)
		if receiverTx.Amount != 30 {
			t.Errorf("Receiver transaction amount = %v, want 30", receiverTx.Amount)
		}
		if receiverTx.FromUsername != "txsender" {
			t.Errorf("Receiver transaction FromUsername = %v, want txsender", receiverTx.FromUsername)
		}
	})

	t.Run("money conservation - sender decrease equals receiver increase", func(t *testing.T) {
		db, svc, mockNotifier := setupCoreServiceTest(t)
		defer db.Close()

		mockNotifier.NotifyUserFunc = func(ctx context.Context, telegramID int64, message string) error {
			return nil
		}

		sender := testutil.CreateTestUser(t, db, 80001, "conservesender", false)
		receiver := testutil.CreateTestUser(t, db, 80002, "conservereceiver", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, sender.ID, currency.ID, 100)
		testutil.CreateTestBalance(t, db, receiver.ID, currency.ID, 50)

		initialTotal := 150.0 // 100 + 50

		err := svc.TransferMoney(ctx, 80001, "conservereceiver", 35, "USD")
		if err != nil {
			t.Fatalf("TransferMoney() error = %v", err)
		}

		var senderBal, receiverBal database.Balance
		db.Conn.Where("user_id = ?", sender.ID).First(&senderBal)
		db.Conn.Where("user_id = ?", receiver.ID).First(&receiverBal)

		finalTotal := senderBal.Amount + receiverBal.Amount
		if finalTotal != initialTotal {
			t.Errorf("Money not conserved: initial=%v, final=%v (sender=%v, receiver=%v)",
				initialTotal, finalTotal, senderBal.Amount, receiverBal.Amount)
		}
	})

	t.Run("rejects transfer to non-existent recipient", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		sender := testutil.CreateTestUser(t, db, 81001, "existingsender", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, sender.ID, currency.ID, 100)

		err := svc.TransferMoney(ctx, 81001, "nonexistentuser", 10, "USD")
		if err == nil {
			t.Error("TransferMoney() should reject non-existent recipient")
		}
	})

	t.Run("transfer succeeds even when notification fails", func(t *testing.T) {
		db, svc, mockNotifier := setupCoreServiceTest(t)
		defer db.Close()

		mockNotifier.NotifyUserFunc = func(ctx context.Context, telegramID int64, message string) error {
			return errors.New("telegram API error")
		}

		sender := testutil.CreateTestUser(t, db, 82001, "notifysender", false)
		receiver := testutil.CreateTestUser(t, db, 82002, "notifyreceiver", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, sender.ID, currency.ID, 100)

		err := svc.TransferMoney(ctx, 82001, "notifyreceiver", 25, "USD")
		if err != nil {
			t.Fatalf("TransferMoney() should succeed despite notification failure: %v", err)
		}

		// Verify transfer completed
		var receiverBal database.Balance
		db.Conn.Where("user_id = ?", receiver.ID).First(&receiverBal)
		if receiverBal.Amount != 25 {
			t.Errorf("Transfer should complete: receiver balance = %v, want 25", receiverBal.Amount)
		}
	})
}

func TestCoreService_SetAdminStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("grants admin status", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		testutil.CreateTestUser(t, db, 83001, "newadmin", false)

		err := svc.SetAdminStatus(ctx, "newadmin", true)
		if err != nil {
			t.Fatalf("SetAdminStatus() error = %v", err)
		}

		var user database.User
		db.Conn.Where("username = ?", "newadmin").First(&user)
		if !user.IsAdmin {
			t.Error("SetAdminStatus() should grant admin status")
		}
	})

	t.Run("revokes admin status", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		testutil.CreateTestUser(t, db, 84001, "exadmin", true)

		err := svc.SetAdminStatus(ctx, "exadmin", false)
		if err != nil {
			t.Fatalf("SetAdminStatus() error = %v", err)
		}

		var user database.User
		db.Conn.Where("username = ?", "exadmin").First(&user)
		if user.IsAdmin {
			t.Error("SetAdminStatus() should revoke admin status")
		}
	})
}

func TestCoreService_ListUsersWithBalances(t *testing.T) {
	ctx := context.Background()

	t.Run("returns all users with their balances", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		user1 := testutil.CreateTestUser(t, db, 85001, "listuser1", false)
		user2 := testutil.CreateTestUser(t, db, 85002, "listuser2", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, user1.ID, currency.ID, 100)
		testutil.CreateTestBalance(t, db, user2.ID, currency.ID, 200)

		users, err := svc.ListUsersWithBalances(ctx)
		if err != nil {
			t.Fatalf("ListUsersWithBalances() error = %v", err)
		}

		if len(users) < 2 {
			t.Errorf("ListUsersWithBalances() returned %d users, want at least 2", len(users))
		}

		// Find our test users
		var found1, found2 bool
		for _, u := range users {
			if u.Username == "listuser1" && u.Balances["USD"] == 100 {
				found1 = true
			}
			if u.Username == "listuser2" && u.Balances["USD"] == 200 {
				found2 = true
			}
		}
		if !found1 {
			t.Error("ListUsersWithBalances() should include listuser1 with USD=100")
		}
		if !found2 {
			t.Error("ListUsersWithBalances() should include listuser2 with USD=200")
		}
	})
}

func TestCoreService_GetTransactionHistory(t *testing.T) {
	ctx := context.Background()

	t.Run("returns transaction history with total count", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		user := testutil.CreateTestUser(t, db, 80001, "historyuser", false)
		currency := testutil.GetDefaultCurrency(t, db)
		balance := testutil.CreateTestBalance(t, db, user.ID, currency.ID, 100)

		// Create test transaction
		tx := &database.Transaction{
			UserID:       user.ID,
			BalanceID:    balance.ID,
			Amount:       50,
			Type:         "transfer_in",
			FromUserID:   999,
			FromUsername: "sender",
			ToUserID:     user.ID,
			ToUsername:   "historyuser",
			Timestamp:    time.Now(),
			BalanceAfter: 150,
		}
		db.Conn.Create(tx)

		history, totalCount, err := svc.GetTransactionHistory(ctx, 80001, 0, 10)
		if err != nil {
			t.Fatalf("GetTransactionHistory() error = %v", err)
		}
		if len(history) != 1 {
			t.Errorf("GetTransactionHistory() count = %v, want 1", len(history))
		}
		if totalCount != 1 {
			t.Errorf("GetTransactionHistory() totalCount = %v, want 1", totalCount)
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		user := testutil.CreateTestUser(t, db, 80002, "manyhistory", false)
		currency := testutil.GetDefaultCurrency(t, db)
		balance := testutil.CreateTestBalance(t, db, user.ID, currency.ID, 1000)

		// Create 15 transactions
		for i := 0; i < 15; i++ {
			tx := &database.Transaction{
				UserID:       user.ID,
				BalanceID:    balance.ID,
				Amount:       float64(i + 1),
				Type:         "transfer_in",
				Timestamp:    time.Now().Add(time.Duration(i) * time.Minute),
				BalanceAfter: float64(100 + i),
			}
			db.Conn.Create(tx)
		}

		history, totalCount, err := svc.GetTransactionHistory(ctx, 80002, 0, 10)
		if err != nil {
			t.Fatalf("GetTransactionHistory() error = %v", err)
		}
		if len(history) != 10 {
			t.Errorf("GetTransactionHistory() count = %v, want 10 (limited)", len(history))
		}
		if totalCount != 15 {
			t.Errorf("GetTransactionHistory() totalCount = %v, want 15", totalCount)
		}
	})

	t.Run("respects offset for pagination", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		user := testutil.CreateTestUser(t, db, 80004, "offsetuser", false)
		currency := testutil.GetDefaultCurrency(t, db)
		balance := testutil.CreateTestBalance(t, db, user.ID, currency.ID, 1000)

		// Create 15 transactions with distinct amounts
		for i := 0; i < 15; i++ {
			tx := &database.Transaction{
				UserID:       user.ID,
				BalanceID:    balance.ID,
				Amount:       float64(15 - i), // 15, 14, 13, ... 1
				Type:         "transfer_in",
				Timestamp:    time.Now().Add(time.Duration(i) * time.Minute),
				BalanceAfter: float64(100 + i),
			}
			db.Conn.Create(tx)
		}

		// Get second page
		history, totalCount, err := svc.GetTransactionHistory(ctx, 80004, 10, 10)
		if err != nil {
			t.Fatalf("GetTransactionHistory() error = %v", err)
		}
		if len(history) != 5 {
			t.Errorf("GetTransactionHistory() count = %v, want 5 (remaining)", len(history))
		}
		if totalCount != 15 {
			t.Errorf("GetTransactionHistory() totalCount = %v, want 15", totalCount)
		}
	})

	t.Run("orders by timestamp descending", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		user := testutil.CreateTestUser(t, db, 80003, "orderuser", false)
		currency := testutil.GetDefaultCurrency(t, db)
		balance := testutil.CreateTestBalance(t, db, user.ID, currency.ID, 100)

		// Create transactions at different times
		tx1 := &database.Transaction{UserID: user.ID, BalanceID: balance.ID, Amount: 1, Timestamp: time.Now().Add(-2 * time.Hour)}
		tx2 := &database.Transaction{UserID: user.ID, BalanceID: balance.ID, Amount: 2, Timestamp: time.Now().Add(-1 * time.Hour)}
		tx3 := &database.Transaction{UserID: user.ID, BalanceID: balance.ID, Amount: 3, Timestamp: time.Now()}
		db.Conn.Create(tx1)
		db.Conn.Create(tx2)
		db.Conn.Create(tx3)

		history, _, _ := svc.GetTransactionHistory(ctx, 80003, 0, 10)
		if history[0].Amount != 3 {
			t.Errorf("First transaction amount = %v, want 3 (most recent)", history[0].Amount)
		}
	})
}

func TestCoreService_AdminSetBalance(t *testing.T) {
	ctx := context.Background()

	t.Run("admin can set balance and user is notified", func(t *testing.T) {
		db, svc, mockNotifier := setupCoreServiceTest(t)
		defer db.Close()

		var notifiedUser int64
		var notificationMsg string
		mockNotifier.NotifyUserFunc = func(ctx context.Context, telegramID int64, message string) error {
			notifiedUser = telegramID
			notificationMsg = message
			return nil
		}

		admin := testutil.CreateTestUser(t, db, 90001, "admin", true)
		target := testutil.CreateTestUser(t, db, 90002, "target", false)
		currency := testutil.GetDefaultCurrency(t, db)
		testutil.CreateTestBalance(t, db, target.ID, currency.ID, 0)

		err := svc.AdminSetBalance(ctx, admin.TelegramID, "target", 500, "USD")
		if err != nil {
			t.Fatalf("AdminSetBalance() error = %v", err)
		}

		var balance database.Balance
		db.Conn.Where("user_id = ?", target.ID).First(&balance)
		if balance.Amount != 500 {
			t.Errorf("AdminSetBalance() balance = %v, want 500", balance.Amount)
		}

		// Verify notification was sent to target user
		if notifiedUser != target.TelegramID {
			t.Errorf("Notification sent to %d, want %d", notifiedUser, target.TelegramID)
		}
		if notificationMsg == "" {
			t.Error("Notification message should not be empty")
		}
	})

	t.Run("non-admin cannot set balance", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		nonAdmin := testutil.CreateTestUser(t, db, 91001, "nonadmin", false)
		testutil.CreateTestUser(t, db, 91002, "target2", false)

		err := svc.AdminSetBalance(ctx, nonAdmin.TelegramID, "target2", 500, "USD")
		if err == nil {
			t.Error("AdminSetBalance() should reject non-admin")
		}
		if err.Error() != "unauthorized" {
			t.Errorf("AdminSetBalance() error = %v, want 'unauthorized'", err)
		}
	})

	t.Run("creates balance if not exists", func(t *testing.T) {
		db, svc, _ := setupCoreServiceTest(t)
		defer db.Close()

		admin := testutil.CreateTestUser(t, db, 92001, "admin2", true)
		testutil.CreateTestUser(t, db, 92002, "newbalanceuser", false)

		err := svc.AdminSetBalance(ctx, admin.TelegramID, "newbalanceuser", 1000, "USD")
		if err != nil {
			t.Fatalf("AdminSetBalance() error = %v", err)
		}

		var count int64
		db.Conn.Model(&database.Balance{}).Joins("JOIN users ON balances.user_id = users.id").
			Where("users.username = ?", "newbalanceuser").Count(&count)
		if count != 1 {
			t.Errorf("AdminSetBalance() should create balance, got count = %v", count)
		}
	})
}

func TestCoreService_AddCurrency(t *testing.T) {
	db, svc, _ := setupCoreServiceTest(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("adds new currency", func(t *testing.T) {
		err := svc.AddCurrency(ctx, "GBP", "British Pound", "£")
		if err != nil {
			t.Fatalf("AddCurrency() error = %v", err)
		}

		var currency database.Currency
		db.Conn.Where("code = ?", "GBP").First(&currency)
		if currency.Name != "British Pound" {
			t.Errorf("AddCurrency() name = %v, want British Pound", currency.Name)
		}
		if currency.Sign != "£" {
			t.Errorf("AddCurrency() sign = %v, want £", currency.Sign)
		}
	})

	t.Run("rejects duplicate code", func(t *testing.T) {
		svc.AddCurrency(ctx, "JPY", "Japanese Yen", "¥")
		err := svc.AddCurrency(ctx, "JPY", "Duplicate Yen", "Y")
		if err == nil {
			t.Error("AddCurrency() should reject duplicate code")
		}
	})
}

func TestCoreService_SetDefaultCurrency(t *testing.T) {
	db, svc, _ := setupCoreServiceTest(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("changes default currency", func(t *testing.T) {
		testutil.CreateTestCurrency(t, db, "CHF", "Swiss Franc", "Fr", false)

		err := svc.SetDefaultCurrency(ctx, "CHF")
		if err != nil {
			t.Fatalf("SetDefaultCurrency() error = %v", err)
		}

		// Verify old default is unset
		var oldDefault database.Currency
		db.Conn.Where("code = ?", "USD").First(&oldDefault)
		if oldDefault.IsDefault {
			t.Error("USD should no longer be default")
		}

		// Verify new default
		var newDefault database.Currency
		db.Conn.Where("code = ?", "CHF").First(&newDefault)
		if !newDefault.IsDefault {
			t.Error("CHF should be new default")
		}
	})
}

func TestCoreService_GetPreviousRecipients(t *testing.T) {
	db, svc, _ := setupCoreServiceTest(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("returns previous recipients", func(t *testing.T) {
		user := testutil.CreateTestUser(t, db, 100001, "sender", false)
		currency := testutil.GetDefaultCurrency(t, db)
		balance := testutil.CreateTestBalance(t, db, user.ID, currency.ID, 100)

		// Create transfer_out transactions
		recipients := []string{"alice", "bob", "charlie"}
		for i, name := range recipients {
			tx := &database.Transaction{
				UserID:     user.ID,
				BalanceID:  balance.ID,
				Amount:     -10,
				Type:       "transfer_out",
				ToUsername: name,
				Timestamp:  time.Now().Add(time.Duration(i) * time.Minute),
			}
			db.Conn.Create(tx)
		}

		result, err := svc.GetPreviousRecipients(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetPreviousRecipients() error = %v", err)
		}
		if len(result) != 3 {
			t.Errorf("GetPreviousRecipients() count = %v, want 3", len(result))
		}
	})

	t.Run("returns empty for user with no transfers", func(t *testing.T) {
		user := testutil.CreateTestUser(t, db, 100002, "notransfers", false)

		result, err := svc.GetPreviousRecipients(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetPreviousRecipients() error = %v", err)
		}
		if len(result) != 0 {
			t.Errorf("GetPreviousRecipients() count = %v, want 0", len(result))
		}
	})
}

func TestCoreService_DisableUser(t *testing.T) {
	db, svc, _ := setupCoreServiceTest(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("soft deletes user", func(t *testing.T) {
		testutil.CreateTestUser(t, db, 110001, "todisable", false)

		err := svc.DisableUser(ctx, "todisable")
		if err != nil {
			t.Fatalf("DisableUser() error = %v", err)
		}

		// User should not be found with normal query
		var user database.User
		result := db.Conn.Where("username = ?", "todisable").First(&user)
		if result.Error == nil {
			t.Error("DisableUser() user should be soft deleted")
		}

		// But should exist with Unscoped
		result = db.Conn.Unscoped().Where("username = ?", "todisable").First(&user)
		if result.Error != nil {
			t.Error("DisableUser() user should exist with Unscoped")
		}
	})
}

func TestCoreService_DestroyUser(t *testing.T) {
	db, svc, _ := setupCoreServiceTest(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("hard deletes user and related data", func(t *testing.T) {
		user := testutil.CreateTestUser(t, db, 120001, "todestroy", false)
		currency := testutil.GetDefaultCurrency(t, db)
		balance := testutil.CreateTestBalance(t, db, user.ID, currency.ID, 100)
		testutil.CreateTestTransaction(t, db, &database.Transaction{
			UserID:    user.ID,
			BalanceID: balance.ID,
			Amount:    50,
			Type:      "transfer_in",
		})

		err := svc.DestroyUser(ctx, "todestroy")
		if err != nil {
			t.Fatalf("DestroyUser() error = %v", err)
		}

		// Verify user is gone (even with Unscoped)
		var count int64
		db.Conn.Unscoped().Model(&database.User{}).Where("username = ?", "todestroy").Count(&count)
		if count != 0 {
			t.Error("DestroyUser() user should be hard deleted")
		}

		// Verify balances are gone
		db.Conn.Unscoped().Model(&database.Balance{}).Where("user_id = ?", user.ID).Count(&count)
		if count != 0 {
			t.Error("DestroyUser() balances should be deleted")
		}

		// Verify transactions are gone
		db.Conn.Unscoped().Model(&database.Transaction{}).Where("user_id = ?", user.ID).Count(&count)
		if count != 0 {
			t.Error("DestroyUser() transactions should be deleted")
		}
	})
}
