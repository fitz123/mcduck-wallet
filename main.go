package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/handlers"
	"github.com/fitz123/mcduck-wallet/pkg/webapp"
	tele "gopkg.in/telebot.v3"
)

const ADMIN_USERNAME = "notbuddy"

func main() {
	// Initialize database
	database.InitDB()

	pref := tele.Settings{
		Token:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Set up bot handlers
	setupBotHandlers(bot)

	// Initialize WebApp
	webapp.InitAuth(os.Getenv("TELEGRAM_BOT_TOKEN"))
	webapp.SetupRoutes()

	// Start the HTTP server for WebApp
	go func() {
		log.Println("Starting WebApp server on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	// Start the bot
	bot.Start()
}

func setupBotHandlers(bot *tele.Bot) {
	bot.Handle("/start", func(c tele.Context) error {
		user, err := handlers.GetOrCreateUser(c.Sender().ID, c.Sender().Username)
		if err != nil {
			return c.Send("Error: " + err.Error())
		}

		if user.Username != c.Sender().Username {
			// Update username if it has changed
			user.Username = c.Sender().Username
			database.DB.Save(user)
		}

		return c.Send(fmt.Sprintf("Welcome to McDuck Wallet, %s! Your personal finance assistant. Your current balance is $%.2f.", user.Username, user.Balance))
	})

	bot.Handle("/balance", func(c tele.Context) error {
		user, err := handlers.GetOrCreateUser(c.Sender().ID, c.Sender().Username)
		if err != nil {
			return c.Send("Error: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Your current balance is $%.2f", user.Balance))
	})

	bot.Handle("/transfer", func(c tele.Context) error {
		args := c.Args()
		if len(args) != 2 {
			return c.Send("Usage: /transfer <@username> <amount>")
		}

		toUsername := strings.TrimPrefix(args[0], "@")
		var toUser database.User
		if err := database.DB.Where("username = ?", toUsername).First(&toUser).Error; err != nil {
			return c.Send("Recipient not found.")
		}

		amount, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return c.Send("Invalid amount. Please enter a number.")
		}

		err = handlers.TransferMoney(c.Sender().ID, toUser.TelegramID, amount)
		if err != nil {
			return c.Send("Transfer failed: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Successfully transferred $%.2f to %s", amount, toUsername))
	})

	bot.Handle("/history", func(c tele.Context) error {
		transactions, err := handlers.GetTransactionHistory(c.Sender().ID)
		if err != nil {
			return c.Send("Error fetching transaction history: " + err.Error())
		}

		if len(transactions) == 0 {
			return c.Send("No transactions found.")
		}

		var response strings.Builder
		response.WriteString("Your recent transactions:\n\n")
		for _, t := range transactions {
			var description string
			if t.Type == "transfer" {
				if t.Amount < 0 {
					description = fmt.Sprintf("Sent $%.2f", -t.Amount)
				} else {
					description = fmt.Sprintf("Received $%.2f", t.Amount)
				}
			} else {
				description = fmt.Sprintf("Deposited $%.2f", t.Amount)
			}
			response.WriteString(fmt.Sprintf("%s - %s\n", t.Timestamp.Format("2006-01-02 15:04:05"), description))
		}

		return c.Send(response.String())
	})

	bot.Handle("/set", func(c tele.Context) error {
		if c.Sender().Username != ADMIN_USERNAME {
			return c.Send("Unauthorized: This command is only available for the admin account.")
		}

		args := c.Args()
		if len(args) != 2 {
			return c.Send("Usage: /set <@username> <key=value>")
		}

		targetUsername := strings.TrimPrefix(args[0], "@")
		var targetUser database.User
		if err := database.DB.Where("username = ?", targetUsername).First(&targetUser).Error; err != nil {
			return c.Send("Target user not found.")
		}

		keyValue := strings.Split(args[1], "=")
		if len(keyValue) != 2 {
			return c.Send("Invalid key=value format.")
		}

		key := keyValue[0]
		value := keyValue[1]

		switch key {
		case "admin":
			isAdmin, err := strconv.ParseBool(value)
			if err != nil {
				return c.Send("Invalid boolean value for admin. Please use 'true' or 'false'.")
			}
			err = handlers.SetAdminStatus(targetUser.TelegramID, isAdmin)
			if err != nil {
				return c.Send("Failed to set admin status: " + err.Error())
			}
			return c.Send(fmt.Sprintf("Successfully set admin status of %s to %v", targetUsername, isAdmin))
		case "balance":
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return c.Send("Invalid amount. Please enter a number.")
			}
			err = handlers.AdminAddMoney(c.Sender().ID, targetUser.TelegramID, amount)
			if err != nil {
				return c.Send("Failed to set balance: " + err.Error())
			}
			return c.Send(fmt.Sprintf("Successfully set balance of %s to $%.2f", targetUsername, amount))
		default:
			return c.Send("Unknown key. Available keys: admin, balance")
		}
	})
}
