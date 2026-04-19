package handlers

import (
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/state"
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
		state.States.Clear(userID)
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

	st := state.States.Get(userID)

	switch st.Key {
	case state.StateAddPyrogram, state.StateAddTelethon,
		state.StateAddDirectPhone, state.StateAddDirectCode, state.StateAddDirect2FA:
		return HandleAddAccountText(b, ctx)

	case state.StateAddDBChannel:
		return HandleDBChannelText(b, ctx)

	case state.StateAddSudoer:
		return HandleSudoerText(b, ctx)

	case state.StateBroadcast:
		return HandleBroadcastText(b, ctx)

	case state.StateJoinManual:
		return HandleJoinManualText(b, ctx)

	case state.StateJoinRange:
		return HandleJoinRangeText(b, ctx)
	}

	return nil
}
