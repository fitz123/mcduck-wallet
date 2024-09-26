// File: ./cmd/mcduck-wallet/main.go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fitz123/mcduck-wallet/internal/bot"
	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/handlers"
	"github.com/fitz123/mcduck-wallet/internal/logger"
	"github.com/fitz123/mcduck-wallet/internal/services"
	"github.com/fitz123/mcduck-wallet/internal/webapp"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load configuration
	cfg := loadConfig()

	// Initialize logger
	logger.Init(cfg.LogLevel)

	// Initialize database
	db, err := database.New(cfg.DatabaseDSN)
	if err != nil {
		logger.Error("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize services
	userService := services.NewUserService(db)
	coreService := services.NewCoreService(db, userService)
	botService := bot.NewBotService(cfg.TelegramToken, userService, coreService)
	webService := webapp.NewWebService(userService, coreService, cfg.TelegramToken)

	// Start the bot
	go botService.Start()

	// Initialize and start the web server
	server := initWebServer(cfg.ServerAddress, webService)
	go func() {
		logger.Info("Starting WebApp server", "address", cfg.ServerAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("WebApp server error", "error", err)
		}
	}()

	// Handle shutdown signals
	handleShutdown(botService, server, db)
}

func loadConfig() *Config {
	// Load configuration from environment variables or files
	return &Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		ServerAddress: ":80",
		DatabaseDSN:   "mcduck_wallet.db",
		LogLevel:      "debug",
	}
}

func initWebServer(addr string, webService *webapp.WebService) *http.Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	handlers.RegisterRoutes(r, webService)

	return &http.Server{
		Addr:    addr,
		Handler: r,
	}
}

func handleShutdown(botService *bot.BotService, server *http.Server, db *database.DB) {
	// Channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-quit
	logger.Info("Received shutdown signal, shutting down gracefully...")

	// Stop the bot
	botService.Stop()
	logger.Info("Telegram bot stopped")

	// Shutdown the web server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}
	logger.Info("WebApp server stopped")

	// Close the database connection
	db.Close()
	logger.Info("Database connection closed")
}

type Config struct {
	TelegramToken string
	ServerAddress string
	DatabaseDSN   string
	LogLevel      string
}
