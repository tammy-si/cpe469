package main

import "fmt"

func main() {
	// test for now
	a := [][]int{
		{1, 2, 3},
		{4, 5, 6},
	}
	b := [][]int {
		{7, 8},
		{9, 10},
		{11, 12},
	}
	
	result := matrixMultiplyConcurrent(a, b)
	fmt.Println("Resultant Matrix:")
	for _, row := range result {
		fmt.Println(row)
	}
}

func matrixMultiplyConcurrent(a [][]int, b [][]int) [][]int {
	n := len(a)
	m := len(b[0])
	p := len(b)

	// Result matrix has the row count of a and column count of b
	finalResult := make([][]int, n)
	for i := range finalResult {
		finalResult[i] = make([]int, m)
	}

	// Channel to collect results
	type result struct {
		row int
		col int
		val int
	}
	resultCh := make(chan result)

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			// func (i, j int) defines an anonymous function that calculates the value for finalResult[i][j]
			go func(i, j int) {
				sum := 0
				for k := 0; k < p; k++ {
					sum += a[i][k] * b[k][j]
				}
				// Put results for the calculated cell into the channel
				resultCh <- result{row: i, col: j, val: sum}
			}(i, j)
		}
	}

	// Collect results for every cell in the result matrix from the channel
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			res := <-resultCh
			finalResult[res.row][res.col] = res.val
		}
	}

	return finalResult
}
