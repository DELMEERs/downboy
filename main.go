package main

import (
	"downboy/notifier"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func CheckURL(urlStr string, wg *sync.WaitGroup, n notifier.Notifier) bool {

	if wg != nil {
		defer wg.Done()
	}
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "http://" + urlStr
	}
	cleanURL := urlStr
	cleanURL = strings.TrimPrefix(cleanURL, "https://")
	cleanURL = strings.TrimPrefix(cleanURL, "http://")

	start := time.Now()
	resp, err := http.Head(urlStr)
	if err != nil {
		errStr := err.Error()
		if lastIndex := strings.LastIndex(errStr, ": "); lastIndex != -1 {
			errStr = errStr[lastIndex+2:]
		}
		n.NotifyError(cleanURL, errStr)
		return false
	}

	duration := time.Since(start).Round(time.Millisecond)
	resp.Body.Close()

	n.NotifySuccess(cleanURL, resp.StatusCode, duration)
	return true

}

func main() {
	ticker := time.NewTicker(10 * time.Second)
	note := notifier.ConsoleNotifier{}
	defer ticker.Stop()

	if len(os.Args[1:]) == 0 {
		fmt.Println("using the application: ./downboy [websites]")
		os.Exit(1)
	}

	var activeWebsites []string

	for _, value := range os.Args[1:] {
		if CheckURL(value, nil, note) {
			activeWebsites = append(activeWebsites, value)
		}
	}

	if len(activeWebsites) == 0 {
		fmt.Println("no working websites found")
		os.Exit(1)
	}

	for {
		var wg sync.WaitGroup
		for _, value := range activeWebsites {
			wg.Add(1)
			go CheckURL(value, &wg, note)
		}
		wg.Wait()
		<-ticker.C
	}

}
