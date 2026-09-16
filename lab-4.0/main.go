package main

import (
	"context"
	"fmt"
	"lab-4.0/pool"
	"time"
)

func main() {
	const jobs = 10

	jobsCh := make(chan pool.Job, jobs)
	for i := 0; i < jobs; i++ {
		jobsCh <- pool.Job{ID: i, Value: i}
	}
	close(jobsCh)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	worker := func(ctx context.Context, n int) (int, error) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return n * n, nil
		}
	}

	results := pool.New(4, worker).Process(ctx, jobsCh)

	done := 0
	for r := range results {
		if r.Err != nil {
			fmt.Printf("job %d failed: %v\n", r.ID, r.Err)
			continue
		}
		fmt.Printf("job %d = %d\n", r.ID, r.Value)
		done++
	}

	fmt.Printf("completed %d/%d jobs\n", done, jobs)
}