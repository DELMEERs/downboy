package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"downboy/internal/checker"
	"downboy/internal/notifier"

	"github.com/charmbracelet/lipgloss"
)

var (
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
)

func main() {
	flag.Parse()
	urls := flag.Args()

	if len(urls) == 0 {
		fmt.Println(errorStyle.Render("using the application: ./downboy [websites]"))
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
		fmt.Println(errorStyle.Render("no working websites found"))
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
