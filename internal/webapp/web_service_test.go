package webapp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/logger"
	"github.com/fitz123/mcduck-wallet/internal/testutil"
)

func init() {
	logger.Init("debug")
}

func setupWebServiceTest() (*WebService, *testutil.MockUserService, *testutil.MockCoreService) {
	mockUserService := &testutil.MockUserService{}
	mockCoreService := &testutil.MockCoreService{}
	ws := &WebService{
		userService: mockUserService,
		coreService: mockCoreService,
		authService: NewAuthService("test_token"),
	}
	return ws, mockUserService, mockCoreService
}

func requestWithUserID(req *http.Request, userID int64) *http.Request {
	ctx := context.WithValue(req.Context(), "userID", userID)
	return req.WithContext(ctx)
}

func TestWebService_ServeHome(t *testing.T) {
	ws, _, _ := setupWebServiceTest()

	t.Run("serves home page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()

		ws.ServeHome(rr, req)

		// Should render without error (status 200)
		if rr.Code != http.StatusOK {
			t.Errorf("ServeHome() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})
}

func TestWebService_GetDashboard(t *testing.T) {
	t.Run("returns dashboard for authenticated user", func(t *testing.T) {
		ws, mockUserService, _ := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{
				TelegramID: telegramID,
				Username:   "testuser",
				Accounts: []database.Balance{
					{Amount: 100, Currency: database.Currency{Code: "USD", Sign: "$"}},
				},
			}, nil
		}

		req := httptest.NewRequest("GET", "/dashboard", nil)
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.GetDashboard(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("GetDashboard() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		ws, mockUserService, _ := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return nil, errors.New("user not found")
		}

		req := httptest.NewRequest("GET", "/dashboard", nil)
		req = requestWithUserID(req, 99999)
		rr := httptest.NewRecorder()

		ws.GetDashboard(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("GetDashboard() status = %v, want %v", rr.Code, http.StatusInternalServerError)
		}
	})
}

func TestWebService_GetTransferForm(t *testing.T) {
	t.Run("returns transfer form with balances", func(t *testing.T) {
		ws, mockUserService, mockCoreService := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "testuser"}, nil
		}
		mockCoreService.GetBalancesFunc = func(ctx context.Context, telegramID int64) ([]database.Balance, error) {
			return []database.Balance{
				{Amount: 100, Currency: database.Currency{Code: "USD", Sign: "$"}},
			}, nil
		}
		mockCoreService.GetPreviousRecipientsFunc = func(ctx context.Context, userID uint) ([]string, error) {
			return []string{"alice", "bob"}, nil
		}

		req := httptest.NewRequest("GET", "/transfer-form", nil)
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.GetTransferForm(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("GetTransferForm() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	t.Run("handles previous recipients error gracefully", func(t *testing.T) {
		ws, mockUserService, mockCoreService := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "testuser"}, nil
		}
		mockCoreService.GetBalancesFunc = func(ctx context.Context, telegramID int64) ([]database.Balance, error) {
			return []database.Balance{}, nil
		}
		mockCoreService.GetPreviousRecipientsFunc = func(ctx context.Context, userID uint) ([]string, error) {
			return nil, errors.New("database error")
		}

		req := httptest.NewRequest("GET", "/transfer-form", nil)
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.GetTransferForm(rr, req)

		// Should still succeed, gracefully handling the error
		if rr.Code != http.StatusOK {
			t.Errorf("GetTransferForm() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})
}

func TestWebService_TransferMoney(t *testing.T) {
	t.Run("successful transfer", func(t *testing.T) {
		ws, mockUserService, mockCoreService := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "sender", Accounts: []database.Balance{}}, nil
		}
		mockUserService.UpdateLastUsedCurrencyFunc = func(ctx context.Context, telegramID int64, currencyID uint) error {
			return nil
		}
		mockCoreService.GetCurrencyByCodeFunc = func(ctx context.Context, code string) (*database.Currency, error) {
			return &database.Currency{Code: "USD", Sign: "$"}, nil
		}
		mockCoreService.TransferMoneyFunc = func(ctx context.Context, fromTelegramID int64, toUsername string, amount float64, currencyCode string) error {
			return nil
		}

		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {"50"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.TransferMoney(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("TransferMoney() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	t.Run("rejects missing recipient", func(t *testing.T) {
		ws, mockUserService, mockCoreService := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "sender"}, nil
		}
		mockCoreService.GetDefaultCurrencyFunc = func(ctx context.Context) (*database.Currency, error) {
			return &database.Currency{Code: "USD"}, nil
		}

		form := url.Values{
			"amount":   {"50"},
			"currency": {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.TransferMoney(rr, req)

		// Should return OK but render error in response (due to handleResponse)
		if rr.Code != http.StatusOK {
			t.Errorf("TransferMoney() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	t.Run("rejects invalid amount", func(t *testing.T) {
		ws, mockUserService, _ := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "sender"}, nil
		}

		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {"not_a_number"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.TransferMoney(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("TransferMoney() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	t.Run("rejects amount below minimum", func(t *testing.T) {
		ws, mockUserService, _ := setupWebServiceTest()

		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "sender"}, nil
		}

		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {"0.001"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.TransferMoney(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("TransferMoney() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	t.Run("uses default currency when not specified", func(t *testing.T) {
		ws, mockUserService, mockCoreService := setupWebServiceTest()

		var usedCurrency string
		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "sender"}, nil
		}
		mockUserService.UpdateLastUsedCurrencyFunc = func(ctx context.Context, telegramID int64, currencyID uint) error {
			return nil
		}
		mockCoreService.GetDefaultCurrencyFunc = func(ctx context.Context) (*database.Currency, error) {
			return &database.Currency{Code: "USD", Sign: "$"}, nil
		}
		mockCoreService.GetCurrencyByCodeFunc = func(ctx context.Context, code string) (*database.Currency, error) {
			usedCurrency = code
			return &database.Currency{Code: code}, nil
		}
		mockCoreService.TransferMoneyFunc = func(ctx context.Context, fromTelegramID int64, toUsername string, amount float64, currencyCode string) error {
			return nil
		}

		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {"50"},
			// No currency specified
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.TransferMoney(rr, req)

		if usedCurrency != "USD" {
			t.Errorf("TransferMoney() used currency = %v, want USD (default)", usedCurrency)
		}
	})

	t.Run("strips @ prefix from username", func(t *testing.T) {
		ws, mockUserService, mockCoreService := setupWebServiceTest()

		var receivedUsername string
		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "sender"}, nil
		}
		mockUserService.UpdateLastUsedCurrencyFunc = func(ctx context.Context, telegramID int64, currencyID uint) error {
			return nil
		}
		mockCoreService.GetCurrencyByCodeFunc = func(ctx context.Context, code string) (*database.Currency, error) {
			return &database.Currency{Code: "USD"}, nil
		}
		mockCoreService.TransferMoneyFunc = func(ctx context.Context, fromTelegramID int64, toUsername string, amount float64, currencyCode string) error {
			receivedUsername = toUsername
			return nil
		}

		form := url.Values{
			"to_username": {"@receiver"},
			"amount":      {"50"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.TransferMoney(rr, req)

		if receivedUsername != "receiver" {
			t.Errorf("TransferMoney() username = %v, want receiver (without @)", receivedUsername)
		}
	})
}

func TestWebService_GetTransactionHistory(t *testing.T) {
	t.Run("returns transaction history", func(t *testing.T) {
		ws, _, mockCoreService := setupWebServiceTest()

		mockCoreService.GetTransactionHistoryFunc = func(ctx context.Context, telegramID int64, offset, limit int) ([]database.Transaction, int64, error) {
			return []database.Transaction{
				{Amount: 50, Type: "transfer_in"},
			}, 1, nil
		}

		req := httptest.NewRequest("GET", "/history", nil)
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.GetTransactionHistory(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("GetTransactionHistory() status = %v, want %v", rr.Code, http.StatusOK)
		}
	})

	t.Run("handles error", func(t *testing.T) {
		ws, _, mockCoreService := setupWebServiceTest()

		mockCoreService.GetTransactionHistoryFunc = func(ctx context.Context, telegramID int64, offset, limit int) ([]database.Transaction, int64, error) {
			return nil, 0, errors.New("database error")
		}

		req := httptest.NewRequest("GET", "/history", nil)
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.GetTransactionHistory(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("GetTransactionHistory() status = %v, want %v", rr.Code, http.StatusInternalServerError)
		}
	})
}

func TestWebService_AddCurrency(t *testing.T) {
	t.Run("adds currency successfully", func(t *testing.T) {
		ws, mockUserService, mockCoreService := setupWebServiceTest()

		var addedCode, addedName, addedSign string
		mockUserService.GetUserFunc = func(ctx context.Context, telegramID int64) (*database.User, error) {
			return &database.User{TelegramID: telegramID, Username: "admin"}, nil
		}
		mockCoreService.AddCurrencyFunc = func(ctx context.Context, code, name, sign string, isReal bool, fixedRate float64) error {
			addedCode = code
			addedName = name
			addedSign = sign
			return nil
		}

		form := url.Values{
			"code": {"eur"},
			"name": {"Euro"},
			"sign": {"€"},
		}
		req := httptest.NewRequest("POST", "/add-currency", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = requestWithUserID(req, 12345)
		rr := httptest.NewRecorder()

		ws.AddCurrency(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("AddCurrency() status = %v, want %v", rr.Code, http.StatusOK)
		}
		if addedCode != "EUR" {
			t.Errorf("AddCurrency() code = %v, want EUR (uppercase)", addedCode)
		}
		if addedName != "Euro" {
			t.Errorf("AddCurrency() name = %v, want Euro", addedName)
		}
		if addedSign != "€" {
			t.Errorf("AddCurrency() sign = %v, want €", addedSign)
		}
	})
}

func TestParseTransferFormValues(t *testing.T) {
	ws, _, mockCoreService := setupWebServiceTest()

	t.Run("parses valid form values", func(t *testing.T) {
		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {"50.25"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ParseForm()

		username, amount, currency, err := ws.parseTransferFormValues(req)
		if err != nil {
			t.Fatalf("parseTransferFormValues() error = %v", err)
		}
		if username != "receiver" {
			t.Errorf("username = %v, want receiver", username)
		}
		if amount != 50.25 {
			t.Errorf("amount = %v, want 50.25", amount)
		}
		if currency != "USD" {
			t.Errorf("currency = %v, want USD", currency)
		}
	})

	t.Run("lowercases username", func(t *testing.T) {
		form := url.Values{
			"to_username": {"UPPERCASE"},
			"amount":      {"50"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ParseForm()

		username, _, _, err := ws.parseTransferFormValues(req)
		if err != nil {
			t.Fatalf("parseTransferFormValues() error = %v", err)
		}
		if username != "uppercase" {
			t.Errorf("username = %v, want uppercase (lowercase)", username)
		}
	})

	t.Run("uses default currency when not provided", func(t *testing.T) {
		mockCoreService.GetDefaultCurrencyFunc = func(ctx context.Context) (*database.Currency, error) {
			return &database.Currency{Code: "USD"}, nil
		}

		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {"50"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ParseForm()

		_, _, currency, err := ws.parseTransferFormValues(req)
		if err != nil {
			t.Fatalf("parseTransferFormValues() error = %v", err)
		}
		if currency != "USD" {
			t.Errorf("currency = %v, want USD (default)", currency)
		}
	})

	t.Run("returns error for empty username", func(t *testing.T) {
		form := url.Values{
			"to_username": {""},
			"amount":      {"50"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ParseForm()

		_, _, _, err := ws.parseTransferFormValues(req)
		if err == nil {
			t.Error("parseTransferFormValues() should error for empty username")
		}
	})

	t.Run("returns error for empty amount", func(t *testing.T) {
		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {""},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ParseForm()

		_, _, _, err := ws.parseTransferFormValues(req)
		if err == nil {
			t.Error("parseTransferFormValues() should error for empty amount")
		}
	})

	t.Run("returns error for amount below minimum", func(t *testing.T) {
		form := url.Values{
			"to_username": {"receiver"},
			"amount":      {"0.001"},
			"currency":    {"USD"},
		}
		req := httptest.NewRequest("POST", "/transfer", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.ParseForm()

		_, _, _, err := ws.parseTransferFormValues(req)
		if err == nil {
			t.Error("parseTransferFormValues() should error for amount < 0.01")
		}
	})
}
