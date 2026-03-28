package domain

import (
	"fmt"
	"time"
)

type Event struct {
	ID          string
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	AllDay      bool
}

func (e *Event) Duration() time.Duration {
	return e.EndTime.Sub(e.StartTime)
}

func (e *Event) IsHappening() bool {
	now := time.Now()
	return now.After(e.StartTime) && now.Before(e.EndTime)
}

func (e *Event) IsPast() bool {
	return time.Now().After(e.EndTime)
}

func (e *Event) FormatTimeRange() string {
	if e.AllDay {
		return "All day"
	}
	return fmt.Sprintf("%s – %s", e.StartTime.Format("15:04"), e.EndTime.Format("15:04"))
}
