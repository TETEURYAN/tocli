package google

import (
	"strings"
	"time"
	"tocli/internal/domain"

	calendar "google.golang.org/api/calendar/v3"
	tasks "google.golang.org/api/tasks/v1"
)

func mapGoogleTaskToDomain(t *tasks.Task, listID string) domain.Task {
	dt := domain.Task{
		ID:     t.Id,
		Title:  t.Title,
		Notes:  t.Notes,
		Status: domain.TaskOpen,
		ListID: listID,
	}

	if strings.EqualFold(t.Status, "completed") {
		dt.Status = domain.TaskCompleted
	}

	if t.Due != "" {
		if parsed, err := time.Parse(time.RFC3339, t.Due); err == nil {
			dt.DueDate = &parsed
		}
	}

	if completed := ptrStr(t.Completed); completed != "" {
		if parsed, err := time.Parse(time.RFC3339, completed); err == nil {
			dt.CompletedAt = &parsed
		}
	}

	return dt
}

func mapGoogleEventToDomain(e *calendar.Event) (domain.Event, error) {
	ev := domain.Event{
		ID:          e.Id,
		Title:       e.Summary,
		Description: e.Description,
		Location:    e.Location,
	}

	if e.Start == nil {
		return ev, nil
	}

	loc := time.Local

	if e.Start.DateTime != "" {
		t, err := time.Parse(time.RFC3339, e.Start.DateTime)
		if err != nil {
			return ev, err
		}
		ev.StartTime = t
		ev.AllDay = false
	} else if e.Start.Date != "" {
		t, err := time.ParseInLocation("2006-01-02", e.Start.Date, loc)
		if err != nil {
			return ev, err
		}
		ev.StartTime = t
		ev.AllDay = true
	}

	if e.End != nil {
		if e.End.DateTime != "" {
			t, err := time.Parse(time.RFC3339, e.End.DateTime)
			if err == nil {
				ev.EndTime = t
			}
		} else if e.End.Date != "" {
			tEnd, err := time.ParseInLocation("2006-01-02", e.End.Date, loc)
			if err == nil {
				// All-day end is exclusive in Calendar API.
				ev.EndTime = tEnd.Add(-time.Nanosecond)
			}
		}
	}

	if ev.EndTime.IsZero() && !ev.StartTime.IsZero() {
		if ev.AllDay {
			ev.EndTime = ev.StartTime.Add(24*time.Hour - time.Nanosecond)
		} else {
			ev.EndTime = ev.StartTime.Add(time.Hour)
		}
	}

	return ev, nil
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
