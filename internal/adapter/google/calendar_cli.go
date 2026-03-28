package google

import (
	"tocli/internal/domain"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type CalendarCLI struct {
	binaryPath string
}

func NewCalendarCLI(binaryPath string) *CalendarCLI {
	if binaryPath == "" {
		binaryPath = "google"
	}
	return &CalendarCLI{binaryPath: binaryPath}
}

type cliEvent struct {
	ID       string       `json:"id"`
	Summary  string       `json:"summary"`
	Location string       `json:"location"`
	Start    cliEventTime `json:"start"`
	End      cliEventTime `json:"end"`
}

type cliEventTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
}

func (c *CalendarCLI) GetEvents(start, end time.Time) ([]domain.Event, error) {
	out, err := c.run(
		"calendar", "events", "list",
		"--timeMin="+start.Format(time.RFC3339),
		"--timeMax="+end.Format(time.RFC3339),
		"--format=json",
	)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	var raw []cliEvent
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse events: %w", err)
	}

	events := make([]domain.Event, len(raw))
	for i, r := range raw {
		events[i] = domain.Event{
			ID:       r.ID,
			Title:    r.Summary,
			Location: r.Location,
		}

		if r.Start.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, r.Start.DateTime); err == nil {
				events[i].StartTime = t
			}
		} else if r.Start.Date != "" {
			if t, err := time.Parse("2006-01-02", r.Start.Date); err == nil {
				events[i].StartTime = t
				events[i].AllDay = true
			}
		}

		if r.End.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, r.End.DateTime); err == nil {
				events[i].EndTime = t
			}
		} else if r.End.Date != "" {
			if t, err := time.Parse("2006-01-02", r.End.Date); err == nil {
				events[i].EndTime = t
			}
		}
	}
	return events, nil
}

func (c *CalendarCLI) run(args ...string) ([]byte, error) {
	cmd := exec.Command(c.binaryPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s: %s", err, string(exitErr.Stderr))
		}
		return nil, err
	}
	return out, nil
}
