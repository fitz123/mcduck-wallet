// File: ./internal/bot/bot_service.go
package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fitz123/mcduck-wallet/internal/database"
	"github.com/fitz123/mcduck-wallet/internal/messages"
	"github.com/fitz123/mcduck-wallet/internal/services"
	tele "gopkg.in/telebot.v3"
)

type BotService struct {
	bot         *tele.Bot
	userService services.UserService
	coreService services.CoreService // Added core service for business logic
}

func NewBotService(token string, userService services.UserService, coreService services.CoreService) *BotService {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	bot, err := tele.NewBot(pref)
	if err != nil {
		panic(err)
	}

	bs := &BotService{
		bot:         bot,
		userService: userService,
		coreService: coreService,
	}
	bs.registerHandlers()
	return bs
}

func (bs *BotService) Start() {
	bs.bot.Start()
}

func (bs *BotService) Stop() {
	bs.bot.Stop()
}

func (bs *BotService) registerHandlers() {
	bs.bot.Handle("/start", bs.handleStart)
	bs.bot.Handle("/balance", bs.handleBalance)
	bs.bot.Handle("/transfer", bs.handleTransfer)
	bs.bot.Handle("/history", bs.handleHistory)
	bs.bot.Handle("/set", bs.handleAdminSet)
	bs.bot.Handle("/listusers", bs.handleAdminListUsers)
	bs.bot.Handle("/disableuser", bs.handleAdminDisableUser)
	bs.bot.Handle("/destroyuser", bs.handleAdminDestroyUser)
	bs.bot.Handle("/adduser", bs.handleAdminAddUser)
	bs.bot.Handle("/addcurrency", bs.handleAddCurrency)
	bs.bot.Handle("/setdefaultcurrency", bs.handleAdminSetDefaultCurrency)
}

func (bs *BotService) handleStart(c tele.Context) error {
	ctx := context.Background()
	user, err := bs.userService.GetUser(ctx, c.Sender().ID)
	if err != nil {
		// User not found, create a new one
		user = &database.User{
			TelegramID: c.Sender().ID,
			Username:   c.Sender().Username,
		}
		err := bs.coreService.AddUser(ctx, c.Sender().ID, c.Sender().Username)
		if err != nil {
			return c.Send("Error creating user: " + err.Error())
		}
	} else {
		// Update username if changed
		if user.Username != c.Sender().Username {
			bs.userService.UpdateUsername(ctx, c.Sender().ID, c.Sender().Username)
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

func (bs *BotService) handleBalance(c tele.Context) error {
	ctx := context.Background()
	balances, err := bs.coreService.GetBalances(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("Error fetching balances: " + err.Error())
	}

	var formattedBalances []string
	for _, balance := range balances {
		formattedBalance := fmt.Sprintf("%s%.0f %s", balance.Currency.Sign, balance.Amount, balance.Currency.Name)
		formattedBalances = append(formattedBalances, formattedBalance)
	}

	response := "Your current balances:\n" + strings.Join(formattedBalances, "\n")
	return c.Send(response)
}

func (bs *BotService) handleTransfer(c tele.Context) error {
	ctx := context.Background()
	args := c.Args()
	if len(args) < 2 || len(args) > 3 {
		return c.Send(messages.UsageTransfer)
	}

	currencyCode := ""
	if len(args) == 3 {
		currencyCode = strings.ToUpper(args[2])
	} else {
		defaultCurrency, err := bs.coreService.GetDefaultCurrency(ctx)
		if err != nil {
			return c.Send("Error fetching default currency: " + err.Error())
		}
		currencyCode = defaultCurrency.Code
	}

	toUsername := strings.TrimPrefix(args[0], "@")
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return c.Send(messages.ErrInvalidAmount)
	}

	err = bs.coreService.TransferMoney(ctx, c.Sender().ID, toUsername, amount, currencyCode)
	if err != nil {
		return c.Send("Transfer failed: " + err.Error())
	}

	return c.Send(fmt.Sprintf(messages.InfoTransferSuccessful, amount, currencyCode, toUsername))
}

func (bs *BotService) handleHistory(c tele.Context) error {
	ctx := context.Background()
	transactions, err := bs.coreService.GetTransactionHistory(ctx, c.Sender().ID)
	if err != nil {
		return c.Send("Error fetching transaction history: " + err.Error())
	}

	formattedTransactions := messages.FormatTransactionHistory(transactions)
	response := fmt.Sprintf("*Transaction History*\n\n%s", strings.Join(formattedTransactions, "\n\n"))
	return c.Send(response, &tele.SendOptions{ParseMode: tele.ModeMarkdown})
}

// Admin handlers

func (bs *BotService) handleAdminSet(c tele.Context) error {
	ctx := context.Background()
	if !bs.userService.IsAdmin(ctx, c.Sender().ID) {
		return c.Send(messages.ErrUnauthorized)
	}

	args := c.Args()
	if len(args) < 2 || len(args) > 3 {
		return c.Send("Usage:\n/set <@username> admin=<true|false>\n/set <@username> balance=<amount> <currency>")
	}

	targetUsername := strings.TrimPrefix(args[0], "@")

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
		err = bs.coreService.SetAdminStatus(ctx, targetUsername, isAdmin)
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
		currencyCode := strings.ToUpper(args[2])

		err = bs.coreService.AdminSetBalance(ctx, c.Sender().ID, targetUsername, amount, currencyCode)
		if err != nil {
			return c.Send("Failed to set balance: " + err.Error())
		}

		currency, err := bs.coreService.GetCurrencyByCode(ctx, currencyCode)
		if err != nil {
			return c.Send("Failed to get currency information: " + err.Error())
		}

		return c.Send(fmt.Sprintf("Successfully set balance of %s to %s%.0f %s", targetUsername, currency.Sign, amount, currency.Name))

	default:
		return c.Send("Unknown key. Available keys: admin, balance")
	}
}

func (bs *BotService) handleAdminListUsers(c tele.Context) error {
	ctx := context.Background()
	if !bs.userService.IsAdmin(ctx, c.Sender().ID) {
		return c.Send(messages.ErrUnauthorized)
	}

	users, err := bs.coreService.ListUsersWithBalances(ctx)
	if err != nil {
		return c.Send("An error occurred while fetching user data.")
	}

	response := "Users and their balances:\n\n"
	for _, user := range users {
		userLine := fmt.Sprintf("%d - @%s:\n", user.TelegramID, user.Username)
		for currencyCode, amount := range user.Balances {
			currency, _ := bs.coreService.GetCurrencyByCode(ctx, currencyCode)
			balanceLine := fmt.Sprintf("  %s%.2f %s\n", currency.Sign, amount, currencyCode)
			userLine += balanceLine
		}
		response += userLine + "\n"
	}

	// Split the response if it's too long
	const maxMessageLength = 4096
	if len(response) > maxMessageLength {
		chunks := splitMessage(response, maxMessageLength)
		for _, chunk := range chunks {
			if err := c.Send(chunk); err != nil {
				return err
			}
		}
		return nil
	}

	return c.Send(response)
}

func (bs *BotService) handleAdminDisableUser(c tele.Context) error {
	ctx := context.Background()
	if !bs.userService.IsAdmin(ctx, c.Sender().ID) {
		return c.Send(messages.ErrUnauthorized)
	}

	args := c.Args()
	if len(args) != 1 {
		return c.Send("Usage: /disableuser <username>")
	}

	username := strings.TrimPrefix(args[0], "@")
	err := bs.coreService.DisableUser(ctx, username)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to disable user: %v", err))
	}

	return c.Send(fmt.Sprintf("User @%s has been successfully disabled.", username))
}

