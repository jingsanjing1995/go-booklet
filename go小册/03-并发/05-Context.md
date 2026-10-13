# 05 · Context

`context.Context` 在 Go 服务里几乎是函数的第一个参数。它传递三件事：**取消信号、截止时间、请求级的键值**。用对了，超时能穿透整条调用链；用错了，goroutine 泄漏或值被塞成垃圾箱。

---

## 1. 接口

```go
type Context interface {
	Deadline() (deadline time.Time, ok bool)
	Done() <-chan struct{}
	Err() error
	Value(key any) any
}
```

- `Done()` 关闭：该停了。
- `Err()`：`Canceled` 或 `DeadlineExceeded`。
- `Value`：只放请求级元数据（trace id），不要放可选参数。

派生树：

```text
Background / TODO
    └── WithCancel / WithTimeout / WithDeadline / WithValue
            └── 再派生...
```

子取消不影响父；父取消会取消所有子。

---

## 2. 从哪来

```go
ctx := context.Background() // 根，main、init、测试顶层
ctx := context.TODO()       // 暂时不知道，占位，应尽快换掉
```

HTTP：

```go
func handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // 客户端断开时会取消
	user, err := svc.Find(ctx, id)
	// ...
}
```

永远不要把 Context 存在结构体里长期复用（少数生命周期等于结构体的例外，如 `Request`）。应当作为参数向下传。

签名约定：

```go
func Find(ctx context.Context, id string) (User, error)
```

`ctx` 是第一个参数，变量就叫 `ctx`。

---

## 3. 取消与超时

```go
ctx, cancel := context.WithCancel(parent)
defer cancel() // 必须调用，释放父链上的资源

go func() {
	<-someSignal
	cancel()
}()
```

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel()
```

超时到了，`ctx.Done()` 关闭，`ctx.Err() == context.DeadlineExceeded`。

**`defer cancel()` 不要忘。** 即使工作已经成功结束，不 cancel 也会让 timer 在父 context 树上多活一会儿，高 QPS 下能看出内存和 goroutine 异常。

手动更早结束：成功后立刻 `cancel()`，不必等超时。

---

## 4. 工人如何响应

```go
func fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
```

自己写循环：

```go
select {
case <-ctx.Done():
	return ctx.Err()
case job, ok := <-jobs:
	if !ok {
		return nil
	}
	return handle(ctx, job)
}
```

只检查开头一次不够：长循环内部也要隔一段看 `ctx.Err()`。

---

## 5. WithValue：克制

```go
type ctxKey struct{} // 未导出类型当键，避免碰撞

func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func TraceID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
```

可以放：trace id、日志字段、登录用户（有争议，团队要统一）。

不要放：连接池、配置全集、可选业务参数。函数需要的东西写在参数列表里，测试才好喂。

`Value` 是沿父链线性查找的，键一多又慢又乱。

---

## 6. 错误怎么对待

```go
if errors.Is(err, context.Canceled) {
	// 客户端走了，通常不必当 500 告警
}
if errors.Is(err, context.DeadlineExceeded) {
	// 超时，可记一条 warning
}
```

把 `ctx.Err()` 往上返回时，用 `%w` 保留判断能力：

```go
return fmt.Errorf("fetch user %s: %w", id, ctx.Err())
```

---

## 7. 常见误用

| 误用 | 后果 |
|------|------|
| 结构体字段存 ctx | 生命周期错乱，取消不了或取消错请求 |
| 不 defer cancel | timer / 子 goroutine 泄漏 |
| 用 `Background` 调下游 HTTP | 请求无法随客户端取消 |
| 把 map、logger、db 塞进 Value | 隐式依赖，测试和可读性崩 |
| 忽略 ctx 继续算很久 | 用户已断开，你还在烧 CPU/配额 |

标准库和 pgx、官方云客户端几乎都接 `ctx`。你自己的导出函数也应接。

对应示例：`codes/07-context`。

---

## 想一想

1. 为什么 cancel 必须被调用，即使函数已经正常返回？
2. 父超时 1s，子再 `WithTimeout(ctx, 5s)`，实际多久取消？
3. 把 `userID` 放进 context 和放进函数参数，各有什么好坏？

## 练习

1. 写一个 200ms 的 sleep 工人，外层 50ms 超时，确认返回 `DeadlineExceeded`。
2. 用未导出 key 存 trace id，打日志时取出来。
3. HTTP handler 里把 `r.Context()` 传给 `http.NewRequestWithContext` 调下游，思考客户端中断时会发生什么。

下一章：[并发模式与陷阱](./06-并发模式与陷阱.md)
