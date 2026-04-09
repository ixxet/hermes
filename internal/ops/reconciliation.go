package ops

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/ixxet/hermes/internal/athena"
)

var (
	ErrWindowRequired      = errors.New("window must be greater than zero")
	ErrBinRequired         = errors.New("bin must be greater than zero")
	ErrBinExceedsWindow    = errors.New("bin must be less than or equal to window")
	ErrHistoryInconsistent = errors.New("stable history is inconsistent with current occupancy")
)

type OccupancyHistoryReader interface {
	OccupancyHistory(context.Context, athena.HistoryFilter) ([]athena.HistoryObservation, error)
}

type ReconciliationAnswer struct {
	FacilityID    string                   `json:"facility_id"`
	SourceService string                   `json:"source_service"`
	WindowStart   string                   `json:"window_start"`
	WindowEnd     string                   `json:"window_end"`
	Current       ReconciliationCurrent    `json:"current"`
	Report        ReconciliationReport     `json:"report"`
	HeatMap       []ReconciliationHeatCell `json:"heat_map"`
	InspectNext   ReconciliationInspect    `json:"inspect_next"`
}

type ReconciliationCurrent struct {
	CurrentCount int    `json:"current_count"`
	ObservedAt   string `json:"observed_at"`
}

type ReconciliationReport struct {
	OpeningCount              int    `json:"opening_count"`
	NetChange                 int    `json:"net_change"`
	CommittedEntries          int    `json:"committed_entries"`
	CommittedExits            int    `json:"committed_exits"`
	FailedObservations        int    `json:"failed_observations"`
	ObservedPassWithoutChange int    `json:"observed_pass_without_change"`
	PeakOccupancy             int    `json:"peak_occupancy"`
	PeakObservedAt            string `json:"peak_observed_at"`
}

type ReconciliationHeatCell struct {
	WindowStart               string `json:"window_start"`
	WindowEnd                 string `json:"window_end"`
	HeatLevel                 int    `json:"heat_level"`
	OccupancyPeak             int    `json:"occupancy_peak"`
	OccupancyEnd              int    `json:"occupancy_end"`
	CommittedEntries          int    `json:"committed_entries"`
	CommittedExits            int    `json:"committed_exits"`
	FailedObservations        int    `json:"failed_observations"`
	ObservedPassWithoutChange int    `json:"observed_pass_without_change"`
}

type ReconciliationInspect struct {
	Category    string `json:"category"`
	Reason      string `json:"reason"`
	WindowStart string `json:"window_start"`
	WindowEnd   string `json:"window_end"`
}

type ReconciliationService struct {
	occupancyReader OccupancyReader
	historyReader   OccupancyHistoryReader
}

func NewReconciliationService(occupancyReader OccupancyReader, historyReader OccupancyHistoryReader) *ReconciliationService {
	return &ReconciliationService{
		occupancyReader: occupancyReader,
		historyReader:   historyReader,
	}
}

