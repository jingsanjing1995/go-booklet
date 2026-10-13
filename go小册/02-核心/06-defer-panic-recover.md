# 06 · defer / panic / recover

这三个关键字管的是「离开函数时要做的事」和「崩溃如何被接住」。用对了，资源不会漏、进程不会因为一个请求的 bug 直接死；用错了，错误被吞掉，现场消失。

---

## 1. defer：离开时必跑（大多时候）

```go
f, err := os.Open(path)
if err != nil {
	return err
}
defer f.Close()
```

规则：

1. **参数立刻求值**，调用推迟到函数返回（含 panic）。
2. 多个 defer **LIFO**，后写的先执行。
3. `os.Exit`、致命信号、运行时直接崩死，defer 不保证执行。
4. 循环里 `defer` 会一直挂到函数结束，不是本轮结束。文件循环应包一层函数：

```go
for _, path := range paths {
	if err := readOne(path); err != nil {
		return err
	}
}

func readOne(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(io.Discard, f)
	return err
}
```

---

## 2. 参数立刻求值的例子

```go
func demo() {
	n := 1
	defer fmt.Println("defer", n) // 记下的是 1
	n = 2
	fmt.Println("now", n)
}
// 输出:
// now 2
// defer 1
```

想看到「返回时的值」，用闭包读变量，而不是把值当参数：

```go
func demo() (n int) {
	n = 1
	defer func() { fmt.Println("n=", n) }()
	n = 2
	return
}
```

---

## 3. defer 改命名返回值

```go
func slurp(path string) (b []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	return io.ReadAll(f)
}
```

匿名函数能看见并修改命名结果。这是处理 Close 错误的标准姿势之一（也可以 `errors.Join`）。

注意：

```go
defer f.Close() // 丢掉 Close 的 error
```

对只读文件通常可接受；对写文件、`os.Create` 后写入，**Close 失败可能意味着数据没落盘**，不要丢。

---

## 4. panic：把栈展开

```go
func must(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}
```

panic 会立刻停掉当前函数，执行已注册的 defer，继续往外展开，直到：

- 被某层 `recover` 住，或
- 到达 goroutine 顶部，程序打印栈并退出（**只死这个进程**；其他 goroutine 的 defer 不保证跑完）

panic 的值是 `any`，习惯传 `error` 或字符串。库代码应少 panic。

运行时也会 panic：越界、nil 指针、向已关闭的 channel 发送、重复关闭 channel、解锁未锁的 Mutex……这些是 bug。

---

## 5. recover：只能在 defer 里生效

```go
func safe(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	fn()
	return nil
}
```

要点：

- `recover()` 在普通函数里调用，拿到的是 nil，**拦不住**。
- 只 recover **当前 goroutine**。A 里的 panic，B 里的 recover 看不见。
- 拦截后如果还想让程序死，再 `panic(r)` 抛出去。
- HTTP 服务器对每个请求的 goroutine 做 recover 是合理的；默默吞掉所有 panic 且不打栈，是事故。

记录栈：

```go
debug.Stack()          // runtime/debug
// 或
fmt.Printf("%+v", r)   // 不够，建议 log 里带 debug.Stack()
```

---

## 6. 该不该 recover？

| 场景 | 建议 |
|------|------|
| `http.Handler` 单请求 | 恢复，记日志，返回 500 |
| 消息消费者单条消息 | 恢复，这条失败，循环继续 |
| `main` 初始化 | 通常让它崩，方便早发现 |
| 库的普通导出函数 | 不要 recover 把错误藏起来 |
| 并发工人 | 工人内部 recover，并把失败经 channel 上报 |

把 panic 当控制流（用 panic 代替 return error，顶层再 recover）会毁掉 `error` 的可见性，也让 `go test`、pprof、竞态检测更难。别学某些「错误处理框架」走这条路。

---

## 7. 和 goroutine 一起时

```go
go func() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("worker panic: %v\n%s", r, debug.Stack())
		}
	}()
	work()
}()
```

子 goroutine **不会**被外层函数的 recover 保护。每个 `go` 出去的入口自己负责。

---

## 想一想

1. 为什么 `defer Close()` 的参数（接收者 `f`）是立刻求值？如果循环里复用同一个 `f` 变量会发生什么？
2. 写文件时忽略 `Close` 错误，可能丢掉什么？
3. 在非 defer 函数里调用 `recover` 为什么没用？

## 练习

1. 写一个函数演示「defer 参数立刻求值」和「闭包读变量」的打印差异。
2. 实现 `func Guard(fn func() error) (err error)`，把 panic 转成 error，普通 error 原样返回。
3. 故意在子 goroutine 里 panic，验证 `main` 的 recover 拦不住。

下一章：[标准库精选](./07-标准库精选.md)
