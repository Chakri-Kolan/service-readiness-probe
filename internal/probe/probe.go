package probe

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Chakri-Kolan/service-readiness-probe/internal/config"
)

const maxBodyBytes = 1 << 20

type Result struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Passed      bool     `json:"passed"`
	Attempts    int      `json:"attempts"`
	StatusCode  int      `json:"status_code,omitempty"`
	LatencyMS   int64    `json:"latency_ms"`
	TLSDaysLeft *int     `json:"tls_days_left,omitempty"`
	Failures    []string `json:"failures,omitempty"`
	CheckedAt   string   `json:"checked_at"`
}

type Executor func(context.Context, config.Target, *http.Client) Result

func Once(ctx context.Context, target config.Target, client *http.Client) Result {
	result := Result{Name: target.Name, URL: sanitizedURL(target.URL), CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	request, err := http.NewRequestWithContext(ctx, target.Method, target.URL, nil)
	if err != nil {
		result.Failures = []string{fmt.Sprintf("create request: %v", err)}
		return result
	}
	request.Header.Set("User-Agent", "service-readiness-probe/1.0")

	started := time.Now()
	response, err := client.Do(request)
	result.LatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		result.Failures = []string{fmt.Sprintf("request failed: %v", err)}
		return result
	}
	defer response.Body.Close()
	result.StatusCode = response.StatusCode

	if response.StatusCode != target.ExpectedStatus {
		result.Failures = append(result.Failures, fmt.Sprintf("status %d, expected %d", response.StatusCode, target.ExpectedStatus))
	}
	if target.MaxLatencyMS > 0 && result.LatencyMS > int64(target.MaxLatencyMS) {
		result.Failures = append(result.Failures, fmt.Sprintf("latency %dms exceeds %dms", result.LatencyMS, target.MaxLatencyMS))
	}
	for name, expected := range target.RequiredHeaders {
		if actual := response.Header.Get(name); actual != expected {
			result.Failures = append(result.Failures, fmt.Sprintf("header %s is %q, expected %q", name, actual, expected))
		}
	}
	if target.BodyContains != "" {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes+1))
		if readErr != nil {
			result.Failures = append(result.Failures, fmt.Sprintf("read body: %v", readErr))
		} else if len(body) > maxBodyBytes {
			result.Failures = append(result.Failures, "response body exceeds 1 MiB inspection limit")
		} else if !strings.Contains(string(body), target.BodyContains) {
			result.Failures = append(result.Failures, fmt.Sprintf("body does not contain %q", target.BodyContains))
		}
	}
	if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
		days := int(time.Until(response.TLS.PeerCertificates[0].NotAfter).Hours() / 24)
		result.TLSDaysLeft = &days
		if days < target.MinTLSDays {
			result.Failures = append(result.Failures, fmt.Sprintf("TLS certificate has %d days left, requires %d", days, target.MinTLSDays))
		}
	}
	result.Passed = len(result.Failures) == 0
	return result
}

func sanitizedURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "invalid-url"
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
