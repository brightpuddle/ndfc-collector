package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_DefaultOutputFile(t *testing.T) {
	cfg := New()
	assert.Equal(t, defaultOutputFile, cfg.Output)
	assert.NotEmpty(t, cfg.Output)
}

func TestNew_DefaultNumerics(t *testing.T) {
	cfg := New()
	assert.Equal(t, 3, cfg.RequestRetryCount)
	assert.Equal(t, 10, cfg.RetryDelay)
	assert.Equal(t, 7, cfg.BatchSize)
	assert.Equal(t, 1000, cfg.PageSize)
	assert.Equal(t, "all", cfg.Endpoint)
}

func TestParseConfig_PreservesDefaultOutputWhenAbsent(t *testing.T) {
	data := "url: https://ndfc.example.com\nusername: admin\n"
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))

	cfg, err := ParseConfig(path)
	require.NoError(t, err)
	assert.Equal(t, defaultOutputFile, cfg.Output,
		"output should fall back to the default when not set in config file")
}

func TestParseConfig_ExplicitOutputOverridesDefault(t *testing.T) {
	data := "output: my-collection.zip\n"
	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))

	cfg, err := ParseConfig(path)
	require.NoError(t, err)
	assert.Equal(t, "my-collection.zip", cfg.Output)
}

func TestParseConfig_MissingFile(t *testing.T) {
	_, err := ParseConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
}

func TestNormalizeURL_StripsHTTPS(t *testing.T) {
	assert.Equal(t, "ndfc.example.com", normalizeURL("https://ndfc.example.com"))
}

func TestNormalizeURL_StripsHTTP(t *testing.T) {
	assert.Equal(t, "ndfc.example.com", normalizeURL("http://ndfc.example.com"))
}

func TestNormalizeURL_NoScheme(t *testing.T) {
	assert.Equal(t, "ndfc.example.com", normalizeURL("ndfc.example.com"))
}
