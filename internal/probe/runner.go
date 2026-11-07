package probe

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/Chakri-Kolan/service-readiness-probe/internal/config"
)

func Run(ctx context.Context, targets []config.Target, workers int, execute Executor) []Result {
	if workers < 1 {
		workers = 1
	}
	if workers > len(targets) {
		workers = len(targets)
	}
	jobs := make(chan config.Target)
	results := make(chan Result, len(targets))
	var wait sync.WaitGroup

	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for target := range jobs {
				results <- runTarget(ctx, target, execute)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, target := range targets {
			select {
			case jobs <- target:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wait.Wait()
		close(results)
	}()

	collected := make([]Result, 0, len(targets))
	for result := range results {
		collected = append(collected, result)
	}
	sort.Slice(collected, func(i, j int) bool { return collected[i].Name < collected[j].Name })
	return collected
}

func runTarget(ctx context.Context, target config.Target, execute Executor) Result {
	client := &http.Client{Timeout: time.Duration(target.TimeoutMS) * time.Millisecond}
	var result Result
	for attempt := 1; attempt <= target.Retries+1; attempt++ {
		result = execute(ctx, target, client)
		result.Attempts = attempt
		if result.Passed || attempt > target.Retries {
			return result
		}
		delay := time.Duration(target.BackoffMS*attempt) * time.Millisecond
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			result.Failures = append(result.Failures, "retry canceled")
			return result
		}
	}
	return result
}
