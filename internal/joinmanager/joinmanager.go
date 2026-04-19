package joinmanager

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/joinids/bot/internal/accounts"
	"github.com/joinids/bot/internal/database"
	"github.com/xelaj/mtproto/telegram"
)

type JoinResult struct {
	Success     int
	AlreadyIn   int
	Failed      int
	Invalid     int
	FloodWaited int
}

type floodEntry struct {
	until time.Time
}

var (
	floodMu    sync.Mutex
	floodUntil = make(map[string]floodEntry)
)

var linkRe = regexp.MustCompile(`https?://(?:t\.me|telegram\.me|telegram\.dog)/[^\s\)\]]+`)

func NormalizeLink(link string) string {
	link = strings.TrimSpace(link)
	link = strings.ReplaceAll(link, "telegram.me", "t.me")
	link = strings.ReplaceAll(link, "telegram.dog", "t.me")
	if strings.HasPrefix(link, "@") {
		link = "https://t.me/" + link[1:]
	}
	if !strings.HasPrefix(link, "http") {
		link = "https://t.me/" + link
	}
	return link
}

func ExtractLinks(text string) []string {
	raw := linkRe.FindAllString(text, -1)
	seen := make(map[string]struct{})
	var out []string
	for _, l := range raw {
		n := NormalizeLink(l)
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			out = append(out, n)
		}
	}
	return out
}

func isAvailable(phone string) bool {
	floodMu.Lock()
	defer floodMu.Unlock()
	entry, ok := floodUntil[phone]
	if !ok {
		return true
	}
	if time.Now().After(entry.until) {
		delete(floodUntil, phone)
		return true
	}
	return false
}

func setFlood(phone string, seconds int) {
	floodMu.Lock()
	defer floodMu.Unlock()
	floodUntil[phone] = floodEntry{until: time.Now().Add(time.Duration(seconds) * time.Second)}
}

func FloodStatus() map[string]time.Duration {
	floodMu.Lock()
	defer floodMu.Unlock()
	out := make(map[string]time.Duration)
	for phone, entry := range floodUntil {
		remaining := time.Until(entry.until)
		if remaining > 0 {
			out[phone] = remaining
		} else {
			delete(floodUntil, phone)
		}
	}
	return out
}

type joinStatus int

const (
	statusSuccess joinStatus = iota
	statusAlready
	statusFlood
	statusInvalid
	statusFailed
)

func joinOne(client *telegram.Client, acc database.Account, link string) (joinStatus, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	isInvite := strings.Contains(link, "/+") || strings.Contains(link, "/joinchat/")
	parts := strings.Split(link, "/")
	slug := parts[len(parts)-1]

	var err error
	var title string

	if isInvite {
		hash := strings.TrimPrefix(slug, "+")
		res, e := client.MessagesImportChatInvite(ctx, hash)
		err = e
		if res != nil {
			switch u := res.(type) {
			case *telegram.UpdatesObj:
				if len(u.Chats) > 0 {
					switch ch := u.Chats[0].(type) {
					case *telegram.ChatObj:
						title = ch.Title
					case *telegram.Channel:
						title = ch.Title
					}
				}
			}
		}
	} else {
		res, e := client.ChannelsJoinChannel(ctx, &telegram.InputChannelObj{})
		if e != nil {
			chRes, e2 := client.ContactsResolveUsername(ctx, slug)
			if e2 != nil {
				err = e2
			} else if len(chRes.Chats) > 0 {
				switch ch := chRes.Chats[0].(type) {
				case *telegram.Channel:
					joinRes, joinErr := client.ChannelsJoinChannel(ctx, &telegram.InputChannelObj{
						ChannelID:  ch.ID,
						AccessHash: ch.AccessHash,
					})
					err = joinErr
					if joinRes != nil {
						title = ch.Title
					}
				case *telegram.ChatObj:
					title = ch.Title
				}
			}
		} else {
			err = nil
			_ = res
		}
	}

	if err == nil {
		if title == "" {
			title = link
		}
		return statusSuccess, title
	}

	errStr := err.Error()

	if strings.Contains(errStr, "USER_ALREADY_PARTICIPANT") {
		return statusAlready, ""
	}

	if strings.Contains(errStr, "FLOOD_WAIT_") {
		var seconds int
		fmt.Sscanf(errStr, "FLOOD_WAIT_%d", &seconds)
		if seconds == 0 {
			seconds = 60
		}
		setFlood(acc.Phone, seconds)
		return statusFlood, ""
	}

	if strings.Contains(errStr, "INVITE_HASH_EXPIRED") ||
		strings.Contains(errStr, "INVITE_HASH_INVALID") ||
		strings.Contains(errStr, "USERNAME_INVALID") ||
		strings.Contains(errStr, "CHANNEL_PRIVATE") {
		return statusInvalid, ""
	}

	return statusFailed, errStr
}

