// File: pkg/messages/messages.go

package messages

const (
	// Error messages
	ErrInsufficientBalance  = "Insufficient balance"
	ErrUserNotFound         = "Recipient not found."
	ErrUnauthorized         = "Unauthorized: user is not an admin"
	ErrInvalidAmount        = "Invalid amount. Please enter a number."
	ErrNegativeAmount       = "Transfer amount must be positive"
	ErrCannotTransferToSelf = "Cannot transfer to self"
	ErrCurrencyMismatch     = "Cannot transfer between different currencies"

	// Info messages
	InfoTransferSuccessful = "Successfully transferred %.2f to @%s"
	InfoWelcome            = "Welcome to McDuck Wallet, @%s! Your personal finance assistant. Your current balance is ¤%.2f.\n\nUse the button below to open the WebApp."
	InfoNoTransactions     = "No transactions found."

	// Command usage
	UsageTransfer = "Usage: /transfer <@username> <amount>"
)
