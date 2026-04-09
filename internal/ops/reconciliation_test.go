package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ixxet/hermes/internal/athena"
)

type stubHistoryReader struct {
	observations []athena.HistoryObservation
	err          error
}

func (s stubHistoryReader) OccupancyHistory(context.Context, athena.HistoryFilter) ([]athena.HistoryObservation, error) {
	if s.err != nil {
		return nil, s.err
	}

	return append([]athena.HistoryObservation(nil), s.observations...), nil
}

func TestReconciliationServiceReturnsDeterministicReportAndHeatMap(t *testing.T) {
	service := NewReconciliationService(
		stubOccupancyReader{
			snapshot: athena.OccupancySnapshot{
				FacilityID:   "ashtonbee",
				CurrentCount: 2,
				ObservedAt:   time.Date(2026, 4, 9, 13, 0, 0, 0, time.UTC),
			},
		},
		stubHistoryReader{
			observations: []athena.HistoryObservation{
				{Direction: "in", Result: "pass", ObservedAt: time.Date(2026, 4, 9, 11, 10, 0, 0, time.UTC), Committed: true},
				{Direction: "in", Result: "pass", ObservedAt: time.Date(2026, 4, 9, 11, 20, 0, 0, time.UTC), Committed: false},
				{Direction: "out", Result: "fail", ObservedAt: time.Date(2026, 4, 9, 11, 40, 0, 0, time.UTC), Committed: false},
				{Direction: "out", Result: "pass", ObservedAt: time.Date(2026, 4, 9, 12, 10, 0, 0, time.UTC), Committed: true},
				{Direction: "in", Result: "pass", ObservedAt: time.Date(2026, 4, 9, 12, 40, 0, 0, time.UTC), Committed: true},
			},
		},
	)

	answer, err := service.AskReconciliation(context.Background(), "ashtonbee", 2*time.Hour, time.Hour)
	if err != nil {
		t.Fatalf("AskReconciliation() error = %v", err)
	}

	if answer.FacilityID != "ashtonbee" {
		t.Fatalf("FacilityID = %q, want ashtonbee", answer.FacilityID)
	}
	if answer.SourceService != "athena" {
		t.Fatalf("SourceService = %q, want athena", answer.SourceService)
	}
	if answer.WindowStart != "2026-04-09T11:00:00Z" || answer.WindowEnd != "2026-04-09T13:00:00Z" {
		t.Fatalf("window = %s..%s, want 2026-04-09T11:00:00Z..2026-04-09T13:00:00Z", answer.WindowStart, answer.WindowEnd)
	}
	if answer.Current.CurrentCount != 2 {
		t.Fatalf("Current.CurrentCount = %d, want 2", answer.Current.CurrentCount)
	}
	if answer.Report.OpeningCount != 1 {
		t.Fatalf("OpeningCount = %d, want 1", answer.Report.OpeningCount)
	}
	if answer.Report.NetChange != 1 {
		t.Fatalf("NetChange = %d, want 1", answer.Report.NetChange)
	}
	if answer.Report.CommittedEntries != 2 {
		t.Fatalf("CommittedEntries = %d, want 2", answer.Report.CommittedEntries)
	}
	if answer.Report.CommittedExits != 1 {
		t.Fatalf("CommittedExits = %d, want 1", answer.Report.CommittedExits)
	}
	if answer.Report.FailedObservations != 1 {
		t.Fatalf("FailedObservations = %d, want 1", answer.Report.FailedObservations)
	}
	if answer.Report.ObservedPassWithoutChange != 1 {
		t.Fatalf("ObservedPassWithoutChange = %d, want 1", answer.Report.ObservedPassWithoutChange)
	}
	if answer.Report.PeakOccupancy != 2 {
		t.Fatalf("PeakOccupancy = %d, want 2", answer.Report.PeakOccupancy)
	}
	if len(answer.HeatMap) != 2 {
		t.Fatalf("len(HeatMap) = %d, want 2", len(answer.HeatMap))
	}
	if answer.HeatMap[0].OccupancyPeak != 2 || answer.HeatMap[0].OccupancyEnd != 2 {
		t.Fatalf("HeatMap[0] = %#v, want first bin peak/end at 2", answer.HeatMap[0])
	}
	if answer.HeatMap[0].ObservedPassWithoutChange != 1 || answer.HeatMap[0].FailedObservations != 1 {
		t.Fatalf("HeatMap[0] = %#v, want first bin issues", answer.HeatMap[0])
	}
	if answer.HeatMap[1].OccupancyPeak != 2 || answer.HeatMap[1].OccupancyEnd != 2 {
		t.Fatalf("HeatMap[1] = %#v, want second bin peak 2 end 2", answer.HeatMap[1])
	}
	if answer.InspectNext.Category != "observation-heavy-window" {
		t.Fatalf("InspectNext.Category = %q, want observation-heavy-window", answer.InspectNext.Category)
	}
	if answer.InspectNext.WindowStart != "2026-04-09T11:00:00Z" {
		t.Fatalf("InspectNext.WindowStart = %q, want first bin", answer.InspectNext.WindowStart)
	}
}

