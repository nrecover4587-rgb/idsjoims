package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/joinids/bot/internal/accounts"
	"github.com/joinids/bot/internal/state"
	"github.com/joinids/bot/internal/database"
	"github.com/joinids/bot/internal/joinmanager"
	"github.com/joinids/bot/internal/keyboards"
	"github.com/xelaj/mtproto/telegram"
)

func HandleJoinMenu(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	auth, _ := isAuthorized(userID)
	if !auth {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ You don't have permission to use this.", nil)
		return err
	}
	state.States.Clear(userID)
	_, err := ctx.EffectiveMessage.Reply(b, "🔗 Choose joining method:", &gotgbot.SendMessageOpts{
		ReplyMarkup: keyboards.JoinMenu(),
	})
	return err
}

func HandleCBJoinDB(b *gotgbot.Bot, ctx *ext.Context) error {
	channels, err := database.Instance.GetAllDBChannels()
	if err != nil || len(channels) == 0 {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ No DB channels configured!", ShowAlert: true})
		return nil
	}
	_, _, err = ctx.EffectiveMessage.EditText(b, "📢 Select a DB channel:", &gotgbot.EditMessageTextOpts{
		ReplyMarkup: keyboards.DBChannelList(channels, true),
	})
	return err
}

func HandleCBJoinDBChannel(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	var idx int
	fmt.Sscanf(ctx.CallbackQuery.Data, "joindbch_%d", &idx)

	channels, _ := database.Instance.GetAllDBChannels()
	if idx >= len(channels) {
		_, _ = ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{Text: "❌ Channel not found!", ShowAlert: true})
		return nil
	}

	state.States.Set(userID, state.StateJoinRange, map[string]interface{}{
		"channel_idx": idx,
	})

	_, _, err := ctx.EffectiveMessage.EditText(b,
		fmt.Sprintf("📢 Channel: *%s*

Choose join scope:", channels[idx].Name),
		&gotgbot.EditMessageTextOpts{
			ParseMode:   "Markdown",
			ReplyMarkup: keyboards.JoinRangeOrAll(idx),
		},
	)
	return err
}

func HandleCBJoinRangeChoice(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	data := ctx.CallbackQuery.Data

	var idx int
	var mode string
	fmt.Sscanf(data, "joinrange_%d_%s", &idx, &mode)

	st := state.States.Get(userID)
	st.Data["channel_idx"] = idx

	if mode == "all" {
		st.Data["start_id"] = 0
		st.Data["end_id"] = 0
		state.States.Set(userID, state.StateJoinRange, st.Data)
		return startJoinProcess(b, ctx, userID, idx, 0, 0)
	}

	state.States.Set(userID, state.StateJoinRange, st.Data)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"🔢 Send the message ID range:

• Single message: `123`
• Range: `100-200`",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleJoinRangeText(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	st := state.States.Get(userID)
	if st.Key != state.StateJoinRange {
		return nil
	}

	text := strings.TrimSpace(ctx.EffectiveMessage.Text)
	var startID, endID int

	if strings.Contains(text, "-") {
		parts := strings.SplitN(text, "-", 2)
		s, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		e, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil {
			_, err := ctx.EffectiveMessage.Reply(b, "❌ Invalid format. Use `100-200` or `123`.", &gotgbot.SendMessageOpts{ParseMode: "Markdown"})
			return err
		}
		startID, endID = s, e
	} else {
		n, err := strconv.Atoi(text)
		if err != nil {
			_, err := ctx.EffectiveMessage.Reply(b, "❌ Invalid format. Use `100-200` or `123`.", &gotgbot.SendMessageOpts{ParseMode: "Markdown"})
			return err
		}
		startID, endID = n, n
	}

	idx, _ := st.Data["channel_idx"].(int)
	state.States.Clear(userID)
	return startJoinProcess(b, ctx, userID, idx, startID, endID)
}

