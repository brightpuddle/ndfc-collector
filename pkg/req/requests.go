// Package req contains the collector requests
package req

//go:generate go run ../../cmd/genscript/main.go

import "ndfc-collector/pkg/ndfc"

// Mod modifies an ndfc Request
type Mod = func(*ndfc.Req)

// Request is an HTTP request.
type Request struct {
	URL       string            // API endpoint URL (after /appcenter/cisco/ndfc/api/v1/, may contain {placeholder} patterns)
	Query     map[string]string // Query parameters
	DependsOn string            // URL template of parent request (empty if no dependency); {placeholder} names in URL are resolved from each parent response item
}

const BaseURL = "/appcenter/cisco/ndfc/api/v1"

// Requests contains all the NDFC API requests to execute.
// Requests with DependsOn are executed after their parent completes;
// one request is issued per item in the parent's response array,
// with {placeholder} names substituted from matching JSON fields.
var Requests = []Request{
	{URL: "/lan-fabric/rest/control/fabrics"},
	{URL: "/fm/about/version"},
	{URL: "/lan-fabric/rest/control/switches/overview"},
	{
		URL:       "/lan-fabric/rest/control/fabrics/{fabricName}/inventory/switchesByFabric",
		DependsOn: "/lan-fabric/rest/control/fabrics",
	},
}

// GetRequests returns normalized requests
func GetRequests() ([]Request, error) {
	return Requests, nil
}
