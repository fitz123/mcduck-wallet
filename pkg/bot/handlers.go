// File: pkg/bot/handlers.go

package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/messages"
	tele "gopkg.in/telebot.v3"
)

type botContext struct {
	c tele.Context
}

func (bc *botContext) GetUserID() int64 {
	return bc.c.Sender().ID
}

func (bc *botContext) GetUsername() string {
	return bc.c.Sender().Username
}

func HandleStart(c tele.Context) error {
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
	webAppURL := "https://3a64-181-111-49-211.ngrok-free.app"
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
}

func HandleBalance(c tele.Context) error {
	ctx := &botContext{c}
	balance, err := core.GetBalance(ctx)
	if err != nil {
		return c.Send(fmt.Sprintf("Error: %v", err))
	}
	return c.Send(fmt.Sprintf(messages.InfoCurrentBalance, balance))
}

func HandleTransfer(c tele.Context) error {
	ctx := &botContext{c}
	args := c.Args()
	if len(args) != 2 {
		return c.Send(messages.UsageTransfer)
	}

	toUsername := strings.TrimPrefix(args[0], "@")
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return c.Send(messages.ErrInvalidAmount)
	}

	err = core.TransferMoney(ctx, toUsername, amount)
	if err != nil {
		return c.Send(fmt.Sprintf("Transfer failed: %v", err))
	}

	return c.Send(fmt.Sprintf(messages.InfoTransferSuccessful, amount, toUsername))
}

func HandleHistory(c tele.Context) error {
	ctx := &botContext{c}
	transactions, err := core.GetTransactionHistory(ctx)
	if err != nil {
		return c.Send(fmt.Sprintf("Error fetching transaction history: %v", err))
	}

	return c.Send(formatTransactionHistory(transactions))
}
