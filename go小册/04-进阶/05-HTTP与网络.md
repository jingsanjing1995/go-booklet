# 05 · HTTP 与网络

标准库 `net/http` 就能支撑很多生产服务。这一章把服务端、客户端、超时、优雅退出和测试写到能落地。框架（Gin、Echo、chi）大多是在这之上加路由糖和中间件，底层模型不变。

---

## 1. 服务端骨架

```go
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /users/{id}", getUser)

	s := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(s.ListenAndServe())
}
```

Go 1.22+ 路由：`GET /users/{id}`，用 `r.PathValue("id")` 取值。

每个请求一个 goroutine。handler **不要再泄漏自己的 goroutine**，有后台任务就用带超时的 context，并在请求结束时取消。

---

## 2. 写响应的规矩

```go
func getUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	u, err := users.Find(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(u); err != nil {
		// 头部可能已写出，只能记日志
		slog.Error("encode", "err", err)
	}
}
```

- 先设 Header，再 `WriteHeader`，再写 Body。
- `http.Error` 会写状态码和文本。
- 不要把内部 `err.Error()` 直接给客户端。
- 读 Body 用 `http.MaxBytesReader(w, r.Body, 1<<20)`。

---

## 3. 中间件

```go
func withLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("req", "m", r.Method, "p", r.URL.Path, "d", time.Since(start))
	})
}

s.Handler = withLog(mux)
```

中间件是 `func(http.Handler) http.Handler`。顺序：外层先执行。恢复 panic：

```go
func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "err", rec)
				http.Error(w, "internal error", 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
```

---

## 4. 客户端：超时是必须的

```go
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

func get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}
```

`http.DefaultClient` 没有 Timeout。生产请自建 Client 复用，不要每次 `http.Get` 还不关 Body。

`Client.Timeout` 覆盖整个往返；`Context` 负责调用链取消。两者可以同时存在，谁先到谁生效。

---

## 5. 优雅退出

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

errCh := make(chan error, 1)
go func() { errCh <- s.ListenAndServe() }()

select {
case <-ctx.Done():
	shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(shut); err != nil {
		return err
	}
case err := <-errCh:
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
}
```

`Shutdown` 停止接新连接，等进行中的请求结束（受传入的 ctx 限制）。handler 应尊重 `r.Context()`。

---

## 6. httptest

```go
func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	health(rec, req)
	if rec.Code != 200 {
		t.Fatalf("code=%d", rec.Code)
	}
}

func TestClientAgainstStub(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true}`)
	}))
	t.Cleanup(ts.Close)
	// 用 ts.URL 当下游
}
```

对应示例：`codes/09-http`。

---

## 7. 更下层：net、UDP、TLS

- `net.Listen("tcp", addr)` + `Accept` 循环：自己写协议时用。
- `tls.Listen` / `http.ListenAndServeTLS`。
- 设 `SetDeadline`，避免一个慢客户端卡死 worker。
- DNS、Keep-Alive、HTTP/2 由 Transport 管理；出问题先看是否复用了 Client、是否 Close Body。

不必一上来就上 gRPC。JSON+HTTP 清楚之后，gRPC 只是另一种编码和代码生成。

---

## 想一想

1. 为什么 `ReadHeaderTimeout` 能缓解慢速攻击？
2. `Shutdown` 之后 handler 里还在写大文件，会发生什么？
3. 每次请求 `http.Get` 且不关 Body，连接池会怎样？

## 练习

1. 写 `GET /users/{id}`，id 为空返回 400，找不到 404，JSON 输出。
2. 给 mux 套上 log + recover 中间件。
3. 用 `httptest.NewServer` 测你的客户端超时：让 stub `sleep` 超过 Timeout。

下一章：[项目结构与工程实践](./06-项目结构与工程实践.md)
