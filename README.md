# Service Readiness Probe

Concurrent Go CLI for validating HTTP service readiness before a deployment or failover. Each target can enforce status, latency, response body, response headers, retries, and minimum TLS certificate lifetime.

## Run

Create a JSON configuration using `examples/targets.json` as a template, then run:

```bash
go run ./cmd/readiness-probe --config examples/targets.json
```

Machine-readable output:

```bash
go run ./cmd/readiness-probe \
  --config examples/targets.json \
  --workers 4 \
  --format json
```

Exit codes:

- `0`: every target passed
- `1`: one or more targets failed
- `2`: invalid configuration or command usage

Only `GET` and `HEAD` are supported. URLs with embedded credentials are rejected, query strings are removed from output, response-body inspection is capped at 1 MiB, and no credentials are stored by the tool.

## Verify

```bash
gofmt -w .
go vet ./...
go test -race -cover ./...
go build ./cmd/readiness-probe
```

Tests use local in-memory HTTP/TLS servers and do not require external services.
