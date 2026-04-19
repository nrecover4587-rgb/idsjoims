# JoinIDs Bot — Go Edition

Telegram bot to manage multiple accounts and auto-join channels/groups.
Built with Go + gotgbot + xelaj/mtproto + MongoDB.

## Setup

### 1. Clone and configure

```bash
cp .env.example .env
# Fill in your values in .env
```

### 2. Run locally

```bash
go mod tidy
go run ./cmd/main.go
```

### 3. Deploy on Heroku

```bash
heroku create your-app-name
heroku stack:set container -a your-app-name
heroku config:set BOT_TOKEN=xxx OWNER_ID=xxx MONGO_URI=xxx API_ID=xxx API_HASH=xxx
git push heroku main
heroku ps:scale worker=1
```

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `BOT_TOKEN` | ✅ | Bot token from @BotFather |
| `OWNER_ID` | ✅ | Your Telegram user ID |
| `MONGO_URI` | ✅ | MongoDB Atlas URI |
| `API_ID` | ✅ | Telegram API ID from my.telegram.org |
| `API_HASH` | ✅ | Telegram API Hash from my.telegram.org |

## Features

- Add accounts via Pyrogram session string, Telethon session string, or direct phone+OTP login
- Add DB channels to fetch join links from
- Join from DB channel (all messages or specific ID range)
- Paste links manually to join
- FloodWait tracking per account with live status
- Sudoer system — owner can grant/revoke access
- Broadcast to all users
- Stats dashboard
- Public account add toggle
