package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Cfg struct {
	BotToken string
	OwnerID  int64
	MongoURI string
	APIID    int
	APIHash  string
}

var C *Cfg

func Load() {
	_ = godotenv.Load()

	ownerID, err := strconv.ParseInt(mustEnv("OWNER_ID"), 10, 64)
	if err != nil {
		log.Fatalf("invalid OWNER_ID: %v", err)
	}

	apiID, err := strconv.Atoi(getEnv("API_ID", "38145963"))
	if err != nil {
		log.Fatalf("invalid API_ID: %v", err)
	}

	C = &Cfg{
		BotToken: mustEnv("BOT_TOKEN"),
		OwnerID:  ownerID,
		MongoURI: mustEnv("MONGO_URI"),
		APIID:    apiID,
		APIHash:  getEnv("API_HASH", "9325201ac0b1f87528cede06dd88484d"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
