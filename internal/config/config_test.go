package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUrlsFromJSON(t *testing.T) {
	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "urls.json")

	content := `["google.com", "https://github.com", "http://example.org"]`
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	urls, err := LoadUrlsFromJSON(jsonPath)
	if err != nil {
		t.Fatalf("expected no error loading JSON, got %v", err)
	}

	if len(urls) != 3 {
		t.Errorf("expected 3 urls, got %d", len(urls))
	}
	if urls[0] != "google.com" || urls[1] != "https://github.com" {
		t.Errorf("unexpected content in parsed JSON urls: %v", urls)
	}
}

func TestLoadUrlsFromJSON_Invalid(t *testing.T) {
	_, err := LoadUrlsFromJSON("non_existent_file.json")
	if err == nil {
		t.Errorf("expected error for missing file, got nil")
	}

	tempDir := t.TempDir()
	jsonPath := filepath.Join(tempDir, "invalid.json")
	_ = os.WriteFile(jsonPath, []byte("invalid json content"), 0644)

	_, err = LoadUrlsFromJSON(jsonPath)
	if err == nil {
		t.Errorf("expected error for invalid json, got nil")
	}
}

func TestLoadSecrets(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test_bot_token")
	t.Setenv("TELEGRAM_CHAT_ID", "test_chat_id")
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/test")

	secrets := LoadSecrets()
	if secrets.Telegram == nil || secrets.Telegram.BotToken != "test_bot_token" {
		t.Errorf("failed to load Telegram secrets correctly")
	}
	if secrets.Discord == nil || secrets.Discord.WebhookURL != "https://discord.com/api/webhooks/test" {
		t.Errorf("failed to load Discord secrets correctly")
	}
}
