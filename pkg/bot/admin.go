// File: pkg/bot/admin_handlers.go

package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
	tele "gopkg.in/telebot.v3"
)

func HandleAdminSet(c tele.Context) error {
	if !core.IsAdmin(c.Sender().ID) {
		return c.Send("Unauthorized: This command is only available for the admin account.")
	}

	args := c.Args()
	if len(args) < 2 || len(args) > 3 {
		return c.Send("Usage:\n/set <@username> admin=<true|false>\n/set <@username> balance=<amount> <currency>")
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

		if len(args) < 3 {
			return c.Send("Please specify the currency code.")
		}
		currencyCode := args[2]

		err = core.AdminSetBalance(c.Sender().ID, targetUser.TelegramID, amount, currencyCode)
		if err != nil {
			return c.Send("Failed to set balance: " + err.Error())
		}

		currency, err := database.GetCurrencyByCode(currencyCode)
		if err != nil {
			return c.Send("Failed to get currency information: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Successfully set balance of %s to %s%.2f %s", targetUsername, currency.Sign, amount, currency.Name))

	default:
		return c.Send("Unknown key. Available keys: admin, balance")
	}
}

func HandleAdminListUsers(c tele.Context) error {
	if !core.IsAdmin(c.Sender().ID) {
		logger.Warn("Unauthorized attempt to list users", "username", c.Sender().Username)
		return c.Send("Unauthorized: This command is only available for admin accounts.")
	}

	users, err := core.ListUsersWithBalances()
	if err != nil {
		logger.Error("Failed to list users", "error", err)
		return c.Send("An error occurred while fetching user data.")
	}

	response := "Users and their balances:\n\n"
	for _, user := range users {
		userLine := fmt.Sprintf("%d - @%s:\n", user.TelegramID, user.Username)
		for currencyCode, amount := range user.Balances {
			currency, _ := database.GetCurrencyByCode(currencyCode)
			balanceLine := fmt.Sprintf("  %s%.2f %s\n", currency.Sign, amount, currencyCode)
			userLine += balanceLine
		}
		response += userLine + "\n"
	}

	logger.Info("Admin listed users", "adminUsername", c.Sender().Username, "responseLength", len(response))

	// If the response is too long, split it into multiple messages
	const maxMessageLength = 4096
	if len(response) > maxMessageLength {
		chunks := splitMessage(response, maxMessageLength)
		for _, chunk := range chunks {
			if err := c.Send(chunk); err != nil {
				logger.Error("Failed to send message chunk", "error", err)
				return err
			}
		}
		return nil
	}

	return c.Send(response)
}

func HandleAdminRemoveUser(c tele.Context) error {
	if !core.IsAdmin(c.Sender().ID) {
		logger.Warn("Unauthorized attempt to remove user", "username", c.Sender().Username)
		return c.Send("Unauthorized: This command is only available for admin accounts.")
	}

	args := c.Args()
	if len(args) != 1 {
		return c.Send("Usage: /removeuser <username>")
	}

	username := strings.TrimPrefix(args[0], "@") // Remove '@' if present

	err := core.RemoveUser(username)
	if err != nil {
		logger.Error("Failed to remove user", "error", err, "username", username)
		return c.Send(fmt.Sprintf("Failed to remove user: %v", err))
	}

	logger.Info("Admin removed user", "adminUsername", c.Sender().Username, "removedUsername", username)
	return c.Send(fmt.Sprintf("User @%s has been successfully removed.", username))
}

// splitMessage splits a long message into chunks of maximum length
func splitMessage(message string, maxLength int) []string {
	var chunks []string
	for len(message) > maxLength {
		chunk := message[:maxLength]
		lastNewline := strings.LastIndex(chunk, "\n")
		if lastNewline != -1 {
			chunk = message[:lastNewline]
			message = message[lastNewline+1:]
		} else {
			message = message[maxLength:]
		}
		chunks = append(chunks, chunk)
	}
	if len(message) > 0 {
		chunks = append(chunks, message)
	}
	return chunks
}

func HandleAdminAddUser(c tele.Context) error {
	if !core.IsAdmin(c.Sender().ID) {
		return c.Send("Unauthorized: This command is only available for admin accounts.")
	}

	args := c.Args()
	if len(args) != 2 {
		return c.Send("Usage: /adduser <telegram_id> <username>")
	}

	telegramID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send("Invalid Telegram ID. Please provide a valid numeric ID.")
	}

	username := args[1]

	err = core.AddUser(c.Sender().ID, telegramID, username)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to add user: %v", err))
	}

	return c.Send(fmt.Sprintf("User @%s with Telegram ID %d has been successfully added.", username, telegramID))
}

func HandleAdminAddCurrency(c tele.Context) error {
	if !core.IsAdmin(c.Sender().ID) {
		return c.Send("Unauthorized: This command is only available for admin accounts.")
	}

	args := c.Args()
	if len(args) != 3 {
		return c.Send("Usage: /addcurrency <code> <name> <sign>")
	}

	code := strings.ToUpper(args[0])
	name := args[1]
	sign := args[2]

	err := core.AddCurrency(c.Sender().ID, code, name, sign)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to add currency: %v", err))
	}

	return c.Send(fmt.Sprintf("Currency %s (%s) with sign %s has been successfully added.", code, name, sign))
}

func HandleAdminSetDefaultCurrency(c tele.Context) error {
	if !core.IsAdmin(c.Sender().ID) {
		return c.Send("Unauthorized: This command is only available for admin accounts.")
	}

	args := c.Args()
	if len(args) != 1 {
		return c.Send("Usage: /setdefaultcurrency <code>")
	}

	code := strings.ToUpper(args[0])

	err := core.SetDefaultCurrency(c.Sender().ID, code)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to set default currency: %v", err))
	}

	return c.Send(fmt.Sprintf("Default currency has been set to %s.", code))
}
