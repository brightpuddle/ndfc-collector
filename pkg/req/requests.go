// Package req contains the collector requests
package req

//go:generate go run ../../cmd/genscript/main.go

import "ndfc-collector/pkg/ndfc"

// Mod modifies an ndfc Request
type Mod = func(*ndfc.Req)

// Request is an HTTP request.
type Request struct {
	URL   string            // API endpoint URL (after /appcenter/cisco/ndfc/api/v1/)
	Query map[string]string // Query parameters
}

const BaseURL = "/appcenter/cisco/ndfc/api/v1"

// Requests contains all the NDFC API requests to execute
var Requests = []Request{
	{URL: "/lan-fabric/rest/control/fabrics"},
	{URL: "/fm/about/version"},
	{URL: "/lan-fabric/rest/control/switches/overview"},
}

// GetRequests returns normalized requests
func GetRequests() ([]Request, error) {
	return Requests, nil
}
