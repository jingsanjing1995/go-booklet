package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {
	fmt.Println("开始时 goroutine:", runtime.NumGoroutine())

	var wg sync.WaitGroup
	errc := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			fmt.Println("工人", id)
			if id == 2 {
				errc <- fmt.Errorf("工人 %d 失败", id)
				return
			}
			errc <- nil
		}(i)
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != nil {
			fmt.Println("收到错误:", err)
		}
	}
	fmt.Println("结束后 goroutine:", runtime.NumGoroutine())
}
