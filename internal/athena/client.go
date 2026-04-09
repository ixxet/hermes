package athena

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

var (
	ErrMalformedResponse        = errors.New("athena occupancy response is malformed")
	ErrHistoryMalformedResponse = errors.New("athena occupancy history response is malformed")
	ErrRequestTimeout           = errors.New("athena occupancy request timed out")
	ErrRequestFailed            = errors.New("athena occupancy request failed")
)

type UpstreamStatusError struct {
	StatusCode int
	Message    string
}

func (e *UpstreamStatusError) Error() string {
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("athena occupancy request failed with status %d", e.StatusCode)
	}

	return fmt.Sprintf("athena occupancy request failed with status %d: %s", e.StatusCode, e.Message)
}

type OccupancySnapshot struct {
	FacilityID   string
	CurrentCount int
	ObservedAt   time.Time
}

type HistoryFilter struct {
	FacilityID string
	Since      time.Time
	Until      time.Time
}

type HistoryObservation struct {
	Direction  string
	Result     string
	ObservedAt time.Time
	Committed  bool
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

type occupancyResponse struct {
	FacilityID   string `json:"facility_id"`
	CurrentCount int    `json:"current_count"`
	ObservedAt   string `json:"observed_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type historyResponse struct {
	FacilityID   string                    `json:"facility_id"`
	Since        string                    `json:"since"`
	Until        string                    `json:"until"`
	Observations []historyObservationEntry `json:"observations"`
}

type historyObservationEntry struct {
	Direction  string `json:"direction"`
	Result     string `json:"result"`
	ObservedAt string `json:"observed_at"`
	Committed  bool   `json:"committed"`
}

func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse athena base url: %w", err)
	}

	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) CurrentOccupancy(ctx context.Context, facilityID string) (OccupancySnapshot, error) {
	query := make(url.Values)
	query.Set("facility", facilityID)
	response, err := c.doGET(ctx, "/api/v1/presence/count", query)
	if err != nil {
		return OccupancySnapshot{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return OccupancySnapshot{}, decodeUpstreamStatus(response)
	}

	var payload occupancyResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return OccupancySnapshot{}, fmt.Errorf("%w: %v", ErrMalformedResponse, err)
	}
	if strings.TrimSpace(payload.FacilityID) == "" || strings.TrimSpace(payload.ObservedAt) == "" {
		return OccupancySnapshot{}, ErrMalformedResponse
	}

	observedAt, err := time.Parse(time.RFC3339, payload.ObservedAt)
	if err != nil {
		return OccupancySnapshot{}, fmt.Errorf("%w: %v", ErrMalformedResponse, err)
	}

	return OccupancySnapshot{
		FacilityID:   payload.FacilityID,
		CurrentCount: payload.CurrentCount,
		ObservedAt:   observedAt.UTC(),
	}, nil
}

func (c *Client) OccupancyHistory(ctx context.Context, filter HistoryFilter) ([]HistoryObservation, error) {
	facilityID := strings.TrimSpace(filter.FacilityID)
	if facilityID == "" {
		return nil, fmt.Errorf("history facility is required")
	}
	if filter.Since.IsZero() {
		return nil, fmt.Errorf("history since is required")
	}
	if filter.Until.IsZero() {
		return nil, fmt.Errorf("history until is required")
	}
	if filter.Until.Before(filter.Since) {
		return nil, fmt.Errorf("history until must be greater than or equal to since")
	}

	query := make(url.Values)
	query.Set("facility", facilityID)
	query.Set("since", filter.Since.UTC().Format(time.RFC3339))
	query.Set("until", filter.Until.UTC().Format(time.RFC3339))

	response, err := c.doGET(ctx, "/api/v1/presence/history", query)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, decodeUpstreamStatus(response)
	}

	var payload historyResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHistoryMalformedResponse, err)
	}
	if strings.TrimSpace(payload.FacilityID) == "" {
		return nil, ErrHistoryMalformedResponse
	}

	observations := make([]HistoryObservation, 0, len(payload.Observations))
	for _, observation := range payload.Observations {
		observedAt, err := time.Parse(time.RFC3339, observation.ObservedAt)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrHistoryMalformedResponse, err)
		}
		if observation.Direction != "in" && observation.Direction != "out" {
			return nil, ErrHistoryMalformedResponse
		}
		if observation.Result != "pass" && observation.Result != "fail" {
			return nil, ErrHistoryMalformedResponse
		}

		observations = append(observations, HistoryObservation{
			Direction:  observation.Direction,
			Result:     observation.Result,
			ObservedAt: observedAt.UTC(),
			Committed:  observation.Committed,
		})
	}

	return observations, nil
}

func (c *Client) doGET(ctx context.Context, endpoint string, query url.Values) (*http.Response, error) {
	requestURL := *c.baseURL
	requestURL.Path = path.Join(c.baseURL.Path, endpoint)
	requestURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build athena request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w: %v", ErrRequestTimeout, err)
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, fmt.Errorf("%w: %v", ErrRequestTimeout, err)
		}

		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}

	return response, nil
}

func decodeUpstreamStatus(response *http.Response) error {
	var upstreamError errorResponse
	if err := json.NewDecoder(response.Body).Decode(&upstreamError); err != nil {
		return &UpstreamStatusError{StatusCode: response.StatusCode}
	}

	return &UpstreamStatusError{
		StatusCode: response.StatusCode,
		Message:    strings.TrimSpace(upstreamError.Error),
	}
}
