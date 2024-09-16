// File: pkg/bot/admin_handlers.go

package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	tele "gopkg.in/telebot.v3"
)

func HandleAdminSet(c tele.Context) error {
	if !core.IsAdmin(c.Sender().Username) {
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
		err = core.AdminSetBalance(c.Sender().ID, targetUser.TelegramID, amount)
		if err != nil {
			return c.Send("Failed to set balance: " + err.Error())
		}
		return c.Send(fmt.Sprintf("Successfully set balance of %s to ¤%.2f", targetUsername, amount))
	default:
		return c.Send("Unknown key. Available keys: admin, balance")
	}
}