func (s *ReconciliationService) AskReconciliation(ctx context.Context, facilityID string, window, bin time.Duration) (ReconciliationAnswer, error) {
	trimmedFacilityID := strings.TrimSpace(facilityID)
	if trimmedFacilityID == "" {
		return ReconciliationAnswer{}, ErrFacilityRequired
	}
	if window <= 0 {
		return ReconciliationAnswer{}, ErrWindowRequired
	}
	if bin <= 0 {
		return ReconciliationAnswer{}, ErrBinRequired
	}
	if bin > window {
		return ReconciliationAnswer{}, ErrBinExceedsWindow
	}

	snapshot, err := s.occupancyReader.CurrentOccupancy(ctx, trimmedFacilityID)
	if err != nil {
		return ReconciliationAnswer{}, err
	}
	if snapshot.ObservedAt.IsZero() {
		return ReconciliationAnswer{}, ErrHistoryInconsistent
	}

	windowEnd := snapshot.ObservedAt.UTC()
	windowStart := windowEnd.Add(-window)

	observations, err := s.historyReader.OccupancyHistory(ctx, athena.HistoryFilter{
		FacilityID: trimmedFacilityID,
		Since:      windowStart,
		Until:      windowEnd,
	})
	if err != nil {
		return ReconciliationAnswer{}, err
	}

	report, heatMap, inspectNext, err := buildReconciliation(snapshot.CurrentCount, windowStart, windowEnd, bin, observations)
	if err != nil {
		return ReconciliationAnswer{}, err
	}

	return ReconciliationAnswer{
		FacilityID:    snapshot.FacilityID,
		SourceService: "athena",
		WindowStart:   windowStart.Format(time.RFC3339),
		WindowEnd:     windowEnd.Format(time.RFC3339),
		Current: ReconciliationCurrent{
			CurrentCount: snapshot.CurrentCount,
			ObservedAt:   windowEnd.Format(time.RFC3339),
		},
		Report:      report,
		HeatMap:     heatMap,
		InspectNext: inspectNext,
	}, nil
}

func buildReconciliation(currentCount int, windowStart, windowEnd time.Time, bin time.Duration, observations []athena.HistoryObservation) (ReconciliationReport, []ReconciliationHeatCell, ReconciliationInspect, error) {
	sorted := append([]athena.HistoryObservation(nil), observations...)
	sort.Slice(sorted, func(i, j int) bool {
		if !sorted[i].ObservedAt.Equal(sorted[j].ObservedAt) {
			return sorted[i].ObservedAt.Before(sorted[j].ObservedAt)
		}
		if sorted[i].Result != sorted[j].Result {
			return sorted[i].Result < sorted[j].Result
		}
		return sorted[i].Direction < sorted[j].Direction
	})

	committedEntries := 0
	committedExits := 0
	failedObservations := 0
	observedPassWithoutChange := 0
	for _, observation := range sorted {
		switch observation.Result {
		case "fail":
			failedObservations++
		case "pass":
			if !observation.Committed {
				observedPassWithoutChange++
				continue
			}
			switch observation.Direction {
			case "in":
				committedEntries++
			case "out":
				committedExits++
			default:
				return ReconciliationReport{}, nil, ReconciliationInspect{}, ErrHistoryInconsistent
			}
		default:
			return ReconciliationReport{}, nil, ReconciliationInspect{}, ErrHistoryInconsistent
		}
	}

	netChange := committedEntries - committedExits
	openingCount := currentCount - netChange
	if openingCount < 0 {
		return ReconciliationReport{}, nil, ReconciliationInspect{}, ErrHistoryInconsistent
	}

	bins := make([]reconciliationBin, 0)
	for start := windowStart.UTC(); start.Before(windowEnd.UTC()); start = start.Add(bin) {
		end := start.Add(bin)
		if end.After(windowEnd.UTC()) {
			end = windowEnd.UTC()
		}
		bins = append(bins, reconciliationBin{
			start: start,
			end:   end,
		})
	}
	if len(bins) == 0 {
		bins = append(bins, reconciliationBin{start: windowStart.UTC(), end: windowEnd.UTC()})
	}

	runningCount := openingCount
	peakOccupancy := openingCount
	peakObservedAt := windowStart.UTC()
	observationIndex := 0
	for index := range bins {
		bins[index].occupancyPeak = runningCount
		for observationIndex < len(sorted) {
			observation := sorted[observationIndex]
			observedAt := observation.ObservedAt.UTC()
			if observedAt.Before(bins[index].start) {
				observationIndex++
				continue
			}
			if !observedAt.Before(bins[index].end) && !(index == len(bins)-1 && observedAt.Equal(bins[index].end)) {
				break
			}

			switch observation.Result {
			case "fail":
				bins[index].failedObservations++
			case "pass":
				if !observation.Committed {
					bins[index].observedPassWithoutChange++
					observationIndex++
					continue
				}

				switch observation.Direction {
				case "in":
					bins[index].committedEntries++
					runningCount++
				case "out":
					bins[index].committedExits++
					runningCount--
				default:
					return ReconciliationReport{}, nil, ReconciliationInspect{}, ErrHistoryInconsistent
				}

				if runningCount < 0 {
					return ReconciliationReport{}, nil, ReconciliationInspect{}, ErrHistoryInconsistent
				}
				if runningCount > bins[index].occupancyPeak {
					bins[index].occupancyPeak = runningCount
				}
				if runningCount > peakOccupancy {
					peakOccupancy = runningCount
					peakObservedAt = observedAt
				}
			}

			observationIndex++
		}

		bins[index].occupancyEnd = runningCount
	}

	if runningCount != currentCount {
		return ReconciliationReport{}, nil, ReconciliationInspect{}, ErrHistoryInconsistent
	}

	heatMap := make([]ReconciliationHeatCell, 0, len(bins))
	maxPeak := 0
	for _, bin := range bins {
		if bin.occupancyPeak > maxPeak {
			maxPeak = bin.occupancyPeak
		}
	}
	for _, bin := range bins {
		heatLevel := 0
		if maxPeak > 0 && bin.occupancyPeak > 0 {
			heatLevel = (bin.occupancyPeak*4 + maxPeak - 1) / maxPeak
		}
		heatMap = append(heatMap, ReconciliationHeatCell{
			WindowStart:               bin.start.Format(time.RFC3339),
			WindowEnd:                 bin.end.Format(time.RFC3339),
			HeatLevel:                 heatLevel,
			OccupancyPeak:             bin.occupancyPeak,
			OccupancyEnd:              bin.occupancyEnd,
			CommittedEntries:          bin.committedEntries,
			CommittedExits:            bin.committedExits,
			FailedObservations:        bin.failedObservations,
			ObservedPassWithoutChange: bin.observedPassWithoutChange,
		})
	}

	inspectNext := chooseInspectTarget(bins)
	report := ReconciliationReport{
		OpeningCount:              openingCount,
		NetChange:                 netChange,
		CommittedEntries:          committedEntries,
		CommittedExits:            committedExits,
		FailedObservations:        failedObservations,
		ObservedPassWithoutChange: observedPassWithoutChange,
		PeakOccupancy:             peakOccupancy,
		PeakObservedAt:            peakObservedAt.Format(time.RFC3339),
	}

	return report, heatMap, inspectNext, nil
}

