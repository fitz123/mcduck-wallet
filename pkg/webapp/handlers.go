package webapp

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
	"github.com/fitz123/mcduck-wallet/pkg/webapp/views"
)

type webContext struct {
	r      *http.Request
	userID int64
}

func (wc *webContext) GetUserID() int64 {
	return wc.userID
}

func ServeHTML(w http.ResponseWriter, r *http.Request) {
	component := views.Index()
	err := component.Render(r.Context(), w)
	if err != nil {
		logger.Error("ServeHTML: Error rendering index", "error", err)
		http.Error(w, "Error rendering index", http.StatusInternalServerError)
		return
	}
}

func GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{r: r, userID: userID}

	balances, err := core.GetBalance(ctx)
	if err != nil {
		logger.Error("Failed to get balances", "error", err, "userID", userID)
		http.Error(w, "Error fetching balances", http.StatusInternalServerError)
		return
	}

	// Convert balances to a slice of BalanceWithCurrency
	balancesWithCurrency := make([]views.BalanceWithCurrency, 0, len(balances))
	for currencyCode, amount := range balances {
		currency, err := database.GetCurrencyByCode(currencyCode)
		if err != nil {
			logger.Error("Failed to get currency", "error", err, "currencyCode", currencyCode)
			continue
		}
		balancesWithCurrency = append(balancesWithCurrency, views.BalanceWithCurrency{
			Amount:   amount,
			Currency: currency,
		})
	}

	component := views.Balances(balancesWithCurrency)
	err = component.Render(r.Context(), w)
	if err != nil {
		logger.Error("Error rendering balances template", "error", err)
		http.Error(w, "Error rendering balances", http.StatusInternalServerError)
		return
	}
}

func TransferMoney(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{r: r, userID: userID}

	toUsername := r.FormValue("to_username")
	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	// Get the currency from the form, or use default if not provided
	currencyCode := r.FormValue("currency")
	if currencyCode == "" {
		defaultCurrency, err := database.GetDefaultCurrency()
		if err != nil {
			logger.Error("Failed to get default currency", "error", err)
			http.Error(w, "Failed to process transfer", http.StatusInternalServerError)
			return
		}
		currencyCode = defaultCurrency.Code
	}

	err = core.TransferMoney(ctx, toUsername, amount, currencyCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the currency details for the response message
	currency, err := database.GetCurrencyByCode(currencyCode)
	if err != nil {
		logger.Error("Failed to get currency details", "error", err, "currencyCode", currencyCode)
		http.Error(w, "Transfer successful, but failed to get currency details", http.StatusInternalServerError)
		return
	}

	response := fmt.Sprintf(messages.InfoTransferSuccessful, currency.Sign, amount, currency.Code, toUsername)
	w.Write([]byte(response))
}

func GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{r: r, userID: userID}

	transactions, err := core.GetTransactionHistory(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	component := views.TransactionHistory(transactions)
	templ.Handler(component).ServeHTTP(w, r)
}
