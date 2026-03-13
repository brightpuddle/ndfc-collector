package cli

import (
	"fmt"
	"path"
	"strings"
	"time"

	"ndfc-collector/pkg/archive"
	"ndfc-collector/pkg/config"
	"ndfc-collector/pkg/ndfc"
	"ndfc-collector/pkg/requests"

	"github.com/brightpuddle/gobits/errors"
	"github.com/brightpuddle/gobits/log"

	"github.com/tidwall/gjson"
)

// GetClient creates an NDFC host client
func GetClient(cfg *config.Config) (ndfc.Client, error) {
	// Sanitize username against quotes
	cfg.Password = strings.ReplaceAll(cfg.Password, "\"", "\\\"")
	client, err := ndfc.NewClient(
		cfg.URL, cfg.Username, cfg.Password,
		ndfc.RequestTimeout(600),
	)
	if err != nil {
		return ndfc.Client{}, errors.WithStack(fmt.Errorf("failed to create NDFC client: %v", err))
	}

	logger := log.New()

	// Authenticate
	logger.Info().Str("host", cfg.URL).Msg("NDFC host")
	logger.Info().Str("user", cfg.Username).Msg("NDFC username")
	logger.Info().Msg("Authenticating to NDFC...")
	if err := client.Login(); err != nil {
		return ndfc.Client{}, errors.WithStack(
			fmt.Errorf("cannot authenticate to NDFC at %s: %v", cfg.URL, err),
		)
	}
	return client, nil
}

func fetchWithRetry(
	client ndfc.Client,
	path string,
	cfg *config.Config,
	mods []func(*ndfc.Req),
) (gjson.Result, error) {
	res, err := client.Get(path, mods...)

	logger := log.New()

	// Retry for requestRetryCount times
	for retries := 0; err != nil && retries < cfg.RequestRetryCount; retries++ {
		logger.Warn().Err(err).Msgf("request failed for %s. Retrying after %d seconds.",
			path, cfg.RetryDelay)
		time.Sleep(time.Second * time.Duration(cfg.RetryDelay))
		res, err = client.Get(path, mods...)
	}
	if err != nil {
		return res, errors.WithStack(fmt.Errorf("request failed for %s: %v", path, err))
	}
	return res, nil
}

// FetchResult fetches data via API, writes it to the provided archive, and returns the result.
func FetchResult(
	client ndfc.Client,
	request requests.Request,
	arc archive.Writer,
	cfg *config.Config,
) (gjson.Result, error) {
	// Request URLs are stored as full host-relative paths derived from the OpenAPI specs.
	fullPath := request.URL
	startTime := time.Now()

	logger := log.New()

	// Convert URL to filename using db_key when available for human-readable names.
	// db_key "inventory/switches" -> "inventory.switches.json"
	// Falls back to URL-based naming for requests without a db_key.
	filename := dbKeyToFilename(request.DBKey)
	if filename == "" {
		filename = urlToFilename(request.URL)
	}

	logger.Debug().Time("start_time", startTime).Msgf("begin: %s", filename)
	logger.Debug().Msgf("fetching %s...", filename)

	mods := []func(*ndfc.Req){}
	for k, v := range request.Query {
		mods = append(mods, ndfc.Query(k, v))
	}

	res, err := fetchWithRetry(client, fullPath, cfg, mods)
	if err != nil {
		return res, err
	}

	logger.Info().Msgf("%s complete", filename)
	if err := arc.Add(filename, []byte(res.Raw)); err != nil {
		return res, err
	}
	logger.Debug().
		TimeDiff("elapsed_time", time.Now(), startTime).
		Msgf("done: %s", filename)
	return res, nil
}

// Fetch fetches data via API and writes it to the provided archive.
func Fetch(
	client ndfc.Client,
	request requests.Request,
	arc archive.Writer,
	cfg *config.Config,
) error {
	_, err := FetchResult(client, request, arc, cfg)
	return err
}

// dbKeyToFilename converts a db_key to a filename.
// Example: "inventory/switches" -> "inventory.switches.json"
// Returns empty string if dbKey is empty.
func dbKeyToFilename(dbKey string) string {
	if dbKey == "" {
		return ""
	}
	return strings.ReplaceAll(dbKey, "/", ".") + ".json"
}

// urlToFilename converts a URL path to a filename
// Example: /lan-fabric/rest/control/fabrics -> lan-fabric.rest.control.fabrics.json
func urlToFilename(url string) string {
	// Remove leading slash
	url = strings.TrimPrefix(url, "/")

	// Replace slashes with dots
	filename := strings.ReplaceAll(url, "/", ".")

	// Get the base name without extension
	base := path.Base(filename)
	if base == "." || base == "" {
		// If no meaningful name, use the whole path
		filename = strings.ReplaceAll(url, "/", ".")
	}

	// Add .json extension
	return filename + ".json"
}
