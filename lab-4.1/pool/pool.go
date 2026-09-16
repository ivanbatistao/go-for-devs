// Package pool implements a generic worker pool. A fixed number of workers
// consume jobs from an input channel and publish results to an output
// channel. Processing stops when the input channel is closed or when the
// provided context is cancelled.
package pool

import (
	"context"
	"sync"
)

// Job is a unit of work. ID identifies the job and Value is passed to workers.
type Job[T any] struct {
	ID    int
	Value T
}

// Result carries the outcome of a single job. Err is non-nil when the worker
// failed to process the job.
type Result[T any] struct {
	ID    int
	Value T
	Err   error
}

// Worker processes a single job value and returns the processed value or an
// error. It receives the pool context and should return promptly when it is
// cancelled.
type Worker[T any] func(ctx context.Context, value T) (T, error)

// Pool runs up to n workers concurrently over the jobs it is given.
type Pool[T any] struct {
	workers int
	worker  Worker[T]
}

// New returns a Pool that runs up to n workers concurrently. A n smaller than
// 1 is treated as 1. A nil worker panics.
func New[T any](n int, worker Worker[T]) *Pool[T] {
	if worker == nil {
		panic("pool: nil worker")
	}
	if n < 1 {
		n = 1
	}
	return &Pool[T]{workers: n, worker: worker}
}

// Process starts the pool workers and consumes jobs from jobs until it is
// closed or ctx is cancelled. A Result is published for every job that
// finishes before cancellation; results of in-flight jobs are dropped when
// ctx is cancelled. The returned channel is closed once all workers have
// stopped, so callers can range over it to completion.
func (p *Pool[T]) Process(ctx context.Context, jobs <-chan Job[T]) <-chan Result[T] {
	results := make(chan Result[T], p.workers)

	var wg sync.WaitGroup
	wg.Add(p.workers)

	for range p.workers {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					value, err := p.worker(ctx, job.Value)
					if ctx.Err() != nil {
						return
					}
					results <- Result[T]{ID: job.ID, Value: value, Err: err}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}