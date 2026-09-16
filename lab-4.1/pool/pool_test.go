package pool

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

func TestProcessAllJobs(t *testing.T) {
	const (
		workers = 4
		jobs    = 100
	)

	jobsCh := make(chan Job[int], jobs)
	for i := range jobs {
		jobsCh <- Job[int]{ID: i, Value: i}
	}
	close(jobsCh)

	worker := func(_ context.Context, v int) (int, error) {
		return v * 2, nil
	}

	results := New(workers, worker).Process(context.Background(), jobsCh)

	seen := make([]bool, jobs)
	got := 0
	for r := range results {
		if r.Err != nil {
			t.Fatalf("job %d: unexpected error %v", r.ID, r.Err)
		}
		if r.ID < 0 || r.ID >= jobs {
			t.Fatalf("unexpected result id %d", r.ID)
		}
		if seen[r.ID] {
			t.Errorf("result id %d seen twice", r.ID)
		}
		seen[r.ID] = true
		got++
	}

	if got != jobs {
		t.Errorf("got %d results, want %d", got, jobs)
	}
}

func TestProcessSingleWorkerInOrder(t *testing.T) {
	const jobs = 50

	jobsCh := make(chan Job[int], jobs)
	for i := range jobs {
		jobsCh <- Job[int]{ID: i, Value: i}
	}
	close(jobsCh)

	worker := func(_ context.Context, v int) (int, error) {
		return v, nil
	}

	results := New(1, worker).Process(context.Background(), jobsCh)

	next := 0
	for r := range results {
		if r.ID != next {
			t.Errorf("result order: got id %d at position %d", r.ID, next)
		}
		next++
	}
	if next != jobs {
		t.Errorf("got %d results, want %d", next, jobs)
	}
}

func TestProcessLimitsConcurrentWorkers(t *testing.T) {
	const (
		workers = 3
		jobs    = 9
	)

	jobsCh := make(chan Job[int], jobs)
	for i := range jobs {
		jobsCh <- Job[int]{ID: i, Value: i}
	}
	close(jobsCh)

	var active, maxActive int32
	started := make(chan struct{})
	release := make(chan struct{})

	worker := func(_ context.Context, v int) (int, error) {
		cur := atomic.AddInt32(&active, 1)
		for {
			prev := atomic.LoadInt32(&maxActive)
			if cur <= prev || atomic.CompareAndSwapInt32(&maxActive, prev, cur) {
				break
			}
		}
		started <- struct{}{}
		<-release
		atomic.AddInt32(&active, -1)
		return v, nil
	}

	results := New(workers, worker).Process(context.Background(), jobsCh)

	for range workers {
		<-started
	}

	if got := atomic.LoadInt32(&maxActive); got != workers {
		t.Errorf("max concurrent workers = %d, want %d", got, workers)
	}

	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range started {
		}
	}()

	close(release)

	got := 0
	for range results {
		got++
	}
	if got != jobs {
		t.Errorf("processed %d jobs, want %d", got, jobs)
	}

	close(started)
	<-drained
}

func TestProcessWorkerError(t *testing.T) {
	jobsCh := make(chan Job[int], 3)
	for i := 0; i < 3; i++ {
		jobsCh <- Job[int]{ID: i, Value: i}
	}
	close(jobsCh)

	worker := func(_ context.Context, v int) (int, error) {
		if v == 1 {
			return 0, errBoom
		}
		return v, nil
	}

	results := New(2, worker).Process(context.Background(), jobsCh)

	failures := 0
	successes := 0
	for r := range results {
		switch {
		case errors.Is(r.Err, errBoom):
			failures++
		case r.Err == nil:
			successes++
		default:
			t.Fatalf("unexpected error %v", r.Err)
		}
	}

	if failures != 1 || successes != 2 {
		t.Errorf("failures=%d successes=%d, want 1 and 2", failures, successes)
	}
}

func TestProcessStopsOnCancellation(t *testing.T) {
	const workers = 4

	jobsCh := make(chan Job[int], workers)
	for i := range workers {
		jobsCh <- Job[int]{ID: i, Value: i}
	}

	ctx, cancel := context.WithCancel(context.Background())

	started := make(chan struct{}, workers)
	worker := func(ctx context.Context, v int) (int, error) {
		started <- struct{}{}
		<-ctx.Done()
		return v, ctx.Err()
	}

	results := New(workers, worker).Process(ctx, jobsCh)

	for range workers {
		<-started
	}

	cancel()

	select {
	case r, ok := <-results:
		if ok {
			t.Fatalf("unexpected result after cancellation: %+v", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("results channel not closed after cancellation")
	}
}

func TestProcessExpiredContextClosesResults(t *testing.T) {
	jobsCh := make(chan Job[int], 10)
	for i := range 10 {
		jobsCh <- Job[int]{ID: i, Value: i}
	}
	close(jobsCh)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Minute))
	defer cancel()

	worker := func(_ context.Context, v int) (int, error) {
		return v, nil
	}

	results := New(4, worker).Process(ctx, jobsCh)

	select {
	case r, ok := <-results:
		if ok {
			t.Fatalf("expired context produced a result: %+v", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("results channel not closed for expired context")
	}
}

func TestWorkerReceivesContext(t *testing.T) {
	jobsCh := make(chan Job[int], 1)
	jobsCh <- Job[int]{ID: 1, Value: 1}
	close(jobsCh)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(3*time.Second))
	defer cancel()

	deadlineSeen := make(chan time.Time, 1)
	worker := func(ctx context.Context, v int) (int, error) {
		d, _ := ctx.Deadline()
		deadlineSeen <- d
		return v, nil
	}

	results := New(1, worker).Process(ctx, jobsCh)

	select {
	case d := <-deadlineSeen:
		if d.IsZero() {
			t.Error("worker received a context without the deadline")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker never received the context")
	}

	for range results {
	}
}

func TestProcessReusesPool(t *testing.T) {
	pool := New(2, func(_ context.Context, v int) (int, error) {
		return v + 1, nil
	})

	for pass := 0; pass < 2; pass++ {
		jobsCh := make(chan Job[int], 10)
		for i := range 10 {
			jobsCh <- Job[int]{ID: i, Value: i}
		}
		close(jobsCh)

		got := 0
		for r := range pool.Process(context.Background(), jobsCh) {
			if r.Value != r.ID+1 {
				t.Errorf("pass %d: got %d, want %d", pass, r.Value, r.ID+1)
			}
			got++
		}
		if got != 10 {
			t.Errorf("pass %d: got %d results, want 10", pass, got)
		}
	}
}

func TestNewClampsZeroWorkers(t *testing.T) {
	jobsCh := make(chan Job[int], 3)
	for i := range 3 {
		jobsCh <- Job[int]{ID: i}
	}
	close(jobsCh)

	results := New(0, func(_ context.Context, v int) (int, error) {
		return v, nil
	}).Process(context.Background(), jobsCh)

	got := 0
	for range results {
		got++
	}
	if got != 3 {
		t.Errorf("got %d results, want 3", got)
	}
}

func TestNewNilWorkerPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New with a nil worker: expected panic")
		}
	}()
	New[int](1, nil)
}