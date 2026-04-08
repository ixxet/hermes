package command

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ixxet/hermes/internal/athena"
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

type observedLogEntry struct {
	Event          string `json:"event"`
	Component      string `json:"component"`
	Tracer         int    `json:"tracer"`
	Question       string `json:"question"`
	RequestID      string `json:"request_id"`
	Facility       string `json:"facility"`
	Upstream       string `json:"upstream"`
	Outcome        string `json:"outcome"`
	DurationMS     *int64 `json:"duration_ms,omitempty"`
	Version        string `json:"version"`
	OccupancyCount *int   `json:"occupancy_count,omitempty"`
	UpstreamStatus *int   `json:"upstream_status,omitempty"`
	ErrorKind      string `json:"error_kind,omitempty"`
	Error          string `json:"error,omitempty"`
}

type recordingOccupancyReader struct {
	called bool
}

func (r *recordingOccupancyReader) CurrentOccupancy(context.Context, string) (athena.OccupancySnapshot, error) {
	r.called = true
	return athena.OccupancySnapshot{}, nil
}

func TestAskOccupancyCommandRequiresFacility(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	loadCalled := false
	askCalled := false

	err := Execute([]string{"ask", "occupancy"}, Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Version:      "v0.1.1",
		Now:          fixedClock(time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 9, 0, 0, 25*int(time.Millisecond), time.UTC)),
		NewRequestID: func() string { return "req-missing-facility" },
		LoadConfig: func() (config.Config, error) {
			loadCalled = true
			return config.Config{
				AthenaBaseURL: "http://127.0.0.1:18080",
				HTTPTimeout:   5 * time.Second,
			}, nil
		},
		NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
			askCalled = true
			return stubOccupancyAsker{}, nil
		},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want missing facility error")
	}
	if !strings.Contains(err.Error(), "required flag(s) \"facility\" not set") {
		t.Fatalf("Execute() error = %q, want missing facility error", err.Error())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty output", stdout.String())
	}
	if loadCalled {
		t.Fatal("LoadConfig() was called for missing required flag")
	}
	if askCalled {
		t.Fatal("NewOccupancyAsker() was called for missing required flag")
	}

	entries := decodeLogEntries(t, stderr.String())
	assertLogSequence(t, entries, []string{"request-start", "request-failed"})
	if entries[1].ErrorKind != "validation_error" {
		t.Fatalf("failure error_kind = %q, want validation_error", entries[1].ErrorKind)
	}
	if entries[1].Facility != "" {
		t.Fatalf("failure facility = %q, want empty", entries[1].Facility)
	}
}

