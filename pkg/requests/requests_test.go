package requests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequests(t *testing.T) {
	reqs, err := GetRequests()
	assert.NoError(t, err)
	assert.NotEmpty(t, reqs)

	for i, req := range reqs {
		assert.NotEmpty(t, req.URL, "Request %d has empty URL", i)
	}
}

// TestDependentRequests verifies the dependent request structure using contrived data.
// A child request's {placeholder} should be resolved from a named field in a parent
// request's response, and the parent URL must exist in the request list.
func TestDependentRequests(t *testing.T) {
	reqs := []Request{
		{URL: "/example/fabrics"},
		{
			URL: "/example/fabrics/{fabricName}/switches",
			DependsOn: map[string]Dependency{
				"fabricName": {URL: "/example/fabrics", Key: "fabricName"},
			},
		},
	}

	byURL := make(map[string]bool, len(reqs))
	for _, r := range reqs {
		byURL[r.URL] = true
	}

	var roots []Request
	for _, r := range reqs {
		if len(r.DependsOn) == 0 {
			roots = append(roots, r)
		}
	}
	assert.NotEmpty(t, roots, "expected at least one root request")

	for _, r := range reqs {
		for placeholder, dep := range r.DependsOn {
			assert.True(t, byURL[dep.URL],
				"request %q placeholder %q depends on %q which is not in the request list",
				r.URL, placeholder, dep.URL)
			assert.NotEmpty(t, dep.Key,
				"request %q placeholder %q has empty Key", r.URL, placeholder)
			assert.Contains(t, r.URL, "{"+placeholder+"}",
				"request URL should contain {%s}", placeholder)
		}
	}
}
