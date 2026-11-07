package probe

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/Chakri-Kolan/service-readiness-probe/internal/config"
)

func TestRunRetriesAndSortsResults(t *testing.T) {
	var mu sync.Mutex
	attempts := map[string]int{}
	execute := func(_ context.Context, target config.Target, _ *http.Client) Result {
		mu.Lock()
		defer mu.Unlock()
		attempts[target.Name]++
		return Result{Name: target.Name, Passed: attempts[target.Name] > 1}
	}
	targets := []config.Target{
		{Name: "zeta", Retries: 1, BackoffMS: 0, TimeoutMS: 100},
		{Name: "alpha", Retries: 1, BackoffMS: 0, TimeoutMS: 100},
	}
	results := Run(context.Background(), targets, 2, execute)
	if len(results) != 2 || results[0].Name != "alpha" || results[1].Name != "zeta" {
		t.Fatalf("unexpected ordering: %+v", results)
	}
	for _, result := range results {
		if !result.Passed || result.Attempts != 2 {
			t.Fatalf("unexpected retry result: %+v", result)
		}
	}
}

func TestRunLimitsWorkerFloor(t *testing.T) {
	execute := func(_ context.Context, target config.Target, _ *http.Client) Result {
		return Result{Name: target.Name, Passed: true}
	}
	results := Run(context.Background(), []config.Target{{Name: "api", TimeoutMS: 100}}, 0, execute)
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("unexpected results: %+v", results)
	}
}
