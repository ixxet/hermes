package ops

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ixxet/hermes/internal/athena"
)

var ErrFacilityRequired = errors.New("facility is required")

type OccupancyReader interface {
	CurrentOccupancy(ctx context.Context, facilityID string) (athena.OccupancySnapshot, error)
}

type OccupancyAnswer struct {
	FacilityID    string `json:"facility_id"`
	CurrentCount  int    `json:"current_count"`
	ObservedAt    string `json:"observed_at"`
	SourceService string `json:"source_service"`
	Notes         string `json:"notes,omitempty"`
}

type OccupancyService struct {
	reader OccupancyReader
}

func NewOccupancyService(reader OccupancyReader) *OccupancyService {
	return &OccupancyService{reader: reader}
}

func (s *OccupancyService) AskOccupancy(ctx context.Context, facilityID string) (OccupancyAnswer, error) {
	trimmedFacilityID := strings.TrimSpace(facilityID)
	if trimmedFacilityID == "" {
		return OccupancyAnswer{}, ErrFacilityRequired
	}

	snapshot, err := s.reader.CurrentOccupancy(ctx, trimmedFacilityID)
	if err != nil {
		return OccupancyAnswer{}, err
	}

	return OccupancyAnswer{
		FacilityID:    snapshot.FacilityID,
		CurrentCount:  snapshot.CurrentCount,
		ObservedAt:    snapshot.ObservedAt.UTC().Format(time.RFC3339),
		SourceService: "athena",
	}, nil
}
