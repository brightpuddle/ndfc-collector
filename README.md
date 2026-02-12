# NDFC Data Collector

This tool collects data from Cisco NDFC (Nexus Dashboard Fabric Controller) to be used by Cisco Services for health check analysis.

Binary releases are available [in the releases tab](https://github.com/ciscotools/ndfc-collector/releases/latest). It's recommended to always use the latest release unless you have a known requirement to use an earlier version.

## Purpose

This tool performs data collection for NDFC health checks. The tool can be run from any computer with network access to the NDFC controller.

Once the collection is complete, the tool will create an `ndfc-collection-data.zip` file in single-fabric mode. In multi-fabric mode, the tool produces one zip per fabric and an aggregate `ndfc-collection.zip` that contains all fabric archives. These files should be provided to the Cisco Services engineer for further analysis.

The tool also creates a log file that can be reviewed and/or provided to Cisco to troubleshoot any issues with the collection process. Note that this file will only be available in a failure scenario; upon successful collection this file is bundled into the `ndfc-collection-data.zip` file along with collection data.

## How it works

The tool collects data from several REST API endpoints on the NDFC controller. The results of these queries are archived in a zip file to be shared with Cisco. Unlike traditional REST APIs, NDFC's API structure varies by endpoint, so each API response is stored as a separate JSON file named after its endpoint path.

The following file can be referenced to see the API queries performed by this tool:

https://github.com/ciscotools/ndfc-collector/blob/main/pkg/req/requests.go

**Note** that this file is part of the CI/CD process for this tool, so is always up to date with the latest query data.

## Safety/Security

-	All of the queries performed by this tool are read-only API calls to retrieve configuration and operational state data.
-	Queries to NDFC are batched and throttled to ensure reduced load on the controller.
-	This tool is open source and can be compiled manually with the Go compiler

This tool only collects the output of the API endpoints listed in the requests file. Credentials are only used at the point of collection and are not stored in any way.

All data provided to Cisco will be maintained under Cisco's [data retention policy](https://www.cisco.com/c/en/us/about/trust-center/global-privacy-policy.html).

### Alternative Collection Method

The binary collector is not strictly required. The release downloads also include a Python script, named `ndfc_collector.py`. This script uses the requests library to generate the same output as the binary collector. Note that the script will need the requests library installed (`pip install requests`). This is a more involved process and doesn't include the batching, throttling, and parallel query capabilities of the binary collector, but can be used as an alternative collection mechanism if required.

## Usage

All command line parameters are optional; the tool will prompt for any missing information. Use the `--help` option to see this output from the CLI.

**Note** that only `url`, `username`, and `password` are typically required. The remainder of the options exist to work around uncommon connectivity challenges, e.g. a long RTT or slow response from NDFC.

### Single Fabric Mode (CLI)

The traditional CLI mode allows collecting from a single fabric:

```bash
./collector --url 10.1.1.1 --username admin --password cisco123
```

### Multi-Fabric Mode (Config File)

For collecting from multiple fabrics in parallel, use a YAML configuration file:

```bash
./collector --config fabrics.yaml
```

Example configuration file:

```yaml
global:
  username: admin
  password: cisco123
  confirm: true

fabrics:
  - name: production
    url: 10.1.1.1
  - name: staging
    url: 10.2.2.2
    username: staging-user
    password: staging-pass
  - name: development
    url: 10.3.3.3
```

For a fully documented example with comments for every option, see [config-example.yaml](config-example.yaml).

### Config File Features

- **Parallel Collection**: All fabrics are collected simultaneously using goroutines
- **Global Settings**: Define common settings once in the `global` section
- **Per-Fabric Overrides**: Override any global setting per fabric
- **Flexible Naming**: 
  - If `name` is specified, output is `{name}.zip`
  - If no `name`, output is `{url}.zip`
- **Aggregate Archive**: After collecting all fabrics, the tool creates `ndfc-collection.zip` containing all per-fabric zip files
- **Fabric Context in Logs**: Each log message includes the fabric name for easy tracking
- **Validation**: Ensures fabric names/URLs are unique and required fields are present

### Supported Global/Fabric Settings

All CLI parameters can be used in the config file:
- `username` - NDFC username
- `password` - NDFC password
- `request_retry_count` - Times to retry failed requests (default: 3)
- `retry_delay` - Seconds to wait before retry (default: 10)
- `batch_size` - Max parallel requests (default: 7)
- `page_size` - Objects per page for large datasets (default: 1000)
- `confirm` - Skip confirmation prompts (default: false)
- `verbose` - Enable debug level logging (default: false)
- `endpoint` - Collect single endpoint (default: all)
- `query` - Query filters for single endpoint

**Note**: `url` must be specified per fabric and is not supported as a global setting.

### Verbose Logging

Enable debug-level logging for detailed progress:

```bash
# Single fabric mode
./collector --url 10.1.1.1 --username admin --password cisco123 --verbose

# Multi-fabric mode
./collector --config fabrics.yaml --verbose

# Or set in config file
global:
  verbose: true
```

Debug logging shows:
- Individual query start/end times
- Detailed fetch progress for each endpoint
- Fine-grained timing information

Standard Info logging shows:
- Batch completion messages
- Authentication status
- Major collection milestones

## Command Line Options

```
NDFC collector
version ...
Usage: collector [--url URL] [--username USERNAME] [--password PASSWORD] [--output OUTPUT] [--config CONFIG] [--request-retry-count REQUEST-RETRY-COUNT] [--retry-delay RETRY-DELAY] [--batch-size BATCH-SIZE] [--page-size PAGE-SIZE] [--confirm] [--verbose] [--endpoint ENDPOINT] [--query QUERY]

Options:
  --url URL              NDFC hostname or IP address [env: NDFC_URL]
  --username USERNAME    NDFC username [env: NDFC_USERNAME]
  --password PASSWORD    NDFC password [env: NDFC_PASSWORD]
  --output OUTPUT, -o OUTPUT
                         Output file [default: ndfc-collection-data.zip]
  --config CONFIG, -c CONFIG
                         Path to YAML configuration file
  --request-retry-count REQUEST-RETRY-COUNT
                         Times to retry a failed request [default: 3]
  --retry-delay RETRY-DELAY
                         Seconds to wait before retry [default: 10]
  --batch-size BATCH-SIZE
                         Max request to send in parallel [default: 7]
  --page-size PAGE-SIZE
                         Object per page for large datasets [default: 1000]
  --confirm, -y          Skip confirmation
  --verbose, -v          Enable verbose (debug level) logging
  --endpoint ENDPOINT    Collect a single endpoint [default: all]
  --query QUERY, -q QUERY
                         Query(s) to filter single endpoint query
  --help, -h             display this help and exit
  --version              display version and exit
```

## Performance and Troubleshooting

In general the collector is expected to run very quickly and have no issues. The collector sends queries in parallel for faster performance. If you encounter issues, you can adjust the `--batch-size` parameter. Setting `--batch-size 1` will make the collector behave synchronously and wait for each query to complete before sending another. This will be slower than sending requests in parallel, but may be helpful for troubleshooting purposes.

These and other configurable settings should not generally need to be modified, but may be useful in corner cases with unusually large configurations, heavily loaded NDFC instances, etc.

### Running code directly from source

Static binaries are provided for convenience and are generally preferred; however, if you'd like to run the code directly from source, e.g. for security auditing, this is also an option.

1.	[Install Go](https://go.dev/doc/install)
2.	Clone the repo
3.	`go mod download`
4.	`go run ./cmd/collector/*.go`

If on Windows, it's recommended to use Powershell or WSL to avoid issues with ANSI escape sequences and path slash direction.

## Architecture

This tool is written in Go and uses the following key libraries:
- **tidwall/gjson & sjson** - Fast JSON parsing/building without struct marshaling
- **golang.org/x/sync/errgroup** - Parallel error handling for batched requests
- **alexflint/go-arg** - CLI argument parsing with struct tags
- **rs/zerolog** - Structured logging
- **gopkg.in/yaml.v3** - YAML configuration file parsing

The code is organized as follows:
- `cmd/collector/` - Main entry point and CLI argument handling
- `pkg/ndfc/` - NDFC API client with authentication
- `pkg/cli/` - Request fetching logic with retry
- `pkg/archive/` - Thread-safe zip file writer
- `pkg/req/` - Request definitions
- `pkg/config/` - YAML configuration file handling
- `pkg/log/` - Logging wrapper

## Development

### Building

```bash
# Build binary
go build -o collector ./cmd/collector/*.go

# Run tests
go test ./...

# Generate Python script
go generate ./...
```

### Release Process

1. Tag version: `git tag v1.2.3`
2. Run `./scripts/release` - this:
   - Runs `go generate ./...` to generate `ndfc_collector.py`
   - Builds cross-platform binaries via goreleaser
   - Packages with README and LICENSE into zip archives
