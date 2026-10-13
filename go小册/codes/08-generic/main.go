package main

import "fmt"

func unique[T comparable](s []T) []T {
	seen := make(map[T]struct{}, len(s))
	out := make([]T, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func sum[T ~int | ~float64](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}

type Stack[T any] struct{ items []T }

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

func main() {
	fmt.Println(unique([]string{"go", "go", "小册"}))
	fmt.Println(sum([]int{1, 2, 3}))

	var st Stack[int]
	st.Push(1)
	st.Push(2)
	v, ok := st.Pop()
	fmt.Println("pop", v, ok)
}
