# 02 · Channel

channel 是带类型的管道，用来在 goroutine 之间传递值，并顺便同步。口号「用通信来共享内存」，落地就是它。

---

## 1. 声明、发送、接收

```go
ch := make(chan int)    // 无缓冲
ch := make(chan int, 8) // 缓冲 8

ch <- 1    // 发送
v := <-ch  // 接收
v, ok := <-ch
```

`ok == false` 表示 channel 已关闭且缓冲区空了。再接收会一直拿到零值。

零值 channel 是 `nil`：

- 对 nil channel 发送或接收：**永远阻塞**
- 关闭 nil channel：**panic**

`select` 里利用 nil 把某个 case 关掉，是高级技巧（模式章）。

---

## 2. 无缓冲 vs 有缓冲

**无缓冲**（同步 channel）：发送方和接收方必须同时就绪，交接后才继续。适合做「信号」和「所有权转移」。

```go
done := make(chan struct{})
go func() {
	work()
	close(done)
}()
<-done
```

`chan struct{}` 不占空间，只传「发生了」。

**有缓冲**：发送在缓冲区未满时不阻塞，接收在非空时不阻塞。适合削峰、工人池任务队列。缓冲大小是容量，不是「越多越好」；过大会掩盖背压，内存里堆满未处理任务。

| | 无缓冲 | 有缓冲 |
|--|--------|--------|
| `make(chan T)` | 同步交接 | 异步到容量上限 |
| 发送阻塞条件 | 没有接收方 | 缓冲满 |
| 典型用途 | 信号、请求-响应 | 队列、批量 |

---

## 3. 关闭：只关一次，且通常由发送方关

```go
close(ch)
```

规则：

1. **只关闭一次**，重复 close panic。
2. **不要关你不拥有的 channel**。约定：发送方关，接收方只读。
3. 关闭后还能读出剩余缓冲；读完 `ok == false`。
4. 向已关闭的 channel **发送**会 panic。
5. 接收方用 `for v := range ch` 最干净，channel 关闭后循环结束。

```go
go func() {
	defer close(out)
	for _, x := range in {
		out <- x * 2
	}
}()

for v := range out {
	fmt.Println(v)
}
```

不必关闭 channel 程序也能退（垃圾回收会收）。关闭的意义是**告诉接收方不会再有新数据**。没有接收方在 `range` 或等结束信号时，可以不关。

---

## 4. 方向：在函数签名里锁死

```go
func produce(out chan<- int) { /* 只能发 */ }
func consume(in <-chan int)  { /* 只能收 */ }
```

编译器阻止你在消费端误 close、误发送。公开 API 请写方向。

---

## 5. 请求-响应

```go
type req struct {
	n    int
	resp chan int
}

func worker(reqs <-chan req) {
	for r := range reqs {
		r.resp <- r.n * r.n
	}
}

resp := make(chan int, 1)
reqs <- req{n: 12, resp: resp}
fmt.Println(<-resp)
```

每个请求自带回信地址。缓冲 1 可以避免 worker 在无人收时卡住（视场景而定）。

---

## 6. 死锁

运行时能检测**整个程序**都阻塞在 channel 上的情况：

```go
ch := make(chan int)
ch <- 1 // 没有接收方，fatal error: all goroutines are asleep - deadlock!
```

检测不到的：还有别的 goroutine 在跑（比如 HTTP 服务），虽然你的那对收发已经互相等死。这类要用超时、`select` + `ctx.Done()` 兜底。

常见死锁配方：

- 无缓冲互相等对方先发
- 缓冲过小的环形依赖
- 忘记启动接收方
- 忘记 close，`range` 永远等
- 锁的顺序和 channel 交错（先锁后发，对端先发后锁）

---

## 7. 不要用 channel 代替 Mutex 做简单计数

```go
// 可以，但往往更绕
ch := make(chan int, 1)
ch <- 0
x := <-ch
x++
ch <- x
```

简单共享状态用 `sync.Mutex` 或 `atomic` 更直。channel 适合**传递数据、传递所有权、发信号、扇入扇出**。工具选对，代码更短。

---

## 8. 容量与背压

工人慢、生产快：无缓冲或小缓冲会让生产方自然变慢（背压）。无限 `go` + 巨大缓冲等于把积压藏进内存。服务里更常见的是：

- 有限工人数
- 有限队列
- 队列满时拒绝或阻塞请求方

---

## 想一想

1. 为什么关闭应该由发送方执行？
2. `for range ch` 和 `v, ok := <-ch` 循环如何对应？
3. 什么时候「不关闭 channel」反而是正确的？

## 练习

1. 无缓冲 channel：一个 goroutine 算平方，main 接收打印。
2. 有缓冲 3：连续发 3 个不启动接收，观察第 4 个是否阻塞（加打印 + 另一边延迟收）。
3. 实现 `func merge(a, b <-chan int) <-chan int`，把两个只读 channel 合成一个（先串行读完也可以，select 版留到下一章）。

对应示例：`codes/06-channel`。

下一章：[Select 与超时](./03-Select与超时.md)
