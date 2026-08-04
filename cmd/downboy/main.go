package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"downboy/internal/checker"
	"downboy/internal/config"
	"downboy/internal/notifier"
	"downboy/internal/theme"
)

var (
	titleStyle   = theme.TitleStyle
	upBadgeStyle = theme.UpBadgeStyle
	downBadge    = theme.DownBadgeStyle
	infoStyle    = theme.InfoStyle
	errorStyle   = theme.ErrorStyle
	summaryStyle = theme.SummaryStyle
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "", "path to json config file containing website urls")
	concurrency := flag.Int("c", 20, "concurrency limit (worker pool size)")
	flag.IntVar(concurrency, "concurrency", 20, "concurrency limit (worker pool size)")

	intervalSec := flag.Int("i", 10, "interval in seconds between continuous monitor checks")
	flag.IntVar(intervalSec, "interval", 10, "interval in seconds between continuous monitor checks")

	timeoutSec := flag.Int("t", 5, "HTTP request timeout per check in seconds")
	flag.IntVar(timeoutSec, "timeout", 5, "HTTP request timeout per check in seconds")

	retries := flag.Int("r", 1, "number of retries for failed checks")
	flag.IntVar(retries, "retries", 1, "number of retries for failed checks")

	once := flag.Bool("once", false, "run single-pass check and exit (useful for CI/CD)")

	flag.Usage = func() {
		fmt.Println(titleStyle.Render("🐕 downboy - parallel website uptime monitor"))
		fmt.Println("usage:")
		fmt.Println("  downboy [flags] [websites...]")
		fmt.Println("\nflags:")
		fmt.Println("  -c, --concurrency <int>   Max concurrent requests (default: 20)")
		fmt.Println("  -i, --interval <int>      Check interval in seconds (default: 10)")
		fmt.Println("  -t, --timeout <int>       HTTP timeout in seconds (default: 5)")
		fmt.Println("  -r, --retries <int>       Max retries before reporting error (default: 1)")
		fmt.Println("      --once                Run single check pass and exit with status code")
		fmt.Println("      --config <path>       Load URLs from a JSON configuration file")
	}

	flag.Parse()
	urls := flag.Args()

	if len(urls) == 0 {
		if *configPath == "" {
			flag.Usage()
			return 1
		}

		configUrls, err := config.LoadUrlsFromJSON(*configPath)
		if err != nil {
			fmt.Println(errorStyle.Render("error loading websites from config:"), err)
			return 1
		}
		urls = configUrls
	}

	if len(urls) == 0 {
		flag.Usage()
		return 1
	}

	activeNotifiers := make([]notifier.Notifier, 0, 2)
	secrets := config.LoadSecrets()

	if secrets.Telegram != nil {
		fmt.Println(infoStyle.Render("[info] telegram notifier enabled"))
		activeNotifiers = append(activeNotifiers, notifier.NewTelegramNotifier(secrets.Telegram.BotToken, secrets.Telegram.ChatID))
	}

	if secrets.Discord != nil {
		fmt.Println(infoStyle.Render("[info] discord webhook notifier enabled"))
		activeNotifiers = append(activeNotifiers, notifier.NewDiscordNotifier(secrets.Discord.WebhookURL))
	}

	note := notifier.NewMultiNotifier(activeNotifiers...)

	opts := checker.CheckOptions{
		Timeout: time.Duration(*timeoutSec) * time.Second,
		Retries: *retries,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	go func() {
		select {
		case <-sigChan:
			cancel()
			fmt.Println("\n" + infoStyle.Render("shutting down downboy..."))
		case <-ctx.Done():
		}
	}()

	fmt.Println(titleStyle.Render("🐕 downboy starting uptime checks..."))
	fmt.Printf("%s URLs: %d | Concurrency: %d | Timeout: %ds | Retries: %d\n\n",
		infoStyle.Render("[config]"), len(urls), *concurrency, *timeoutSec, *retries)

	if *once {
		results := runWorkerPool(ctx, urls, *concurrency, note, opts)
		allUp := printSummary(results)
		if !allUp {
			return 1
		}
		return 0
	}

	// Continuous mode
	ticker := time.NewTicker(time.Duration(*intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		fmt.Println(infoStyle.Render(fmt.Sprintf("[%s] running health checks...", time.Now().Format("15:04:05"))))
		_ = runWorkerPool(ctx, urls, *concurrency, note, opts)

		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
		}
	}
}

func runWorkerPool(ctx context.Context, urls []string, concurrency int, note notifier.Notifier, opts checker.CheckOptions) []checker.Result {
	urlCount := len(urls)
	if concurrency <= 0 {
		concurrency = 20
	}
	if concurrency > urlCount {
		concurrency = urlCount
	}

	consoleNotif := notifier.NewConsoleNotifier(urls)
	defer consoleNotif.Stop()

	notifiers := make([]notifier.Notifier, 0, 2)
	notifiers = append(notifiers, consoleNotif)
	if note != nil {
		notifiers = append(notifiers, note)
	}
	multiNote := notifier.NewMultiNotifier(notifiers...)

	jobs := make(chan string, urlCount)
	resultsChan := make(chan checker.Result, urlCount)

	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for targetURL := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					res := checker.CheckURLWithOptions(ctx, targetURL, nil, multiNote, opts)
					resultsChan <- res
				}
			}
		}()
	}

	for _, u := range urls {
		jobs <- u
	}
	close(jobs)

	wg.Wait()
	close(resultsChan)

	// memory pre-allocation: pre-size slice capacity to urlCount to eliminate slice grow re-allocations
	results := make([]checker.Result, 0, urlCount)
	for r := range resultsChan {
		results = append(results, r)
	}

	// speed & memory optimization: slices.SortFunc uses type-safe sorting without reflection overhead
	slices.SortFunc(results, func(a, b checker.Result) int {
		if a.IsUp != b.IsUp {
			if !a.IsUp {
				return -1
			}
			return 1
		}
		return strings.Compare(a.CleanURL, b.CleanURL)
	})

	return results
}

func printSummary(results []checker.Result) bool {
	upCount := 0
	downCount := 0
	var totalDuration time.Duration

	for _, r := range results {
		if r.IsUp {
			upCount++
		} else {
			downCount++
		}
		totalDuration += r.Duration
	}

	avgLatency := time.Duration(0)
	if len(results) > 0 {
		avgLatency = totalDuration / time.Duration(len(results))
	}

	summaryText := fmt.Sprintf(
		"Total: %d | %s | %s | Avg Response: %s",
		len(results),
		upBadgeStyle.Render(fmt.Sprintf("%d UP", upCount)),
		downBadge.Render(fmt.Sprintf("%d DOWN", downCount)),
		avgLatency.Round(time.Millisecond),
	)

	fmt.Println(summaryStyle.Render(summaryText))
	return downCount == 0
}
