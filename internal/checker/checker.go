package checker

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"downboy/internal/notifier"
)

// httpHead is swapped out in tests to avoid real network/DNS calls.
var httpHead = http.Head

// CheckURL sends a HEAD request to validate website availability.
func CheckURL(url string, wg *sync.WaitGroup, n notifier.Notifier) bool {
	if wg != nil {
		defer wg.Done()
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "http://" + url
	}

	clean := strings.TrimPrefix(url, "https://")
	clean = strings.TrimPrefix(clean, "http://")

	start := time.Now()
	resp, err := httpHead(url)
	if err != nil {
		errStr := err.Error()
		if idx := strings.LastIndex(errStr, ": "); idx != -1 {
			errStr = errStr[idx+2:]
		}
		n.NotifyError(clean, errStr)
		return false
	}

	duration := time.Since(start).Round(time.Millisecond)
	resp.Body.Close()

	n.NotifySuccess(clean, resp.StatusCode, duration)
	return true
}
