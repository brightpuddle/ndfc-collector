package req

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequests(t *testing.T) {
	reqs, err := GetRequests()
	assert.NoError(t, err)
	assert.NotEmpty(t, reqs)

	// Verify the expected endpoints are present
	expectedCount := 3
	assert.Equal(t, expectedCount, len(reqs), "GetRequests() returned %d requests, expected %d", len(reqs), expectedCount)

	// Verify each request has a URL
	for i, req := range reqs {
		assert.NotEmpty(t, req.URL, "Request %d has empty URL", i)
	}
}

func TestBaseURL(t *testing.T) {
	expected := "/appcenter/cisco/ndfc/api/v1"
	assert.Equal(t, expected, BaseURL)
}
