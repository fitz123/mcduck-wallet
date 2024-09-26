package webapp

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"github.com/fitz123/mcduck-wallet/pkg/webapp/views"
)

// parseTransferFormValues parses and validates the form values from the request.
func parseTransferFormValues(r *http.Request) (string, float64, string, error) {
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
		defaultCurrency, err := database.GetDefaultCurrency()
		if err != nil {
			return "", 0, "", fmt.Errorf("Failed to get default currency")
		}
		currencyCode = defaultCurrency.Code
	}

	return toUsername, amount, currencyCode, nil
}

func handleResponse(w http.ResponseWriter, r *http.Request, ctx *webContext, response Response) {
	success := response.Error == nil
	message := response.Message

	if !success {
		logger.Error(response.Message, "error", response.Error)
		// Set the error message to be displayed
		message = response.Error.Error()
	}

	// Fetch balances
	balances, err := core.GetBalance(ctx)
	if err != nil {
		logger.Error("Failed to get balances", "error", err)
		balances = []database.Balance{} // Default to an empty set
		message = "Failed to fetch balances"
		success = false
	}

	// Render the main content (dashboard)
	component := views.MainContent(balances, message, success)
	if err := component.Render(r.Context(), w); err != nil {
		logger.Error("Error rendering response", "error", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

func WithWebContext(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserIDFromContext(r)
		ctx := &webContext{r: r, userID: userID}
		ctxKey := "webContext"
		r = r.WithContext(context.WithValue(r.Context(), ctxKey, ctx))
		next(w, r)
	}
}
