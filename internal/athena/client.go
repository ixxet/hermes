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

var ErrMalformedResponse = errors.New("athena occupancy response is malformed")

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
	requestURL := *c.baseURL
	requestURL.Path = path.Join(c.baseURL.Path, "/api/v1/presence/count")

	query := requestURL.Query()
	query.Set("facility", facilityID)
	requestURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return OccupancySnapshot{}, fmt.Errorf("build athena occupancy request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return OccupancySnapshot{}, fmt.Errorf("athena occupancy request timed out: %w", err)
		}

		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return OccupancySnapshot{}, fmt.Errorf("athena occupancy request timed out: %w", err)
		}

		return OccupancySnapshot{}, fmt.Errorf("athena occupancy request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var upstreamError errorResponse
		if err := json.NewDecoder(response.Body).Decode(&upstreamError); err != nil {
			return OccupancySnapshot{}, &UpstreamStatusError{StatusCode: response.StatusCode}
		}

		return OccupancySnapshot{}, &UpstreamStatusError{
			StatusCode: response.StatusCode,
			Message:    strings.TrimSpace(upstreamError.Error),
		}
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
