<p align="center">
  <img src="logo/duck.svg" width="120" alt="McDuck Wallet">
</p>

<h1 align="center">McDuck Wallet</h1>

<p align="center">
  A family finance Telegram Mini App that teaches kids financial literacy through real wallet mechanics.
</p>

---

McDuck Wallet is a home project built around a simple idea: kids learn about money by using it. Every family member gets their own multi-currency wallet inside Telegram. We give allowances, lend to each other, transfer between currencies, and track everything -- just like a real bank, but within the family.

The built-in currency exchange with live rates is especially handy when traveling -- kids can see exactly how much their pocket money is worth in the local currency.

## How it works

**Telegram Bot** -- quick commands for transfers, balances, and admin ops:

```
/balance                              -- check your wallets
/transfer @kid 50 EUR "for ice cream" -- send money with a note
/exchange 100 USD EUR                 -- convert currencies at live rates
/history                              -- paginated transaction log
/addcurrency GEM Gems 💎 10           -- create a custom family currency
```

**Web Mini App** -- tap "Open McDuck Wallet" in the bot for a full UI with:
- Dashboard with all balances at a glance
- Transfer form with recipient autocomplete and currency memory
- Live exchange rate preview as you type
- Infinite-scroll transaction history
- Custom currency creation

## Features

- **Multi-currency wallets** -- real currencies (USD, EUR, RUB, ...) and custom ones (chore tokens, game coins, whatever the family invents)
- **Live exchange rates** -- fetched from ExchangeRate-API, cached for 2 hours, with graceful fallback
- **Custom currencies** -- set a fixed rate to USD and mix them freely with real currencies
- **Transfer notes** -- attach a reason to any transfer ("birthday gift", "borrowed for lunch")
- **Notifications** -- Telegram messages for every transfer, exchange, and admin action
- **Admin controls** -- set balances, manage users, adjust exchange rates
- **Telegram-native auth** -- HMAC-SHA256 validation of Mini App init data, no passwords

## Tech stack

Go 1.23 | Chi | GORM + SQLite | Templ | HTMX + Pico CSS | Telebot v3

## Quick start

```bash
# Set up
cp .env.example .env
# Fill in TELEGRAM_BOT_TOKEN and WEBAPP_URL

# Development (hot reload)
air

# Or run directly
make run

# Tests
make test
```

## Deployment

```bash
# First-time setup (creates user, firewall, systemd service)
make deploy

# Update existing deployment
make update

# Verify deployment
make verify
```

Builds a static Linux amd64 binary, transfers it via SCP, and manages a systemd service on the remote host.

## Project structure

```
cmd/mcduck-wallet/main.go        -- entry point
internal/
  bot/                            -- Telegram bot commands
  webapp/                         -- HTTP handlers + Telegram auth
    views/                        -- Templ templates (HTMX partials)
  services/                       -- business logic (transfers, exchange, notifications)
  database/                       -- GORM models, SQLite connection
  handlers/                       -- Chi route registration
scripts/                          -- deploy, update, verify
logo/                             -- duck assets
```

## License

Private family project. Not currently accepting contributions.
