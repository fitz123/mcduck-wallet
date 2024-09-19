// File: pkg/webapp/handlers.go

package webapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
)

type webContext struct {
	r      *http.Request
	userID int64
}

func (wc *webContext) GetUserID() int64 {
	return wc.userID
}

func ServeHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webapp/index.html")
}

func GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{r: r, userID: userID}

	balance, err := core.GetBalance(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the default currency
	var defaultCurrency database.Currency
	if err := database.DB.First(&defaultCurrency).Error; err != nil {
		http.Error(w, "Error fetching currency information", http.StatusInternalServerError)
		return
	}

	// Create a response object
	response := struct {
		Sign     string  `json:"sign"`
		Value    float64 `json:"value"`
		Currency string  `json:"currency"`
	}{
		Sign:     defaultCurrency.Sign,
		Value:    balance,
		Currency: defaultCurrency.Name,
	}

	// Encode the response object as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, fmt.Sprintf(messages.InfoTransferSuccessful, amount, toUsername))
}

func GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{r: r, userID: userID}

	transactions, err := core.GetTransactionHistory(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(formatTransactionHistory(transactions))
}
