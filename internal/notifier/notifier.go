package notifier

import (
	"fmt"
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

type TelegramNotifier struct{}

func (TelegramNotifier) NotifySuccess(url string, statusCode int, duration time.Duration) {
	fmt.Printf("[TG BOT] Successfully checked %s (%d)\n", url, statusCode)
}

func (TelegramNotifier) NotifyError(url string, err string) {
	fmt.Printf("[TG BOT] ALERT! Website %s is down: %s\n", url, err)
}
