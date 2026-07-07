package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"downboy/internal/checker"
	"downboy/internal/notifier"
)

func main() {
	urls := os.Args[1:]
	if len(urls) == 0 {
		fmt.Println("using the application: ./downboy [websites]")
		os.Exit(1)
	}

	note := notifier.ConsoleNotifier{}

	var activeWebsites []string
	for _, url := range urls {
		if checker.CheckURL(url, nil, note) {
			activeWebsites = append(activeWebsites, url)
		}
	}
	if len(activeWebsites) == 0 {
		fmt.Println("no working websites found")
		os.Exit(1)
	}

	for {
		var wg sync.WaitGroup
		for _, url := range activeWebsites {
			wg.Add(1)
			go checker.CheckURL(url, &wg, note)
		}
		wg.Wait()
		time.Sleep(10 * time.Second)
	}
}