func (bs *BotService) handleAdminDestroyUser(c tele.Context) error {
	ctx := context.Background()
	if !bs.userService.IsAdmin(ctx, c.Sender().ID) {
		return c.Send(messages.ErrUnauthorized)
	}

	args := c.Args()
	if len(args) != 1 {
		return c.Send("Usage: /destroyuser <username>")
	}

	username := strings.TrimPrefix(args[0], "@")
	err := bs.coreService.DestroyUser(ctx, username)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to destroy user: %v", err))
	}

	return c.Send(fmt.Sprintf("User @%s has been successfully destroyed.", username))
}

func (bs *BotService) handleAdminAddUser(c tele.Context) error {
	ctx := context.Background()
	if !bs.userService.IsAdmin(ctx, c.Sender().ID) {
		return c.Send(messages.ErrUnauthorized)
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
	err = bs.coreService.AddUser(ctx, telegramID, username)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to add user: %v", err))
	}

	return c.Send(fmt.Sprintf("User @%s with Telegram ID %d has been successfully added.", username, telegramID))
}

func (bs *BotService) handleAddCurrency(c tele.Context) error {
	ctx := context.Background()

	args := c.Args()
	if len(args) != 3 {
		return c.Send("Usage: /addcurrency <code> <name> <sign>")
	}

	code := strings.ToUpper(args[0])
	name := args[1]
	sign := args[2]

	err := bs.coreService.AddCurrency(ctx, code, name, sign)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to add currency: %v", err))
	}

	return c.Send(fmt.Sprintf("Currency %s (%s) with sign %s has been successfully added.", code, name, sign))
}

func (bs *BotService) handleAdminSetDefaultCurrency(c tele.Context) error {
	ctx := context.Background()
	if !bs.userService.IsAdmin(ctx, c.Sender().ID) {
		return c.Send(messages.ErrUnauthorized)
	}

	args := c.Args()
	if len(args) != 1 {
		return c.Send("Usage: /setdefaultcurrency <code>")
	}

	code := strings.ToUpper(args[0])

	err := bs.coreService.SetDefaultCurrency(ctx, code)
	if err != nil {
		return c.Send(fmt.Sprintf("Failed to set default currency: %v", err))
	}

	return c.Send(fmt.Sprintf("Default currency has been set to %s.", code))
}

// Helper function to split long messages
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
