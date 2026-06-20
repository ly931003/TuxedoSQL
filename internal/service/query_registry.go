package service

import (
	"context"
	"fmt"
	"sync"
)

type QueryRegistry struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewQueryRegistry() *QueryRegistry {
	return &QueryRegistry{
		cancels: make(map[string]context.CancelFunc),
	}
}

func (r *QueryRegistry) Register(id string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.cancels[id] = cancel
}

func (r *QueryRegistry) Cancel(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cancel, ok := r.cancels[id]
	if !ok {
		return fmt.Errorf("查询 %s 不存在或已完成", id)
	}

	cancel()
	delete(r.cancels, id)
	return nil
}

func (r *QueryRegistry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.cancels, id)
}
