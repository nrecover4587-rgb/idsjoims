package keyboards

import (
	"fmt"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/joinids/bot/internal/database"
)

func MainMenu(isOwner bool) gotgbot.ReplyKeyboardMarkup {
	rows := [][]gotgbot.KeyboardButton{
		{{Text: "➕ Add Account"}, {Text: "📋 My Accounts"}},
		{{Text: "🔗 Join Channels"}, {Text: "📢 DB Channels"}},
	}
	if isOwner {
		rows = append(rows,
			[]gotgbot.KeyboardButton{{Text: "👥 Manage Sudoers"}, {Text: "📊 Statistics"}},
			[]gotgbot.KeyboardButton{{Text: "⚙️ Settings"}, {Text: "📣 Broadcast"}},
		)
	}
	return gotgbot.ReplyKeyboardMarkup{Keyboard: rows, ResizeKeyboard: true}
}

func AddAccount() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "Pyrogram Session", CallbackData: "add_pyrogram"}},
		{{Text: "Telethon Session", CallbackData: "add_telethon"}},
		{{Text: "Direct Login (Phone)", CallbackData: "add_direct"}},
		{{Text: "« Back", CallbackData: "back_main"}},
	}}
}

func AccountList(accs []database.Account) gotgbot.InlineKeyboardMarkup {
	var rows [][]gotgbot.InlineKeyboardButton
	for i, acc := range accs {
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("📱 %s (%s)", acc.Phone, acc.Type), CallbackData: fmt.Sprintf("acc_%d", i)},
		})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "« Back", CallbackData: "back_main"}})
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func AccountDetail(idx int) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "🗑 Delete Account", CallbackData: fmt.Sprintf("del_acc_%d", idx)}},
		{{Text: "« Back to Accounts", CallbackData: "show_accounts"}},
	}}
}

func JoinMenu() gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "📢 Join from DB Channel", CallbackData: "join_db"}},
		{{Text: "✏️ Paste Links Manually", CallbackData: "join_manual"}},
		{{Text: "« Back", CallbackData: "back_main"}},
	}}
}

func DBChannelList(channels []database.DBChannel, forJoin bool) gotgbot.InlineKeyboardMarkup {
	var rows [][]gotgbot.InlineKeyboardButton
	for i, ch := range channels {
		cb := fmt.Sprintf("dbch_%d", i)
		if forJoin {
			cb = fmt.Sprintf("joindbch_%d", i)
		}
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("📢 %s", ch.Name), CallbackData: cb},
		})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ Add DB Channel", CallbackData: "add_db_channel"}})
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "« Back", CallbackData: "back_main"}})
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func DBChannelDetail(idx int) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "🚀 Use for Joining", CallbackData: fmt.Sprintf("joindbch_%d", idx)}},
		{{Text: "🗑 Delete Channel", CallbackData: fmt.Sprintf("del_dbch_%d", idx)}},
		{{Text: "« Back", CallbackData: "show_db_channels"}},
	}}
}

func JoinRangeOrAll(chIdx int) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "📋 All Messages", CallbackData: fmt.Sprintf("joinrange_%d_all", chIdx)}},
		{{Text: "🔢 Specific Range", CallbackData: fmt.Sprintf("joinrange_%d_range", chIdx)}},
		{{Text: "« Back", CallbackData: "show_db_channels"}},
	}}
}

func SudoerList(sudoers []int64) gotgbot.InlineKeyboardMarkup {
	var rows [][]gotgbot.InlineKeyboardButton
	for _, id := range sudoers {
		rows = append(rows, []gotgbot.InlineKeyboardButton{
			{Text: fmt.Sprintf("👤 %d", id), CallbackData: fmt.Sprintf("sudo_%d", id)},
		})
	}
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "➕ Add Sudoer", CallbackData: "add_sudoer"}})
	rows = append(rows, []gotgbot.InlineKeyboardButton{{Text: "« Back", CallbackData: "back_main"}})
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func SudoerDetail(id int64) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "🗑 Remove Sudoer", CallbackData: fmt.Sprintf("del_sudo_%d", id)}},
		{{Text: "« Back", CallbackData: "show_sudoers"}},
	}}
}

func Settings(publicAdd bool) gotgbot.InlineKeyboardMarkup {
	status := "❌ Disabled"
	if publicAdd {
		status = "✅ Enabled"
	}
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: fmt.Sprintf("Public Add Accounts: %s", status), CallbackData: "toggle_public_add"}},
		{{Text: "« Back", CallbackData: "back_main"}},
	}}
}

func Confirm(action string) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "✅ Confirm", CallbackData: "confirm_" + action}, {Text: "❌ Cancel", CallbackData: "back_main"}},
	}}
}