type reconciliationBin struct {
	start                     time.Time
	end                       time.Time
	committedEntries          int
	committedExits            int
	failedObservations        int
	observedPassWithoutChange int
	occupancyPeak             int
	occupancyEnd              int
}

func chooseInspectTarget(bins []reconciliationBin) ReconciliationInspect {
	bestIssueIndex := -1
	bestIssueScore := -1
	for index, bin := range bins {
		score := bin.failedObservations + bin.observedPassWithoutChange
		if score > bestIssueScore {
			bestIssueScore = score
			bestIssueIndex = index
		}
	}
	if bestIssueIndex >= 0 && bestIssueScore > 0 {
		bin := bins[bestIssueIndex]
		return ReconciliationInspect{
			Category:    "observation-heavy-window",
			Reason:      "highest non-committed or failed observation activity in stable history",
			WindowStart: bin.start.Format(time.RFC3339),
			WindowEnd:   bin.end.Format(time.RFC3339),
		}
	}

	bestPeakIndex := 0
	bestPeak := -1
	for index, bin := range bins {
		if bin.occupancyPeak > bestPeak {
			bestPeak = bin.occupancyPeak
			bestPeakIndex = index
		}
	}
	bin := bins[bestPeakIndex]
	return ReconciliationInspect{
		Category:    "peak-occupancy-window",
		Reason:      "highest occupancy pressure in stable history",
		WindowStart: bin.start.Format(time.RFC3339),
		WindowEnd:   bin.end.Format(time.RFC3339),
	}
}
