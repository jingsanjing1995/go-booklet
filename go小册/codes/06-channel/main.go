package main

import (
	"fmt"
	"sync"
)

func squares(in <-chan int, out chan<- int) {
	defer close(out)
	for n := range in {
		out <- n * n
	}
}

func merge(a, b <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for a != nil || b != nil {
			select {
			case v, ok := <-a:
				if !ok {
					a = nil
					continue
				}
				out <- v
			case v, ok := <-b:
				if !ok {
					b = nil
					continue
				}
				out <- v
			}
		}
	}()
	return out
}

func main() {
	in := make(chan int)
	out := make(chan int)
	go squares(in, out)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for v := range out {
			fmt.Println("平方:", v)
		}
	}()
	for i := 1; i <= 3; i++ {
		in <- i
	}
	close(in)
	wg.Wait()

	left := make(chan int, 2)
	right := make(chan int, 2)
	left <- 10
	left <- 20
	right <- 30
	close(left)
	close(right)
	fmt.Println("merge:")
	for v := range merge(left, right) {
		fmt.Println(" ", v)
	}
}