func HandleCBJoinManual(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	state.States.Set(userID, state.StateJoinManual, nil)
	_, _, err := ctx.EffectiveMessage.EditText(b,
		"✏️ Paste the links you want to join (one per line or space-separated):

Supports `t.me/username` and `t.me/+hash` formats.",
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return err
}

func HandleJoinManualText(b *gotgbot.Bot, ctx *ext.Context) error {
	userID := ctx.EffectiveUser.Id
	st := state.States.Get(userID)
	if st.Key != state.StateJoinManual {
		return nil
	}

	links := joinmanager.ExtractLinks(ctx.EffectiveMessage.Text)
	if len(links) == 0 {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ No valid Telegram links found in your message.", nil)
		return err
	}

	state.States.Clear(userID)
	return runJoin(b, ctx, links)
}

func joinDBChannel(client *telegram.Client, chUsername string, isPrivate bool) error {
	bg := context.Background()

	if isPrivate {
		hash := strings.TrimPrefix(chUsername, "https://t.me/+")
		hash = strings.TrimPrefix(hash, "https://t.me/joinchat/")
		hash = strings.TrimPrefix(hash, "+")
		_, err := client.MessagesImportChatInvite(bg, hash)
		if err != nil && strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			return nil
		}
		return err
	}

	username := strings.TrimPrefix(chUsername, "https://t.me/")
	username = strings.TrimPrefix(username, "@")

	resolved, err := client.ContactsResolveUsername(bg, username)
	if err != nil {
		return fmt.Errorf("resolve failed: %w", err)
	}
	if len(resolved.Chats) == 0 {
		return fmt.Errorf("channel not found: %s", username)
	}

	switch ch := resolved.Chats[0].(type) {
	case *telegram.Channel:
		_, err = client.ChannelsJoinChannel(bg, &telegram.InputChannelObj{
			ChannelID:  ch.ID,
			AccessHash: ch.AccessHash,
		})
		if err != nil && strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			return nil
		}
		return err
	case *telegram.ChatObj:
		return nil
	default:
		return fmt.Errorf("unexpected chat type")
	}
}

