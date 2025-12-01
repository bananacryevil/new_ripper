package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func main() {
	// Open the file
	file, err := os.Open("episodes.txt")
	if err != nil {
		log.Fatalf("Failed to open episodes.txt: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Skip the header line
	if scanner.Scan() {
		_ = scanner.Text() // "NUM:KEY"
	}

	// Ensure the output directory exists
	outputDir := "./tv/SSNHP"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	var wg sync.WaitGroup
	// Use a buffered channel as a semaphore to limit concurrency.
	// Adjust the buffer size (e.g., 5) to control how many downloads run at once.
	concurrencyLimit := 5
	sem := make(chan struct{}, concurrencyLimit)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			log.Printf("Skipping invalid line: %s", line)
			continue
		}

		num := parts[0]
		key := parts[1]

		wg.Add(1)
		go func(n, k string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore

			// Construct the output path
			outputPath := fmt.Sprintf("./tv/SSNHP/%s.mp4", n)

			// Command: java -jar abyss-dl.jar ${KEY} h -o ./tv/SSNHP/{$NUM}.mp4
			cmd := exec.Command("java", "-jar", "abyss-dl.jar", k, "h", "-o", outputPath)

			// Connect stdout and stderr to see progress/errors
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			fmt.Printf("Starting processing for NUM: %s, KEY: %s\n", n, k)
			if err := cmd.Run(); err != nil {
				log.Printf("Error processing NUM: %s: %v\n", n, err)
			} else {
				fmt.Printf("Completed NUM: %s\n", n)
			}
		}(num, key)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	fmt.Println("All tasks completed.")
}