func TestAskOccupancyCommandOutputsStableJSONShapeAndStructuredSuccessLog(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Execute([]string{"ask", "occupancy", "--facility", "ashtonbee"}, Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Version:      "v0.1.1",
		Now:          fixedClock(time.Date(2026, 4, 8, 9, 10, 0, 0, time.UTC), time.Date(2026, 4, 8, 9, 10, 0, 25*int(time.Millisecond), time.UTC)),
		NewRequestID: func() string { return "req-success" },
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
	if got := stdout.String(); got != "{\"facility_id\":\"ashtonbee\",\"current_count\":9,\"observed_at\":\"2026-04-02T16:00:00Z\",\"source_service\":\"athena\"}\n" {
		t.Fatalf("stdout = %q", got)
	}

	entries := decodeLogEntries(t, stderr.String())
	assertLogSequence(t, entries, []string{"request-start", "request-complete"})
	if entries[0].Outcome != "started" {
		t.Fatalf("start outcome = %q, want started", entries[0].Outcome)
	}
	if entries[1].Outcome != "success" {
		t.Fatalf("completion outcome = %q, want success", entries[1].Outcome)
	}
	if entries[1].OccupancyCount == nil || *entries[1].OccupancyCount != 9 {
		t.Fatalf("completion occupancy_count = %#v, want 9", entries[1].OccupancyCount)
	}
	if entries[1].DurationMS == nil || *entries[1].DurationMS != 25 {
		t.Fatalf("completion duration_ms = %#v, want 25", entries[1].DurationMS)
	}
	assertBaseFields(t, entries, "req-success", "ashtonbee", "v0.1.1")
}

func TestAskOccupancyCommandSupportsTextOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Execute([]string{"ask", "occupancy", "--facility", "ashtonbee", "--format", "text"}, Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Version:      "v0.1.1",
		Now:          fixedClock(time.Date(2026, 4, 8, 9, 20, 0, 0, time.UTC), time.Date(2026, 4, 8, 9, 20, 0, 25*int(time.Millisecond), time.UTC)),
		NewRequestID: func() string { return "req-text" },
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
	entries := decodeLogEntries(t, stderr.String())
	assertLogSequence(t, entries, []string{"request-start", "request-complete"})
}

func TestAskOccupancyCommandHelpDoesNotEmitObservabilityLogs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	loadCalled := false
	askCalled := false

	err := Execute([]string{"ask", "occupancy", "--help"}, Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Version:      "v0.1.1",
		Now:          fixedClock(time.Date(2026, 4, 8, 9, 25, 0, 0, time.UTC)),
		NewRequestID: func() string { return "req-help" },
		LoadConfig: func() (config.Config, error) {
			loadCalled = true
			return config.Config{}, nil
		},
		NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
			askCalled = true
			return stubOccupancyAsker{}, nil
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Read current facility occupancy from ATHENA") {
		t.Fatalf("stdout = %q, want occupancy help output", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want no observability output", stderr.String())
	}
	if loadCalled {
		t.Fatal("LoadConfig() was called for help output")
	}
	if askCalled {
		t.Fatal("NewOccupancyAsker() was called for help output")
	}
}

func TestAskOccupancyCommandStructuredFailureLogs(t *testing.T) {
	invalidFormatLoadCalled := false
	invalidFormatAskCalled := false
	configValidationAskCalled := false

	testCases := []struct {
		name              string
		args              []string
		loadConfig        func() (config.Config, error)
		newOccupancyAsker func(config.Config) (OccupancyAsker, error)
		wantErr           error
		wantKind          string
		wantStatus        *int
		wantFacility      string
		assertExtra       func(*testing.T)
	}{
		{
			name: "invalid format",
			args: []string{"ask", "occupancy", "--facility", "ashtonbee", "--format", "yaml"},
			loadConfig: func() (config.Config, error) {
				invalidFormatLoadCalled = true
				return config.Config{}, nil
			},
			newOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				invalidFormatAskCalled = true
				return nil, nil
			},
			wantErr:      ErrInvalidFormat,
			wantKind:     "validation_error",
			wantFacility: "ashtonbee",
			assertExtra: func(t *testing.T) {
				if invalidFormatLoadCalled {
					t.Fatal("LoadConfig() called for invalid format")
				}
				if invalidFormatAskCalled {
					t.Fatal("NewOccupancyAsker() called for invalid format")
				}
			},
		},
		{
			name: "upstream status",
			args: []string{"ask", "occupancy", "--facility", "ashtonbee"},
			loadConfig: func() (config.Config, error) {
				return config.Config{
					AthenaBaseURL: "http://127.0.0.1:18080",
					HTTPTimeout:   5 * time.Second,
				}, nil
			},
			newOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				return stubOccupancyAsker{
					err: &athena.UpstreamStatusError{StatusCode: 500, Message: "read path unavailable"},
				}, nil
			},
			wantKind:     "upstream_error",
			wantStatus:   intPointer(500),
			wantFacility: "ashtonbee",
		},
		{
			name: "timeout",
			args: []string{"ask", "occupancy", "--facility", "ashtonbee"},
			loadConfig: func() (config.Config, error) {
				return config.Config{
					AthenaBaseURL: "http://127.0.0.1:18080",
					HTTPTimeout:   5 * time.Second,
				}, nil
			},
			newOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				return stubOccupancyAsker{
					err: fmt.Errorf("%w: context deadline exceeded", athena.ErrRequestTimeout),
				}, nil
			},
			wantKind:     "upstream_timeout",
			wantFacility: "ashtonbee",
		},
		{
			name: "malformed upstream body",
			args: []string{"ask", "occupancy", "--facility", "ashtonbee"},
			loadConfig: func() (config.Config, error) {
				return config.Config{
					AthenaBaseURL: "http://127.0.0.1:18080",
					HTTPTimeout:   5 * time.Second,
				}, nil
			},
			newOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				return stubOccupancyAsker{
					err: fmt.Errorf("%w: unexpected EOF", athena.ErrMalformedResponse),
				}, nil
			},
			wantKind:     "decode_error",
			wantFacility: "ashtonbee",
		},
		{
			name: "config validation failure",
			args: []string{"ask", "occupancy", "--facility", "ashtonbee", "--timeout", "-1s"},
			loadConfig: func() (config.Config, error) {
				return config.Config{
					AthenaBaseURL: "http://127.0.0.1:18080",
					HTTPTimeout:   5 * time.Second,
				}, nil
			},
			newOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
				configValidationAskCalled = true
				return nil, nil
			},
			wantErr:      config.ErrHTTPTimeoutInvalid,
			wantKind:     "config_error",
			wantFacility: "ashtonbee",
			assertExtra: func(t *testing.T) {
				if configValidationAskCalled {
					t.Fatal("NewOccupancyAsker() called for config validation failure")
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			invalidFormatLoadCalled = false
			invalidFormatAskCalled = false
			configValidationAskCalled = false

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			err := Execute(testCase.args, Dependencies{
				Stdout:            &stdout,
				Stderr:            &stderr,
				Version:           "v0.1.1",
				Now:               fixedClock(time.Date(2026, 4, 8, 10, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 10, 0, 0, 25*int(time.Millisecond), time.UTC)),
				NewRequestID:      func() string { return "req-failure" },
				LoadConfig:        testCase.loadConfig,
				NewOccupancyAsker: testCase.newOccupancyAsker,
			})
			if err == nil {
				t.Fatal("Execute() error = nil, want failure")
			}
			if testCase.wantErr != nil && !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Execute() error = %v, want %v", err, testCase.wantErr)
			}
			if stdout.String() != "" {
				t.Fatalf("stdout = %q, want empty output", stdout.String())
			}

			entries := decodeLogEntries(t, stderr.String())
			assertLogSequence(t, entries, []string{"request-start", "request-failed"})
			assertBaseFields(t, entries, "req-failure", testCase.wantFacility, "v0.1.1")
			if entries[1].ErrorKind != testCase.wantKind {
				t.Fatalf("failure error_kind = %q, want %q", entries[1].ErrorKind, testCase.wantKind)
			}
			if entries[1].DurationMS == nil || *entries[1].DurationMS != 25 {
				t.Fatalf("failure duration_ms = %#v, want 25", entries[1].DurationMS)
			}
			if !reflect.DeepEqual(entries[1].UpstreamStatus, testCase.wantStatus) {
				t.Fatalf("failure upstream_status = %#v, want %#v", entries[1].UpstreamStatus, testCase.wantStatus)
			}
			if strings.TrimSpace(entries[1].Error) == "" {
				t.Fatal("failure error field is empty")
			}
			if testCase.assertExtra != nil {
				testCase.assertExtra(t)
			}
		})
	}
}

