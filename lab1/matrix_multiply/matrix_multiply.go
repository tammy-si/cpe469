package main

import ("fmt"
		"math/rand/v2")

func s_matrix(a [][]int, b [][]int) [][]int {
}
		
func main() {
	fmt.Println(matrix_gen(10,10))
	var a = [2][3]int{{1, 2, 3}, {4, 5, 6}}
	var b = [3][2]int{{7, 8}, {9, 10}, {11, 12}}
}

func matrix_gen(r int, c int) [][]int {
	var i int
	var j int
	var matrix [][]int

	for i:=0, i<r, i++{
		for j:=0, j<c, j++{
			matrix[i][j] = rand.Int(100)
		}
	}
	return matrix
}