package handlers

import (
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/bot"
)

var menuTexts = map[string]bool{
	"➕ Add Account":    true,
	"📋 My Accounts":   true,
	"🔗 Join Channels": true,
	"📢 DB Channels":   true,
	"👥 Manage Sudoers": true,
	"📊 Statistics":    true,
	"⚙️ Settings":      true,
	"📣 Broadcast":     true,
}

func HandleTextRouter(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	text := strings.TrimSpace(ctx.EffectiveMessage.Text)

	if strings.HasPrefix(text, "/") {
		return nil
	}

	if menuTexts[text] {
		bot.States.Clear(userID)
		switch text {
		case "➕ Add Account":
			return HandleAddAccountMenu(b, ctx)
		case "📋 My Accounts":
			return HandleMyAccounts(b, ctx)
		case "🔗 Join Channels":
			return HandleJoinMenu(b, ctx)
		case "📢 DB Channels":
			return HandleDBChannelsMenu(b, ctx)
		case "👥 Manage Sudoers":
			return HandleManageSudoers(b, ctx)
		case "📊 Statistics":
			return HandleStatistics(b, ctx)
		case "⚙️ Settings":
			return HandleSettings(b, ctx)
		case "📣 Broadcast":
			return HandleBroadcast(b, ctx)
		}
	}

	st := bot.States.Get(userID)

	switch st.Key {
	case bot.StateAddPyrogram, bot.StateAddTelethon,
		bot.StateAddDirectPhone, bot.StateAddDirectCode, bot.StateAddDirect2FA:
		return HandleAddAccountText(b, ctx)

	case bot.StateAddDBChannel:
		return HandleDBChannelText(b, ctx)

	case bot.StateAddSudoer:
		return HandleSudoerText(b, ctx)

	case bot.StateBroadcast:
		return HandleBroadcastText(b, ctx)

	case bot.StateJoinManual:
		return HandleJoinManualText(b, ctx)

	case bot.StateJoinRange:
		return HandleJoinRangeText(b, ctx)
	}

	return nil
}
