package main

import (
	"log"
	"net/http"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/joinids/bot/internal/bot"
	"github.com/joinids/bot/internal/config"
	"github.com/joinids/bot/internal/database"
)

func main() {
	config.Load()

	if err := database.Connect(); err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	log.Println("MongoDB connected.")

	b, err := gotgbot.NewBot(config.C.BotToken, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{Timeout: 30 * time.Second},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	log.Printf("Bot started: @%s", b.Username)

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Printf("Handler error: %v", err)
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	bot.Setup(b, dispatcher)

	updater := ext.NewUpdater(dispatcher, nil)

	if err := updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout:        10,
			AllowedUpdates: []string{"message", "callback_query"},
		},
	}); err != nil {
		log.Fatalf("Failed to start polling: %v", err)
	}

	log.Println("Bot is polling for updates...")
	updater.Idle()
}
