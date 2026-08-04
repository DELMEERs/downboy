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
	"slices"
	"strconv"
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

	cachedSuccessBracket = successStyle.Render("[ ✓ ]")
	cachedFailureBracket = failureStyle.Render("[ X ]")
	cachedSpinnerFrames  []string

	bufPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
)

func init() {
	cachedSpinnerFrames = make([]string, len(SpinnerFrames))
	for i, frame := range SpinnerFrames {
		cachedSpinnerFrames[i] = spinnerStyle.Render(frame)
	}
}

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

// cleanURL removes protocol prefixes using string slicing (zero-allocation when prefix present)
func cleanURL(raw string) string {
	if strings.HasPrefix(raw, "https://") {
		return raw[8:]
	}
	if strings.HasPrefix(raw, "http://") {
		return raw[7:]
	}
	return raw
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
		items:    make([]*checkItem, 0, len(urls)),
		itemMap:  make(map[string]*checkItem, len(urls)*2),
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
				cn.frame = (cn.frame + 1) % len(cachedSpinnerFrames)
				cn.redrawListLocked()
			}
			cn.mu.Unlock()
		}
	}
}

// formatLineLockedToBuf writes the item state directly to a bytes.Buffer to avoid intermediate string allocations
func (cn *ConsoleNotifier) formatLineLockedToBuf(buf *bytes.Buffer, item *checkItem) {
	switch item.status {
	case statusSuccess:
		buf.WriteString(cachedSuccessBracket)
		buf.WriteByte(' ')
		buf.WriteString(item.cleanURL)
		if item.statusCode > 0 {
			buf.WriteString(" - ")
			buf.WriteString(strconv.Itoa(item.statusCode))
			buf.WriteString(" (")
			buf.WriteString(item.duration.String())
			buf.WriteByte(')')
		}
	case statusFailure:
		buf.WriteString(cachedFailureBracket)
		buf.WriteByte(' ')
		buf.WriteString(item.cleanURL)
		if item.errStr != "" {
			buf.WriteString(" - ")
			buf.WriteString(item.errStr)
		}
	default:
		buf.WriteString(cachedSpinnerFrames[cn.frame])
		buf.WriteByte(' ')
		buf.WriteString(item.cleanURL)
	}
}

func (cn *ConsoleNotifier) renderListLocked() {
	if len(cn.items) == 0 {
		return
	}
	w := cn.getWriter()
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	for _, item := range cn.items {
		cn.formatLineLockedToBuf(buf, item)
		buf.WriteByte('\n')
	}
	_, _ = w.Write(buf.Bytes())
	cn.linesDrawn = len(cn.items)
}

func (cn *ConsoleNotifier) redrawListLocked() {
	if len(cn.items) == 0 {
		return
	}
	w := cn.getWriter()
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	if cn.linesDrawn > 0 {
		fmt.Fprintf(buf, "\033[%dA", cn.linesDrawn)
	}
	for _, item := range cn.items {
		buf.WriteString("\033[2K\r")
		cn.formatLineLockedToBuf(buf, item)
		buf.WriteByte('\n')
	}
	_, _ = w.Write(buf.Bytes())
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
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	buf.WriteString(cachedSuccessBracket)
	buf.WriteByte(' ')
	buf.WriteString(clean)
	buf.WriteString(" - ")
	buf.WriteString(strconv.Itoa(statusCode))
	buf.WriteString(" (")
	buf.WriteString(duration.String())
	buf.WriteString(")\n")

	_, _ = w.Write(buf.Bytes())
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
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)

	buf.WriteString(cachedFailureBracket)
	buf.WriteByte(' ')
	buf.WriteString(clean)
	buf.WriteString(" - ")
	buf.WriteString(errStr)
	buf.WriteByte('\n')

	_, _ = w.Write(buf.Bytes())
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

	// memory and speed optimization: slices.SortFunc uses type-safe sorting without reflection overhead
	slices.SortFunc(cn.items, func(a, b *checkItem) int {
		if a.status != b.status {
			if a.status == statusFailure {
				return -1
			}
			if b.status == statusFailure {
				return 1
			}
		}
		return strings.Compare(a.cleanURL, b.cleanURL)
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

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(formData.Encode()))
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

// concrete struct for JSON encoding eliminates map[string]interface{} interface boxing overhead
type discordPayload struct {
	Content string `json:"content"`
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
	payload := discordPayload{
		Content: fmt.Sprintf("🚨 **ALERT! Website is DOWN**\n**URL:** `%s`\n**Error:** `%s`\n**Time:** `%s`",
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

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, dn.webhookURL, bytes.NewReader(bodyBytes))
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

// direct loop eliminates goroutine spawning & sync.WaitGroup overhead for memory/console updates
func (mn *MultiNotifier) NotifySuccess(url string, statusCode int, duration time.Duration) {
	for _, n := range mn.notifiers {
		n.NotifySuccess(url, statusCode, duration)
	}
}

// direct loop eliminates goroutine spawning & sync.WaitGroup overhead for memory/console updates
func (mn *MultiNotifier) NotifyError(url string, err string) {
	for _, n := range mn.notifiers {
		n.NotifyError(url, err)
	}
}
