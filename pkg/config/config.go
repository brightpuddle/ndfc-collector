// Package config provides configuration handling for the NDFC collector.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/brightpuddle/gobits/errors"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

const defaultOutputFile = "ndfc-collection-data.zip"

// Config holds all settings for the NDFC collector.
type Config struct {
	URL               string            `yaml:"url"`
	Output            string            `yaml:"output"`
	Username          string            `yaml:"username"`
	Password          string            `yaml:"password"`
	RequestRetryCount int               `yaml:"request_retry_count"`
	RetryDelay        int               `yaml:"retry_delay"`
	BatchSize         int               `yaml:"batch_size"`
	PageSize          int               `yaml:"page_size"`
	Confirm           bool              `yaml:"confirm"`
	Verbose           bool              `yaml:"verbose"`
	Endpoint          string            `yaml:"endpoint"`
	Query             map[string]string `yaml:"query"`
}

// New returns a Config with default values.
func New() Config {
	return Config{
		Output:            defaultOutputFile,
		RequestRetryCount: 3,
		RetryDelay:        10,
		BatchSize:         7,
		PageSize:          1000,
		Endpoint:          "all",
	}
}

// ParseConfig reads and parses a YAML configuration file.
// Fields absent from the file retain their default values from New().
func ParseConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(fmt.Errorf("failed to read config file: %w", err))
	}

	cfg := New()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.WithStack(fmt.Errorf("failed to parse config file: %w", err))
	}

	return &cfg, nil
}

// NormalizeAndPrompt fills missing required values interactively and normalizes inputs.
func (c *Config) NormalizeAndPrompt() error {
	if c.URL == "" {
		c.URL = input("NDFC URL:")
	}
	c.URL = normalizeURL(c.URL)

	if c.Username == "" {
		c.Username = input("NDFC username:")
	}
	if c.Password == "" {
		c.Password = inputPassword("NDFC password:")
	}

	if c.URL == "" {
		return fmt.Errorf("url is required")
	}
	return nil
}

// input collects CLI input.
func input(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s ", prompt)
	s, _ := reader.ReadString('\n')
	return strings.Trim(s, "\r\n")
}

func inputPassword(prompt string) string {
	fmt.Print(prompt + " ")
	pwd, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return string(pwd)
}

func normalizeURL(url string) string {
	url, _ = strings.CutPrefix(url, "http://")
	url, _ = strings.CutPrefix(url, "https://")
	return url
}
