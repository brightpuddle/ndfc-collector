package req

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequests(t *testing.T) {
	reqs, err := GetRequests()
	assert.NoError(t, err)
	assert.NotEmpty(t, reqs)

	// Verify each request has a URL
	for i, req := range reqs {
		assert.NotEmpty(t, req.URL, "Request %d has empty URL", i)
	}

	// Verify root requests (no DependsOn) are present
	var roots []Request
	for _, r := range reqs {
		if r.DependsOn == "" {
			roots = append(roots, r)
		}
	}
	assert.NotEmpty(t, roots, "Expected at least one root request")

	// Verify dependent requests reference a URL that exists in the list
	byURL := make(map[string]bool, len(reqs))
	for _, r := range reqs {
		byURL[r.URL] = true
	}
	for _, r := range reqs {
		if r.DependsOn != "" {
			assert.True(t, byURL[r.DependsOn],
				"Request %q depends on %q which is not in the request list", r.URL, r.DependsOn)
			assert.True(t, strings.Contains(r.URL, "{"),
				"Dependent request %q should contain a {placeholder}", r.URL)
		}
	}
}

func TestBaseURL(t *testing.T) {
	expected := "/appcenter/cisco/ndfc/api/v1"
	assert.Equal(t, expected, BaseURL)
}
