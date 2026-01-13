package main

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

func main() {
	big_a := matrix_gen(10000, 10000)
	big_b := matrix_gen(10000, 10000)

	var concurrentResult [][]float32
	var SequentialResult [][]float32

	var sumCon float32
	var sumSeq float32

	for _, goroutines := range []int{1, 2, 4, 8, 16, 32} {
		start := time.Now()
		concurrentResult = matrixMultiplyConcurrent(big_a, big_b, goroutines)
		elasped := time.Since(start)

		sumCon = 0
		for i := 0; i < len(concurrentResult); i++ {
			sumCon += concurrentResult[i][i]
		}
		fmt.Printf("Concurrent (%d goroutines) time: %v, diagonal sum: %v\n", goroutines, elasped, sumCon)
	}

	start2 := time.Now()
	SequentialResult = matrixMultiplySeq(big_a, big_b)
	for i := 0; i < len(SequentialResult); i++ {
		sumSeq += SequentialResult[i][i]
	}

	if sumCon != sumSeq {
		fmt.Printf("Sequential Result != concurrentResult\n")
	}

	elasped2 := time.Since(start2)

	fmt.Printf("Sequential time : %v\n", elasped2)
}

func matrixMultiplyConcurrent(a [][]float32, b [][]float32, numGoroutines int) [][]float32 {
	n := len(a)
	m := len(b[0])
	p := len(b)

	// Result matrix has the row count of a and column count of b
	finalResult := make([][]float32, n)
	for i := range finalResult {
		finalResult[i] = make([]float32, m)
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Compute row ranges for each goroutine, rounds up
	chunkSize := (n + numGoroutines - 1) / numGoroutines

	for g := 0; g < numGoroutines; g++ {
		startRow := g * chunkSize
		endRow := startRow + chunkSize
		if endRow > n {
			endRow = n
		}

		go func(start, end int) {
			for i := start; i < end; i++ {
				for j := 0; j < m; j++ {
					var sum float32
					for k := 0; k < p; k++ {
						sum += a[i][k] * b[k][j]
					}
					finalResult[i][j] = sum
				}
			}
			wg.Done()
		}(startRow, endRow)
	}

	wg.Wait()

	return finalResult
}

func matrixMultiplySeq(a [][]float32, b [][]float32) [][]float32 {
	rowA := len(a)
	colA := len(a[0])

	rowB := len(b)
	colB := len(b[0])

	// Check if matrices can be multiplied
	if colA != rowB {
		fmt.Println("cA != rB")
		return nil
	}

	// Initialize result matrix C
	C := make([][]float32, rowA)
	for i := range C {
		C[i] = make([]float32, colB)
	}

	// Perform multiplication
	for i := 0; i < rowA; i++ {
		for j := 0; j < colB; j++ {
			for k := 0; k < colA; k++ {
				C[i][j] += a[i][k] * b[k][j]
			}
		}
	}

	return C
}

// taken from matrix_multipliy.go
func matrix_gen(r int, c int) [][]float32 {
	matrix := make([][]float32, r)
	for i := range matrix {
		matrix[i] = make([]float32, c)
	}

	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			matrix[i][j] = rand.Float32() * 100
		}
	}
	return matrix
}
