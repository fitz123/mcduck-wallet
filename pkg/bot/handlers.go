// File: pkg/bot/handlers.go

package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fitz123/mcduck-wallet/pkg/core"
	"github.com/fitz123/mcduck-wallet/pkg/database"
	"github.com/fitz123/mcduck-wallet/pkg/logger"
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
	user, err := core.GetUser(c.Sender().ID)
	if err != nil {
		if err == core.ErrUserNotFound {
			user, err = core.CreateUser(c.Sender().ID, c.Sender().Username)
			if err != nil {
				return c.Send("Error creating user: " + err.Error())
			}
		} else {
			return c.Send("Error: " + err.Error())
		}
	} else {
		// Update username if it has changed
		if user.Username != c.Sender().Username {
			err = core.UpdateUsername(c.Sender().ID, c.Sender().Username)
			if err != nil {
				return c.Send("Error updating username: " + err.Error())
			}
		}
	}

	// Create a keyboard with a WebApp button
	webAppURL := "https://mcduck.120912.xyz"
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

	return c.Send(fmt.Sprintf(messages.InfoWelcome, user.Username), keyboard)
}

func HandleBalance(c tele.Context) error {
	ctx := &botContext{c}
	balances, err := core.GetBalance(ctx)
	if err != nil {
		return c.Send(fmt.Sprintf("Error: %v", err))
	}

	var formattedBalances []string
	for _, balance := range balances {
		currencyCode := balance.Currency.Code
		currency, err := database.GetCurrencyByCode(currencyCode)
		if err != nil {
			logger.Error("Failed to get currency", "error", err, "currencyCode", currencyCode)
			continue
		}
		formattedBalance := fmt.Sprintf("%s%.0f %s", currency.Sign, balance.Amount, currency.Name)
		formattedBalances = append(formattedBalances, formattedBalance)
	}

	response := "Your current balances:\n" + strings.Join(formattedBalances, "\n")
	return c.Send(response)
}

func HandleTransfer(c tele.Context) error {
	ctx := &botContext{c}
	args := c.Args()
	if len(args) < 2 || len(args) > 3 {
		return c.Send(messages.UsageTransfer)
	}

	defaultCurrency, err := database.GetDefaultCurrency()
	currencyCode := defaultCurrency.Code

	if len(args) == 3 {
		currencyCode = strings.ToUpper(args[2])
	}

	toUsername := strings.TrimPrefix(args[0], "@")
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return c.Send(messages.ErrInvalidAmount)
	}

	err = core.TransferMoney(ctx, toUsername, amount, currencyCode)
	if err != nil {
		return c.Send(fmt.Sprintf("Transfer failed: %v", err))
	}

	return c.Send(fmt.Sprintf(messages.InfoTransferSuccessful, amount, currencyCode, toUsername))
}

func HandleHistory(c tele.Context) error {
	ctx := &botContext{c}
	transactions, err := core.GetTransactionHistory(ctx)
	if err != nil {
		return c.Send(fmt.Sprintf("Error fetching transaction history: %v", err))
	}

	formattedTransactions := messages.FormatTransactionHistory(transactions)
	response := fmt.Sprintf("*Transaction History*\n\n%s", strings.Join(formattedTransactions, "\n\n"))
	return c.Send(response, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}
