package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/state"
	"github.com/joinids/bot/internal/database"
	"github.com/joinids/bot/internal/keyboards"
)

func HandleManageSudoers(b *gotgbot.Bot, ctx *ext.Context) error {
	if !isOwner(ctx.EffectiveUser.Id) {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ Owner only.", nil)
		return err
	}
	sudoers, _ := database.Instance.GetSudoers()
	_, err := ctx.EffectiveMessage.Reply(b,
		fmt.Sprintf("👥 Sudoers: *%d*

Select one to manage:", len(sudoers)),
		&gotgbot.SendMessageOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.SudoerList(sudoers)},
	)
	return err
}

func HandleCBShowSudoers(b *gotgbot.Bot, ctx *ext.Context) error {
	sudoers, _ := database.Instance.GetSudoers()
	_, _, err := ctx.EffectiveMessage.EditText(b,
		fmt.Sprintf("👥 Sudoers: *%d*

Select one to manage:", len(sudoers)),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.SudoerList(sudoers)},
	)
	return err
}

func HandleCBAddSudoer(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	if !isOwner(userID) {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Owner only!", ShowAlert: true})
		return nil
	}
	state.States.Set(userID, state.StateAddSudoer, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📝 Send the Telegram *user ID* to add as sudoer:

Example: `123456789`",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleCBSudoerDetail(b *gotgbot.Bot, ctx *ext.Context) error {
	var sudoerID int64
	fmt.Sscanf(ctx.CallbackQuery.Data, "sudo_%d", &sudoerID)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		fmt.Sprintf("👤 Sudoer ID: `%d`

Manage this sudoer:", sudoerID),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.SudoerDetail(sudoerID)},
	)
	return err
}

func HandleCBRemoveSudoer(b *gotgbot.Bot, ctx *ext.Context) error {
	var sudoerID int64
	fmt.Sscanf(ctx.CallbackQuery.Data, "del_sudo_%d", &sudoerID)
	_ = database.Instance.RemoveSudoer(sudoerID)
	_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "✅ Sudoer removed!", ShowAlert: true})
	return HandleCBShowSudoers(b, ctx)
}

func HandleSudoerText(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	st := state.States.Get(userID)
	if st.Key != state.StateAddSudoer {
		return nil
	}

	newID, err := strconv.ParseInt(strings.TrimSpace(ctx.EffectiveMessage.Text), 10, 64)
	if err != nil {
		_, err = ctx.EffectiveMessage.Reply(b, "❌ Invalid user ID. Send a numeric ID.", nil)
		return err
	}

	if newID == userID {
		_, err = ctx.EffectiveMessage.Reply(b, "❌ You cannot add yourself as sudoer.", nil)
		state.States.Clear(userID)
		return err
	}

	_ = database.Instance.AddSudoer(newID)
	state.States.Clear(userID)
	_, err = ctx.EffectiveMessage.Reply(b, fmt.Sprintf("✅ User `%d` added as sudoer!", newID), &gotgbot.SendMessageOpts{ParseMode: "Markdown"})
	return err
}

func HandleStatistics(b *gotgbot.Bot, ctx *ext.Context) error {
	if !isOwner(ctx.EffectiveUser.Id) {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ Owner only.", nil)
		return err
	}

	users, _ := database.Instance.GetAllUsers()
	accs, _ := database.Instance.GetAllAccounts()
	sudoers, _ := database.Instance.GetSudoers()
	channels, _ := database.Instance.GetAllDBChannels()

	text := fmt.Sprintf(
		"📊 *Bot Statistics*

"+
			"👥 Total Users: `%d`
"+
			"📱 Total Accounts: `%d`
"+
			"⭐ Sudoers: `%d`
"+
			"📢 DB Channels: `%d`",
		len(users), len(accs), len(sudoers), len(channels),
	)
	_, err := ctx.EffectiveMessage.Reply(b, text, &gotgbot.SendMessageOpts{ParseMode: "Markdown"})
	return err
}

func HandleSettings(b *gotgbot.Bot, ctx *ext.Context) error {
	if !isOwner(ctx.EffectiveUser.Id) {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ Owner only.", nil)
		return err
	}

	val, _ := database.Instance.GetSetting("public_add_enabled")
	publicAdd := val != nil && val.(bool)

	_, err := ctx.EffectiveMessage.Reply(b,
		"⚙️ *Bot Settings*",
		&gotgbot.SendMessageOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.Settings(publicAdd)},
	)
	return err
}

func HandleCBTogglePublicAdd(b *gotgbot.Bot, ctx *ext.Context) error {
	if !isOwner(ctx.EffectiveUser.Id) {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Owner only!", ShowAlert: true})
		return nil
	}

	val, _ := database.Instance.GetSetting("public_add_enabled")
	current := val != nil && val.(bool)
	newVal := !current
	_ = database.Instance.SetSetting("public_add_enabled", newVal)

	status := "❌ Disabled"
	if newVal {
		status = "✅ Enabled"
	}
	_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: fmt.Sprintf("Public Add: %s", status), ShowAlert: true})
	_, _, err := ctx.EffectiveMessage.EditText(b, "⚙️ *Bot Settings*", &gotgbot.EditMessageTextOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboards.Settings(newVal),
	})
	return err
}

func HandleBroadcast(b *gotgbot.Bot, ctx *ext.Context) error {
	if !isOwner(ctx.EffectiveUser.Id) {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ Owner only.", nil)
		return err
	}
	state.States.Set(ctx.EffectiveUser.Id, state.StateBroadcast, nil)
	_, err := ctx.EffectiveMessage.Reply(b, "📣 Send the message to broadcast to all users:", nil)
	return err
}

func HandleBroadcastText(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	st := state.States.Get(userID)
	if st.Key != state.StateBroadcast {
		return nil
	}

	users, _ := database.Instance.GetAllUsers()
	msg, _ := ctx.EffectiveMessage.Reply(b,
		fmt.Sprintf("📢 Broadcasting to %d users...

✅ 0 ❌ 0", len(users)),
		nil,
	)

	success, failed := 0, 0
	for i, uid := range users {
		_, err := b.SendMessage(uid, ctx.EffectiveMessage.Text, nil)
		if err != nil {
			failed++
		} else {
			success++
		}

		if (i+1)%20 == 0 || i+1 == len(users) {
			_, _ = msg.EditText(b,
				fmt.Sprintf("📢 Broadcasting...

✅ Success: %d
❌ Failed: %d
📊 Progress: %d/%d",
					success, failed, i+1, len(users)),
				nil,
			)
		}
	}

	state.States.Clear(userID)
	_, _ = msg.EditText(b,
		fmt.Sprintf("✅ *Broadcast Completed!*

✅ Success: %d
❌ Failed: %d
📊 Total: %d",
			success, failed, len(users)),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return nil
}
