package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fitz123/mcduck-wallet/pkg/commands"
	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
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

type botContext struct {
	c tele.Context
}

func (bc *botContext) GetUserID() int64 {
	return bc.c.Sender().ID
}

func (bc *botContext) GetUsername() string {
	return bc.c.Sender().Username
}

func (bc *botContext) Reply(message string) error {
	return bc.c.Reply(message)
}

func setupBotHandlers(bot *tele.Bot) {
	bot.Handle("/start", func(c tele.Context) error {
		user, err := core.GetOrCreateUser(c.Sender().ID, c.Sender().Username)
		if err != nil {
			return c.Send("Error: " + err.Error())
		}

		if user.Username != c.Sender().Username {
			// Update username if it has changed
			user.Username = c.Sender().Username
			database.DB.Save(user)
		}

		// Create a keyboard with a WebApp button
		webAppURL := "https://07ac-181-111-49-211.ngrok-free.app"
		webAppButton := tele.InlineButton{
			Text: "Open McDuck Wallet",
			WebApp: &tele.WebApp{
				URL: webAppURL,
			},
		}

		keyboard := &tele.ReplyMarkup{
			InlineKeyboard: [][]tele.InlineButton{
				{webAppButton},
			},
		}

		return c.Send(fmt.Sprintf("Welcome to McDuck Wallet, %s! Your personal finance assistant. Your current balance is ¤%.2f.\n\nUse the button below to open the WebApp.", user.Username, user.Balance), keyboard)
	})

	bot.Handle("/balance", func(c tele.Context) error {
		return commands.Balance(&botContext{c})
	})

	bot.Handle("/transfer", func(c tele.Context) error {
		return commands.Transfer(&botContext{c}, c.Args()...)
	})

	bot.Handle("/history", func(c tele.Context) error {
		return commands.History(&botContext{c})
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
			err = core.SetAdminStatus(targetUser.TelegramID, isAdmin)
			if err != nil {
				return c.Send("Failed to set admin status: " + err.Error())
			}
			return c.Send(fmt.Sprintf("Successfully set admin status of %s to %v", targetUsername, isAdmin))
		case "balance":
			amount, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return c.Send("Invalid amount. Please enter a number.")
			}
			err = core.AdminAddMoney(c.Sender().ID, targetUser.TelegramID, amount)
			if err != nil {
				return c.Send("Failed to set balance: " + err.Error())
			}
			return c.Send(fmt.Sprintf("Successfully set balance of %s to ¤%.2f", targetUsername, amount))
		default:
			return c.Send("Unknown key. Available keys: admin, balance")
		}
	})
}
