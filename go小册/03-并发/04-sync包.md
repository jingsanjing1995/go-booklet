# 04 · sync 包

channel 适合传递和信号；**同一块内存被多个 goroutine 读写**时，用 `sync` 里的原语。忘记同步的代价是：结果错、偶发崩、`-race` 报警。

---

## 1. 数据竞争一句话

两个以上 goroutine 同时访问同一变量，且至少一次是写，又没有 happens-before 关系 → 数据竞争。Go 认为这是未定义行为级别的 bug。

```bash
go test -race ./...
go run -race .
```

能用 race detector 抓住的，不要靠「我看它没问题」。

---

## 2. Mutex 与 RWMutex

```go
type Cache struct {
	mu sync.Mutex
	m  map[string]int
}

func (c *Cache) Add(k string, v int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = make(map[string]int)
	}
	c.m[k] = v
}

func (c *Cache) Get(k string) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[k]
	return v, ok
}
```

规矩：

- Mutex **不能复制**（含把它当值嵌在结构体再传值）。结构体带锁就用指针。
- 锁的范围尽量小，但必须覆盖「读改写」整段。
- 先锁 A 再锁 B，全项目保持同一顺序，避免死锁。
- Unlock 未 Lock 的 Mutex 会 panic。
- 零值 Mutex 可用，不必 New。

读多写少可以用 `RWMutex`：`RLock` 共享，`Lock` 独占。注意：

- 读锁不能升级成写锁（会死锁）。
- 短临界区上 RWMutex 未必比 Mutex 快，先测。

---

## 3. WaitGroup

见 Goroutine 章。跨版本代码以 `Add/Done/Wait` 为主。不要在 `Wait` 之后再 `Add`（除非你非常清楚生命周期已经重新开始）。

`WaitGroup` 嵌入结构体时同样不能复制。

---

## 4. Once

```go
var (
	once sync.Once
	cli  *Client
)

func Client() *Client {
	once.Do(func() {
		cli = mustNew()
	})
	return cli
}
```

`Do` 保证 `f` 只执行一次，即使被很多 goroutine 同时调用。`f` 里 panic 后，之后的 `Do` 认为已经做过（不会重试）。需要「失败可重试」不要用 Once。

---

## 5. Cond（少用）

`sync.Cond` 基于 Mutex，提供 `Wait` / `Signal` / `Broadcast`。绝大多数场景用 channel 更清晰。只有在「等某个复杂条件，且唤醒语义必须和锁绑在一起」时才考虑它。入门可以跳过。

---

## 6. Pool

```go
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func encode() []byte {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	// 写入 buf
	// 注意：若把 buf.Bytes() 逃逸出去，Put 回去后会被别人覆盖
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out
}
```

Pool 给 GC 减压（复用 buffer）。对象必须 `Reset`，且**不能把内部切片交给外界后还放回池**。GC 压力不大时不必上 Pool，过早优化。

---

## 7. Map 与原子

`sync.Map`：适合「键稳定、写很少、很多 goroutine 读」或「键互不相同」的场景。普通 `map + Mutex` 更直观，优先用它。`sync.Map` 的 API 是 `Store/Load/LoadOrStore/Range`，类型是 `any`。

`sync/atomic`：无锁改单个字。

```go
var n atomic.Int64 // 1.19+ 封装类型
n.Add(1)
fmt.Println(n.Load())
```

适合计数器、开关。复杂不变量仍用 Mutex。不要用「普通 int 加减 + 凭感觉」代替 atomic。

比较并交换：

```go
var flag atomic.Bool
if flag.CompareAndSwap(false, true) {
	// 我是那个把它打开的人
}
```

---

## 8. 和 channel 怎么选

| 问题 | 优先 |
|------|------|
| 所有权转移、流水线、信号 | channel |
| 结构体内部小状态、map 缓存 | Mutex |
| 只加一个计数 | atomic |
| 初始化一次 | Once |
| 复用 buffer | Pool |
| 取消/超时 | context + select |

可以组合：Mutex 保护内部 map，对外用 channel 收任务。不要叠三层「channel 里再传带锁结构体的指针」除非模型真的需要。

---

## 想一想

1. 为什么带 Mutex 的结构体要传指针？复制会发生什么？
2. `RWMutex` 读锁升级写锁为什么会死锁？
3. `sync.Pool` 取出的对象为什么必须 Reset？

## 练习

1. 实现线程安全的 `Counter`，分别用 Mutex 和 `atomic.Int64`，跑 `-race`。
2. 用 Mutex 包一层 `map[string]int`，写测试并发增减。
3. 用 `Once` 初始化一个假「连接」，打印它只被 New 了一次。

下一章：[Context](./05-Context.md)
