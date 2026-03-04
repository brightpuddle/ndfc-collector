// Package requests contains the collector requests
package requests

//go:generate go run ../../cmd/genscript/main.go

import (
	_ "embed"
	"fmt"

	"ndfc-collector/pkg/ndfc"

	"gopkg.in/yaml.v3"
)

// Mod modifies an ndfc Request
type Mod = func(*ndfc.Req)

// Dependency describes how one URL placeholder is resolved from a parent request.
// URL is the template URL of the parent request; Key is the JSON field name in
// each parent response item whose value should be substituted for the placeholder.
type Dependency struct {
	URL string `yaml:"url"` // URL template of the parent request
	Key string `yaml:"key"` // JSON field name in the parent response item
}

// Request is an HTTP request.
type Request struct {
	URL       string                // API endpoint URL (after /appcenter/cisco/ndfc/api/v1/, may contain {placeholder} patterns)
	Query     map[string]string     // Query parameters
	DependsOn map[string]Dependency // maps each URL {placeholder} name to the parent request and JSON key that supplies its value
	// Storage metadata (used by vetr for ingestion; ignored by collector HTTP logic)
	DBKey     string `yaml:"db_key"`    // canonical key prefix (slashes→dots for filename, used as buntDB prefix)
	ListPath  string `yaml:"list_path"` // dot-notation path to the item array in the response
	IDField   string `yaml:"id_field"`  // JSON field used as the unique row identifier
}

const BaseURL = "/api/v1"

//go:embed requests.yaml
var requestsYAML []byte

// yamlRequests is the intermediate representation used to parse requests.yaml.
type yamlRequests struct {
	Requests []struct {
		URL       string                       `yaml:"url"`
		DBKey     string                       `yaml:"db_key"`
		ListPath  string                       `yaml:"list_path"`
		IDField   string                       `yaml:"id_field"`
		Query     map[string]string            `yaml:"query"`
		DependsOn map[string]struct {
			URL string `yaml:"url"`
			Key string `yaml:"key"`
		} `yaml:"depends_on"`
	} `yaml:"requests"`
}

// GetRequests parses requests.yaml and returns normalized requests.
func GetRequests() ([]Request, error) {
	var raw yamlRequests
	if err := yaml.Unmarshal(requestsYAML, &raw); err != nil {
		return nil, fmt.Errorf("parsing requests.yaml: %w", err)
	}
	reqs := make([]Request, 0, len(raw.Requests))
	for _, r := range raw.Requests {
		req := Request{
			URL:      r.URL,
			DBKey:    r.DBKey,
			ListPath: r.ListPath,
			IDField:  r.IDField,
			Query:    r.Query,
		}
		if len(r.DependsOn) > 0 {
			req.DependsOn = make(map[string]Dependency, len(r.DependsOn))
			for placeholder, dep := range r.DependsOn {
				req.DependsOn[placeholder] = Dependency{URL: dep.URL, Key: dep.Key}
			}
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}
