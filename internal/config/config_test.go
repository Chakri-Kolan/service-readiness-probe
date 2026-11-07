package config

import (
	"strings"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	cfg, err := Load(strings.NewReader(`{"targets":[{"name":"api","url":"https://example.com/health"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	target := cfg.Targets[0]
	if target.Method != "GET" || target.ExpectedStatus != 200 || target.TimeoutMS != 5000 || target.BackoffMS != 100 {
		t.Fatalf("unexpected defaults: %+v", target)
	}
}

func TestLoadRejectsUnsafeAndInvalidTargets(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		needle string
	}{
		{"empty", `{"targets":[]}`, "at least one"},
		{"credentials", `{"targets":[{"name":"api","url":"https://user:pass@example.com"}]}`, "credentials"},
		{"scheme", `{"targets":[{"name":"api","url":"ftp://example.com"}]}`, "http or https"},
		{"method", `{"targets":[{"name":"api","url":"https://example.com","method":"POST"}]}`, "GET or HEAD"},
		{"head body", `{"targets":[{"name":"api","url":"https://example.com","method":"HEAD","body_contains":"ok"}]}`, "HEAD"},
		{"tls on http", `{"targets":[{"name":"api","url":"http://example.com","min_tls_days":7}]}`, "https"},
		{"duplicate", `{"targets":[{"name":"api","url":"https://one.example.com"},{"name":"api","url":"https://two.example.com"}]}`, "duplicate"},
		{"unknown", `{"targets":[{"name":"api","url":"https://example.com","unexpected":true}]}`, "unknown field"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(strings.NewReader(test.input))
			if err == nil || !strings.Contains(err.Error(), test.needle) {
				t.Fatalf("error = %v, want containing %q", err, test.needle)
			}
		})
	}
}

func TestLoadRejectsTrailingJSON(t *testing.T) {
	_, err := Load(strings.NewReader(`{"targets":[{"name":"api","url":"https://example.com"}]} {}`))
	if err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("unexpected error: %v", err)
	}
}
