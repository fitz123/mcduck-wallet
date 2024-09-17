// File: pkg/webapp/handlers.go

package webapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fitz123/mcduck-wallet/pkg/core"
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

	message := fmt.Sprintf(messages.InfoCurrentBalance, balance)

	json.NewEncoder(w).Encode(message)
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
