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

package archive

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWriter(t *testing.T) {
	tmpfile := "/tmp/test-archive.zip"
	defer os.Remove(tmpfile)

	arc, err := NewWriter(tmpfile)
	assert.NoError(t, err)

	// Add a test file
	testContent := []byte("test content")
	err = arc.Add("test.json", testContent)
	assert.NoError(t, err)

	// Close the archive
	err = arc.Close()
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(tmpfile)
	assert.NoError(t, err)
}
