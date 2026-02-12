package cli

import (
	"fmt"
	"path"
	"strings"
	"time"

	"ndfc-collector/pkg/archive"
	"ndfc-collector/pkg/config"
	"ndfc-collector/pkg/ndfc"
	"ndfc-collector/pkg/req"

	"ndfc-collector/pkg/log"

	"github.com/tidwall/gjson"
)

// getLogger returns a logger with fabric context if a fabric name is set.
func getLogger(cfg config.FabricConfig) log.Logger {
	if cfg.GetFabricName() != "" {
		return log.WithFabric(cfg.GetFabricName())
	}
	return log.New()
}

// GetClient creates an NDFC host client
func GetClient(cfg config.FabricConfig) (ndfc.Client, error) {
	// Sanitize username against quotes
	cfg.Password = strings.ReplaceAll(cfg.Password, "\"", "\\\"")
	client, err := ndfc.NewClient(
		cfg.URL, cfg.Username, cfg.Password,
		ndfc.RequestTimeout(600),
	)
	if err != nil {
		return ndfc.Client{}, fmt.Errorf("failed to create NDFC client: %v", err)
	}

	// Get logger with fabric context
	logger := getLogger(cfg)

	// Authenticate
	logger.Info().Str("host", cfg.URL).Msg("NDFC host")
	logger.Info().Str("user", cfg.Username).Msg("NDFC username")
	logger.Info().Msg("Authenticating to NDFC...")
	if err := client.Login(); err != nil {
		return ndfc.Client{}, fmt.Errorf("cannot authenticate to NDFC at %s: %v", cfg.URL, err)
	}
	return client, nil
}

func fetchWithRetry(
	client ndfc.Client,
	path string,
	cfg config.FabricConfig,
	mods []func(*ndfc.Req),
) (gjson.Result, error) {
	res, err := client.Get(path, mods...)

	// Get logger with fabric context
	logger := getLogger(cfg)

	// Retry for requestRetryCount times
	for retries := 0; err != nil && retries < cfg.GetRequestRetryCount(); retries++ {
		logger.Warn().Err(err).Msgf("request failed for %s. Retrying after %d seconds.",
			path, cfg.GetRetryDelay())
		time.Sleep(time.Second * time.Duration(cfg.GetRetryDelay()))
		res, err = client.Get(path, mods...)
	}
	if err != nil {
		return res, fmt.Errorf("request failed for %s: %v", path, err)
	}
	return res, nil
}

// Fetch fetches data via API and writes it to the provided archive.
func Fetch(client ndfc.Client, request req.Request, arc archive.Writer, cfg config.FabricConfig) error {
	// Construct full path
	fullPath := req.BaseURL + request.URL
	startTime := time.Now()

	// Get logger with fabric context
	logger := getLogger(cfg)

	// Convert URL to filename: /lan-fabric/rest/control/fabrics -> lan-fabric.rest.control.fabrics.json
	filename := urlToFilename(request.URL)

	logger.Debug().Time("start_time", startTime).Msgf("begin: %s", filename)
	logger.Debug().Msgf("fetching %s...", filename)

	mods := []func(*ndfc.Req){}
	for k, v := range request.Query {
		mods = append(mods, ndfc.Query(k, v))
	}

	res, err := fetchWithRetry(client, fullPath, cfg, mods)
	if err != nil {
		return err
	}

	logger.Info().Msgf("%s complete", filename)
	err = arc.Add(filename, []byte(res.Raw))
	if err != nil {
		return err
	}
	logger.Debug().
		TimeDiff("elapsed_time", time.Now(), startTime).
		Msgf("done: %s", filename)
	return nil
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
