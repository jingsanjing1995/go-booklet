# 03 · Select 与超时

`select` 同时等待多个 channel 操作，哪一个先就绪就走哪一个。超时、取消、多路合并，都靠它。

---

## 1. 基本形态

```go
select {
case v := <-in:
	fmt.Println("got", v)
case out <- n:
	fmt.Println("sent", n)
case <-time.After(time.Second):
	fmt.Println("timeout")
case <-ctx.Done():
	return ctx.Err()
}
```

规则：

- 所有 case 都是 **channel 的发送或接收**（或 `default`）。
- 多个 case 同时就绪时，**随机选一个**，避免饿死。
- 没有任何 case 就绪且没有 `default`：阻塞。
- 有 `default`：不阻塞，走 default。

空 `select{}` 会永远阻塞当前 goroutine，偶尔用来让 `main` 挂住（不如 `<-ctx.Done()` 或 `WaitGroup` 清晰）。

---

## 2. 超时：After 与 Timer

```go
select {
case v := <-ch:
	use(v)
case <-time.After(3 * time.Second):
	return errors.New("timeout")
}
```

`time.After` 每次调用创建一个 timer。在**热循环**里不要用它，会堆积未触发的 timer：

```go
// 更好
t := time.NewTimer(3 * time.Second)
defer t.Stop()

select {
case v := <-ch:
	if !t.Stop() {
		<-t.C // 已经触发则抽干，避免泄漏到下次复用
	}
	use(v)
case <-t.C:
	return errors.New("timeout")
}
```

复用 timer 用 `t.Reset(d)`。第一次 Stop/Reset 的细节容易写错，封装成小函数或直接用 `context.WithTimeout` 更稳。

---

## 3. 和 Context 配合（首选）

```go
ctx, cancel := context.WithTimeout(parent, 3*time.Second)
defer cancel()

select {
case v := <-ch:
	use(v)
case <-ctx.Done():
	return ctx.Err() // context.DeadlineExceeded 或 Canceled
}
```

Context 可以一路往下传，所有下游的 HTTP、SQL、自己的 `select` 都能同时停。这比每个函数自己 `time.After` 更一致。下一章专讲 Context。

---

## 4. 非阻塞尝试

```go
select {
case ch <- v:
	// 送进去了
default:
	// 满了，丢弃或返回忙碌
}
```

适用：指标上报、尽力而为的日志、有限队列满时快速失败。不要用它写自旋空转：

```go
for {
	select {
	case v := <-ch:
		use(v)
	default:
		// 疯狂空转，吃满一个核
	}
}
```

要么阻塞收，要么 `default` 里 `runtime.Gosched()` / sleep——但那已经说明模型不对。

---

## 5. 关掉某个 case：置 nil

```go
var inA, inB <-chan int = a, b
for inA != nil || inB != nil {
	select {
	case v, ok := <-inA:
		if !ok {
			inA = nil
			continue
		}
		out <- v
	case v, ok := <-inB:
		if !ok {
			inB = nil
			continue
		}
		out <- v
	}
}
```

channel 关掉后，`<-inA` 会**立刻**读到零值。如果不把 `inA` 置 nil，这个 case 会一直就绪，饿死其它逻辑。置 nil 后该 case 永阻塞，等于从 select 里摘掉。这是 fan-in 的标准写法。

---

## 6. 取消进行中的发送

无缓冲发送本身可能一直卡住。用 select 保护：

```go
select {
case out <- v:
	return nil
case <-ctx.Done():
	return ctx.Err()
}
```

接收侧同样：

```go
select {
case v, ok := <-in:
	if !ok {
		return io.EOF
	}
	return handle(v)
case <-ctx.Done():
	return ctx.Err()
}
```

---

## 7. 公平与饥饿

多 case 同时就绪时随机选择，长期看公平。若你总是先在循环外读某一个 channel，再 select 另一个，可能饿死第二条路。能放进同一个 `select` 的就放在一起。

---

## 想一想

1. 为什么热循环里的 `time.After` 会泄漏？
2. 关闭的 channel 为什么会让 select case「永远就绪」？
3. `default` 分支在工人池满载时适合做什么（以及不适合做什么）？

## 练习

1. 写 `func recvTimeout(ch <-chan int, d time.Duration) (int, error)`。
2. 实现真正的 `merge(a, b <-chan int) <-chan int`，用 select + nil 摘除。
3. 用非阻塞 send 实现「队列满则返回 `ErrBusy`」。

下一章：[sync 包](./04-sync包.md)
