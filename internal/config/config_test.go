package config

import (
	"errors"
	"testing"
	"time"
)

func TestConfigWithOverridesValidatesRequiredValues(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	_, err = cfg.WithOverrides("", 0)
	if !errors.Is(err, ErrAthenaBaseURLRequired) {
		t.Fatalf("WithOverrides() error = %v, want %v", err, ErrAthenaBaseURLRequired)
	}
}

func TestConfigWithOverridesAcceptsFlagOverrides(t *testing.T) {
	cfg := Config{}

	resolved, err := cfg.WithOverrides("http://127.0.0.1:18080", 3*time.Second)
	if err != nil {
		t.Fatalf("WithOverrides() error = %v", err)
	}
	if resolved.AthenaBaseURL != "http://127.0.0.1:18080" {
		t.Fatalf("AthenaBaseURL = %q, want %q", resolved.AthenaBaseURL, "http://127.0.0.1:18080")
	}
	if resolved.HTTPTimeout != 3*time.Second {
		t.Fatalf("HTTPTimeout = %s, want %s", resolved.HTTPTimeout, 3*time.Second)
	}
}

func TestConfigWithOverridesRejectsInvalidValues(t *testing.T) {
	testCases := []struct {
		name    string
		cfg     Config
		baseURL string
		timeout time.Duration
	}{
		{
			name:    "invalid base url",
			cfg:     Config{HTTPTimeout: 5 * time.Second},
			baseURL: "://bad",
			timeout: 5 * time.Second,
		},
		{
			name:    "missing host",
			cfg:     Config{HTTPTimeout: 5 * time.Second},
			baseURL: "http://",
			timeout: 5 * time.Second,
		},
		{
			name:    "negative timeout override on existing config",
			cfg:     Config{AthenaBaseURL: "http://127.0.0.1:18080", HTTPTimeout: 5 * time.Second},
			timeout: -1 * time.Second,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := testCase.cfg.WithOverrides(testCase.baseURL, testCase.timeout)
			if err == nil {
				t.Fatal("WithOverrides() error = nil, want validation error")
			}
		})
	}
}
