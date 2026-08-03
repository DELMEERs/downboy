package checker

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"downboy/internal/notifier"
)

// CheckOptions configures the behavior of URL checks.
type CheckOptions struct {
	Timeout time.Duration
	Retries int
	Client  *http.Client
}

// DefaultOptions returns sensible defaults for checking URLs.
func DefaultOptions() CheckOptions {
	return CheckOptions{
		Timeout: 5 * time.Second,
		Retries: 1,
		Client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Result holds metrics and status details for a single URL check.
type Result struct {
	URL        string
	CleanURL   string
	StatusCode int
	Duration   time.Duration
	Err        error
	IsUp       bool
}

// httpHead is swapped out in tests to avoid real network/DNS calls.
var httpHead func(url string) (*http.Response, error) = http.Head

// CheckURL sends a request to validate website availability (backwards compatible).
func CheckURL(url string, wg *sync.WaitGroup, n notifier.Notifier) bool {
	res := CheckURLWithOptions(context.Background(), url, wg, n, DefaultOptions())
	return res.IsUp
}

// CheckURLWithOptions executes a website health check with context, timeout, and retries.
func CheckURLWithOptions(ctx context.Context, rawURL string, wg *sync.WaitGroup, n notifier.Notifier, opts CheckOptions) Result {
	if wg != nil {
		defer wg.Done()
	}

	url := rawURL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	clean := strings.TrimPrefix(url, "https://")
	clean = strings.TrimPrefix(clean, "http://")

	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: opts.Timeout}
	}

	var lastErr error
	var resp *http.Response
	var duration time.Duration

	attempts := opts.Retries + 1
	if attempts < 1 {
		attempts = 1
	}

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*150) * time.Millisecond)
		}

		reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
		start := time.Now()

		resp, lastErr = doRequestWithContext(reqCtx, opts.Client, url)
		duration = time.Since(start).Round(time.Millisecond)
		cancel()

		if lastErr == nil {
			break
		}
	}

	if lastErr != nil {
		errStr := lastErr.Error()
		if idx := strings.LastIndex(errStr, ": "); idx != -1 {
			errStr = errStr[idx+2:]
		}
		if n != nil {
			n.NotifyError(clean, errStr)
		}
		return Result{
			URL:        url,
			CleanURL:   clean,
			StatusCode: 0,
			Duration:   duration,
			Err:        lastErr,
			IsUp:       false,
		}
	}

	defer resp.Body.Close()

	if n != nil {
		n.NotifySuccess(clean, resp.StatusCode, duration)
	}

	// Server responded without network error
	isUp := true

	return Result{
		URL:        url,
		CleanURL:   clean,
		StatusCode: resp.StatusCode,
		Duration:   duration,
		Err:        nil,
		IsUp:       isUp,
	}
}

func doRequestWithContext(ctx context.Context, client *http.Client, targetURL string) (*http.Response, error) {
	// If httpHead function has been overridden for mocking in unit tests:
	if httpHead != nil && reflect.ValueOf(httpHead).Pointer() != reflect.ValueOf(http.Head).Pointer() {
		return httpHead(targetURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Fallback to GET if HEAD method is not allowed (HTTP 405)
	if resp.StatusCode == http.StatusMethodNotAllowed {
		_ = resp.Body.Close()
		getReq, getErr := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, http.NoBody)
		if getErr != nil {
			return nil, getErr
		}
		return client.Do(getReq)
	}

	return resp, nil
}
