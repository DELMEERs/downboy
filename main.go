package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args[1:]) == 0 {
		fmt.Println("using the application: ./downboy [website]")
		os.Exit(1)
	}

	for _, value := range os.Args[1:] {

		urlStr := value

		if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
			urlStr = "http://" + urlStr
		}

		start := time.Now()

		resp, err := http.Head(urlStr)
		if err != nil {
			errStr := err.Error()
			if lastIndex := strings.LastIndex(errStr, ": "); lastIndex != -1 {
				errStr = errStr[lastIndex+2:]
			}

			fmt.Printf("[error] %s - %v\n", value, errStr)
			continue
		}

		duration := time.Since(start)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			fmt.Printf("[%d] %s - %s\n", resp.StatusCode, value, duration.Round(time.Millisecond))
		} else if resp.StatusCode >= 400 {
			fmt.Printf("[%d] %s - %s\n", resp.StatusCode, value, duration.Round(time.Millisecond))
		}
	}

}
