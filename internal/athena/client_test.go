package athena

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientCurrentOccupancyConsumesAthenaReadSurface(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/api/v1/presence/count" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/api/v1/presence/count")
		}
		if facility := r.URL.Query().Get("facility"); facility != "ashtonbee" {
			t.Fatalf("facility = %q, want %q", facility, "ashtonbee")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"facility_id":"ashtonbee","current_count":9,"observed_at":"2026-04-02T16:00:00Z"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	snapshot, err := client.CurrentOccupancy(context.Background(), "ashtonbee")
	if err != nil {
		t.Fatalf("CurrentOccupancy() error = %v", err)
	}
	if snapshot.FacilityID != "ashtonbee" {
		t.Fatalf("FacilityID = %q, want %q", snapshot.FacilityID, "ashtonbee")
	}
	if snapshot.CurrentCount != 9 {
		t.Fatalf("CurrentCount = %d, want 9", snapshot.CurrentCount)
	}
	wantObservedAt := time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC)
	if !snapshot.ObservedAt.Equal(wantObservedAt) {
		t.Fatalf("ObservedAt = %s, want %s", snapshot.ObservedAt, wantObservedAt)
	}
}

func TestClientCurrentOccupancyMapsUpstreamFailuresClearly(t *testing.T) {
	testCases := []struct {
		name     string
		handler  http.HandlerFunc
		wantErr  string
		checkErr func(error) bool
		timeout  time.Duration
	}{
		{
			name: "upstream error response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"read path unavailable"}`))
			},
			wantErr: "status 500: read path unavailable",
			checkErr: func(err error) bool {
				var upstreamErr *UpstreamStatusError
				return errors.As(err, &upstreamErr) && upstreamErr.StatusCode == http.StatusInternalServerError
			},
		},
		{
			name: "malformed upstream json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"facility_id":`))
			},
			checkErr: func(err error) bool { return errors.Is(err, ErrMalformedResponse) },
			wantErr:  "malformed",
		},
		{
			name: "invalid observed at",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"facility_id":"ashtonbee","current_count":9,"observed_at":"not-a-time"}`))
			},
			checkErr: func(err error) bool { return errors.Is(err, ErrMalformedResponse) },
			wantErr:  "malformed",
		},
		{
			name: "timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(50 * time.Millisecond)
				_, _ = w.Write([]byte(`{"facility_id":"ashtonbee","current_count":9,"observed_at":"2026-04-02T16:00:00Z"}`))
			},
			wantErr:  "timed out",
			timeout:  10 * time.Millisecond,
			checkErr: func(err error) bool { return errors.Is(err, ErrRequestTimeout) },
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(testCase.handler)
			defer server.Close()

			timeout := testCase.timeout
			if timeout == 0 {
				timeout = 2 * time.Second
			}

			client, err := NewClient(server.URL, timeout)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			_, err = client.CurrentOccupancy(context.Background(), "ashtonbee")
			if err == nil {
				t.Fatal("CurrentOccupancy() error = nil, want failure")
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("CurrentOccupancy() error = %q, want substring %q", err.Error(), testCase.wantErr)
			}
			if testCase.checkErr != nil && !testCase.checkErr(err) {
				t.Fatalf("CurrentOccupancy() error = %v, want specific classification", err)
			}
		})
	}
}

func TestClientCurrentOccupancyClassifiesTransportFailure(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	_, err = client.CurrentOccupancy(context.Background(), "ashtonbee")
	if err == nil {
		t.Fatal("CurrentOccupancy() error = nil, want transport failure")
	}
	if !errors.Is(err, ErrRequestFailed) {
		t.Fatalf("CurrentOccupancy() error = %v, want %v", err, ErrRequestFailed)
	}
}
