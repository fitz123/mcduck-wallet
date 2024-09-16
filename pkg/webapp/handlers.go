// File: pkg/webapp/handlers.go

package webapp

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/commands"
)

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
	// Split the message into lines
	lines := strings.Split(message, "\n")

	// Join the lines with HTML line breaks
	htmlMessage := strings.Join(lines, "<br>")

	// Set the content type to HTML
	wc.w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Write the HTML-formatted message
	_, err := fmt.Fprint(wc.w, htmlMessage)
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
	ctx := &webContext{w: w, r: r, userID: userID}
	if err := commands.History(ctx); err != nil {
		handleError(w, err)
	}
}

func handleError(w http.ResponseWriter, err error) {
	log.Printf("Error: %v", err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
