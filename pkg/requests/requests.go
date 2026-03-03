// Package requests contains the collector requests
package requests

//go:generate go run ../../cmd/genscript/main.go

import "ndfc-collector/pkg/ndfc"

// Mod modifies an ndfc Request
type Mod = func(*ndfc.Req)

// Dependency describes how one URL placeholder is resolved from a parent request.
// URL is the template URL of the parent request; Key is the JSON field name in
// each parent response item whose value should be substituted for the placeholder.
type Dependency struct {
	URL string // URL template of the parent request
	Key string // JSON field name in the parent response item
}

// Request is an HTTP request.
type Request struct {
	URL       string                // API endpoint URL (after /appcenter/cisco/ndfc/api/v1/, may contain {placeholder} patterns)
	Query     map[string]string     // Query parameters
	DependsOn map[string]Dependency // maps each URL {placeholder} name to the parent request and JSON key that supplies its value
}

const BaseURL = "/api/v1"

// Requests contains all the NDFC API requests to execute.
// Requests with a non-empty DependsOn are executed after their parent(s) complete;
// one request is issued per item in the parent's response array, with each
// {placeholder} substituted using the Dependency's Key field from that item.
var Requests = []Request{
	{URL: "/manage/inventory/switches"},
}

// GetRequests returns normalized requests
func GetRequests() ([]Request, error) {
	return Requests, nil
}
