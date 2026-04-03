package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ixxet/hermes/internal/athena"
)

type stubOccupancyReader struct {
	snapshot athena.OccupancySnapshot
	err      error
}

func (s stubOccupancyReader) CurrentOccupancy(context.Context, string) (athena.OccupancySnapshot, error) {
	if s.err != nil {
		return athena.OccupancySnapshot{}, s.err
	}

	return s.snapshot, nil
}

func TestOccupancyServiceAskOccupancyReturnsStableShape(t *testing.T) {
	service := NewOccupancyService(stubOccupancyReader{
		snapshot: athena.OccupancySnapshot{
			FacilityID:   "ashtonbee",
			CurrentCount: 9,
			ObservedAt:   time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC),
		},
	})

	answer, err := service.AskOccupancy(context.Background(), "ashtonbee")
	if err != nil {
		t.Fatalf("AskOccupancy() error = %v", err)
	}
	if answer.FacilityID != "ashtonbee" {
		t.Fatalf("FacilityID = %q, want %q", answer.FacilityID, "ashtonbee")
	}
	if answer.CurrentCount != 9 {
		t.Fatalf("CurrentCount = %d, want 9", answer.CurrentCount)
	}
	if answer.ObservedAt != "2026-04-02T16:00:00Z" {
		t.Fatalf("ObservedAt = %q, want %q", answer.ObservedAt, "2026-04-02T16:00:00Z")
	}
	if answer.SourceService != "athena" {
		t.Fatalf("SourceService = %q, want %q", answer.SourceService, "athena")
	}
}

func TestOccupancyServiceAskOccupancyValidatesInputsAndPropagatesFailures(t *testing.T) {
	_, err := NewOccupancyService(stubOccupancyReader{}).AskOccupancy(context.Background(), "   ")
	if !errors.Is(err, ErrFacilityRequired) {
		t.Fatalf("AskOccupancy(blank) error = %v, want %v", err, ErrFacilityRequired)
	}

	upstreamErr := errors.New("athena unavailable")
	_, err = NewOccupancyService(stubOccupancyReader{err: upstreamErr}).AskOccupancy(context.Background(), "ashtonbee")
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("AskOccupancy(upstream) error = %v, want %v", err, upstreamErr)
	}
}
