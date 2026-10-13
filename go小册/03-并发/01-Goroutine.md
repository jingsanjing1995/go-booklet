# 01 · Goroutine

`go f()` 会启动一个由 Go 运行时调度的轻量执行体。它不是操作系统线程：创建成本低，栈从很小长起，成千上万个同时活着是常态。这一章先建立「启动、等待、不要泄漏」的肌肉记忆。

---

## 1. 启动

```go
go doWork(ctx, id)

go func() {
	fmt.Println("hi")
}()
```

`main` 返回时，**整个进程结束**，其它 goroutine 会被直接掐掉，来不及收尾。所以「主函数里 `go` 一下就结束」常常什么都打印不出来。

```go
func main() {
	go fmt.Println("maybe")
	// 没有等待，进程可能已经退了
}
```

---

## 2. 等待一组工作：WaitGroup

```go
var wg sync.WaitGroup
for i := 0; i < 5; i++ {
	wg.Add(1)
	go func(id int) {
		defer wg.Done()
		work(id)
	}(i)
}
wg.Wait()
```

规矩：

- `Add` 在 `go` **之前**调用，且在同一个 goroutine 里加，避免竞态。
- 每个 goroutine `Done` 一次，常用 `defer`。
- `Add` 的计数不能变成负数，否则 panic。
- WaitGroup 不能被复制，用指针传递。

需要收集错误时，单靠 WaitGroup 不够，还要 channel 或 `errgroup`（见后文和模式章）。

---

## 3. 它到底有多「轻」

粗粒度图像：

```text
你的代码：成千上万 goroutine
        ↓
运行时调度器（GMP）
        ↓
少量 OS 线程（默认约等于 CPU 核数，可 GOMAXPROCS 调整）
```

阻塞在 channel、网络、syscall 上时，调度器可以去跑别的 goroutine。所以「一连接一 goroutine」在 HTTP 服务器里是自然模型。

代价：

- 每个 goroutine 仍占内存（栈 + 上下文）。泄漏一万个，内存会看见。
- 无限制地 `go` 可能把下游（数据库、远端 API）打爆。要有并发上限。

GMP 细节放在高级篇。现在只要相信：**你写的是逻辑并发，不是手搓线程。**

---

## 4. 泄漏长什么样

```go
func leak() {
	go func() {
		time.Sleep(time.Hour) // 没人取消，没人等
	}()
}
```

每次调用 `leak` 就多一个活一个小时的 goroutine。请求处理函数里这样做，流量一来就炸。

识别信号：

- `runtime.NumGoroutine()` 持续爬升
- pprof 的 `goroutine` 画像里大量卡在同一处 `chan receive` / `sleep`

对策：传入 `context.Context`，在退出路径取消；或用能关闭的 channel 通知。

```go
func worker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := step(); err != nil {
				return err
			}
		}
	}
}
```

---

## 5. 闭包捕获

Go 1.22 起循环变量每轮重新声明，下面代码打印 0..2 是符合预期的：

```go
for i := 0; i < 3; i++ {
	go func() { fmt.Println(i) }()
}
```

仍然建议**当参数传入**，在旧版本和代码评审里都更安全、更可读。

捕获外层的 `err`、`resp` 等可变量时，先把需要的值拷到局部：

```go
resp := resp
go func() { handle(resp) }()
```

---

## 6. GOMAXPROCS 与并行

- **并发**：一段时间内逻辑上有多件事在推进（一个核也能）。
- **并行**：同一时刻真的在多个核上跑。

`runtime.GOMAXPROCS(n)` 控制同时执行 Go 代码的 OS 线程数。默认等于 CPU 数。CPU 密集可以保持默认；I/O 密集通常也不需要乱调。调之前先用基准和 pprof 说话。

`runtime.Gosched()` 让出当前 P，几乎用不到。`runtime.LockOSThread()` 会把当前 goroutine 钉在线程上（cgo、需要线程本地状态时），会降低调度灵活性。

---

## 7. 错误与 panic 不会自动汇合

子 goroutine 里 `return err` 没人接就丢了。panic 会杀死整个进程（除非它自己 recover）。

最低配置：

```go
errc := make(chan error, 1)
go func() {
	errc <- work()
}()
if err := <-errc; err != nil {
	return err
}
```

多个工人用带缓冲 channel 或 `golang.org/x/sync/errgroup`：

```go
g, ctx := errgroup.WithContext(ctx)
for _, id := range ids {
	id := id
	g.Go(func() error { return fetch(ctx, id) })
}
return g.Wait()
```

`errgroup` 能在第一个错误时取消其余工作，比手写 WaitGroup+channel 少很多样板。

---

## 想一想

1. 为什么 `WaitGroup.Add` 必须在 `go` 之前？
2. 「轻量」是不是意味着可以无限 `go`？瓶颈通常在哪？
3. `main` 结束和「所有 goroutine 结束」是什么关系？

## 练习

1. 启动 10 个 goroutine 打印自己的 id，用 WaitGroup 等它们结束。
2. 用 `runtime.NumGoroutine()` 观察：写一个故意泄漏的函数，再改成 `context` 可取消。
3. 让一个子 goroutine 返回 error，在 main 里接收并打印。

对应示例：`codes/05-goroutine`。

下一章：[Channel](./02-Channel.md)
