package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"downboy/internal/checker"
	"downboy/internal/config"
	"downboy/internal/notifier"

	"github.com/charmbracelet/lipgloss"
)

var (
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
)

func main() {
	configPath := flag.String("config", "", "path to json config file containing website urls")

	flag.Usage = func() {
		fmt.Println("usage of downboy:")
		fmt.Println("  ./downboy [websites...]       Provide space-separated URLs directly")
		fmt.Println("  ./downboy --config <path>     Load URLs from a JSON config file")
	}

	flag.Parse()
	urls := flag.Args()

	if len(urls) == 0 {
		if *configPath == "" {
			flag.Usage()
			os.Exit(1)
		}

		configUrls, err := config.LoadUrlsFromJSON(*configPath)
		if err != nil {
			fmt.Println(errorStyle.Render("error loading websites from config:"), err)
			os.Exit(1)
		}
		urls = configUrls
	}

	if len(urls) == 0 {
		flag.Usage()
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
