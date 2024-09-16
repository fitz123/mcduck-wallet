// File: pkg/webapp/handlers.go

package webapp

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fitz123/mcduck-wallet/pkg/commands"
	"github.com/fitz123/mcduck-wallet/pkg/core"
)

var templates = template.Must(template.ParseFiles(filepath.Join("webapp", "templates", "transaction_history.html")))

type webContext struct {
	w      http.ResponseWriter
	r      *http.Request
	userID int64
}

func (wc *webContext) GetUserID() int64 {
	return wc.userID
}

func (wc *webContext) GetUsername() string {
	// In a web context, we might not always have the username readily available.
	// You could fetch it from the database if needed, or return an empty string.
	return ""
}

func (wc *webContext) Reply(message string) error {
	_, err := fmt.Fprint(wc.w, message)
	return err
}

// SetupRoutes sets up the HTTP routes for the WebApp
func SetupRoutes() {
	http.HandleFunc("/balance", AuthMiddleware(getBalance))
	http.HandleFunc("/transfer", AuthMiddleware(transferMoney))
	http.HandleFunc("/history", AuthMiddleware(getTransactionHistory))
	http.HandleFunc("/", serveHTML)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webapp/index.html")
}

func getBalance(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	ctx := &webContext{w: w, r: r, userID: userID}
	if err := commands.Balance(ctx); err != nil {
		handleError(w, err)
	}
}

func transferMoney(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	toUsername := r.FormValue("to_username")
	amount := r.FormValue("amount")
	ctx := &webContext{w: w, r: r, userID: userID}
	if err := commands.Transfer(ctx, toUsername, amount); err != nil {
		handleError(w, err)
	}
}

func getTransactionHistory(w http.ResponseWriter, r *http.Request) {
	userID := GetUserIDFromContext(r)
	transactions, err := core.GetTransactionHistory(userID)
	if err != nil {
		http.Error(w, "Error fetching transaction history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	historyText := commands.BuildTransactionHistory(transactions)

	// Convert newlines to <br> tags for HTML display
	historyHTML := strings.ReplaceAll(historyText, "\n", "<br>")

	fmt.Fprint(w, historyHTML)
}

func handleError(w http.ResponseWriter, err error) {
	log.Printf("Error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
