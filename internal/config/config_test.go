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
		baseURL string
		timeout time.Duration
	}{
		{
			name:    "invalid base url",
			baseURL: "://bad",
			timeout: 5 * time.Second,
		},
		{
			name:    "missing host",
			baseURL: "http://",
			timeout: 5 * time.Second,
		},
		{
			name:    "non positive timeout",
			baseURL: "http://127.0.0.1:18080",
			timeout: -1 * time.Second,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Config{}.WithOverrides(testCase.baseURL, testCase.timeout)
			if err == nil {
				t.Fatal("WithOverrides() error = nil, want validation error")
			}
		})
	}
}
