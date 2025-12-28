// File: ./internal/webapp/web_service.go
package webapp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/logger"
	"github.com/fitz123/mcduck-wallet/internal/messages"
	"github.com/fitz123/mcduck-wallet/internal/services"
	"github.com/fitz123/mcduck-wallet/internal/webapp/views"
)

type WebService struct {
	userService services.UserService
	coreService services.CoreService
	authService *AuthService
}

func NewWebService(userService services.UserService, coreService services.CoreService, botToken string) *WebService {
	return &WebService{
		userService: userService,
		coreService: coreService,
		authService: NewAuthService(botToken),
	}
}

func (ws *WebService) ServeHome(w http.ResponseWriter, r *http.Request) {
	component := views.InitialLoadingIndex()
	_ = component.Render(r.Context(), w)
}

func (ws *WebService) GetTransferForm(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())

	user, err := ws.userService.GetUser(r.Context(), userID)
	if err != nil {
		logger.Error("Failed to get user", "error", err)
		http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	balances, err := ws.coreService.GetBalances(r.Context(), userID)
	if err != nil {
		logger.Error("Failed to get balances", "error", err)
		http.Error(w, "Failed to fetch balances", http.StatusInternalServerError)
		return
	}

	previousRecipients, err := ws.coreService.GetPreviousRecipients(r.Context(), user.ID)
	if err != nil {
		logger.Error("Failed to get previous recipients", "error", err)
		previousRecipients = []string{}
	}

	component := views.TransferForm(balances, previousRecipients, user)
	if err := component.Render(r.Context(), w); err != nil {
		logger.Error("Error rendering transfer form", "error", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

func (ws *WebService) GetDashboard(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	user, err := ws.userService.GetUser(r.Context(), userID)
	if err != nil {
		logger.Error(messages.ErrUserNotFound, "error", err)
		http.Error(w, messages.ErrUserNotFound, http.StatusInternalServerError)
		return
	}

	component := views.MainContent(user, "", true)
	if err := component.Render(r.Context(), w); err != nil {
		logger.Error("Error rendering dashboard", "error", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

func (ws *WebService) TransferMoney(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	r.ParseForm()

	toUsername, amount, currencyCode, err := ws.parseTransferFormValues(r)
	if err != nil {
		ws.handleResponse(w, r, userID, Response{
			Message:    err.Error(),
			Error:      err,
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	// Get currency ID for updating last used currency
	currency, err := ws.coreService.GetCurrencyByCode(r.Context(), currencyCode)
	if err != nil {
		ws.handleResponse(w, r, userID, Response{
			Message:    "Failed to get currency information",
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// Update last used currency
	err = ws.userService.UpdateLastUsedCurrency(r.Context(), userID, currency.ID)
	if err != nil {
		logger.Error("Failed to update last used currency", "error", err)
		// Don't return error to user, just log it
	}

	err = ws.coreService.TransferMoney(r.Context(), userID, toUsername, amount, currencyCode)
	if err != nil {
		ws.handleResponse(w, r, userID, Response{
			Message:    "Transfer failed",
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	ws.handleResponse(w, r, userID, Response{
		Message: fmt.Sprintf(messages.InfoTransferSuccessful, amount, currencyCode, toUsername),
	})
}

func (ws *WebService) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())

	// Parse pagination params
	offset := 0
	limit := 10
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	transactions, totalCount, err := ws.coreService.GetTransactionHistory(r.Context(), userID, offset, limit)
	if err != nil {
		logger.Error("Failed to get transaction history", "error", err)
		http.Error(w, "Failed to fetch transaction history", http.StatusInternalServerError)
		return
	}

	// Check if this is an HTMX request for infinite scroll
	isHTMX := r.Header.Get("HX-Request") == "true"
	hasMore := int64(offset+len(transactions)) < totalCount

	if isHTMX {
		// Return only the transaction items for infinite scroll
		component := views.TransactionItems(transactions, offset+limit, hasMore)
		templ.Handler(component).ServeHTTP(w, r)
	} else {
		// Return full page
		component := views.TransactionHistory(transactions, offset+limit, hasMore)
		templ.Handler(component).ServeHTTP(w, r)
	}
}

func (ws *WebService) AuthMiddleware(next http.Handler) http.Handler {
	return ws.authService.AuthMiddleware(next)
}

func (ws *WebService) GetExchangeForm(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())

	user, err := ws.userService.GetUser(r.Context(), userID)
	if err != nil {
		logger.Error("Failed to get user", "error", err)
		http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	balances, err := ws.coreService.GetBalances(r.Context(), userID)
	if err != nil {
		logger.Error("Failed to get balances", "error", err)
		http.Error(w, "Failed to fetch balances", http.StatusInternalServerError)
		return
	}

	currencies, err := ws.coreService.ListCurrencies(r.Context())
	if err != nil {
		logger.Error("Failed to get currencies", "error", err)
		http.Error(w, "Failed to fetch currencies", http.StatusInternalServerError)
		return
	}

	component := views.ExchangeForm(balances, currencies, user)
	if err := component.Render(r.Context(), w); err != nil {
		logger.Error("Error rendering exchange form", "error", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

func (ws *WebService) GetExchangePreview(w http.ResponseWriter, r *http.Request) {
	fromCurrency := strings.ToUpper(r.URL.Query().Get("from_currency"))
	toCurrency := strings.ToUpper(r.URL.Query().Get("to_currency"))
	amountStr := r.URL.Query().Get("amount")

	// Validate inputs
	if fromCurrency == "" || toCurrency == "" || amountStr == "" {
		component := views.ExchangePreviewError("Enter amount to see preview")
		component.Render(r.Context(), w)
		return
	}

	if fromCurrency == toCurrency {
		component := views.ExchangePreviewError("Select different currencies")
		component.Render(r.Context(), w)
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount < 0.01 {
		component := views.ExchangePreviewError("Enter valid amount")
		component.Render(r.Context(), w)
		return
	}

	// Get exchange rate
	rate, err := ws.coreService.GetExchangeRate(r.Context(), fromCurrency, toCurrency)
	if err != nil {
		logger.Error("Failed to get exchange rate", "error", err)
		component := views.ExchangePreviewError("Unable to get exchange rate")
		component.Render(r.Context(), w)
		return
	}

	toAmount := amount * rate
	component := views.ExchangePreview(amount, fromCurrency, toAmount, toCurrency, rate)
	component.Render(r.Context(), w)
}

func (ws *WebService) ExchangeMoney(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	r.ParseForm()

	fromCurrency := strings.ToUpper(r.FormValue("from_currency"))
	toCurrency := strings.ToUpper(r.FormValue("to_currency"))
	amountStr := r.FormValue("amount")

	if fromCurrency == "" || toCurrency == "" {
		ws.handleResponse(w, r, userID, Response{
			Message:    "Both currencies are required",
			Error:      fmt.Errorf("both currencies are required"),
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	if fromCurrency == toCurrency {
		ws.handleResponse(w, r, userID, Response{
			Message:    "Cannot exchange to same currency",
			Error:      fmt.Errorf("cannot exchange to same currency"),
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount < 0.01 {
		ws.handleResponse(w, r, userID, Response{
			Message:    "Invalid amount (minimum 0.01)",
			Error:      fmt.Errorf("invalid amount"),
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	err = ws.coreService.ExchangeMoney(r.Context(), userID, fromCurrency, toCurrency, amount)
	if err != nil {
		ws.handleResponse(w, r, userID, Response{
			Message:    "Exchange failed",
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	ws.handleResponse(w, r, userID, Response{
		Message: fmt.Sprintf("Successfully exchanged %.2f %s to %s", amount, fromCurrency, toCurrency),
	})
}

func (ws *WebService) GetAddCurrencyForm(w http.ResponseWriter, r *http.Request) {
	component := views.AddCurrencyForm()
	if err := component.Render(r.Context(), w); err != nil {
		logger.Error("Error rendering add currency form", "error", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

func (ws *WebService) AddCurrency(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())
	r.ParseForm()

	code := strings.ToUpper(r.FormValue("code"))
	name := r.FormValue("name")
	sign := r.FormValue("sign")
	isReal := r.FormValue("is_real") == "true"
	fixedRate := 0.0
	if rateStr := r.FormValue("fixed_rate"); rateStr != "" {
		if rate, err := strconv.ParseFloat(rateStr, 64); err == nil {
			fixedRate = rate
		}
	}

	err := ws.coreService.AddCurrency(r.Context(), code, name, sign, isReal, fixedRate)
	if err != nil {
		ws.handleResponse(w, r, userID, Response{
			Message:    "Failed to add currency",
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	ws.handleResponse(w, r, userID, Response{
		Message: fmt.Sprintf("Currency %s (%s) with sign %s has been successfully added.", code, name, sign),
	})
}

// Helper functions

func (ws *WebService) parseTransferFormValues(r *http.Request) (string, float64, string, error) {
	toUsername := strings.TrimPrefix(r.FormValue("to_username"), "@")
	toUsername = strings.ToLower(toUsername)
	if toUsername == "" {
		return "", 0, "", fmt.Errorf("Recipient username is required")
	}

	amountStr := r.FormValue("amount")
	if amountStr == "" {
		return "", 0, "", fmt.Errorf("Amount is required")
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return "", 0, "", fmt.Errorf("Invalid amount")
	}

	if amount < 0.01 {
		return "", 0, "", fmt.Errorf("Amount must be at least 0.01")
	}

	currencyCode := r.FormValue("currency")
	if currencyCode == "" {
		defaultCurrency, err := ws.coreService.GetDefaultCurrency(r.Context())
		if err != nil {
			return "", 0, "", fmt.Errorf("Failed to get default currency")
		}
		currencyCode = defaultCurrency.Code
	}

	return toUsername, amount, currencyCode, nil
}

type Response struct {
	Message    string
	Error      error
	StatusCode int
}

func (ws *WebService) handleResponse(w http.ResponseWriter, r *http.Request, userID int64, response Response) {
	success := response.Error == nil
	message := response.Message

	if !success {
		logger.Error(response.Message, "error", response.Error)
		message = response.Error.Error()
	}

	user, err := ws.userService.GetUser(r.Context(), userID)
	if err != nil {
		logger.Error("Failed to get user", "error", err)
		user = &database.User{}
		message = "Failed to fetch user data"
		success = false
	}

	component := views.MainContent(user, message, success)
	if err := component.Render(r.Context(), w); err != nil {
		logger.Error("Error rendering response", "error", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
