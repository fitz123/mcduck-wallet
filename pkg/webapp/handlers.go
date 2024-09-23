package webapp

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/fitz123/mcduck-wallet/pkg/core"
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

	balance, currency, err := core.GetBalance(ctx)
	if err != nil {
		logger.Error("Failed to get balance", "error", err, "userID", userID)
		http.Error(w, "Error fetching balance", http.StatusInternalServerError)
		return
	}

	component := views.Balance(balance, currency)
	err = component.Render(r.Context(), w)
	if err != nil {
		logger.Error("Error rendering balance template", "error", err)
		http.Error(w, "Error rendering balance", http.StatusInternalServerError)
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

	err = core.TransferMoney(ctx, toUsername, amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(fmt.Sprintf(messages.InfoTransferSuccessful, amount, toUsername)))
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
