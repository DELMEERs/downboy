package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Secrets struct {
	TelegramBotToken string
	TelegramChatID   string
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

func LoadSecrets() (*Secrets, error) {
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")

	if token == "" || chatID == "" {
		return nil, fmt.Errorf("missing TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID in environment")
	}

	return &Secrets{
		TelegramBotToken: token,
		TelegramChatID:   chatID,
	}, nil
}
