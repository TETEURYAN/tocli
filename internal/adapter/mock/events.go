package mock

import (
	"tocli/internal/domain"
	"time"
)

type EventRepo struct {
	events []domain.Event
}

func NewEventRepo() *EventRepo {
	r := &EventRepo{}
	r.seed()
	return r
}

func (r *EventRepo) GetEvents(start, end time.Time) ([]domain.Event, error) {
	var result []domain.Event
	for _, e := range r.events {
		if (e.StartTime.Equal(start) || e.StartTime.After(start)) && e.StartTime.Before(end) {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *EventRepo) seed() {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	r.events = []domain.Event{
		{
			ID:        "evt-1",
			Title:     "Daily Standup",
			StartTime: today.Add(9 * time.Hour),
			EndTime:   today.Add(9*time.Hour + 15*time.Minute),
			Location:  "Google Meet",
		},
		{
			ID:        "evt-2",
			Title:     "Sprint Planning",
			StartTime: today.Add(10 * time.Hour),
			EndTime:   today.Add(11 * time.Hour),
			Location:  "Room 3A",
		},
		{
			ID:        "evt-3",
			Title:     "Lunch with Alex",
			StartTime: today.Add(12 * time.Hour),
			EndTime:   today.Add(13 * time.Hour),
			Location:  "Cafeteria",
		},
		{
			ID:        "evt-4",
			Title:     "Code Review Session",
			StartTime: today.Add(14 * time.Hour),
			EndTime:   today.Add(15 * time.Hour),
		},
		{
			ID:        "evt-5",
			Title:     "1:1 with Manager",
			StartTime: today.Add(16 * time.Hour),
			EndTime:   today.Add(16*time.Hour + 30*time.Minute),
			Location:  "Google Meet",
		},
		{
			ID:          "evt-6",
			Title:       "Team Building",
			StartTime:   today.Add(24 * time.Hour),
			EndTime:      today.Add(24*time.Hour + 2*time.Hour),
			Description: "Quarterly team event",
			Location:    "Rooftop Lounge",
		},
		{
			ID:        "evt-7",
			Title:     "Design Review",
			StartTime: today.Add(48*time.Hour + 10*time.Hour),
			EndTime:   today.Add(48*time.Hour + 11*time.Hour),
		},
	}
}
