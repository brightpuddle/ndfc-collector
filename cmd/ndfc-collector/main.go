package main

import (
	"fmt"
	"os"
	"path/filepath"

	"ndfc-collector/pkg/archive"
	"ndfc-collector/pkg/cli"
	"ndfc-collector/pkg/req"

	"github.com/brightpuddle/gobits/log"
)

func pause(msg string) {
	fmt.Println(msg)
	var throwaway string
	fmt.Scanln(&throwaway)
}

func main() {
	cfg, err := readArgs()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading configuration.")
	}
	if cfg == nil {
		return // version flag
	}

	if cfg.Verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// Initialize NDFC HTTP client
	client, err := cli.GetClient(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing NDFC client.")
	}

	// Create results archive
	outputFile := cfg.Output
	arc, err := archive.NewWriter(outputFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating archive file: %s.", outputFile)
	}

	// Initiate requests
	reqs, err := req.GetRequests()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading requests.")
	}

	// Allow overriding in-built queries with a single endpoint query
	if cfg.Endpoint != "all" {
		reqs = []req.Request{{
			URL:   cfg.Endpoint,
			Query: cfg.Query,
		}}
	}

	// Batch and fetch queries in parallel
	collectErr := collectFabric(client, arc, reqs, cfg)

	arc.Close()
	log.Info().Msg("====== Complete ======")

	path, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot read current working directory")
	}
	outPath := filepath.Join(path, outputFile)

	if collectErr != nil {
		log.Warn().Err(collectErr).Msg("some data could not be fetched")
		log.Info().Msgf("Available data written to %s.", outPath)
	} else {
		log.Info().Msg("Collection complete.")
		log.Info().Msgf("Please provide %s to Cisco Services for further analysis.", outPath)
	}
	if !cfg.Confirm {
		pause("Press enter to exit.")
	}
}
