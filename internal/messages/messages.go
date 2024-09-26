// File: ./internal/messages/messages.go
package messages

const (
	InfoWelcome            = "Welcome to McDuck Wallet, @%s! Your personal finance assistant.\nUse the button below to open the WebApp."
	InfoTransferSuccessful = "Successfully transferred %.0f %s to @%s"
	InfoNoTransactions     = "No transactions found"
	ErrUserNotFound        = "User not found."
	ErrInvalidAmount       = "Invalid amount. Please enter a number."
	ErrUnauthorized        = "Unauthorized: This command is only available for admin accounts."
	UsageTransfer          = "Usage: /transfer <@username> <amount> [<currency_code>]"
	// Add other messages as needed
)
