package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type ctxKey struct{}

func withTrace(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

func traceID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

func worker(ctx context.Context) error {
	select {
	case <-time.After(200 * time.Millisecond):
		fmt.Println("工人完成", "trace", traceID(ctx))
		return nil
	case <-ctx.Done():
		return fmt.Errorf("工人中断: %w", ctx.Err())
	}
}

func main() {
	parent := withTrace(context.Background(), "t-001")
	ctx, cancel := context.WithTimeout(parent, 50*time.Millisecond)
	defer cancel()

	err := worker(ctx)
	if errors.Is(err, context.DeadlineExceeded) {
		fmt.Println("预期超时:", err)
	}
}
