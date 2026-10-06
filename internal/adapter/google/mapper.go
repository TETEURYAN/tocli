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
			// Google sends the due date as midnight UTC. Keep the calendar date and put it at
			// local midnight; converting the instant would show e.g. "21:00 of the day before".
			p := parsed.UTC()
			local := time.Date(p.Year(), p.Month(), p.Day(), 0, 0, 0, 0, time.Local)
			dt.DueDate = &local
		}
	}

	if completed := ptrStr(t.Completed); completed != "" {
		if parsed, err := time.Parse(time.RFC3339, completed); err == nil {
			local := parsed.Local()
			dt.CompletedAt = &local
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
		URL:         e.HtmlLink,
		MeetLink:    meetLink(e),
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
		ev.StartTime = t.Local()
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
				ev.EndTime = t.Local()
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

// meetLink is the event's video-call link: the classic Hangouts field, or the first video entry
// point of its conference data.
func meetLink(e *calendar.Event) string {
	if e.HangoutLink != "" {
		return e.HangoutLink
	}
	if e.ConferenceData != nil {
		for _, ep := range e.ConferenceData.EntryPoints {
			if ep != nil && ep.EntryPointType == "video" && ep.Uri != "" {
				return ep.Uri
			}
		}
	}
	return ""
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
