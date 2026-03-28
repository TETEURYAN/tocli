package google

import (
	"context"
	"fmt"
	"net/http"

	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	tasks "google.golang.org/api/tasks/v1"
)

// Client bundles Google Calendar and Tasks API services using one HTTP client.
type Client struct {
	Calendar *calendar.Service
	Tasks    *tasks.Service
}

// NewClient builds API services with an OAuth-authorized HTTP client.
func NewClient(ctx context.Context, httpClient *http.Client) (*Client, error) {
	calSvc, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("calendar service: %w", err)
	}
	taskSvc, err := tasks.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("tasks service: %w", err)
	}
	return &Client{Calendar: calSvc, Tasks: taskSvc}, nil
}
