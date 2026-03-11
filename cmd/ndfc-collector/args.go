// Package main is the entry point for the NDFC collector.
package main

import (
	"ndfc-collector/pkg/config"

	"github.com/alecthomas/kong"
)

var version = "(dev)"

// Args are command line parameters.
type Args struct {
	URL               string            `kong:"--url,env='NDFC_URL',help='NDFC hostname or IP address'"`
	Username          string            `kong:"--username,env='NDFC_USERNAME',help='NDFC username'"`
	Password          string            `kong:"--password,env='NDFC_PASSWORD',help='NDFC password'"`
	Output            string            `kong:"-o,default='ndfc-collection-data.zip',help='Output file'"`
	ConfigFile        string            `kong:"-c,--config,help='Path to YAML configuration file'"`
	RequestRetryCount int               `kong:"--request-retry-count,default='3',help='Times to retry a failed request'"`
	RetryDelay        int               `kong:"--retry-delay,default='10',help='Seconds to wait before retry'"`
	BatchSize         int               `kong:"--batch-size,default='7',help='Max request to send in parallel'"`
	PageSize          int               `kong:"--page-size,default='1000',help='Object per page for large datasets'"`
	Confirm           bool              `kong:"-y,help='Skip confirmation'"`
	Verbose           bool              `kong:"-v,--verbose,help='Enable verbose (debug level) logging'"`
	Endpoint          string            `kong:"--endpoint,default='all',help='Collect a single endpoint'"`
	Query             map[string]string `kong:"-q,help='Query(s) to filter single endpoint query'"`
	Version           bool              `kong:"--version,help='Show version'"`
}

// readArgs collects the CLI args and returns a config.Config.
func readArgs() (*config.Config, error) {
	var args Args
	_ = kong.Parse(&args)

	if args.Version {
		println("NDFC Collector", version)
		return nil, nil
	}

	var cfg *config.Config
	if args.ConfigFile != "" {
		var err error
		cfg, err = config.ParseConfig(args.ConfigFile)
		if err != nil {
			return nil, err
		}
	} else {
		c := config.New()
		cfg = &c
		cfg.URL = args.URL
		cfg.Output = args.Output
		cfg.Username = args.Username
		cfg.Password = args.Password
		cfg.RequestRetryCount = args.RequestRetryCount
		cfg.RetryDelay = args.RetryDelay
		cfg.BatchSize = args.BatchSize
		cfg.PageSize = args.PageSize
		cfg.Confirm = args.Confirm
		cfg.Verbose = args.Verbose
		cfg.Endpoint = args.Endpoint
		cfg.Query = args.Query
	}

	if err := cfg.NormalizeAndPrompt(); err != nil {
		return nil, err
	}

	return cfg, nil
}
