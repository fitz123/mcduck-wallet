// File: main.go

package main

import (
	"log"
	"net/http"
	"os"
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
	logger.Info("Bot handlers set up")

	// Initialize WebApp
	webapp.InitAuth(os.Getenv("TELEGRAM_BOT_TOKEN"))
	logger.Info("WebApp authentication initialized")

	// Set up webapp routes
	http.HandleFunc("/balance", webapp.AuthMiddleware(webapp.GetBalance))
	http.HandleFunc("/transfer", webapp.AuthMiddleware(webapp.TransferMoney))
	http.HandleFunc("/history", webapp.AuthMiddleware(webapp.GetTransactionHistory))
	http.HandleFunc("/", webapp.ServeHTML)
	logger.Info("WebApp routes set up")

	// Start the bot in a separate goroutine
	go func() {
		logger.Info("Starting Telegram bot...")
		teleBot.Start()
	}()

	// Start the HTTP server for WebApp
	logger.Info("Starting WebApp server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
