package webapp

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/fitz123/mcduck-wallet/pkg/core"
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

type Response struct {
	Message    string
	Error      error
	StatusCode int
}

func ServeHTML(w http.ResponseWriter, r *http.Request) {
	component := views.InitialLoadingIndex()
	_ = component.Render(r.Context(), w)
}

func GetTransferForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("webContext").(*webContext)
	balances, err := core.GetBalance(ctx)
	if err != nil {
		handleResponse(w, r, ctx, Response{
			Message:    "Error fetching user balances",
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	component := views.TransferForm(balances)
	err = component.Render(r.Context(), w)
	if err != nil {
		handleResponse(w, r, ctx, Response{
			Message:    "Error rendering transfer form",
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}
}

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("webContext").(*webContext)

	// Respond with an empty message (no specific success message)
	handleResponse(w, r, ctx, Response{})
}

func GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("webContext").(*webContext)

	transactions, err := core.GetTransactionHistory(ctx)
	if err != nil {
		handleResponse(w, r, ctx, Response{
			Message:    "Error fetching transaction history",
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})

		return
	}
	component := views.TransactionHistory(transactions)
	templ.Handler(component).ServeHTTP(w, r)
}

func TransferMoney(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context().Value("webContext").(*webContext)

	// Parse form values
	toUsername, amount, currencyCode, err := parseTransferFormValues(r)
	if err != nil {
		// Respond with an error using the unified response handling
		handleResponse(w, r, ctx, Response{
			Message:    err.Error(),
			Error:      err,
			StatusCode: http.StatusBadRequest,
		})
		return
	}

	// Process the money transfer
	err = core.TransferMoney(ctx, toUsername, amount, currencyCode)
	if err != nil {
		// Handle transfer failure
		handleResponse(w, r, ctx, Response{
			Error:      err,
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	// Respond with a success message
	handleResponse(w, r, ctx, Response{
		Message: fmt.Sprintf(messages.InfoTransferSuccessful, amount, currencyCode, toUsername),
	})
}
