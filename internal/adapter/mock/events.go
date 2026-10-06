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

	// A wider spread of past and future appointments, so the search palette has something to find.
	day := func(days int, hour, minutes int) (time.Time, time.Time) {
		start := today.AddDate(0, 0, days).Add(time.Duration(hour) * time.Hour)
		return start, start.Add(time.Duration(minutes) * time.Minute)
	}
	add := func(id, title, location, description string, days, hour, minutes int, allDay bool) {
		start, end := day(days, hour, minutes)
		r.events = append(r.events, domain.Event{
			ID: id, Title: title, Location: location, Description: description,
			StartTime: start, EndTime: end, AllDay: allDay,
		})
	}
	add("evt-8", "Daily Standup", "Google Meet", "", -1, 9, 15, false)
	add("evt-9", "Daily Standup", "Google Meet", "", -2, 9, 15, false)
	add("evt-10", "Reunião de alinhamento", "Sala 2", "Alinhar entregas da semana", -7, 15, 60, false)
	add("evt-11", "Sprint Planning", "Room 3A", "", -14, 10, 60, false)
	add("evt-12", "Consulta com dentista", "Clínica Sorriso", "Levar exames", -30, 14, 45, false)
	add("evt-13", "Reunião de planejamento trimestral", "Auditório", "Metas do próximo trimestre", -90, 9, 120, false)
	add("evt-14", "Planning offsite", "Rooftop Lounge", "Two-day planning session", -200, 9, 480, false)
	add("evt-15", "Sprint Planning", "Room 3A", "", 14, 10, 60, false)
	add("evt-16", "Apresentação do TCC", "Auditório", "Defesa final", 21, 14, 90, false)
	add("evt-17", "Aniversário da Katie", "", "", 20, 0, 0, true)
	add("evt-18", "Workshop de design", "Sala 4", "Design system e acessibilidade", 60, 13, 180, false)
	add("evt-19", "Revisão de código", "Google Meet", "Pull requests pendentes: https://github.com/example/tocli/pulls", 3, 11, 45, false)

	// Detail-panel samples: a video call, a long multi-line description, and a calendar page.
	for i := range r.events {
		e := &r.events[i]
		switch e.ID {
		case "evt-1", "evt-5", "evt-19":
			e.MeetLink = "https://meet.google.com/abc-defg-hij"
		case "evt-2":
			e.Description = "Plan the next sprint.\n\nAgenda:\n- Review the backlog and re-estimate the top items\n" +
				"- Capacity: who is out, who is on call\n- Pick the sprint goal\n- Risks and dependencies on other teams\n" +
				"- Demo plan for the end of the sprint\n\nBring your estimates. Notes live in https://docs.example.com/sprint-planning (read-only for guests).\n" +
				"If the room is taken, we move to the 4th floor kitchen and use the shared screen there.\n\n" +
				"Last sprint's retro actions:\n- Smaller pull requests\n- Keep the board up to date before the standup\n- Pair on the flaky test"
			e.URL = "https://calendar.google.com/calendar/event?eid=evt-2"
		case "evt-6":
			e.URL = "https://calendar.google.com/calendar/event?eid=evt-6"
		}
	}
}
