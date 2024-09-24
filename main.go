package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/bot"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	"github.com/fitz123/mcduck-wallet/pkg/webapp"
	tele "gopkg.in/telebot.v3"
)

func main() {
	// Initialize database
	database.InitDB()
	logger.Info("Database initialized")

	// Initialize bot
	pref := tele.Settings{
		Token:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	teleBot, err := tele.NewBot(pref)
	if err != nil {
		logger.Error("Failed to create bot", "error", err)
		log.Fatal(err)
	}
	logger.Info("Telegram bot created")

	// Set up bot handlers
	teleBot.Handle("/start", bot.HandleStart)
	teleBot.Handle("/balance", bot.HandleBalance)
	teleBot.Handle("/transfer", bot.HandleTransfer)
	teleBot.Handle("/history", bot.HandleHistory)
	teleBot.Handle("/set", bot.HandleAdminSet)
	teleBot.Handle("/listusers", bot.HandleAdminListUsers)
	teleBot.Handle("/removeuser", bot.HandleAdminRemoveUser)
	teleBot.Handle("/adduser", bot.HandleAdminAddUser)
	teleBot.Handle("/addcurrency", bot.HandleAdminAddCurrency)
	teleBot.Handle("/setdefaultcurrency", bot.HandleAdminSetDefaultCurrency)
	logger.Info("Bot handlers set up")

	// Initialize WebApp
	webapp.InitAuth(os.Getenv("TELEGRAM_BOT_TOKEN"))
	logger.Info("WebApp authentication initialized")

	// Set up webapp routes
	mux := http.NewServeMux()
	mux.HandleFunc("/balance", webapp.AuthMiddleware(webapp.GetBalance))
	mux.HandleFunc("/transfer", webapp.AuthMiddleware(webapp.TransferMoney))
	mux.HandleFunc("/history", webapp.AuthMiddleware(webapp.GetTransactionHistory))
	mux.HandleFunc("/", webapp.ServeHTML)
	logger.Info("WebApp routes set up")

	// Create a new server
	server := &http.Server{
		Addr:    ":80",
		Handler: mux,
	}

	// Channel to signal shutdown
	shutdown := make(chan struct{})

	// Set up signal catching
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start the bot in a separate goroutine
	go func() {
		logger.Info("Starting Telegram bot...")
		teleBot.Start()
	}()

	// Start the HTTP server in a separate goroutine
	go func() {
		logger.Info("Starting WebApp server on :80")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("WebApp server error", "error", err)
		}
	}()

	// Wait for interrupt signal
	go func() {
		<-signalChan
		logger.Info("Received interrupt signal, shutting down gracefully...")

		// Stop the bot
		teleBot.Stop()
		logger.Info("Telegram bot stopped")

		// Create a deadline for server shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt to gracefully shutdown the server
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Server forced to shutdown", "error", err)
		}

		logger.Info("Server stopped")

		// Close the database connection
		if db, err := database.DB.DB(); err == nil {
			db.Close()
			logger.Info("Database connection closed")
		}

		// Signal successful shutdown
		close(shutdown)
	}()

	// Wait for shutdown to complete
	<-shutdown
	logger.Info("Shutdown complete")
}
