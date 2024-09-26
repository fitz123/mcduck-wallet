// File: ./internal/handlers/handlers.go
package handlers

import (
	"github.com/fitz123/mcduck-wallet/internal/webapp"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, webService *webapp.WebService) {
	r.Get("/", webService.ServeHome)
	r.Route("/", func(r chi.Router) {
		r.Use(webService.AuthMiddleware)
		r.Get("/dashboard", webService.GetDashboard)
		r.Get("/transfer-form", webService.GetTransferForm)
		r.Post("/transfer", webService.TransferMoney)
		r.Get("/history", webService.GetTransactionHistory)
	})
}
