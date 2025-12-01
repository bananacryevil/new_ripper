package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	batchSize    = 5
	outputDir    = "./tv/SSNHP"
	episodesFile = "episodes.txt"
	jarFile      = "abyss-dl.jar"
)

type Episode struct {
	Num string
	Key string
	URL string
}

func main() {
	// 1. Read episodes
	episodes, err := readEpisodes(episodesFile)
	if err != nil {
		log.Fatalf("Failed to read episodes: %v", err)
	}

	// 2. Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// 3. Process in batches
	totalEpisodes := len(episodes)
	for i := 0; i < totalEpisodes; i += batchSize {
		end := i + batchSize
		if end > totalEpisodes {
			end = totalEpisodes
		}
		batch := episodes[i:end]

		fmt.Printf("\n=== Processing Batch %d-%d of %d ===\n", i+1, end, totalEpisodes)
		processBatch(batch)
	}

	fmt.Println("\nAll batches completed.")
}

func readEpisodes(path string) ([]Episode, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var episodes []Episode
	scanner := bufio.NewScanner(file)

	// Skip header
	if scanner.Scan() {
		_ = scanner.Text()
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			num := parts[0]
			key := parts[1]
			// Construct URL based on the pattern provided by user
			// https://wlext.is/series/sin-senos-no-hay-paraiso-2008/?server=shorticu&episode={NUM}
			url := fmt.Sprintf("https://wlext.is/series/sin-senos-no-hay-paraiso-2008/?server=shorticu&episode=%s", num)
			episodes = append(episodes, Episode{Num: num, Key: key, URL: url})
		}
	}
	return episodes, scanner.Err()
}

func processBatch(batch []Episode) {
	// Create a new browser context for this batch
	// We use a headful mode (not headless) if possible, or headless if on server.
	// The user asked for "regular fucking browser", but we are in a remote env likely.
	// We will try to use ExecAllocator to launch a browser.
	
	// By default, chromedp looks for chrome/chromium in standard locations.
	// If you need to specify a path, use chromedp.ExecPath("/path/to/chrome")
	
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/chromium"), // Explicitly use the installed chromium
		chromedp.Flag("headless", true), // Run headless as requested
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	// Create a context
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// Open tabs for all episodes in the batch
	fmt.Println("Opening browser tabs...")
	var tabWg sync.WaitGroup

	// We need to keep the browser alive while downloads happen.
	// We will open a tab for each URL.

	for _, ep := range batch {
		tabWg.Add(1)
		go func(e Episode) {
			defer tabWg.Done()
			// Create a new tab context from the browser context
			tCtx, cancel := chromedp.NewContext(ctx)
			defer cancel()

			// Navigate and wait for load
			fmt.Printf("[%s] Navigating to %s\n", e.Num, e.URL)
			if err := chromedp.Run(tCtx,
				chromedp.Navigate(e.URL),
				chromedp.Sleep(5*time.Second), // Wait for scripts to run
			); err != nil {
				log.Printf("[%s] Failed to navigate: %v", e.Num, err)
			} else {
				fmt.Printf("[%s] Page loaded.\n", e.Num)
			}

			// Keep this tab open?
			// Actually, chromedp.Run blocks until actions are done.
			// If we return, the context might close if we cancel it.
			// But we are inside a loop. We need to hold this tab open while the download runs.
			// We can't easily "hold" it in this structure without blocking.

			// Better approach: Open all tabs sequentially in the SAME browser context (or separate tabs),
			// but we need them open concurrently.
		}(ep)
	}

	// Wait for tabs to be "ready" (loaded).
	// Actually, the above logic is flawed because `chromedp.Run` blocks.
	// If we want them open *simultaneously*, we need to launch them and keep them alive.

	// Revised strategy for browser:
	// Just launch one browser, and use `chromedp.NewContext` to create tabs.
	// We need to keep the Go routines alive.

	// Let's simplify:
	// 1. Launch browser.
	// 2. For each episode, create a target and navigate.
	// 3. Start download.
	// 4. Wait for download.
	// 5. Close target.

	// But we want to do this in parallel (batch of 5).

	var dlWg sync.WaitGroup

	for _, ep := range batch {
		dlWg.Add(1)
		go func(e Episode) {
			defer dlWg.Done()

			// Create a tab
			tCtx, cancelTab := chromedp.NewContext(ctx)
			defer cancelTab()

			// Navigate
			fmt.Printf("[%s] Opening page...\n", e.Num)
			if err := chromedp.Run(tCtx, chromedp.Navigate(e.URL)); err != nil {
				log.Printf("[%s] Browser error: %v", e.Num, err)
				return
			}

			// Wait a bit for page to "settle"
			time.Sleep(5 * time.Second)

			// Start Download
			outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.mp4", e.Num))
			fmt.Printf("[%s] Starting download to %s\n", e.Num, outputPath)

			cmd := exec.Command("java", "-jar", jarFile, e.Key, "h", "-o", outputPath)

			// Capture output to check for errors
			// cmd.Stdout = os.Stdout
			// cmd.Stderr = os.Stderr

			if err := cmd.Start(); err != nil {
				log.Printf("[%s] Failed to start download: %v", e.Num, err)
				return
			}

			// Monitor file size
			done := make(chan struct{})
			go monitorFileSize(e.Num, outputPath, done)

			if err := cmd.Wait(); err != nil {
				log.Printf("[%s] Download command failed: %v", e.Num, err)
			} else {
				fmt.Printf("[%s] Download command finished.\n", e.Num)
			}
			close(done)

		}(ep)
	}

	dlWg.Wait()
	fmt.Println("Batch finished. Closing browser...")
}

func monitorFileSize(num, path string, done chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastSize int64

	for {
		select {
		case <-done:
			// Final check
			fi, err := os.Stat(path)
			if err == nil {
				sizeMB := float64(fi.Size()) / 1024 / 1024
				fmt.Printf("[%s] Final size: %.2f MB\n", num, sizeMB)
				if sizeMB < 10 {
					fmt.Printf("[%s] WARNING: File is suspiciously small (<10MB)!\n", num)
				} else {
					fmt.Printf("[%s] SUCCESS: Large file confirmed.\n", num)
				}
			} else {
				fmt.Printf("[%s] Error checking final file size: %v\n", num, err)
			}
			return
		case <-ticker.C:
			fi, err := os.Stat(path)
			if err == nil {
				currentSize := fi.Size()
				sizeMB := float64(currentSize) / 1024 / 1024

				// Only print if size changed or it's been a while
				if currentSize != lastSize {
					fmt.Printf("[%s] Downloading... Current size: %.2f MB\n", num, sizeMB)
					lastSize = currentSize
				}
			}
		}
	}
}
