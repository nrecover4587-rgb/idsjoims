package handlers

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/bot"
	"github.com/joinids/bot/internal/database"
	"github.com/joinids/bot/internal/keyboards"
)

func HandleDBChannelsMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	auth, _ := isAuthorized(userID)
	if !auth {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ You don't have permission.", nil)
		return err
	}

	channels, _ := database.Instance.GetAllDBChannels()
	_, err := ctx.EffectiveMessage.Reply(b,
		fmt.Sprintf("📢 DB Channels: *%d*\n\nSelect a channel:", len(channels)),
		&gotgbot.SendMessageOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.DBChannelList(channels, false)},
	)
	return err
}

func HandleCBShowDBChannels(b *gotgbot.Bot, ctx *ext.Context) error {
	channels, _ := database.Instance.GetAllDBChannels()
	_, _, err := ctx.EffectiveMessage.EditText(b,
		fmt.Sprintf("📢 DB Channels: *%d*\n\nSelect a channel:", len(channels)),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown", ReplyMarkup: keyboards.DBChannelList(channels, false)},
	)
	return err
}

func HandleCBAddDBChannel(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	bot.States.Set(userID, bot.StateAddDBChannel, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"📝 Send the DB channel link or username:\n\n"+
			"• Public: `https://t.me/username` or `@username`\n"+
			"• Private: `https://t.me/+hash`\n"+
			"• Format: `Name | @username`",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleCBDBChannelDetail(b *gotgbot.Bot, ctx *ext.Context) error {
	var idx int
	fmt.Sscanf(ctx.CallbackQuery.Data, "dbch_%d", &idx)

	channels, _ := database.Instance.GetAllDBChannels()
	if idx >= len(channels) {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Channel not found!", ShowAlert: true})
		return nil
	}

	ch := channels[idx]
	text := fmt.Sprintf("📢 *DB Channel Details*\n\nName: `%s`\nUsername: `%s`\nPrivate: `%v`", ch.Name, ch.Username, ch.IsPrivate)
	_, _, err := ctx.EffectiveMessage.EditText(b, text, &gotgbot.EditMessageTextOpts{
		ParseMode:   "Markdown",
		ReplyMarkup: keyboards.DBChannelDetail(idx),
	})
	return err
}

func HandleCBDeleteDBChannel(b *gotgbot.Bot, ctx *ext.Context) error {
	var idx int
	fmt.Sscanf(ctx.CallbackQuery.Data, "del_dbch_%d", &idx)

	channels, _ := database.Instance.GetAllDBChannels()
	if idx >= len(channels) {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Channel not found!", ShowAlert: true})
		return nil
	}

	_ = database.Instance.DeleteDBChannel(channels[idx].ID)
	_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "✅ Channel deleted!", ShowAlert: true})
	return HandleCBShowDBChannels(b, ctx)
}

func HandleDBChannelText(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	st := bot.States.Get(userID)
	if st.Key != bot.StateAddDBChannel {
		return nil
	}

	text := strings.TrimSpace(ctx.EffectiveMessage.Text)
	var name, username string
	isPrivate := false

	if strings.Contains(text, "/+") || strings.Contains(text, "/joinchat/") {
		parts := strings.Split(text, "/")
		hash := parts[len(parts)-1]
		name = fmt.Sprintf("Private Channel %s", hash[:min(8, len(hash))])
		username = text
		isPrivate = true
	} else if strings.Contains(text, "|") {
		parts := strings.SplitN(text, "|", 2)
		name = strings.TrimSpace(parts[0])
		username = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(parts[1]), "@"))
	} else if strings.HasPrefix(text, "https://t.me/") {
		username = strings.TrimPrefix(text, "https://t.me/")
		name = fmt.Sprintf("Channel @%s", username)
	} else if strings.HasPrefix(text, "@") {
		username = strings.TrimPrefix(text, "@")
		name = fmt.Sprintf("Channel @%s", username)
	} else {
		_, err := ctx.EffectiveMessage.Reply(b,
			"❌ Invalid format! Send either:\n• `https://t.me/username`\n• `@username`\n• `https://t.me/+hash`\n• `Name | @username`",
			&gotgbot.SendMessageOpts{ParseMode: "Markdown"},
		)
		return err
	}

	ch := database.DBChannel{Name: name, Username: username, IsPrivate: isPrivate}
	if err := database.Instance.AddDBChannel(ch); err != nil {
		_, err = ctx.EffectiveMessage.Reply(b, "❌ Failed to save channel to DB.", nil)
		bot.States.Clear(userID)
		return err
	}

	bot.States.Clear(userID)
	_, err := ctx.EffectiveMessage.Reply(b, fmt.Sprintf("✅ DB Channel *%s* added!", name), &gotgbot.SendMessageOpts{ParseMode: "Markdown"})
	return err
}