func startJoinProcess(b *gotgbot.Bot, ctx *ext.Context, userID int64, chIdx, startID, endID int) error {
	channels, _ := database.Instance.GetAllDBChannels()
	if chIdx >= len(channels) {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ Channel not found.", nil)
		return err
	}
	ch := channels[chIdx]

	accs, _ := database.Instance.GetAllAccounts()
	if len(accs) == 0 {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ No accounts added. Add accounts first.", nil)
		return err
	}

	msg, _ := ctx.EffectiveMessage.Reply(b,
		fmt.Sprintf("⏳ *Step 1/3* — Joining DB channel with all accounts...

👥 Accounts: %d", len(accs)),
		&gotgbot.SendMessageOpts{ParseMode: "Markdown"},
	)

	var joined []database.Account
	var failedNames []string

	for i, acc := range accs {
		client, err := accounts.GetClient(acc)
		if err != nil {
			failedNames = append(failedNames, fmt.Sprintf("%s (client error)", acc.Phone))
			continue
		}

		joinErr := joinDBChannel(client, ch.Username, ch.IsPrivate)
		if joinErr != nil {
			short := joinErr.Error()
			if len(short) > 50 {
				short = short[:50]
			}
			failedNames = append(failedNames, fmt.Sprintf("%s (%s)", acc.Phone, short))
		} else {
			joined = append(joined, acc)
		}

		if (i+1)%5 == 0 || i+1 == len(accs) {
			_, _ = msg.EditText(b,
				fmt.Sprintf("⏳ *Step 1/3* — Joining DB channel...

👥 Progress: %d/%d
✅ Joined: %d  ❌ Failed: %d",
					i+1, len(accs), len(joined), len(failedNames)),
				&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
			)
		}
	}

	if len(joined) == 0 {
		failList := ""
		for _, f := range failedNames {
			failList += fmt.Sprintf("• %s
", f)
		}
		_, _ = msg.EditText(b,
			fmt.Sprintf("❌ *No accounts could join DB channel!*

Failed (%d):
%s", len(failedNames), failList),
			&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
		)
		return nil
	}

	_, _ = msg.EditText(b,
		fmt.Sprintf("✅ Step 1 done: *%d/%d* accounts joined

⏳ *Step 2/3* — Fetching links...", len(joined), len(accs)),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)

	links, msgCount, err := joinmanager.FetchLinksFromChannel(joined[0], ch.Username, startID, endID)
	if err != nil {
		short := err.Error()
		if len(short) > 200 {
			short = short[:200]
		}
		_, _ = msg.EditText(b,
			fmt.Sprintf("❌ Failed to fetch links:
`%s`", short),
			&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
		)
		return nil
	}

	if len(links) == 0 {
		rangeText := "All"
		if startID > 0 {
			rangeText = fmt.Sprintf("%d → %d", startID, endID)
		}
		_, _ = msg.EditText(b,
			fmt.Sprintf("❌ No links found.

Checked: %d messages
Range: %s", msgCount, rangeText),
			&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
		)
		return nil
	}

	_, _ = msg.EditText(b,
		fmt.Sprintf("✅ Step 2 done: Found *%d* links from %d messages

⏳ *Step 3/3* — Joining links...

📊 Progress: 0%%",
			len(links), msgCount),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)

	lastEdit := time.Now()

	result := joinmanager.JoinLinks(joined, links, func(done, total int, r joinmanager.JoinResult) {
		if time.Since(lastEdit) < 3*time.Second && done != total {
			return
		}
		lastEdit = time.Now()

		pct := float64(done) / float64(total) * 100
		floodInfo := buildFloodInfo()

		_, _ = msg.EditText(b,
			fmt.Sprintf("🚀 *Step 3/3 — Joining links...*

📊 Progress: %.1f%% (%d/%d)
✅ Success: %d
⚠️ Already in: %d
❌ Failed: %d
🚫 Invalid: %d%s",
				pct, done, total, r.Success, r.AlreadyIn, r.Failed, r.Invalid, floodInfo),
			&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
		)
	})

	failList := ""
	for _, f := range failedNames {
		failList += fmt.Sprintf("• %s
", f)
	}

	finalText := fmt.Sprintf(
		"✅ *All Steps Completed!*

"+
			"*Step 1 — DB Channel:*
✅ Joined: %d  ❌ Failed: %d

"+
			"*Step 2 — Links Found:* %d

"+
			"*Step 3 — Join Results:*
✅ Success: %d
⚠️ Already in: %d
❌ Failed: %d
🚫 Invalid: %d",
		len(joined), len(failedNames),
		len(links),
		result.Success, result.AlreadyIn, result.Failed, result.Invalid,
	)
	if failList != "" {
		finalText += fmt.Sprintf("

*Failed accounts:*
%s", failList)
	}

	_, _ = msg.EditText(b, finalText, &gotgbot.EditMessageTextOpts{ParseMode: "Markdown"})
	return nil
}

func runJoin(b *gotgbot.Bot, ctx *ext.Context, links []string) error {
	accs, _ := database.Instance.GetAllAccounts()
	if len(accs) == 0 {
		_, err := ctx.EffectiveMessage.Reply(b, "❌ No accounts added. Add accounts first.", nil)
		return err
	}

	msg, _ := ctx.EffectiveMessage.Reply(b,
		fmt.Sprintf("⏳ Joining *%d* links with *%d* accounts...

📊 Progress: 0%%", len(links), len(accs)),
		&gotgbot.SendMessageOpts{ParseMode: "Markdown"},
	)

	lastEdit := time.Now()

	result := joinmanager.JoinLinks(accs, links, func(done, total int, r joinmanager.JoinResult) {
		if time.Since(lastEdit) < 3*time.Second && done != total {
			return
		}
		lastEdit = time.Now()

		pct := float64(done) / float64(total) * 100
		floodInfo := buildFloodInfo()

		_, _ = msg.EditText(b,
			fmt.Sprintf("🚀 *Joining links...*

📊 Progress: %.1f%% (%d/%d)
✅ Success: %d
⚠️ Already in: %d
❌ Failed: %d
🚫 Invalid: %d%s",
				pct, done, total, r.Success, r.AlreadyIn, r.Failed, r.Invalid, floodInfo),
			&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
		)
	})

	_, _ = msg.EditText(b,
		fmt.Sprintf("✅ *Done!*

✅ Success: %d
⚠️ Already in: %d
❌ Failed: %d
🚫 Invalid: %d",
			result.Success, result.AlreadyIn, result.Failed, result.Invalid),
		&gotgbot.EditMessageTextOpts{ParseMode: "Markdown"},
	)
	return nil
}

func buildFloodInfo() string {
	floods := joinmanager.FloodStatus()
	if len(floods) == 0 {
		return ""
	}
	out := "

⏰ *FloodWait:*"
	for phone, remaining := range floods {
		mins := int(remaining.Minutes())
		secs := int(remaining.Seconds()) % 60
		out += fmt.Sprintf("
• `%s`: %dm %ds", phone, mins, secs)
	}
	return out
}
