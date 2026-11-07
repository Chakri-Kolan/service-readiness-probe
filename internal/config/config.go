package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Targets []Target `json:"targets"`
}

type Target struct {
	Name            string            `json:"name"`
	URL             string            `json:"url"`
	Method          string            `json:"method,omitempty"`
	ExpectedStatus  int               `json:"expected_status,omitempty"`
	TimeoutMS       int               `json:"timeout_ms,omitempty"`
	MaxLatencyMS    int               `json:"max_latency_ms,omitempty"`
	Retries         int               `json:"retries,omitempty"`
	BackoffMS       int               `json:"backoff_ms,omitempty"`
	BodyContains    string            `json:"body_contains,omitempty"`
	RequiredHeaders map[string]string `json:"required_headers,omitempty"`
	MinTLSDays      int               `json:"min_tls_days,omitempty"`
}

func Load(r io.Reader) (Config, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Config{}, err
	}
	for i := range cfg.Targets {
		applyDefaults(&cfg.Targets[i])
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func LoadFile(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()
	return Load(file)
}

func (c Config) Validate() error {
	if len(c.Targets) == 0 {
		return fmt.Errorf("config must contain at least one target")
	}
	names := make(map[string]struct{}, len(c.Targets))
	for i, target := range c.Targets {
		if err := validateTarget(target); err != nil {
			return fmt.Errorf("target %d: %w", i+1, err)
		}
		if _, exists := names[target.Name]; exists {
			return fmt.Errorf("target %d: duplicate name %q", i+1, target.Name)
		}
		names[target.Name] = struct{}{}
	}
	return nil
}

func applyDefaults(target *Target) {
	if target.Method == "" {
		target.Method = "GET"
	}
	target.Method = strings.ToUpper(target.Method)
	if target.ExpectedStatus == 0 {
		target.ExpectedStatus = 200
	}
	if target.TimeoutMS == 0 {
		target.TimeoutMS = 5000
	}
	if target.BackoffMS == 0 {
		target.BackoffMS = 100
	}
}

func validateTarget(target Target) error {
	if strings.TrimSpace(target.Name) == "" {
		return fmt.Errorf("name is required")
	}
	parsed, err := url.Parse(target.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("url must be an absolute http or https URL")
	}
	if parsed.User != nil {
		return fmt.Errorf("url must not contain embedded credentials")
	}
	if target.Method != "GET" && target.Method != "HEAD" {
		return fmt.Errorf("method must be GET or HEAD")
	}
	if target.Method == "HEAD" && target.BodyContains != "" {
		return fmt.Errorf("body_contains cannot be used with HEAD")
	}
	if target.ExpectedStatus < 100 || target.ExpectedStatus > 599 {
		return fmt.Errorf("expected_status must be between 100 and 599")
	}
	if target.TimeoutMS <= 0 || target.MaxLatencyMS < 0 || target.Retries < 0 || target.BackoffMS < 0 || target.MinTLSDays < 0 {
		return fmt.Errorf("timeouts, retries, backoff, latency, and TLS days must be non-negative")
	}
	if parsed.Scheme != "https" && target.MinTLSDays > 0 {
		return fmt.Errorf("min_tls_days requires an https URL")
	}
	for name := range target.RequiredHeaders {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("required header name cannot be empty")
		}
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing config data: %w", err)
	}
	return fmt.Errorf("decode config: multiple JSON values are not allowed")
}
