# NDFC Collector - AI Coding Agent Instructions

## Project Overview

This is a Go-based data collector that queries Cisco NDFC (Nexus Dashboard Fabric Controller) via REST API. It fetches configuration and operational data for health checks performed by Cisco Services. The tool produces an `ndfc-collection-data.zip` file containing JSON responses from various NDFC API endpoints.

**Key architectural components:**
- `cmd/collector/main.go` - Entry point with batch orchestration logic
- `pkg/ndfc/client.go` - HTTP client using session-based authentication
- `pkg/cli/cli.go` - API fetching with retry logic
- `pkg/req/requests.go` - Request definitions with URL-based structure
- `pkg/archive/archive.go` - Thread-safe zip writer using mutex locks

## Critical Patterns

### Request Configuration
All API queries are defined in [pkg/req/requests.go](pkg/req/requests.go). Each entry specifies:
- `URL`: NDFC API endpoint path (after `/appcenter/cisco/ndfc/api/v1/`)
- `Query`: Optional query parameters

**When modifying queries:** Update `requests.go`, then run `go generate ./...` to regenerate the `ndfc_collector.py` Python script alternative.

### Concurrency & Batching
The collector processes requests in parallel batches (default: 7 concurrent requests). See [cmd/collector/main.go#L63-L79](cmd/collector/main.go#L63-L79):
```go
for i := 0; i < len(reqs); i += args.BatchSize {
    var g errgroup.Group
    // Launch batch of requests in parallel
    for j := i; j < i+args.BatchSize && j < len(reqs); j++ {
        g.Go(func() error {
            return cli.Fetch(client, req, arc, cfg)
        })
    }
    err = g.Wait()
}
```

**File Naming:** NDFC responses are stored using filenames derived from the URL path. For example, `/lan-fabric/rest/control/fabrics` becomes `lan-fabric.rest.control.fabrics.json`.

### Authentication
The NDFC client uses session-based authentication via cookies. Unlike ACI, NDFC doesn't require periodic token refresh - the session is maintained by the HTTP client's cookie jar.

### Error Handling & Retries
Failed requests retry up to 3 times with 10-second delays ([cli.go#L67-L78](pkg/cli/cli.go#L67-L78)). Exception: "dataset is too big" errors immediately trigger pagination instead of retry.

## Development Workflow

### Building & Testing
```bash
# Run from source
go run ./cmd/collector/*.go

# Run tests (uses gock for HTTP mocking)
go test ./...

# Build release binaries (requires goreleaser)
./scripts/release
```

### Release Process
1. Tag version: `git tag v1.2.3`
2. Run `./scripts/release` - this:
   - Runs `python make_script.py` to generate `vetr-collector.sh`
   - Builds cross-platform binaries via goreleaser
   - Packages with README and LICENSE into zip archives

**Note:** `.goreleaser.yml` defines build targets: Windows/Linux/Darwin (arm64 for macOS). CGO is disabled for static binaries.

### Testing Patterns
Tests use [gock](https://github.com/h2non/gock) to mock HTTP responses. See [pkg/aci/client_test.go](pkg/aci/client_test.go):
```go
func testClient() Client {
    client, _ := NewClient(testHost, "usr", "pwd")
    gock.InterceptClient(client.HTTPClient)
    return client
}
```

Always call `defer gock.Off()` to clean up mocks after tests.

## Project-Specific Conventions

### Logging
Uses [zerolog](https://github.com/rs/zerolog) throughout. Log levels in [pkg/log/log.go](pkg/log/log.go):
- `log.Info()` - User-facing progress messages
- `log.Debug()` - Timing/diagnostic info (start/end times)
- `log.Warn()` - Retry attempts, non-fatal issues
- `log.Fatal()` - Unrecoverable errors (exits program)

### File Organization
- **Packages are thin:** Each `pkg/` subdirectory has 2-4 files (implementation + tests)
- **No internal pkg:** All packages are directly under `pkg/`
- **Single binary:** Only one cmd entry point at `cmd/collector/`

### CLI Argument Handling
Uses [go-arg](https://github.com/alexflint/go-arg) for structured CLI parsing. Arguments support environment variables (e.g., `NDFC_URL`, `NDFC_USERNAME`). Interactive prompts fill missing required values.

**Important:** Passwords with quotes are escaped: `strings.ReplaceAll(cfg.Password, "\"", "\\\"")` to handle special characters in NDFC passwords.

## External Dependencies

- **tidwall/gjson & sjson** - Fast JSON parsing/building without struct marshaling
- **golang.org/x/sync/errgroup** - Parallel error handling for batched requests
- **alexflint/go-arg** - CLI argument parsing with struct tags
- **rs/zerolog** - Structured logging
- **h2non/gock** - HTTP mocking for tests

## Common Gotchas

1. **Archive writes must be thread-safe:** Use `zipMux.Lock()` in [archive.go#L44](pkg/archive/archive.go#L44) since parallel goroutines write to the same zip file.

2. **URL normalization:** User input is stripped of `http://` and `https://` prefixes, then `https://` is re-added in `ndfc.NewClient`.

3. **Version injection:** The `version` variable in [main.go](cmd/collector/main.go) is set via `-ldflags` during build: `-X main.version=$TAG`.

4. **Dual collection methods:** Binary collector (this codebase) and Python script (`ndfc_collector.py`) must stay in sync. Always run `go generate ./...` after modifying `requests.go`.
