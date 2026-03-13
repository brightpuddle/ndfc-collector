# NDFC Data Collector

This tool collects data from Cisco NDFC (Nexus Dashboard Fabric Controller) to
be used by Cisco Services for health check analysis.

Binary releases are available
[in the releases tab](https://github.com/ciscotools/ndfc-collector/releases/latest).
It's recommended to always use the latest release unless you have a known
requirement to use an earlier version.

## Purpose

This tool performs data collection for NDFC health checks. The tool can be run
from any computer with network access to the NDFC controller.

Once the collection is complete, the tool will create an
`ndfc-collection-data.zip` file that should be provided to the Cisco Services
engineer for further analysis.

The tool also creates a log file that can be reviewed and/or provided to Cisco
to troubleshoot any issues with the collection process. Note that this file will
only be available in a failure scenario; upon successful collection this file is
bundled into the `ndfc-collection-data.zip` file along with collection data.

## How it works

The tool collects data from several REST API endpoints on the NDFC controller.
The results of these queries are archived in a zip file to be shared with Cisco.
Unlike traditional REST APIs, NDFC's API structure varies by endpoint, so each
API response is stored as a separate JSON file named after its endpoint path.

Some endpoints depend on data from other endpoints. For example, the security
segmentation VRF inventory request requires a fabric name supplied as a query
parameter:

```
/api/v1/analyze/securitySegmentation/vrfs?fabricName={fabricName}
```

The collector automatically handles these **dependent queries**: it first
fetches the parent endpoint (for example `/api/v1/manage/fabrics`), then issues
one child request per item in the parent's response array. Each `{placeholder}`
in the child URL or query string is resolved using the `Dependency.Key` JSON
field from the parent response item. Child requests run in parallel within their
dependency level, so there is no unnecessary serialisation.

The canonical query list lives in
<https://github.com/ciscotools/ndfc-collector/blob/main/pkg/requests/requests.yaml>.
Each request URL is a full host-relative path copied from the OpenAPI spec's
`servers[0].url` + operation path.

## Safety/Security

- All of the queries performed by this tool are read-only API calls to retrieve
  configuration and operational state data.
- Queries to NDFC are batched and throttled to ensure reduced load on the
  controller.
- This tool is open source and can be compiled manually with the Go compiler

This tool only collects the output of the API endpoints listed in the requests
file. Credentials are only used at the point of collection and are not stored in
any way.

All data provided to Cisco will be maintained under Cisco's
[data retention policy](https://www.cisco.com/c/en/us/about/trust-center/global-privacy-policy.html).

### Alternative Collection Method

The binary collector is not strictly required. The release downloads also
include a Python script, named `ndfc_collector.py`. This script uses the
requests library to generate the same output as the binary collector. Note that
the script will need the requests library installed (`pip install requests`).
This is a more involved process and doesn't include the batching, throttling,
and parallel query capabilities of the binary collector, but can be used as an
alternative collection mechanism if required.

## Usage

All command line parameters are optional; the tool will prompt for any missing
information. Use the `--help` option to see this output from the CLI.

**Note** that only `url`, `username`, and `password` are typically required. The
remainder of the options exist to work around uncommon connectivity challenges,
e.g. a long RTT or slow response from NDFC.

### CLI

```bash
./ndfc-collector --url 10.1.1.1 --username admin --password cisco123
```

### Config File

All options can also be provided via a YAML configuration file:

```bash
./ndfc-collector --config collector.yaml
```

Example configuration file:

```yaml
url: 10.1.1.1
username: admin
password: cisco123
confirm: true
```

For a fully documented example with comments for every option, see
[config-example.yaml](config-example.yaml).

### Supported Settings

All CLI parameters are also supported in the config file:

- `url` - NDFC hostname or IP address
- `username` - NDFC username
- `password` - NDFC password
- `output` - Output zip file name (default: ndfc-collection-data.zip)
- `request_retry_count` - Times to retry failed requests (default: 3)
- `retry_delay` - Seconds to wait before retry (default: 10)
- `batch_size` - Max parallel requests (default: 7)
- `page_size` - Objects per page for large datasets (default: 1000)
- `confirm` - Skip confirmation prompts (default: false)
- `verbose` - Enable debug level logging (default: false)
- `endpoint` - Collect single endpoint (default: all)
- `query` - Query filters for single endpoint

### Verbose Logging

Enable debug-level logging for detailed progress:

```bash
./ndfc-collector --url 10.1.1.1 --username admin --password cisco123 --verbose
```

Or set in config file:

```yaml
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
Usage: ndfc-collector [--url URL] [--username USERNAME] [--password PASSWORD] [--output OUTPUT] [--config CONFIG] [--request-retry-count REQUEST-RETRY-COUNT] [--retry-delay RETRY-DELAY] [--batch-size BATCH-SIZE] [--page-size PAGE-SIZE] [--confirm] [--verbose] [--endpoint ENDPOINT] [--query QUERY]

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

In general the collector is expected to run very quickly and have no issues. The
collector sends queries in parallel for faster performance. If you encounter
issues, you can adjust the `--batch-size` parameter. Setting `--batch-size 1`
will make the collector behave synchronously and wait for each query to complete
before sending another. This will be slower than sending requests in parallel,
but may be helpful for troubleshooting purposes.

These and other configurable settings should not generally need to be modified,
but may be useful in corner cases with unusually large configurations, heavily
loaded NDFC instances, etc.

### Running code directly from source

Static binaries are provided for convenience and are generally preferred;
however, if you'd like to run the code directly from source, e.g. for security
auditing, this is also an option.

1. [Install Go](https://go.dev/doc/install)
2. Clone the repo
3. `go mod download`
4. `go run ./cmd/collector/*.go`

If on Windows, it's recommended to use Powershell or WSL to avoid issues with
ANSI escape sequences and path slash direction.

## Architecture

This tool is written in Go and uses the following key libraries:

- **tidwall/gjson & sjson** - Fast JSON parsing/building without struct
  marshaling
- **golang.org/x/sync/errgroup** - Parallel error handling for batched requests
- **alecthomas/kong** - CLI argument parsing with struct tags
- **rs/zerolog** - Structured logging
- **gopkg.in/yaml.v3** - YAML configuration file parsing

The code is organized as follows:

- `cmd/ndfc-collector/` - Main entry point and CLI argument handling
- `cmd/ndfc-collector/collect.go` - Dependency-aware collection engine
- `pkg/ndfc/` - NDFC API client with authentication
- `pkg/cli/` - Request fetching logic with retry (`Fetch` and `FetchResult`)
- `pkg/archive/` - Thread-safe zip file writer
- `pkg/req/` - Request definitions (including dependent query relationships)
- `pkg/config/` - YAML configuration file handling

## Development

### Building

```bash
# Build binary
go build -o collector ./cmd/ndfc-collector/*.go

# Run tests
go test ./...

# Generate Python script (run after modifying requests.yaml)
go generate ./...
```

### Release Process

1. Tag version: `git tag v1.2.3`
2. Run `./scripts/release` - this:
   - Runs `go generate ./...` to generate `ndfc_collector.py`
   - Builds cross-platform binaries via goreleaser
   - Packages with README and LICENSE into zip archives
