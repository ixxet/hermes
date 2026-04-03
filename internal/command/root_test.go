package command

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ixxet/hermes/internal/config"
	"github.com/ixxet/hermes/internal/ops"
)

type stubOccupancyAsker struct {
	answer ops.OccupancyAnswer
	err    error
}

func (s stubOccupancyAsker) AskOccupancy(context.Context, string) (ops.OccupancyAnswer, error) {
	if s.err != nil {
		return ops.OccupancyAnswer{}, s.err
	}

	return s.answer, nil
}

func TestAskOccupancyCommandRequiresFacility(t *testing.T) {
	var stdout bytes.Buffer
	err := Execute([]string{"ask", "occupancy"}, Dependencies{
		Stdout: &stdout,
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				AthenaBaseURL: "http://127.0.0.1:18080",
				HTTPTimeout:   5 * time.Second,
			}, nil
		},
		NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
			return stubOccupancyAsker{}, nil
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want missing facility error")
	}
	if !strings.Contains(err.Error(), "required flag(s) \"facility\" not set") {
		t.Fatalf("Execute() error = %q, want missing facility error", err.Error())
	}
}

func TestAskOccupancyCommandOutputsStableJSONShape(t *testing.T) {
	var stdout bytes.Buffer

	err := Execute([]string{"ask", "occupancy", "--facility", "ashtonbee"}, Dependencies{
		Stdout: &stdout,
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				AthenaBaseURL: "http://127.0.0.1:18080",
				HTTPTimeout:   5 * time.Second,
			}, nil
		},
		NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
			return stubOccupancyAsker{
				answer: ops.OccupancyAnswer{
					FacilityID:    "ashtonbee",
					CurrentCount:  9,
					ObservedAt:    "2026-04-02T16:00:00Z",
					SourceService: "athena",
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body := stdout.String()
	if !strings.Contains(body, `"facility_id":"ashtonbee"`) {
		t.Fatalf("body = %q, want facility_id", body)
	}
	if !strings.Contains(body, `"current_count":9`) {
		t.Fatalf("body = %q, want current_count", body)
	}
	if !strings.Contains(body, `"source_service":"athena"`) {
		t.Fatalf("body = %q, want source_service", body)
	}
}

func TestAskOccupancyCommandSupportsTextOutputAndClearErrors(t *testing.T) {
	t.Run("text output", func(t *testing.T) {
		var stdout bytes.Buffer
		err := Execute([]string{"ask", "occupancy", "--facility", "ashtonbee", "--format", "text"}, Dependencies{
			Stdout: &stdout,
			LoadConfig: func() (config.Config, error) {
				return config.Config{
					AthenaBaseURL: "http://127.0.0.1:18080",
					HTTPTimeout:   5 * time.Second,
				}, nil
			},
			NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				return stubOccupancyAsker{
					answer: ops.OccupancyAnswer{
						FacilityID:    "ashtonbee",
						CurrentCount:  9,
						ObservedAt:    "2026-04-02T16:00:00Z",
						SourceService: "athena",
					},
				}, nil
			},
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if got := stdout.String(); got != "facility_id=ashtonbee current_count=9 observed_at=2026-04-02T16:00:00Z source_service=athena\n" {
			t.Fatalf("stdout = %q", got)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		err := Execute([]string{"ask", "occupancy", "--facility", "ashtonbee", "--format", "yaml"}, Dependencies{
			LoadConfig: func() (config.Config, error) {
				return config.Config{
					AthenaBaseURL: "http://127.0.0.1:18080",
					HTTPTimeout:   5 * time.Second,
				}, nil
			},
			NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				return stubOccupancyAsker{}, nil
			},
		})
		if err == nil {
			t.Fatal("Execute() error = nil, want invalid format error")
		}
		if !strings.Contains(err.Error(), "format must be one of: json, text") {
			t.Fatalf("Execute() error = %q, want invalid format error", err.Error())
		}
	})

	t.Run("upstream failure", func(t *testing.T) {
		upstreamErr := errors.New("athena occupancy request failed with status 500: read path unavailable")
		err := Execute([]string{"ask", "occupancy", "--facility", "ashtonbee"}, Dependencies{
			LoadConfig: func() (config.Config, error) {
				return config.Config{
					AthenaBaseURL: "http://127.0.0.1:18080",
					HTTPTimeout:   5 * time.Second,
				}, nil
			},
			NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				return stubOccupancyAsker{err: upstreamErr}, nil
			},
		})
		if !errors.Is(err, upstreamErr) {
			t.Fatalf("Execute() error = %v, want %v", err, upstreamErr)
		}
	})
}
