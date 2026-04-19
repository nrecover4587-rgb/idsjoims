package bot

import (
	"log"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	h "github.com/joinids/bot/internal/handlers"
)

func Setup(b *gotgbot.Bot, dispatcher *ext.Dispatcher) {
	dispatcher.AddHandler(handlers.NewCommand("start", h.HandleStart))

	dispatcher.AddHandler(handlers.NewMessage(message.Text, h.HandleTextRouter))

	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("back_main"), h.HandleBackMain))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("add_pyrogram"), h.HandleCBAddPyrogram))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("add_telethon"), h.HandleCBAddTelethon))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("add_direct"), h.HandleCBAddDirect))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("show_accounts"), h.HandleCBShowAccounts))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("acc_"), h.HandleCBAccountDetail))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("del_acc_"), h.HandleCBDeleteAccount))

	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("join_db"), h.HandleCBJoinDB))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("join_manual"), h.HandleCBJoinManual))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("joindbch_"), h.HandleCBJoinDBChannel))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("joinrange_"), h.HandleCBJoinRangeChoice))

	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("add_db_channel"), h.HandleCBAddDBChannel))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("show_db_channels"), h.HandleCBShowDBChannels))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("dbch_"), h.HandleCBDBChannelDetail))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("del_dbch_"), h.HandleCBDeleteDBChannel))

	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("add_sudoer"), h.HandleCBAddSudoer))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("show_sudoers"), h.HandleCBShowSudoers))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("sudo_"), h.HandleCBSudoerDetail))
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Prefix("del_sudo_"), h.HandleCBRemoveSudoer))

	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("toggle_public_add"), h.HandleCBTogglePublicAdd))

	log.Println("All handlers registered.")
}
