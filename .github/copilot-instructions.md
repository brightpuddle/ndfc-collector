# NDFC Collector - AI Coding Agent Instructions

## Project Overview

This is a Go-based data collector that queries Cisco NDFC (Nexus Dashboard
Fabric Controller) via REST API. It fetches configuration and operational data
for health checks performed by Cisco Services. The tool produces an
`ndfc-collection-data.zip` file containing JSON responses from various NDFC API
endpoints.

**Key architectural components:**

- `cmd/ndfc-collector/main.go` - Entry point and CLI argument handling
- `cmd/ndfc-collector/collect.go` - Dependency-aware collection engine
- `pkg/ndfc/client.go` - HTTP client using session-based authentication
- `pkg/cli/cli.go` - API fetching with retry logic (`Fetch` and `FetchResult`)
- `pkg/req/requests.go` - Request definitions with dependency relationships
- `pkg/archive/archive.go` - Thread-safe zip writer using mutex locks

## Critical Patterns

### Request Configuration

All API queries are defined in [pkg/req/requests.go](pkg/req/requests.go). Each
entry specifies:

- `URL`: NDFC API endpoint path (after `/appcenter/cisco/ndfc/api/v1/`). May
  contain `{placeholder}` names for dependent queries.
- `Query`: Optional query parameters.
- `DependsOn`: URL template of a parent request. When set, this request is
  executed once per item in the parent's response array, with `{placeholder}`
  names in the URL substituted from matching JSON field names in each item.

Example of a dependent request:

```go
{
    URL:       "/lan-fabric/rest/control/fabrics/{fabricName}/inventory/switchesByFabric",
    DependsOn: "/lan-fabric/rest/control/fabrics",
}
```

This produces one request per fabric returned by the `/fabrics` endpoint,
e.g. `…/fabrics/MyFabric/inventory/switchesByFabric`.

**When modifying queries:** Update `requests.go`, then run `go generate ./...`
to regenerate the `ndfc_collector.py` Python script alternative.

### Concurrency & Batching

The collector groups requests into topological dependency levels
(`cmd/ndfc-collector/collect.go`). Within each level all expanded requests are
batched and run in parallel (default: 7 concurrent requests) using
`errgroup.Group`, preserving the original homegrown batching approach:

```go
for i := 0; i < len(expanded); i += batchSize {
    var g errgroup.Group
    for _, er := range expanded[i:end] {
        er := er
        g.Go(func() error {
            res, err := cli.FetchResult(client, fetchReq, arc, cfg)
            ...
        })
    }
    g.Wait()
}
```

Dependent requests are only expanded after their parent level has completed,
so ordering is guaranteed automatically.

**File Naming:** NDFC responses are stored using filenames derived from the
resolved URL path (placeholders already substituted). For example,
`/lan-fabric/rest/control/fabrics/MyFabric/inventory/switchesByFabric` becomes
`lan-fabric.rest.control.fabrics.MyFabric.inventory.switchesByFabric.json`.

### Authentication

The NDFC client uses session-based authentication via cookies. Unlike ACI, NDFC
doesn't require periodic token refresh - the session is maintained by the HTTP
client's cookie jar.

### Error Handling & Retries

Failed requests retry up to 3 times with 10-second delays
([cli.go](pkg/cli/cli.go)). If a parent request fails, its children are
silently skipped (no parent results → no expanded child requests).

## Development Workflow

### Building & Testing

```bash
# Run from source
go run ./cmd/ndfc-collector/*.go

# Run tests
go test ./...

# Generate Python script (run after modifying requests.go)
go generate ./...

# Build release binaries (requires goreleaser)
./scripts/release
```

### Release Process

1. Tag version: `git tag v1.2.3`
2. Run `./scripts/release` - this:
   - Runs `go generate ./...` to generate `ndfc_collector.py`
   - Builds cross-platform binaries via goreleaser
   - Packages with README and LICENSE into zip archives

**Note:** `.goreleaser.yml` defines build targets: Windows/Linux/Darwin (arm64
for macOS). CGO is disabled for static binaries.

### Testing Patterns

New collection logic tests live in
`cmd/ndfc-collector/collect_test.go` and test pure functions directly
(`substituteURL`, `mergeCtx`, `buildLevels`, `expandLevel`).

## Project-Specific Conventions

### Logging

Uses [zerolog](https://github.com/rs/zerolog) throughout. Log levels:

- `log.Info()` - User-facing progress messages
- `log.Debug()` - Timing/diagnostic info (start/end times)
- `log.Warn()` - Retry attempts, non-fatal issues
- `log.Fatal()` - Unrecoverable errors (exits program)

### File Organization

- **Packages are thin:** Each `pkg/` subdirectory has 2-4 files
  (implementation + tests)
- **No internal pkg:** All packages are directly under `pkg/`
- **Single binary:** Only one cmd entry point at `cmd/ndfc-collector/`

### CLI Argument Handling

Uses [kong](https://github.com/alecthomas/kong) for structured CLI parsing.
Arguments support environment variables (e.g., `NDFC_URL`, `NDFC_USERNAME`).
Interactive prompts fill missing required values.

**Important:** Passwords with quotes are escaped:
`strings.ReplaceAll(cfg.Password, "\"", "\\\"")` to handle special characters in
NDFC passwords.

## External Dependencies

- **tidwall/gjson & sjson** - Fast JSON parsing/building without struct
  marshaling
- **golang.org/x/sync/errgroup** - Parallel error handling for batched requests
- **alecthomas/kong** - CLI argument parsing with struct tags
- **rs/zerolog** - Structured logging

## Common Gotchas

1. **Archive writes must be thread-safe:** `archive.FileWriter.Add` uses
   `zipMux.Lock()` since parallel goroutines write to the same zip file.

2. **URL normalization:** User input is stripped of `http://` and `https://`
   prefixes, then `https://` is re-added in `ndfc.NewClient`.

3. **Version injection:** The `version` variable in
   `cmd/ndfc-collector/main.go` is set via `-ldflags` during build:
   `-X main.version=$TAG`.

4. **Dual collection methods:** Binary collector (this codebase) and Python
   script (`ndfc_collector.py`) must stay in sync. Always run
   `go generate ./...` after modifying `requests.go`.

5. **Dependent request expansion:** If a parent request returns no results
   (empty array or failed), its child requests produce no expanded requests and
   are silently skipped. This is intentional — if `/fabrics` fails there are
   no fabric names to substitute.

