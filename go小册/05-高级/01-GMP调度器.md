# 01 · GMP 调度器

前面把 goroutine 当成「很轻的线程」来用。这一章打开盒子：运行时怎样把海量 G 放到少量系统线程上跑。读完你应能解释：阻塞 syscall 会不会卡死整个进程、`GOMAXPROCS` 在调什么、为什么有 work stealing。

这是模型课，不是要你改 runtime 源码。细节随版本微调，概念稳定。

---

## 1. 三个字母

| 缩写 | 全称 | 角色 |
|------|------|------|
| **G** | goroutine | 你的代码 + 栈 + 状态 |
| **M** | machine | 操作系统线程 |
| **P** | processor | 调度上下文：本地运行队列、内存缓存等 |

约束：**要执行 Go 代码，M 必须持有一个 P。** P 的数量默认等于 `GOMAXPROCS`（通常 = CPU 核数）。

```text
M0 ─持有─ P0 ── 本地队列 [G1 G2 G3]
M1 ─持有─ P1 ── 本地队列 [G4 G5]
M2  （无 P，可能卡在 syscall 里，或在休眠）
全局队列 [G6 G7 ...]
```

P 是「逻辑处理器」。G 在 P 的本地队列里排队。M 是真正在 CPU 上跑的人。

---

## 2. 一次调度循环（简化）

某个 M 持有 P，反复：

1. 从本地队列取 G，执行。
2. 本地空了，去全局队列拿一批。
3. 还没有，从其它 P **偷**（work stealing）。
4. 再没有，去网络轮询器看有没有就绪的 G。
5. 还没有，M 休眠，P 交回空闲列表。

新 `go f()` 的 G 优先进**当前 P 的本地队列**，减少锁竞争。偶尔会有 G 被放进全局队列，避免不公平。

---

## 3. 阻塞会怎样

**channel / mutex / select 阻塞**：G 休眠，M 放下它，从 P 的队列拿下一个 G。P 还在，CPU 不闲着。

**系统调用（文件、cgo、部分网络旧路径）**：

- M 带着 G 进入 syscall，P 被**抢走**交给别的 M，以免 P 闲置。
- syscall 返回后，M 要重新抢一个 P；抢不到就把 G 放全局队列，自己休眠。

所以「一个 goroutine 读慢磁盘」不会让所有 Go 代码停摆。但会多消耗线程。线程太多，调度和内存都会疼。

**网络**：现代运行时用 netpoll（epoll/kqueue/IOCP）。G 等数据时把 fd 交给 poller，M/P 去干别的。数据到了，G 被标记就绪。这是「一连接一 G」能成立的关键。

---

## 4. 抢占

早期协作式：只有函数调用时才找得到调度点，空转 `for {}` 会饿死别人。

现在有**基于信号的异步抢占**（Go 1.14+）：运行时定时向线程发信号，即使是无调用的热循环也能被打断。你仍不该写死循环占核，但「一个忙 G 卡死调度器」已经少见。

---

## 5. GOMAXPROCS 与线程数

- `P` 数 ≈ `GOMAXPROCS`：同时执行 **Go 代码** 的并行度。
- `M` 数可以 **大于** P：因为有些 M 陷在 syscall 里，需要额外线程顶上。

`runtime.NumCPU()`、`runtime.GOMAXPROCS(0)` 查看。容器里旧版本曾误读宿主机核数，导致 P 过多；较新版本尊重 cgroup 限额。

乱加大 `GOMAXPROCS` 不会让 I/O 服务线性变快，反而增加调度开销。CPU 密集且可并行的计算，保持默认或按实测微调。

---

## 6. 和你写的代码的对应

| 你看到的 | 调度器里发生的 |
|----------|----------------|
| `go f()` | 新建 G，入队 |
| `ch <- x` 阻塞 | G 挂起，M 跑其它 G |
| `http.Listen` | 多数时间在 netpoll 等 |
| `file.Read` 大文件 | 可能进 syscall，P 被移交 |
| `cgo` 调慢 C | 类似 syscall，占着 M |
| `runtime.LockOSThread` | 这个 G 绑死某个 M |
| 泄漏的 G 在 `<-ch` | 一个 G 永远占着栈内存 |

`runtime.NumGoroutine()` 包含系统 goroutine（GC、poller），比你 `go` 出去的要多几个。

---

## 7. 观察

```bash
GODEBUG=schedtrace=1000 ./app     # 每秒打调度摘要
GODEBUG=scheddetail=1,schedtrace=1000 ./app
curl http://localhost:6060/debug/pprof/goroutine?debug=2
```

pprof goroutine 画像能看见谁卡在 `chan receive`、`select`、`syscall`。这是查泄漏的第一工具。

---

## 想一想

1. 为什么「M 必须持有 P 才能跑 Go 代码」，却允许 M 的数量多于 P？
2. 空转 `for { counter++ }` 在 1.14 前后对其他 goroutine 的影响差在哪？
3. 大量并发 `os.ReadFile` 小文件，和大量并发 HTTP 客户端，谁更容易把 M 撑爆？

## 练习

1. 写一个只 `select{}` 的程序，打印 `NumGoroutine`。
2. 启动 10000 个 `time.Sleep(10s)` 的 G，观察内存和 `NumGoroutine`。
3. 用 pprof 看一个故意泄漏的服务，指出卡在哪一行。

下一章：[内存模型与 GC](./02-内存模型与GC.md)
