package search_functions

import(
	"sync"
)

func CountConcurrent(text, term string, workers int) int {
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
			c := CountSequential(segment, term)

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

func CountSequential(text, term string) int {
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
