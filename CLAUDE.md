# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

McDuck Wallet is a Telegram Mini App wallet service with dual interfaces:
- **Telegram Bot**: Command-based interaction via `gopkg.in/telebot.v3`
- **Web Mini App**: HTMX + Pico CSS frontend with Templ templates

**Tech Stack**: Go 1.23, Chi router, GORM + SQLite, Templ templating, Telebot v3

## Build & Development Commands

```bash
# Development with hot reload (watches .go, .templ files)
air

# Build for production (Linux x86_64 static binary)
make build

# Run tests
make test

# Deploy to remote server (builds, transfers, restarts)
make deploy

# Generate Templ templates (required before build)
templ generate ./...
```

## Server Deployment

The app runs on remote server via nohup (not systemd). Deploy restarts the process:
```bash
ssh <server> 'pkill mcduck-wallet || true; cd ~/mcduck-wallet && nohup ./bin/mcduck-wallet > /dev/null 2>&1 &'
```

Environment: `TELEGRAM_BOT_TOKEN` must be set. Server listens on port 80.

## Architecture

```
cmd/mcduck-wallet/main.go    # Entry point, wires services together
internal/
├── services/                # Business logic layer
│   ├── core_service.go      # Wallet ops: transfer, balance, history
│   ├── user_service.go      # User CRUD, admin checks
│   └── notification_service.go  # Telegram notifications
├── database/                # GORM models & DB connection
│   ├── models.go            # User, Balance, Transaction, Currency
│   └── database.go          # SQLite connection, auto-migrations
├── webapp/                  # Web Mini App
│   ├── web_service.go       # HTTP handlers
│   ├── auth.go              # Telegram HMAC-SHA256 auth
│   └── views/*.templ        # UI templates (generate *_templ.go)
├── bot/                     # Telegram bot command handlers
└── handlers/                # Chi route registration
```

**Service Pattern**: All services are interfaces (CoreService, UserService, NotificationService) with constructor injection.

**Authentication**: Telegram Mini App init data validated via HMAC-SHA256 with bot token.

## Key Implementation Details

- Multi-currency support with configurable default currency
- Transfer minimum: 0.01, self-transfer blocked
- Transaction history limited to last 10
- Notifications sent async (non-blocking on transfer failure)
- User soft-deletion via GORM DeletedAt
- Amount formatting: `%.2f` for currency display

## Database

SQLite file: `mcduck_wallet.db` (auto-created with migrations on startup)

Models: User → Balance → Currency, User → Transaction