func TestAskOccupancyCommandBlankFacilityStaysValidationOnly(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	reader := &recordingOccupancyReader{}

	err := Execute([]string{"ask", "occupancy", "--facility", "   "}, Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Version:      "v0.1.1",
		Now:          fixedClock(time.Date(2026, 4, 8, 10, 30, 0, 0, time.UTC), time.Date(2026, 4, 8, 10, 30, 0, 25*int(time.Millisecond), time.UTC)),
		NewRequestID: func() string { return "req-blank-facility" },
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				AthenaBaseURL: "http://127.0.0.1:18080",
				HTTPTimeout:   5 * time.Second,
			}, nil
		},
		NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
			return ops.NewOccupancyService(reader), nil
		},
	})
	if !errors.Is(err, ops.ErrFacilityRequired) {
		t.Fatalf("Execute() error = %v, want %v", err, ops.ErrFacilityRequired)
	}
	if reader.called {
		t.Fatal("CurrentOccupancy() was called for blank facility")
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty output", stdout.String())
	}

	entries := decodeLogEntries(t, stderr.String())
	assertLogSequence(t, entries, []string{"request-start", "request-failed"})
	assertBaseFields(t, entries, "req-blank-facility", "", "v0.1.1")
	if entries[1].ErrorKind != "validation_error" {
		t.Fatalf("failure error_kind = %q, want validation_error", entries[1].ErrorKind)
	}
}

