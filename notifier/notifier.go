package notifier

import (
	"fmt"
	"time"
)

type Notifier interface {
	NotifySuccess(urlStr string, statusCode int, duration time.Duration)
	NotifyError(urlStr string, errStr string)
}

type ConsoleNotifier struct{}

func (c ConsoleNotifier) NotifySuccess(urlStr string, statusCode int, duration time.Duration) {
	fmt.Printf("[%d] %s - %s\n", statusCode, urlStr, duration)
}

func (c ConsoleNotifier) NotifyError(urlStr string, errStr string) {
	fmt.Printf("[error] %s - %s\n", urlStr, errStr)
}

type TelegramNotifier struct{}

func (t TelegramNotifier) NotifySuccess(urlStr string, statusCode int, duration time.Duration) {
	fmt.Printf("[TG BOT] Successfully checked %s (%d)\n", urlStr, statusCode)
}

func (t TelegramNotifier) NotifyError(urlStr string, errStr string) {
	fmt.Printf("[TG BOT] ALERT! Website %s is down: %s\n", urlStr, errStr)
}
