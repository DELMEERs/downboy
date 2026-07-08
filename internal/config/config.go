package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type AppSecrets struct {
	Telegram *TelegramSecrets
	Discord  *DiscordSecrets
}

type TelegramSecrets struct {
	BotToken string
	ChatID   string
}

type DiscordSecrets struct {
	WebhookURL string
}

// the function takes a file name and returns a list of URLs or an error
func LoadUrlsFromJSON(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var urls []string

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&urls); err != nil {
		return nil, fmt.Errorf("JSON parsing error: %w", err)
	}

	return urls, nil
}

func LoadSecrets() *AppSecrets {
	_ = godotenv.Load()

	secrets := &AppSecrets{}
	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	tgChatID := os.Getenv("TELEGRAM_CHAT_ID")

	if tgToken != "" && tgChatID != "" {
		secrets.Telegram = &TelegramSecrets{
			BotToken: tgToken,
			ChatID:   tgChatID,
		}
	}

	discordWebhook := os.Getenv("DISCORD_WEBHOOK_URL")
	if discordWebhook != "" {
		secrets.Discord = &DiscordSecrets{
			WebhookURL: discordWebhook,
		}
	}

	return secrets
}
