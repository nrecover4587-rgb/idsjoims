package handlers

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/accounts"
	"github.com/joinids/bot/internal/state"
	"github.com/joinids/bot/internal/database"
	"github.com/joinids/bot/internal/keyboards"
)

func HandleAddAccountMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	auth, _ := isAuthorized(userID)

	pubAdd, _ := database.Instance.GetSetting("public_add_enabled")
	publicOK := pubAdd != nil && pubAdd.(bool)

	if !auth && !publicOK {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ You don't have permission to add accounts.", nil)
		return err
	}

	state.States.Clear(userID)
	_, err := ctx.EffectiveMessage.Reply(b, "📱 Choose a method to add your account:", &gotgbot.SendMessageOpts{
		ReplyMarkup: keyboards.AddAccount(),
	})
	return err
}

func HandleCBAddPyrogram(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	state.States.Set(userID, state.StateAddPyrogram, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📝 Send your *Pyrogram* session string.

Generate one at @StringSessionBot",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleCBAddTelethon(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	state.States.Set(userID, state.StateAddTelethon, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📝 Send your *Telethon* session string.

Generate one at @StringSessionBot",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleCBAddDirect(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	state.States.Set(userID, state.StateAddDirectPhone, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📞 Send your phone number with country code.

Example: `+919876543210`",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleAddAccountText(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	st := state.States.Get(userID)
	text := strings.TrimSpace(ctx.EffectiveMessage.Text)

	switch st.Key {
	case state.StateAddPyrogram:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Validating Pyrogram session...", nil)
		acc, err := accounts.ValidatePyrogramSession(text, userID)
		if err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Invalid session:
`%s`", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		state.States.Clear(userID)
		_, _ = msg.EditText(b, fmt.Sprintf("✅ Account *%s* added successfully!", acc.Phone), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})

	case state.StateAddTelethon:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Validating Telethon session...", nil)
		acc, err := accounts.ValidateTelethonSession(text, userID)
		if err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Invalid session:
`%s`", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		state.States.Clear(userID)
		_, _ = msg.EditText(b, fmt.Sprintf("✅ Account *%s* added successfully!", acc.Phone), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})

	case state.StateAddDirectPhone:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Sending OTP...", nil)
		if err := accounts.StartDirectLogin(userID, text); err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Failed to send OTP:
`%s`", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		state.States.Set(userID, state.StateAddDirectCode, map[string]interface{}{"phone": text})
		_, _ = msg.EditText(b,
			fmt.Sprintf("📨 OTP sent to `%s`

Send the code with spaces:
Example: `1 2 3 4 5`", text),
			&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
		)

	case state.StateAddDirectCode:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Verifying code...", nil)
		acc, needs2FA, err := accounts.SubmitCode(userID, text)
		if err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Error:
`%s`

Start again from ➕ Add Account", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if needs2FA {
			state.States.Set(userID, state.StateAddDirect2FA, nil)
			_, _ = msg.EditText(b, "🔐 Two-step verification is enabled.

Send your *2FA password*:", &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		state.States.Clear(userID)
		_, _ = msg.EditText(b, fmt.Sprintf("✅ Account *%s* logged in and saved!", acc.Phone), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})

	case state.StateAddDirect2FA:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Checking 2FA password...", nil)
		acc, err := accounts.Submit2FA(userID, text)
		if err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Wrong password:
`%s`

Start again from ➕ Add Account", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			state.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		state.States.Clear(userID)
		_, _ = msg.EditText(b, fmt.Sprintf("✅ Account *%s* logged in and saved!", acc.Phone), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})

	default:
		return nil
	}

	return nil
}

func HandleMyAccounts(b *gotgbot.Bot, ctx *ext.Context) error {
	accs, err := database.Instance.GetAllAccounts()
	if err != nil || len(accs) == 0 {
		_, err = ctx.EffectiveMessage.Reply(b, "📭 No accounts added yet.", nil)
		return err
	}
	_, err = ctx.EffectiveMessage.Reply(b,
		fmt.Sprintf("📋 Total Accounts: *%d*

Select an account:", len(accs)),
		&gotgbot.SendMessageOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.AccountList(accs)},
	)
	return err
}

func HandleCBShowAccounts(b *gotgbot.Bot, ctx *ext.Context) error {
	accs, err := database.Instance.GetAllAccounts()
	if err != nil || len(accs) == 0 {
		_, _, err = ctx.EffectiveMessage.EditText(b, "📭 No accounts added yet.", nil)
		return err
	}
	_, _, err = ctx.EffectiveMessage.EditText(b,
		fmt.Sprintf("📋 Total Accounts: *%d*

Select an account:", len(accs)),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.AccountList(accs)},
	)
	return err
}

func HandleCBAccountDetail(b *gotgbot.Bot, ctx *ext.Context) error {
	var idx int
	fmt.Sscanf(ctx.CallbackQuery.Data, "acc_%d", &idx)

	accs, _ := database.Instance.GetAllAccounts()
	if idx >= len(accs) {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Account not found!", ShowAlert: true})
		return nil
	}

	acc := accs[idx]
	text := fmt.Sprintf("📱 *Account Details*

Phone: `%s`
Type: `%s`
Telegram UID: `%d`", acc.Phone, acc.Type, acc.TelegramUID)
	_, _, err := ctx.EffectiveMessage.EditText(b, text, &gotgbot.EditMessageTextOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboards.AccountDetail(idx),
	})
	return err
}

func HandleCBDeleteAccount(b *gotgbot.Bot, ctx *ext.Context) error {
	var idx int
	fmt.Sscanf(ctx.CallbackQuery.Data, "del_acc_%d", &idx)

	accs, _ := database.Instance.GetAllAccounts()
	if idx >= len(accs) {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Account not found!", ShowAlert: true})
		return nil
	}

	acc := accs[idx]
	accounts.ReleaseClient(acc.Phone)
	_ = database.Instance.DeleteAccount(acc.ID)

	_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "✅ Account deleted!", ShowAlert: true})
	return HandleCBShowAccounts(b, ctx)
}
