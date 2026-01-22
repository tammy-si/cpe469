package main

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
	"lab2/search_functions"
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
	seqCount := search_functions.CountSequential(text, term)
	seqElasped := time.Since(seqStart)

	fmt.Printf("Found %q %d times without go Routines. It took %v.\n", term, seqCount, seqElasped)

	workers := runtime.NumCPU()
	goStart := time.Now()
	count := search_functions.CountConcurrent(text, term, workers)
	goElasped := time.Since(goStart)

	fmt.Printf("Found %q %d times using %d workers. It took %v.\n", term, count, workers, goElasped)

	if seqCount != count {
		fmt.Printf("Word count without go routines != Word count with go routines")
	}
}
