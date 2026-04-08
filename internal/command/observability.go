package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ixxet/hermes/internal/athena"
	"github.com/ixxet/hermes/internal/config"
	"github.com/ixxet/hermes/internal/ops"
)

const tracerID = 14

var requestSequence atomic.Uint64

type occupancyTrace struct {
	writer    io.Writer
	now       func() time.Time
	requestID string
	facility  string
	version   string
	startedAt time.Time
	finished  bool
}

type occupancyLogEntry struct {
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

type classifiedError struct {
	kind           string
	upstreamStatus *int
}

func newOccupancyTrace(writer io.Writer, now func() time.Time, requestID, facility, version string) *occupancyTrace {
	return &occupancyTrace{
		writer:    writer,
		now:       now,
		requestID: requestID,
		facility:  strings.TrimSpace(facility),
		version:   version,
	}
}

func nextRequestID() string {
	return fmt.Sprintf("occ-%06d", requestSequence.Add(1))
}

func occupancyInvocationFacility(args []string) (string, bool) {
	if len(args) < 2 || args[0] != "ask" || args[1] != "occupancy" {
		return "", false
	}

	var facility string
	for index := 2; index < len(args); index++ {
		switch argument := args[index]; {
		case argument == "--facility" && index+1 < len(args):
			facility = args[index+1]
			index++
		case strings.HasPrefix(argument, "--facility="):
			facility = strings.TrimPrefix(argument, "--facility=")
		}
	}

	return strings.TrimSpace(facility), true
}

func (t *occupancyTrace) Start() {
	if t == nil || t.finished {
		return
	}

	t.startedAt = t.now()
	t.log(occupancyLogEntry{
		Event:   "request-start",
		Outcome: "started",
	})
}

func (t *occupancyTrace) Complete(answer ops.OccupancyAnswer) {
	if t == nil || t.finished {
		return
	}

	t.finished = true
	durationMS := t.durationMilliseconds()
	occupancyCount := answer.CurrentCount
	if strings.TrimSpace(answer.FacilityID) != "" {
		t.facility = strings.TrimSpace(answer.FacilityID)
	}

	t.log(occupancyLogEntry{
		Event:          "request-complete",
		Outcome:        "success",
		DurationMS:     &durationMS,
		OccupancyCount: &occupancyCount,
	})
}

func (t *occupancyTrace) Fail(err error) {
	if t == nil || t.finished || err == nil {
		return
	}

	t.finished = true
	durationMS := t.durationMilliseconds()
	classified := classifyOccupancyError(err)
	t.log(occupancyLogEntry{
		Event:          "request-failed",
		Outcome:        "failed",
		DurationMS:     &durationMS,
		UpstreamStatus: classified.upstreamStatus,
		ErrorKind:      classified.kind,
		Error:          err.Error(),
	})
}

func (t *occupancyTrace) durationMilliseconds() int64 {
	if t.startedAt.IsZero() {
		return 0
	}

	duration := t.now().Sub(t.startedAt).Milliseconds()
	if duration < 0 {
		return 0
	}

	return duration
}

func (t *occupancyTrace) log(entry occupancyLogEntry) {
	if t == nil || t.writer == nil {
		return
	}

	entry.Component = "hermes"
	entry.Tracer = tracerID
	entry.Question = "occupancy"
	entry.RequestID = t.requestID
	entry.Facility = t.facility
	entry.Upstream = "athena"
	entry.Version = t.version

	encoder := json.NewEncoder(t.writer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(entry)
}

func classifyOccupancyError(err error) classifiedError {
	if err == nil {
		return classifiedError{kind: "unexpected_error"}
	}

	switch {
	case errors.Is(err, ErrInvalidFormat), errors.Is(err, ops.ErrFacilityRequired), isCobraValidationError(err):
		return classifiedError{kind: "validation_error"}
	case errors.Is(err, config.ErrAthenaBaseURLRequired),
		errors.Is(err, config.ErrAthenaBaseURLInvalid),
		errors.Is(err, config.ErrAthenaBaseURLIncomplete),
		errors.Is(err, config.ErrHTTPTimeoutInvalid),
		errors.Is(err, config.ErrHTTPTimeoutParse):
		return classifiedError{kind: "config_error"}
	case errors.Is(err, athena.ErrRequestTimeout):
		return classifiedError{kind: "upstream_timeout"}
	case errors.Is(err, athena.ErrMalformedResponse):
		return classifiedError{kind: "decode_error"}
	case errors.Is(err, athena.ErrRequestFailed):
		return classifiedError{kind: "upstream_error"}
	}

	var upstreamErr *athena.UpstreamStatusError
	if errors.As(err, &upstreamErr) {
		return classifiedError{
			kind:           "upstream_error",
			upstreamStatus: &upstreamErr.StatusCode,
		}
	}

	return classifiedError{kind: "unexpected_error"}
}

func isCobraValidationError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "required flag(s)")
}
