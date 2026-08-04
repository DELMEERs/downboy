package checker

import (
	"context"
	"io"
	"net"
	"net/http"
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

var (
	// defaultTransport reuses HTTP keep-alive connections across health check passes to minimize latency and socket overhead.
	defaultTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// defaultClient is the default shared client instance.
	defaultClient = &http.Client{
		Transport: defaultTransport,
		Timeout:   5 * time.Second,
	}
)

// DefaultOptions returns sensible defaults for checking URLs.
func DefaultOptions() CheckOptions {
	return CheckOptions{
		Timeout: 5 * time.Second,
		Retries: 1,
		Client:  defaultClient,
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

// httpHead is overridden in unit tests to avoid real network/DNS calls
// nil by default to avoid reflection overhead during production execution
var httpHead func(url string) (*http.Response, error)

// normalizeURL normalizes a raw URL string into a full URL (with scheme) and clean URL (without scheme)
// memory optimization: uses slice slicing rather than string trim/concatenation allocations when prefixes exist
func normalizeURL(raw string) (fullURL string, cleanURL string) {
	if strings.HasPrefix(raw, "https://") {
		return raw, raw[8:]
	}
	if strings.HasPrefix(raw, "http://") {
		return raw, raw[7:]
	}
	return "http://" + raw, raw
}

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

	url, clean := normalizeURL(rawURL)

	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.Client == nil {
		if opts.Timeout == 5*time.Second {
			opts.Client = defaultClient
		} else {
			opts.Client = &http.Client{
				Transport: defaultTransport,
				Timeout:   opts.Timeout,
			}
		}
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

	// drain response body up to 4KB to enable TCP connection reuse in HTTP Keep-Alive transport pool
	defer func() {
		if resp.Body != nil {
			_, _ = io.CopyN(io.Discard, resp.Body, 4096)
			_ = resp.Body.Close()
		}
	}()

	if n != nil {
		n.NotifySuccess(clean, resp.StatusCode, duration)
	}

	return Result{
		URL:        url,
		CleanURL:   clean,
		StatusCode: resp.StatusCode,
		Duration:   duration,
		Err:        nil,
		IsUp:       true,
	}
}

func doRequestWithContext(ctx context.Context, client *http.Client, targetURL string) (*http.Response, error) {
	// If httpHead hook is non-nil, use it (used for mock unit tests without reflection)
	if httpHead != nil {
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
		if resp.Body != nil {
			_, _ = io.CopyN(io.Discard, resp.Body, 4096)
			_ = resp.Body.Close()
		}
		getReq, getErr := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, http.NoBody)
		if getErr != nil {
			return nil, getErr
		}
		return client.Do(getReq)
	}

	return resp, nil
}
