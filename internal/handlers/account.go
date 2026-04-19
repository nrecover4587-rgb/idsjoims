package handlers

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/accounts"
	"github.com/joinids/bot/internal/bot"
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

	bot.States.Clear(userID)
	_, err := ctx.EffectiveMessage.Reply(b, "📱 Choose a method to add your account:", &gotgbot.SendMessageOpts{
		ReplyMarkup: keyboards.AddAccount(),
	})
	return err
}

func HandleCBAddPyrogram(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	bot.States.Set(userID, bot.StateAddPyrogram, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📝 Send your *Pyrogram* session string.\n\nGenerate one at @StringSessionBot",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleCBAddTelethon(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	bot.States.Set(userID, bot.StateAddTelethon, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📝 Send your *Telethon* session string.\n\nGenerate one at @StringSessionBot",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleCBAddDirect(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	bot.States.Set(userID, bot.StateAddDirectPhone, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📞 Send your phone number with country code.\n\nExample: `+919876543210`",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleAddAccountText(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	st := bot.States.Get(userID)
	text := strings.TrimSpace(ctx.EffectiveMessage.Text)

	switch st.Key {
	case bot.StateAddPyrogram:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Validating Pyrogram session...", nil)
		acc, err := accounts.ValidatePyrogramSession(text, userID)
		if err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Invalid session:\n`%s`", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		bot.States.Clear(userID)
		_, _ = msg.EditText(b, fmt.Sprintf("✅ Account *%s* added successfully!", acc.Phone), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})

	case bot.StateAddTelethon:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Validating Telethon session...", nil)
		acc, err := accounts.ValidateTelethonSession(text, userID)
		if err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Invalid session:\n`%s`", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		bot.States.Clear(userID)
		_, _ = msg.EditText(b, fmt.Sprintf("✅ Account *%s* added successfully!", acc.Phone), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})

	case bot.StateAddDirectPhone:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Sending OTP...", nil)
		if err := accounts.StartDirectLogin(userID, text); err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Failed to send OTP:\n`%s`", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		bot.States.Set(userID, bot.StateAddDirectCode, map[string]interface{}{"phone": text})
		_, _ = msg.EditText(b,
			fmt.Sprintf("📨 OTP sent to `%s`\n\nSend the code with spaces:\nExample: `1 2 3 4 5`", text),
			&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
		)

	case bot.StateAddDirectCode:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Verifying code...", nil)
		acc, needs2FA, err := accounts.SubmitCode(userID, text)
		if err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Error:\n`%s`\n\nStart again from ➕ Add Account", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if needs2FA {
			bot.States.Set(userID, bot.StateAddDirect2FA, nil)
			_, _ = msg.EditText(b, "🔐 Two-step verification is enabled.\n\nSend your *2FA password*:", &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		bot.States.Clear(userID)
		_, _ = msg.EditText(b, fmt.Sprintf("✅ Account *%s* logged in and saved!", acc.Phone), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})

	case bot.StateAddDirect2FA:
		msg, _ := ctx.EffectiveMessage.Reply(b, "⏳ Checking 2FA password...", nil)
		acc, err := accounts.Submit2FA(userID, text)
		if err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, fmt.Sprintf("❌ Wrong password:\n`%s`\n\nStart again from ➕ Add Account", err.Error()), &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
			return nil
		}
		if err := database.Instance.AddAccount(acc); err != nil {
			bot.States.Clear(userID)
			_, _ = msg.EditText(b, "❌ Failed to save account to DB.", nil)
			return nil
		}
		bot.States.Clear(userID)
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
		fmt.Sprintf("📋 Total Accounts: *%d*\n\nSelect an account:", len(accs)),
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
		fmt.Sprintf("📋 Total Accounts: *%d*\n\nSelect an account:", len(accs)),
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
	text := fmt.Sprintf("📱 *Account Details*\n\nPhone: `%s`\nType: `%s`\nTelegram UID: `%d`", acc.Phone, acc.Type, acc.TelegramUID)
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
