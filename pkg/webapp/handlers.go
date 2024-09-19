// File: pkg/webapp/handlers.go

package webapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
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
	if r.URL.Path != "/" {
		r.URL.Path += ".html"
	}
	http.FileServer(http.Dir("webapp")).ServeHTTP(w, r)
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

	response := struct {
		Value    float64 `json:"value"`
		Sign     string  `json:"sign"`
		Currency string  `json:"currency"`
	}{
		Value:    balance,
		Sign:     currency.Sign,
		Currency: currency.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode balance response", "error", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}

	logger.Info("Balance retrieved successfully", "userID", userID, "balance", balance, "currency", currency.Code)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(transactions)
}
