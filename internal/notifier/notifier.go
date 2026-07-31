package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Notifier is an interface that defines methods for sending status notifications.
type Notifier interface {
	NotifySuccess(url string, statusCode int, duration time.Duration)
	NotifyError(url string, err string)
}

type ConsoleNotifier struct{}

func (ConsoleNotifier) NotifySuccess(targetURL string, statusCode int, duration time.Duration) {
	fmt.Printf("[%d] %s - %s\n", statusCode, targetURL, duration)
}

func (ConsoleNotifier) NotifyError(targetURL string, errStr string) {
	fmt.Printf("[error] %s - %s\n", targetURL, errStr)
}

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: token,
		chatID:   chatID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (tn *TelegramNotifier) NotifySuccess(targetURL string, statusCode int, duration time.Duration) {
	// Optional: telegram success notification
}

func (tn *TelegramNotifier) NotifyError(targetURL string, errStr string) {
	message := fmt.Sprintf("🚨 *ALERT! Website is DOWN*\n\n*URL:* %s\n*Error:* %s\n*Time:* %s",
		targetURL, errStr, time.Now().Format("2006-01-02 15:04:05"))

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", tn.botToken)

	formData := url.Values{}
	formData.Set("chat_id", tn.chatID)
	formData.Set("text", message)
	formData.Set("parse_mode", "Markdown")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBufferString(formData.Encode()))
		if err != nil {
			fmt.Printf("[tg] error creating request: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := tn.client.Do(req)
		if err != nil {
			fmt.Printf("[tg] failed to send request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("[tg] telegram API returned status: %s\nresponse: %s\n",
				resp.Status, string(body))
		}
	}()
}

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (dn *DiscordNotifier) NotifySuccess(targetURL string, statusCode int, duration time.Duration) {}

func (dn *DiscordNotifier) NotifyError(targetURL string, errStr string) {
	payload := map[string]interface{}{
		"content": fmt.Sprintf("🚨 **ALERT! Website is DOWN**\n**URL:** `%s`\n**Error:** `%s`\n**Time:** `%s`",
			targetURL, errStr, time.Now().Format("2006-01-02 15:04:05")),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[discord] failed to marshal JSON: %v\n", err)
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, dn.webhookURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			fmt.Printf("[discord] error creating request: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := dn.client.Do(req)
		if err != nil {
			fmt.Printf("[discord] failed to send request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			respBody, _ := io.ReadAll(resp.Body)
			fmt.Printf("[discord] API returned status: %s\nresponse: %s\n",
				resp.Status, string(respBody))
		}
	}()
}

type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (mn *MultiNotifier) NotifySuccess(url string, statusCode int, duration time.Duration) {
	var wg sync.WaitGroup
	for _, n := range mn.notifiers {
		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			notifier.NotifySuccess(url, statusCode, duration)
		}(n)
	}
	wg.Wait()
}

func (mn *MultiNotifier) NotifyError(url string, err string) {
	var wg sync.WaitGroup
	for _, n := range mn.notifiers {
		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			notifier.NotifyError(url, err)
		}(n)
	}
	wg.Wait()
}
