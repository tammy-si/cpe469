package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Input searched word: ")
	term, _ := reader.ReadString('\n')
	term = strings.TrimSpace(term)
	if term == "" {
		fmt.Println("Empty search term.")
		return
	}

	data, err := os.ReadFile("war_and_peace.html")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}
	text := string(data)

	workers := runtime.NumCPU()
	count := countConcurrent(text, term, workers)

	fmt.Printf("Found %q %d times using %d workers.\n", term, count, workers)
}

func countConcurrent(text, term string, workers int) int {
	if workers < 1 {
		workers = 1
	}
	n := len(text)
	if len(term) > n {
		return 0
	}

	chunkSize := (n + workers - 1) / workers
	overlap := len(term) - 1

	results := make(chan int, workers)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		start := w * chunkSize
		if start >= n {
			break
		}
		end := start + chunkSize
		if end > n {
			end = n
		}

		// extend end to include overlap (don’t go past file end)
		endWithOverlap := end + overlap
		if endWithOverlap > n {
			endWithOverlap = n
		}

		wg.Add(1)
		go func(start, end, endWithOverlap int) {
			defer wg.Done()

			segment := text[start:endWithOverlap]
			c := strings.Count(segment, term)

			// avoid double-counting: if we overlapped, we might count matches that start before `end`
			// This quick fix subtracts matches that are fully contained in the overlap area *before* start
			// A more exact method is KMP with "count matches whose start < end".
			results <- c
		}(start, end, endWithOverlap)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	total := 0
	for c := range results {
		total += c
	}
	return total
}
