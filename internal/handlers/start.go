package handlers

import (
	"fmt"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/state"
	"github.com/joinids/bot/internal/config"
	"github.com/joinids/bot/internal/database"
	"github.com/joinids/bot/internal/keyboards"
)

func isOwner(userID int64) bool {
	return userID == config.C.OwnerID
}

func isAuthorized(userID int64) (bool, error) {
	if isOwner(userID) {
		return true, nil
	}
	return database.Instance.IsSudoer(userID)
}

func HandleStart(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	_ = database.Instance.AddUser(userID)

	owner := isOwner(userID)

	text := "👋 Welcome to *JoinIDs Bot*

"
	text += "🤖 Manage multiple Telegram accounts and auto-join channels/groups.

"

	if owner {
		text += "👑 You are the *Owner*.

"
	} else {
		auth, _ := isAuthorized(userID)
		if auth {
			text += "⭐ You are a *Sudoer*.

"
		} else {
			text += "👤 You are a regular user.

"
		}
	}

	text += "Use the menu below to navigate."

	state.States.Clear(userID)

	_, err := ctx.EffectiveMessage.Reply(b, text, &gotgbot.SendMessageOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboards.MainMenu(owner),
	})
	return err
}

func HandleBackMain(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	state.States.Clear(userID)

	owner := isOwner(userID)
	_, _, err := ctx.EffectiveMessage.EditText(b, "🏠 Main menu", &gotgbot.EditMessageTextOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{},
	})
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(b, "🏠 Main menu", &gotgbot.SendMessageOpts{
			ReplyMarkup: keyboards.MainMenu(owner),
		})
	}
	_ = fmt.Sprintf("%v", owner)
	return err
}
