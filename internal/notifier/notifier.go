package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"downboy/internal/theme"
)

// Notifier is an interface that defines methods for sending status notifications.
type Notifier interface {
	NotifySuccess(url string, statusCode int, duration time.Duration)
	NotifyError(url string, err string)
}

type itemStatus int

const (
	statusInProgress itemStatus = iota
	statusSuccess
	statusFailure
)

var SpinnerFrames = []string{"[   ]", "[-  ]", "[-- ]", "[---]", "[ --]", "[  -]"}

var (
	successStyle = theme.SuccessStyle
	failureStyle = theme.FailureStyle
	spinnerStyle = theme.SpinnerStyle
)

type checkItem struct {
	rawURL     string
	cleanURL   string
	status     itemStatus
	statusCode int
	duration   time.Duration
	errStr     string
}

type ConsoleNotifier struct {
	mu         sync.Mutex
	urls       []string
	items      []*checkItem
	itemMap    map[string]*checkItem
	writer     io.Writer
	frame      int
	ticker     *time.Ticker
	stopChan   chan struct{}
	doneChan   chan struct{}
	stopped    bool
	linesDrawn int
}

func cleanURL(raw string) string {
	clean := strings.TrimPrefix(raw, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	return clean
}

func NewConsoleNotifier(urls []string) *ConsoleNotifier {
	return NewConsoleNotifierWithWriter(urls, os.Stdout)
}

func NewConsoleNotifierWithWriter(urls []string, w io.Writer) *ConsoleNotifier {
	if w == nil {
		w = os.Stdout
	}

	cn := &ConsoleNotifier{
		urls:     urls,
		itemMap:  make(map[string]*checkItem),
		writer:   w,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}

	for _, u := range urls {
		clean := cleanURL(u)
		item := &checkItem{
			rawURL:   u,
			cleanURL: clean,
			status:   statusInProgress,
		}
		cn.items = append(cn.items, item)
		cn.itemMap[u] = item
		cn.itemMap[clean] = item
	}

	if len(urls) > 0 {
		cn.mu.Lock()
		cn.renderListLocked()
		cn.mu.Unlock()

		cn.ticker = time.NewTicker(100 * time.Millisecond)
		go cn.animate()
	}

	return cn
}

func (cn *ConsoleNotifier) getWriter() io.Writer {
	if cn.writer != nil {
		return cn.writer
	}
	return os.Stdout
}

func (cn *ConsoleNotifier) animate() {
	for {
		select {
		case <-cn.stopChan:
			close(cn.doneChan)
			return
		case <-cn.ticker.C:
			cn.mu.Lock()
			if !cn.stopped {
				cn.frame = (cn.frame + 1) % len(SpinnerFrames)
				cn.redrawListLocked()
			}
			cn.mu.Unlock()
		}
	}
}

func (cn *ConsoleNotifier) formatLineLocked(item *checkItem) string {
	switch item.status {
	case statusSuccess:
		bracket := successStyle.Render("[ ✓ ]")
		if item.statusCode > 0 {
			return fmt.Sprintf("%s %s - %d (%s)", bracket, item.cleanURL, item.statusCode, item.duration)
		}
		return fmt.Sprintf("%s %s", bracket, item.cleanURL)
	case statusFailure:
		bracket := failureStyle.Render("[ X ]")
		if item.errStr != "" {
			return fmt.Sprintf("%s %s - %s", bracket, item.cleanURL, item.errStr)
		}
		return fmt.Sprintf("%s %s", bracket, item.cleanURL)
	default:
		frameStr := SpinnerFrames[cn.frame]
		bracket := spinnerStyle.Render(frameStr)
		return fmt.Sprintf("%s %s", bracket, item.cleanURL)
	}
}

func (cn *ConsoleNotifier) renderListLocked() {
	if len(cn.items) == 0 {
		return
	}
	w := cn.getWriter()
	for _, item := range cn.items {
		fmt.Fprintln(w, cn.formatLineLocked(item))
	}
	cn.linesDrawn = len(cn.items)
}

func (cn *ConsoleNotifier) redrawListLocked() {
	if len(cn.items) == 0 {
		return
	}
	w := cn.getWriter()
	if cn.linesDrawn > 0 {
		fmt.Fprintf(w, "\033[%dA", cn.linesDrawn)
	}
	for _, item := range cn.items {
		fmt.Fprintf(w, "\033[2K\r%s\n", cn.formatLineLocked(item))
	}
	cn.linesDrawn = len(cn.items)
}

func (cn *ConsoleNotifier) NotifySuccess(targetURL string, statusCode int, duration time.Duration) {
	cn.mu.Lock()
	defer cn.mu.Unlock()

	clean := cleanURL(targetURL)
	if cn.itemMap != nil {
		item, ok := cn.itemMap[targetURL]
		if !ok {
			item, ok = cn.itemMap[clean]
		}
		if ok {
			item.status = statusSuccess
			item.statusCode = statusCode
			item.duration = duration
			if !cn.stopped {
				cn.redrawListLocked()
			}
			return
		}
	}

	w := cn.getWriter()
	bracket := successStyle.Render("[ ✓ ]")
	fmt.Fprintf(w, "%s %s - %d (%s)\n", bracket, clean, statusCode, duration)
}

func (cn *ConsoleNotifier) NotifyError(targetURL string, errStr string) {
	cn.mu.Lock()
	defer cn.mu.Unlock()

	clean := cleanURL(targetURL)
	if cn.itemMap != nil {
		item, ok := cn.itemMap[targetURL]
		if !ok {
			item, ok = cn.itemMap[clean]
		}
		if ok {
			item.status = statusFailure
			item.errStr = errStr
			if !cn.stopped {
				cn.redrawListLocked()
			}
			return
		}
	}

	w := cn.getWriter()
	bracket := failureStyle.Render("[ X ]")
	fmt.Fprintf(w, "%s %s - %s\n", bracket, clean, errStr)
}

func (cn *ConsoleNotifier) Stop() {
	cn.mu.Lock()
	if cn.stopped {
		cn.mu.Unlock()
		return
	}
	cn.stopped = true
	if cn.ticker != nil {
		cn.ticker.Stop()
		close(cn.stopChan)
	} else {
		close(cn.doneChan)
	}
	cn.mu.Unlock()

	if cn.ticker != nil {
		<-cn.doneChan
	}

	cn.mu.Lock()
	defer cn.mu.Unlock()

	sort.Slice(cn.items, func(i, j int) bool {
		if cn.items[i].status != cn.items[j].status {
			if cn.items[i].status == statusFailure {
				return true
			}
			if cn.items[j].status == statusFailure {
				return false
			}
		}
		return cn.items[i].cleanURL < cn.items[j].cleanURL
	})

	cn.redrawListLocked()
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

func (*TelegramNotifier) NotifySuccess(_ string, _ int, _ time.Duration) {
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

func (*DiscordNotifier) NotifySuccess(_ string, _ int, _ time.Duration) {}

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
