package service

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueryRegistryRegisterAndCancel(t *testing.T) {
	tests := []struct {
		name    string
		queryID string
	}{
		{name: "注册后取消应调用 cancel", queryID: "query-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewQueryRegistry()
			called := make(chan struct{})

			registry.Register(tt.queryID, func() {
				close(called)
			})

			if err := registry.Cancel(tt.queryID); err != nil {
				t.Fatalf("Cancel() 返回错误: %v", err)
			}

			select {
			case <-called:
			case <-time.After(time.Second):
				t.Fatal("期望 cancel func 被调用，但超时未触发")
			}
		})
	}
}

func TestQueryRegistryCancelUnknownID(t *testing.T) {
	tests := []struct {
		name    string
		queryID string
	}{
		{name: "取消不存在的查询应报错", queryID: "missing-query"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewQueryRegistry()

			if err := registry.Cancel(tt.queryID); err == nil {
				t.Fatal("期望返回错误，但没有")
			}
		})
	}
}

func TestQueryRegistryRemove(t *testing.T) {
	tests := []struct {
		name    string
		queryID string
	}{
		{name: "移除后取消应报错", queryID: "query-removed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewQueryRegistry()
			registry.Register(tt.queryID, func() {})

			registry.Remove(tt.queryID)

			if err := registry.Cancel(tt.queryID); err == nil {
				t.Fatal("期望返回错误，但没有")
			}
		})
	}
}

func TestQueryRegistryConcurrent(t *testing.T) {
	tests := []struct {
		name        string
		goroutines  int
		iterations  int
		wantMinimum int32
	}{
		{name: "并发注册取消移除应线程安全", goroutines: 8, iterations: 50, wantMinimum: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewQueryRegistry()
			var calledCount atomic.Int32
			var wg sync.WaitGroup

			for i := 0; i < tt.goroutines; i++ {
				wg.Add(1)
				go func(worker int) {
					defer wg.Done()

					for j := 0; j < tt.iterations; j++ {
						queryID := generateQueryID()
						registry.Register(queryID, func() {
							calledCount.Add(1)
						})

						if (worker+j)%2 == 0 {
							_ = registry.Cancel(queryID)
							continue
						}

						registry.Remove(queryID)
						_ = registry.Cancel(queryID)
					}
				}(i)
			}

			wg.Wait()

			if got := calledCount.Load(); got < tt.wantMinimum {
				t.Fatalf("cancel func 调用次数 = %d, 期望至少 %d", got, tt.wantMinimum)
			}
		})
	}
}
