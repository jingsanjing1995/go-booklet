package main

import (
	"fmt"
	"slices"
	"strings"
)

func clone(s []int) []int {
	out := make([]int, len(s))
	copy(out, s)
	return out
}

func first(s []int, n int) []int {
	if n > len(s) {
		n = len(s)
	}
	return slices.Clone(s[:n])
}

func wordCount(text string) map[string]int {
	freq := make(map[string]int)
	for _, w := range strings.Fields(text) {
		freq[w]++
	}
	return freq
}

func main() {
	a := []int{1, 2, 3, 4, 5}
	b := a[:2]
	b = append(b, 99)
	fmt.Println("共享数组时 append 可能改到 a:", a)

	src := []int{1, 2, 3, 4, 5}
	c := first(src, 2)
	c = append(c, 77)
	fmt.Println("clone 后的 first 不会改 src:", src, "c:", c)
	fmt.Println("clone(src):", clone(src))

	freq := wordCount("go go 小册 小册 小册")
	fmt.Println("词频:", freq)
}
