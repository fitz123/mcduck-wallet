// File: ./internal/webapp/views/views.go
package views

import (
	"context"

	"github.com/a-h/templ"
	"github.com/fitz123/mcduck-wallet/internal/database"
)

func InitialLoadingIndex() templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w templ.Writer) error {
		return w.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>McDuck Wallet WebApp</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://telegram.org/js/telegram-web-app.js"></script>
    <script src="https://unpkg.com/htmx.org@2.0.2"></script>
</head>
<body>
    <main data-page="main"></main>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            let tg = window.Telegram.WebApp;
            tg.expand();
            tg.ready();
            htmx.ajax('GET', '/dashboard', {target: 'body', swap: 'innerHTML'});
        });
    </script>
</body>
</html>`)
	})
}

func MainContent(balances []database.Balance, message string, isSuccess bool) templ.Component {
	// Implement rendering of the main content with balances and message
	return templ.ComponentFunc(func(ctx context.Context, w templ.Writer) error {
		// Render the balances and messages
		return nil
	})
}

func TransferForm(balances []database.Balance) templ.Component {
	// Implement rendering of the transfer form
	return templ.ComponentFunc(func(ctx context.Context, w templ.Writer) error {
		// Render the transfer form
		return nil
	})
}

func TransactionHistory(transactions []database.Transaction) templ.Component {
	// Implement rendering of the transaction history
	return templ.ComponentFunc(func(ctx context.Context, w templ.Writer) error {
		// Render the transaction history
		return nil
	})
}
