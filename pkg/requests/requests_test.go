// SPDX-License-Identifier: Apache-2.0

// Copyright 2026 Cisco Systems, Inc. and their affiliates

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
