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
