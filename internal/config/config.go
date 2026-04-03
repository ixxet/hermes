package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultHTTPTimeout = 5 * time.Second

var ErrAthenaBaseURLRequired = errors.New("HERMES_ATHENA_BASE_URL or --athena-base-url is required")

type Config struct {
	AthenaBaseURL string
	HTTPTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		AthenaBaseURL: strings.TrimSpace(os.Getenv("HERMES_ATHENA_BASE_URL")),
		HTTPTimeout:   defaultHTTPTimeout,
	}

	if rawTimeout := strings.TrimSpace(os.Getenv("HERMES_HTTP_TIMEOUT")); rawTimeout != "" {
		timeout, err := time.ParseDuration(rawTimeout)
		if err != nil {
			return Config{}, fmt.Errorf("HERMES_HTTP_TIMEOUT is invalid: %w", err)
		}
		cfg.HTTPTimeout = timeout
	}

	return cfg, nil
}

func (c Config) WithOverrides(baseURL string, timeout time.Duration) (Config, error) {
	if trimmed := strings.TrimSpace(baseURL); trimmed != "" {
		c.AthenaBaseURL = trimmed
	}
	if timeout != 0 {
		c.HTTPTimeout = timeout
	}

	return c, c.Validate()
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.AthenaBaseURL) == "" {
		return ErrAthenaBaseURLRequired
	}

	parsed, err := url.Parse(strings.TrimSpace(c.AthenaBaseURL))
	if err != nil {
		return fmt.Errorf("athena base url is invalid: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("athena base url must include scheme and host")
	}
	if c.HTTPTimeout <= 0 {
		return errors.New("http timeout must be greater than zero")
	}

	return nil
}
