package webapp

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	component := views.Index("", false)
	_ = component.Render(r.Context(), w)
}

func GetTransferForm(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{r: r, userID: userID}

	balances, err := core.GetBalance(ctx)
	if err != nil {
		logger.Error("GetTransferForm: Error fetching user balances", "error", err)
		http.Error(w, "Error fetching user balances", http.StatusInternalServerError)
		return
	}

	component := views.TransferForm(balances)
	err = component.Render(r.Context(), w)
	if err != nil {
		logger.Error("GetTransferForm: Error rendering transfer form", "error", err)
		http.Error(w, "Error rendering transfer form", http.StatusInternalServerError)
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

	component := views.Balances(balances)
	err = component.Render(r.Context(), w)
	if err != nil {
		logger.Error("Error rendering balances template", "error", err)
		http.Error(w, "Error rendering balances", http.StatusInternalServerError)
		return
	}
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

func TransferMoney(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{r: r, userID: userID}

	toUsername := strings.TrimPrefix(r.FormValue("to_username"), "@")
	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		indexComponent := views.Index("Invalid amount", false)
		err = indexComponent.Render(r.Context(), w)
		if err != nil {
			logger.Error("Error rendering index page after transfer error", "error", err)
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}
		return
	}

	currencyCode := r.FormValue("currency")
	if currencyCode == "" {
		defaultCurrency, err := database.GetDefaultCurrency()
		if err != nil {
			logger.Error("TransferMoney: Failed to get default currency", "error", err)
			indexComponent := views.Index("Failed to process transfer", false)
			err = indexComponent.Render(r.Context(), w)
			if err != nil {
				logger.Error("Error rendering index page after transfer error", "error", err)
				http.Error(w, "Error rendering page", http.StatusInternalServerError)
			}
			return
		}
		currencyCode = defaultCurrency.Code
	}

	err = core.TransferMoney(ctx, toUsername, amount, currencyCode)
	if err != nil {
		logger.Error("TransferMoney: Failed to transfer money", "error", err)
		indexComponent := views.Index(err.Error(), false)
		err = indexComponent.Render(r.Context(), w)
		if err != nil {
			logger.Error("Error rendering index page after transfer error", "error", err)
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}
		return
	}

	currency, err := database.GetCurrencyByCode(currencyCode)
	if err != nil {
		logger.Error("TransferMoney: Failed to get currency details", "error", err, "currencyCode", currencyCode)
		indexComponent := views.Index("Transfer successful, but failed to get currency details", true)
		err = indexComponent.Render(r.Context(), w)
		if err != nil {
			logger.Error("Error rendering index page after transfer", "error", err)
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}
		return
	}

	response := fmt.Sprintf(messages.InfoTransferSuccessful, amount, currency.Code, toUsername)
	indexComponent := views.Index(response, true)
	err = indexComponent.Render(r.Context(), w)
	if err != nil {
		logger.Error("Error rendering index page after transfer", "error", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
