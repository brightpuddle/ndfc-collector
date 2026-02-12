package req

import (
	"testing"
)

func TestGetRequests(t *testing.T) {
	reqs, err := GetRequests()
	if err != nil {
		t.Fatalf("GetRequests() error = %v", err)
	}

	if len(reqs) == 0 {
		t.Fatal("GetRequests() returned empty list")
	}

	// Verify the expected endpoints are present
	expectedCount := 3
	if len(reqs) != expectedCount {
		t.Errorf("GetRequests() returned %d requests, expected %d", len(reqs), expectedCount)
	}

	// Verify each request has a URL
	for i, req := range reqs {
		if req.URL == "" {
			t.Errorf("Request %d has empty URL", i)
		}
	}
}

func TestBaseURL(t *testing.T) {
	expected := "/appcenter/cisco/ndfc/api/v1"
	if BaseURL != expected {
		t.Errorf("BaseURL = %q, want %q", BaseURL, expected)
	}
}