func TestReconciliationServiceValidatesInputsAndPropagatesFailures(t *testing.T) {
	service := NewReconciliationService(stubOccupancyReader{}, stubHistoryReader{})

	testCases := []struct {
		name     string
		facility string
		window   time.Duration
		bin      time.Duration
		wantErr  error
		prepare  func() *ReconciliationService
	}{
		{
			name:     "blank facility",
			facility: "   ",
			window:   time.Hour,
			bin:      time.Hour,
			wantErr:  ErrFacilityRequired,
		},
		{
			name:     "missing window",
			facility: "ashtonbee",
			bin:      time.Hour,
			wantErr:  ErrWindowRequired,
		},
		{
			name:     "missing bin",
			facility: "ashtonbee",
			window:   time.Hour,
			wantErr:  ErrBinRequired,
		},
		{
			name:     "bin exceeds window",
			facility: "ashtonbee",
			window:   time.Hour,
			bin:      2 * time.Hour,
			wantErr:  ErrBinExceedsWindow,
		},
		{
			name:     "occupancy failure",
			facility: "ashtonbee",
			window:   time.Hour,
			bin:      time.Hour,
			wantErr:  errors.New("athena unavailable"),
			prepare: func() *ReconciliationService {
				return NewReconciliationService(
					stubOccupancyReader{err: errors.New("athena unavailable")},
					stubHistoryReader{},
				)
			},
		},
		{
			name:     "history failure",
			facility: "ashtonbee",
			window:   time.Hour,
			bin:      time.Hour,
			wantErr:  errors.New("history unavailable"),
			prepare: func() *ReconciliationService {
				return NewReconciliationService(
					stubOccupancyReader{
						snapshot: athena.OccupancySnapshot{
							FacilityID:   "ashtonbee",
							CurrentCount: 2,
							ObservedAt:   time.Date(2026, 4, 9, 13, 0, 0, 0, time.UTC),
						},
					},
					stubHistoryReader{err: errors.New("history unavailable")},
				)
			},
		},
		{
			name:     "inconsistent history",
			facility: "ashtonbee",
			window:   time.Hour,
			bin:      time.Hour,
			wantErr:  ErrHistoryInconsistent,
			prepare: func() *ReconciliationService {
				return NewReconciliationService(
					stubOccupancyReader{
						snapshot: athena.OccupancySnapshot{
							FacilityID:   "ashtonbee",
							CurrentCount: 0,
							ObservedAt:   time.Date(2026, 4, 9, 13, 0, 0, 0, time.UTC),
						},
					},
					stubHistoryReader{
						observations: []athena.HistoryObservation{
							{Direction: "in", Result: "pass", ObservedAt: time.Date(2026, 4, 9, 12, 10, 0, 0, time.UTC), Committed: true},
						},
					},
				)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			target := service
			if testCase.prepare != nil {
				target = testCase.prepare()
			}

			_, err := target.AskReconciliation(context.Background(), testCase.facility, testCase.window, testCase.bin)
			if err == nil {
				t.Fatal("AskReconciliation() error = nil, want failure")
			}
			if !errors.Is(err, testCase.wantErr) && err.Error() != testCase.wantErr.Error() {
				t.Fatalf("AskReconciliation() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
