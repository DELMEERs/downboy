package notifier

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Notifier is an interface that defines methods for sending status notifications.
type Notifier interface {
	NotifySuccess(url string, statusCode int, duration time.Duration)
	NotifyError(url string, err string)
}

type ConsoleNotifier struct{}

func (ConsoleNotifier) NotifySuccess(url string, statusCode int, duration time.Duration) {
	fmt.Printf("[%d] %s - %s\n", statusCode, url, duration)
}

func (ConsoleNotifier) NotifyError(url string, err string) {
	fmt.Printf("[error] %s - %s\n", url, err)
}

type TelegramNotifier struct {
	botToken string
	chatID   string
}

func NewTelegramNotifier(token, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: token,
		chatID:   chatID,
	}
}

func (tn *TelegramNotifier) NotifySuccess(url string, statusCode int, duration time.Duration) {

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
		resp, err := http.PostForm(apiURL, formData)
		if err != nil {
			fmt.Printf("[tg] failed to send request: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("[tg] telegram API returned status: %s\nresponse: %s\n",
				resp.Status, string(body))
			return
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
	for _, n := range mn.notifiers {
		n.NotifySuccess(url, statusCode, duration)
	}
}

func (mn *MultiNotifier) NotifyError(url string, err string) {
	for _, n := range mn.notifiers {
		n.NotifyError(url, err)
	}
}
