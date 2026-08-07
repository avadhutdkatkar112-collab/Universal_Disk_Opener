// Package worker implements a concurrent worker pool for background tasks.
// It handles sector reading, indexing, preview generation, and search.
package worker

import (
	"sync"
)

// Task represents a unit of work.
type Task struct {
	ID       string
	Priority int
	Fn       func() (interface{}, error)
}

// Result holds the result of a task.
type Result struct {
	TaskID string
	Value  interface{}
	Err    error
}

// Pool manages a pool of workers.
type Pool struct {
	tasks    chan Task
	results  chan Result
	workers  int
	wg       sync.WaitGroup
	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewPool creates a worker pool with the given number of workers.
func NewPool(workers int) *Pool {
	if workers <= 0 {
		workers = 4
	}
	return &Pool{
		tasks:   make(chan Task, 100),
		results: make(chan Result, 100),
		workers: workers,
		stopCh:  make(chan struct{}),
	}
}

// Start begins processing tasks.
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

// Submit adds a task to the pool.
func (p *Pool) Submit(task Task) {
	p.tasks <- task
}

// Results returns the results channel.
func (p *Pool) Results() <-chan Result {
	return p.results
}

// Stop stops the worker pool.
func (p *Pool) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
		close(p.tasks)
		p.wg.Wait()
		close(p.results)
	})
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			value, err := task.Fn()
			select {
			case p.results <- Result{TaskID: task.ID, Value: value, Err: err}:
			case <-p.stopCh:
				return
			}
		case <-p.stopCh:
			return
		}
	}
}
