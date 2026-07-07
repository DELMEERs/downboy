package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func CheckURL(urlStr string) {

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
		fmt.Printf("[error] %s - %v\n", urlStr, errStr)
		return
	}

	duration := time.Since(start).Round(time.Millisecond)
	resp.Body.Close()

	fmt.Printf("[%d] %s - %s\n", resp.StatusCode, urlStr, duration)

}

func main() {
	if len(os.Args[1:]) == 0 {
		fmt.Println("using the application: ./downboy [websites]")
		os.Exit(1)
	}

	for _, value := range os.Args[1:] {
		CheckURL(value)
	}

}
