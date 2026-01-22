package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
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

	seqStart := time.Now()
	seqCount := countSequential(text, term)
	seqElasped := time.Since(seqStart)

	fmt.Printf("Found %q %d times without go Routines. It took %v.\n", term, seqCount, seqElasped)

	workers := runtime.NumCPU()
	goStart := time.Now()
	count := countConcurrent(text, term, workers)
	goElasped := time.Since(goStart)

	fmt.Printf("Found %q %d times using %d workers. It took %v.\n", term, count, workers, goElasped)

	if seqCount != count {
		fmt.Printf("Word count without go routines != Word count with go routines")
	}
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
			c := countSequential(segment, term)

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

func countSequential(text, term string) int {
	if len(term) == 0 || len(term) > len(text) {
		return 0
	}

	count := 0
	for i := 0; i <= len(text) - len(term); i++ {
		match := true
		// check to make sure all characters in our text window match term
		for j := 0; j < len(term); j++ {
			if text[i+j] != term[j]  {
				match = false
				break
			}
		}
		if match {
			count++
		}
	}
	return count
}