type ProgressFunc func(done, total int, result JoinResult)

func JoinLinks(accs []database.Account, links []string, onProgress ProgressFunc) JoinResult {
	seen := make(map[string]struct{})
	var deduped []string
	for _, l := range links {
		n := NormalizeLink(l)
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			deduped = append(deduped, n)
		}
	}

	result := JoinResult{}
	total := len(deduped)

	activeClients := make(map[string]*telegram.Client)
	for _, acc := range accs {
		client, err := accounts.GetClient(acc)
		if err != nil {
			continue
		}
		activeClients[acc.Phone] = client
	}

	for i, link := range deduped {
		joined := false

		for _, acc := range accs {
			if !isAvailable(acc.Phone) {
				continue
			}

			client, ok := activeClients[acc.Phone]
			if !ok {
				continue
			}

			status, _ := joinOne(client, acc, link)

			switch status {
			case statusSuccess:
				result.Success++
				joined = true
			case statusAlready:
				result.AlreadyIn++
				joined = true
			case statusInvalid:
				result.Invalid++
				joined = true
			case statusFlood:
				result.FloodWaited++
				continue
			case statusFailed:
				result.Failed++
			}

			if joined {
				break
			}

			time.Sleep(3 * time.Second)
		}

		if !joined {
			result.Failed++
		}

		if onProgress != nil {
			onProgress(i+1, total, result)
		}
	}

	return result
}

func FetchLinksFromChannel(acc database.Account, channelUsername string, startID, endID int) ([]string, int, error) {
	client, err := accounts.GetClient(acc)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	isPrivate := strings.Contains(channelUsername, "/+") || strings.Contains(channelUsername, "/joinchat/")

	var inputPeer telegram.InputPeer

	if isPrivate {
		parts := strings.Split(channelUsername, "/")
		hash := strings.TrimPrefix(parts[len(parts)-1], "+")
		res, err := client.MessagesImportChatInvite(ctx, hash)
		if err != nil && !strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			return nil, 0, fmt.Errorf("failed to join private channel: %w", err)
		}
		_ = res
		resolved, err := client.ContactsResolveUsername(ctx, hash)
		if err != nil || len(resolved.Chats) == 0 {
			return nil, 0, fmt.Errorf("could not resolve private channel after join")
		}
		switch ch := resolved.Chats[0].(type) {
		case *telegram.Channel:
			inputPeer = &telegram.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
		default:
			return nil, 0, fmt.Errorf("unexpected chat type")
		}
	} else {
		username := strings.TrimPrefix(channelUsername, "@")
		username = strings.TrimPrefix(username, "https://t.me/")
		resolved, err := client.ContactsResolveUsername(ctx, username)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to resolve channel: %w", err)
		}
		if len(resolved.Chats) == 0 {
			return nil, 0, fmt.Errorf("channel not found: %s", username)
		}
		switch ch := resolved.Chats[0].(type) {
		case *telegram.Channel:
			inputPeer = &telegram.InputPeerChannel{ChannelID: ch.ID, AccessHash: ch.AccessHash}
		case *telegram.ChatObj:
			inputPeer = &telegram.InputPeerChat{ChatID: ch.ID}
		default:
			return nil, 0, fmt.Errorf("unexpected chat type")
		}
	}

	var allLinks []string
	totalMessages := 0
	offsetID := 0

	for {
		history, err := client.MessagesGetHistory(ctx, inputPeer, &telegram.MessagesGetHistoryParams{
			OffsetID:  int32(offsetID),
			AddOffset: 0,
			Limit:     100,
			MaxID:     0,
			MinID:     0,
		})
		if err != nil {
			return allLinks, totalMessages, fmt.Errorf("failed to fetch messages: %w", err)
		}

		var messages []telegram.Message
		switch h := history.(type) {
		case *telegram.MessagesMessagesObj:
			messages = h.Messages
		case *telegram.MessagesMessagesSlice:
			messages = h.Messages
		case *telegram.MessagesChannelMessages:
			messages = h.Messages
		}

		if len(messages) == 0 {
			break
		}

		for _, raw := range messages {
			msg, ok := raw.(*telegram.MessageObj)
			if !ok {
				continue
			}

			msgID := int(msg.ID)

			if startID > 0 && endID > 0 {
				if msgID < startID || msgID > endID {
					if msgID < startID {
						goto done
					}
					continue
				}
			} else if startID > 0 && endID == 0 {
				if msgID != startID {
					continue
				}
			}

			totalMessages++
			if msg.Message != "" {
				allLinks = append(allLinks, ExtractLinks(msg.Message)...)
			}

			offsetID = int(msg.ID)
		}

		if startID > 0 && offsetID <= startID {
			break
		}
	}

done:
	return allLinks, totalMessages, nil
}
