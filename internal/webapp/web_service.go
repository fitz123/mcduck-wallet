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

func (ws *WebService) GetTransferForm(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r.Context())

	balances, err := ws.coreService.GetBalances(r.Context(), userID)
	if err != nil {
		logger.Error("Failed to get balances", "error", err)
		http.Error(w, "Failed to fetch balances", http.StatusInternalServerError)
		return
	}

	component := views.TransferForm(balances)
	if err := component.Render(r.Context(), w); err != nil {
		logger.Error("Error rendering transfer form", "error", err)
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

	transactions, err := ws.coreService.GetTransactionHistory(r.Context(), userID)
	if err != nil {
		logger.Error("Failed to get transaction history", "error", err)
		http.Error(w, "Failed to fetch transaction history", http.StatusInternalServerError)
		return
	}

	component := views.TransactionHistory(transactions)
	templ.Handler(component).ServeHTTP(w, r)
}

func (ws *WebService) AuthMiddleware(next http.Handler) http.Handler {
	return ws.authService.AuthMiddleware(next)
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

	err := ws.coreService.AddCurrency(r.Context(), code, name, sign)
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
