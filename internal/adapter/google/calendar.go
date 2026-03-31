package google

import (
	"context"
	"time"
	"tocli/internal/domain"

	calendar "google.golang.org/api/calendar/v3"
)

// CalendarRepo implements domain.EventRepository using Google Calendar API.
type CalendarRepo struct {
	svc        *calendar.Service
	ctx        context.Context
	calendarID string
}

// NewCalendarRepo uses the user's primary calendar by default.
func NewCalendarRepo(ctx context.Context, client *Client) *CalendarRepo {
	return &CalendarRepo{
		svc:        client.Calendar,
		ctx:        ctx,
		calendarID: "primary",
	}
}

func (r *CalendarRepo) GetEvents(start, end time.Time) ([]domain.Event, error) {
	var all []domain.Event
	pageToken := ""
	for {
		call := r.svc.Events.List(r.calendarID).
			Context(r.ctx).
			TimeMin(start.Format(time.RFC3339)).
			TimeMax(end.Format(time.RFC3339)).
			SingleEvents(true).
			OrderBy("startTime").
			MaxResults(250)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, wrapAPIError("list calendar events", err)
		}
		for _, item := range resp.Items {
			if item == nil {
				continue
			}
			ev, err := mapGoogleEventToDomain(item)
			if err != nil {
				continue
			}
			all = append(all, ev)
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return all, nil
}
