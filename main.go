package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func CheckURL(urlStr string, wg *sync.WaitGroup) {

	defer wg.Done()
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
		fmt.Printf("[error] %s - %v\n", cleanURL, errStr)
		return
	}

	duration := time.Since(start).Round(time.Millisecond)
	resp.Body.Close()

	fmt.Printf("[%d] %s - %s\n", resp.StatusCode, cleanURL, duration)

}

func main() {
	var wg sync.WaitGroup

	if len(os.Args[1:]) == 0 {
		fmt.Println("using the application: ./downboy [websites]")
		os.Exit(1)
	}

	for _, value := range os.Args[1:] {
		wg.Add(1)
		go CheckURL(value, &wg)
	}
	wg.Wait()

}