func TestAskOccupancyCommandObservabilityRemainsLowNoiseAcrossRepeatedRuns(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		first := runOccupancy(t, []string{"ask", "occupancy", "--facility", "ashtonbee"}, "req-success-1", stubOccupancyAsker{
			answer: ops.OccupancyAnswer{
				FacilityID:    "ashtonbee",
				CurrentCount:  9,
				ObservedAt:    "2026-04-02T16:00:00Z",
				SourceService: "athena",
			},
		})
		second := runOccupancy(t, []string{"ask", "occupancy", "--facility", "ashtonbee"}, "req-success-2", stubOccupancyAsker{
			answer: ops.OccupancyAnswer{
				FacilityID:    "ashtonbee",
				CurrentCount:  9,
				ObservedAt:    "2026-04-02T16:00:00Z",
				SourceService: "athena",
			},
		})

		if !reflect.DeepEqual(normalizeEntries(first), normalizeEntries(second)) {
			t.Fatalf("normalized logs differ:\nfirst=%#v\nsecond=%#v", normalizeEntries(first), normalizeEntries(second))
		}
	})

	t.Run("failure", func(t *testing.T) {
		first := runOccupancy(t, []string{"ask", "occupancy", "--facility", "ashtonbee"}, "req-failure-1", stubOccupancyAsker{
			err: &athena.UpstreamStatusError{StatusCode: 500, Message: "read path unavailable"},
		})
		second := runOccupancy(t, []string{"ask", "occupancy", "--facility", "ashtonbee"}, "req-failure-2", stubOccupancyAsker{
			err: &athena.UpstreamStatusError{StatusCode: 500, Message: "read path unavailable"},
		})

		if !reflect.DeepEqual(normalizeEntries(first), normalizeEntries(second)) {
			t.Fatalf("normalized logs differ:\nfirst=%#v\nsecond=%#v", normalizeEntries(first), normalizeEntries(second))
		}
	})
}

func runOccupancy(t *testing.T, args []string, requestID string, asker stubOccupancyAsker) []observedLogEntry {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	_ = Execute(args, Dependencies{
		Stdout:       &stdout,
		Stderr:       &stderr,
		Version:      "v0.1.1",
		Now:          fixedClock(time.Date(2026, 4, 8, 11, 0, 0, 0, time.UTC), time.Date(2026, 4, 8, 11, 0, 0, 25*int(time.Millisecond), time.UTC)),
		NewRequestID: func() string { return requestID },
		LoadConfig: func() (config.Config, error) {
			return config.Config{
				AthenaBaseURL: "http://127.0.0.1:18080",
				HTTPTimeout:   5 * time.Second,
			}, nil
		},
		NewOccupancyAsker: func(config.Config) (OccupancyAsker, error) {
			return asker, nil
		},
	})

	return decodeLogEntries(t, stderr.String())
}

func decodeLogEntries(t *testing.T, raw string) []observedLogEntry {
	t.Helper()

	var entries []observedLogEntry
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		var entry observedLogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("json.Unmarshal(log) error = %v", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner.Err() = %v", err)
	}

	return entries
}

func assertBaseFields(t *testing.T, entries []observedLogEntry, requestID, facility, version string) {
	t.Helper()

	for _, entry := range entries {
		if entry.Component != "hermes" {
			t.Fatalf("component = %q, want hermes", entry.Component)
		}
		if entry.Tracer != 14 {
			t.Fatalf("tracer = %d, want 14", entry.Tracer)
		}
		if entry.Question != "occupancy" {
			t.Fatalf("question = %q, want occupancy", entry.Question)
		}
		if entry.RequestID != requestID {
			t.Fatalf("request_id = %q, want %q", entry.RequestID, requestID)
		}
		if entry.Facility != facility {
			t.Fatalf("facility = %q, want %q", entry.Facility, facility)
		}
		if entry.Upstream != "athena" {
			t.Fatalf("upstream = %q, want athena", entry.Upstream)
		}
		if entry.Version != version {
			t.Fatalf("version = %q, want %q", entry.Version, version)
		}
	}
}

func assertLogSequence(t *testing.T, entries []observedLogEntry, want []string) {
	t.Helper()

	if len(entries) != len(want) {
		t.Fatalf("len(logs) = %d, want %d; logs=%v", len(entries), len(want), entries)
	}
	for index, event := range want {
		if entries[index].Event != event {
			t.Fatalf("logs[%d].event = %q, want %q", index, entries[index].Event, event)
		}
	}
}

func normalizeEntries(entries []observedLogEntry) []observedLogEntry {
	normalized := make([]observedLogEntry, len(entries))
	copy(normalized, entries)
	for index := range normalized {
		normalized[index].RequestID = "<request>"
	}
	return normalized
}

func fixedClock(times ...time.Time) func() time.Time {
	index := 0
	return func() time.Time {
		if len(times) == 0 {
			return time.Time{}
		}
		if index >= len(times) {
			return times[len(times)-1]
		}
		current := times[index]
		index++
		return current
	}
}

func intPointer(value int) *int {
	return &value
}
