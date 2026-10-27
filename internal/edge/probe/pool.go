// Package probe provides tier probing (L0-L3) and concurrency orchestration for CLI adapters.
package probe

import (
	"context"
	"runtime"
	"sync"
)

// Task defines an executable unit of probe work.
type Task func(context.Context) error

// Pool manages concurrent task execution with a bounded worker pool.
type Pool struct {
	limit int
}

// NewPool creates a worker pool bounded by maxWorkers. Defaults to runtime.NumCPU() if <= 0.
func NewPool(maxWorkers int) *Pool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	return &Pool{limit: maxWorkers}
}

// Run executes all tasks using the bounded worker pool and waits for all to finish.
func (p *Pool) Run(ctx context.Context, tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}

	limit := p.limit
	if limit <= 0 {
		limit = runtime.NumCPU()
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

loop:
	for _, task := range tasks {
		if ctx.Err() != nil {
			break
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break loop
		}

		wg.Add(1)
		go func(t Task) {
			defer func() {
				<-sem
				wg.Done()
			}()
			if err := t(ctx); err != nil {
				once.Do(func() { firstErr = err })
			}
		}(task)
	}

	wg.Wait()
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}
