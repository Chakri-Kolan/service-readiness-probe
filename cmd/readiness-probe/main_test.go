package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRejectsMissingConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != exitError {
		t.Fatalf("exit code = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), "config is required") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunRejectsInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"targets":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--config", path}, &stdout, &stderr); code != exitError {
		t.Fatalf("exit code = %d, want %d", code, exitError)
	}
}

func TestAllPassed(t *testing.T) {
	if allPassed(nil) {
		t.Fatal("empty results must not pass")
	}
}

func TestRunProbesTargetAndWritesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Ready", "true")
		_, _ = w.Write([]byte("ready"))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"targets":[{"name":"api","url":"` + server.URL + `","body_contains":"ready","required_headers":{"X-Ready":"true"}}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--config", path, "--format", "json"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit code = %d; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"passed": true`) {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunValidatesFlags(t *testing.T) {
	tests := [][]string{
		{"--config", "missing.json", "--format", "yaml"},
		{"--config", "missing.json", "--workers", "0"},
	}
	for _, args := range tests {
		if code := run(args, &bytes.Buffer{}, &bytes.Buffer{}); code != exitError {
			t.Fatalf("run(%v) = %d, want %d", args, code, exitError)
		}
	}
}
