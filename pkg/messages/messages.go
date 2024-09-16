// File: pkg/messages/messages.go

package messages

const (
	// Error messages
	ErrInsufficientBalance = "Insufficient balance"
	ErrUserNotFound        = "User not found"
	ErrUnauthorized        = "Unauthorized: user is not an admin"
	ErrInvalidAmount       = "Invalid amount. Please enter a number."
	ErrRecipientNotFound   = "Recipient not found."
	ErrNegativeAmount      = "Transfer amount must be positive"

	// Info messages
	InfoCurrentBalance     = "Your current balance is ¤%.2f"
	InfoTransferSuccessful = "Successfully transferred ¤%.2f to %s"
	InfoWelcome            = "Welcome to McDuck Wallet, %s! Your personal finance assistant. Your current balance is ¤%.2f.\n\nUse the button below to open the WebApp."
	InfoNoTransactions     = "No transactions found."
	InfoRecentTransactions = "Your recent transactions:\n\n"

	// Command usage
	UsageTransfer = "Usage: /transfer <@username> <amount>"
	UsageSet      = "Usage: /set <@username> <key=value>"

	// Transaction descriptions
	TransactionSent      = "Sent ¤%.2f to %s"
	TransactionReceived  = "Received ¤%.2f from %s"
	TransactionDeposited = "Deposited ¤%.2f"
)
