// File: ./internal/messages/messages.go
package messages

const (
	InfoWelcome            = "Welcome to McDuck Wallet, @%s! Your personal finance assistant.\nUse the button below to open the WebApp."
	InfoTransferSuccessful = "Successfully transferred %.2f %s to @%s"
	InfoNoTransactions     = "No transactions found"
	ErrUserNotFound        = "User not found."
	ErrInvalidAmount       = "Invalid amount. Please provide a number greater than 0.01."
	ErrUnauthorized        = "Unauthorized: This command is only available for admin accounts."
	UsageTransfer          = "Usage: /transfer <@username> <amount> [<currency_code>]"
)
