package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Chakri-Kolan/service-readiness-probe/internal/config"
	"github.com/Chakri-Kolan/service-readiness-probe/internal/probe"
)

const (
	exitOK      = 0
	exitFailure = 1
	exitError   = 2
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("readiness-probe", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "path to JSON configuration")
	format := fs.String("format", "text", "output format: text or json")
	workers := fs.Int("workers", 4, "maximum concurrent probes")
	if err := fs.Parse(args); err != nil {
		return exitError
	}
	if *configPath == "" {
		fmt.Fprintln(stderr, "config is required")
		return exitError
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "format must be text or json")
		return exitError
	}
	if *workers < 1 {
		fmt.Fprintln(stderr, "workers must be at least 1")
		return exitError
	}
	cfg, err := config.LoadFile(*configPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitError
	}
	results := probe.Run(context.Background(), cfg.Targets, *workers, probe.Once)
	passed := allPassed(results)
	if *format == "json" {
		payload := struct {
			Passed  bool           `json:"passed"`
			Results []probe.Result `json:"results"`
		}{passed, results}
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(payload); err != nil {
			fmt.Fprintln(stderr, err)
			return exitError
		}
	} else {
		writeText(stdout, results, passed)
	}
	if !passed {
		return exitFailure
	}
	return exitOK
}

func allPassed(results []probe.Result) bool {
	for _, result := range results {
		if !result.Passed {
			return false
		}
	}
	return len(results) > 0
}

func writeText(w io.Writer, results []probe.Result, passed bool) {
	for _, result := range results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(w, "%-4s %-20s status=%d latency=%dms attempts=%d", status, result.Name, result.StatusCode, result.LatencyMS, result.Attempts)
		if result.TLSDaysLeft != nil {
			fmt.Fprintf(w, " tls_days=%d", *result.TLSDaysLeft)
		}
		if len(result.Failures) > 0 {
			fmt.Fprintf(w, " reason=%s", strings.Join(result.Failures, "; "))
		}
		fmt.Fprintln(w)
	}
	if passed {
		fmt.Fprintln(w, "readiness: PASS")
	} else {
		fmt.Fprintln(w, "readiness: FAIL")
	}
}
