package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Chakri-Kolan/service-readiness-probe/internal/config"
)

func TestOncePassesStatusHeaderBodyAndTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ready", "true")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}))
	defer server.Close()
	target := config.Target{
		Name: "api", URL: server.URL + "/health?key=not-reported", Method: "GET",
		ExpectedStatus: 200, BodyContains: "ready", RequiredHeaders: map[string]string{"X-Ready": "true"},
	}
	result := Once(context.Background(), target, server.Client())
	if !result.Passed || result.TLSDaysLeft == nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	if strings.Contains(result.URL, "key=") {
		t.Fatalf("result URL leaked query string: %s", result.URL)
	}
}

func TestOnceCollectsValidationFailures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("starting"))
	}))
	defer server.Close()
	target := config.Target{
		Name: "api", URL: server.URL, Method: "GET", ExpectedStatus: 200,
		BodyContains: "ready", RequiredHeaders: map[string]string{"X-Ready": "true"},
	}
	result := Once(context.Background(), target, server.Client())
	if result.Passed || len(result.Failures) != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestOnceEnforcesLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	target := config.Target{Name: "slow", URL: server.URL, Method: "GET", ExpectedStatus: 200, MaxLatencyMS: 5}
	result := Once(context.Background(), target, server.Client())
	if result.Passed || !strings.Contains(strings.Join(result.Failures, " "), "latency") {
		t.Fatalf("expected latency failure: %+v", result)
	}
}

func TestOnceHandlesConnectionFailure(t *testing.T) {
	target := config.Target{Name: "down", URL: "http://127.0.0.1:1", Method: "GET", ExpectedStatus: 200}
	client := &http.Client{Timeout: 100 * time.Millisecond}
	result := Once(context.Background(), target, client)
	if result.Passed || !strings.Contains(result.Failures[0], "request failed") {
		t.Fatalf("expected request failure: %+v", result)
	}
}
