package google

import (
	"testing"
	"time"

	calendar "google.golang.org/api/calendar/v3"
	tasks "google.golang.org/api/tasks/v1"
)

// inZone runs f with time.Local set to a fixed offset, restoring it afterwards.
func inZone(t *testing.T, name string, offsetHours int, f func()) {
	t.Helper()
	old := time.Local
	time.Local = time.FixedZone(name, offsetHours*3600)
	defer func() { time.Local = old }()
	f()
}

var zones = []struct {
	name   string
	offset int
}{
	{"UTC-3 (Brasília)", -3}, {"UTC", 0}, {"UTC+9 (Tokyo)", 9}, {"UTC-12", -12}, {"UTC+14", 14},
}

func TestDueForGoogleKeepsTheLocalCalendarDate(t *testing.T) {
	for _, z := range zones {
		inZone(t, z.name, z.offset, func() {
			// Whatever the time of day the user typed, the date they typed must survive.
			for _, hour := range []int{0, 1, 12, 21, 23} {
				local := time.Date(2026, 4, 7, hour, 30, 0, 0, time.Local)
				got := dueForGoogle(local)
				if want := "2026-04-07T00:00:00Z"; got != want {
					t.Errorf("%s at %02d:30: dueForGoogle = %s, want %s", z.name, hour, got, want)
				}
			}
		})
	}
}

func TestGoogleDueMapsToLocalMidnightOfTheSameDate(t *testing.T) {
	for _, z := range zones {
		inZone(t, z.name, z.offset, func() {
			task := mapGoogleTaskToDomain(&tasks.Task{Id: "1", Title: "x", Due: "2026-04-07T00:00:00.000Z"}, "l")
			if task.DueDate == nil {
				t.Fatalf("%s: due date lost", z.name)
			}
			d := task.DueDate.In(time.Local)
			if d.Year() != 2026 || d.Month() != 4 || d.Day() != 7 || d.Hour() != 0 || d.Minute() != 0 {
				t.Errorf("%s: due = %v, want 2026-04-07 00:00 local (a date, not 21:00 of the day before)", z.name, d)
			}
		})
	}
}

func TestDueRoundTripIsStableInEveryZone(t *testing.T) {
	for _, z := range zones {
		inZone(t, z.name, z.offset, func() {
			typed := time.Date(2026, 12, 31, 0, 0, 0, 0, time.Local)
			sent := dueForGoogle(typed)
			back := mapGoogleTaskToDomain(&tasks.Task{Id: "1", Title: "x", Due: sent}, "l")
			if back.DueDate == nil || !back.DueDate.Equal(typed) {
				t.Errorf("%s: typed %v, came back %v", z.name, typed, back.DueDate)
			}
		})
	}
}

func TestTaskWithoutDueHasNoDueDate(t *testing.T) {
	if task := mapGoogleTaskToDomain(&tasks.Task{Id: "1", Title: "x"}, "l"); task.DueDate != nil {
		t.Errorf("due = %v, want nil", task.DueDate)
	}
}

func TestMeetLinkAndEventURLMapping(t *testing.T) {
	ev, err := mapGoogleEventToDomain(&calendar.Event{
		Id: "1", Summary: "Standup", HtmlLink: "https://calendar.google.com/e/1", HangoutLink: "https://meet.google.com/aaa",
		Start: &calendar.EventDateTime{DateTime: "2026-04-07T10:00:00-03:00"},
		End:   &calendar.EventDateTime{DateTime: "2026-04-07T10:30:00-03:00"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ev.URL != "https://calendar.google.com/e/1" || ev.MeetLink != "https://meet.google.com/aaa" {
		t.Errorf("URL=%q MeetLink=%q", ev.URL, ev.MeetLink)
	}

	// Without the classic hangout link, the video entry point of the conference data is used.
	conf := &calendar.Event{Id: "2", Start: &calendar.EventDateTime{Date: "2026-04-07"},
		ConferenceData: &calendar.ConferenceData{EntryPoints: []*calendar.EntryPoint{
			{EntryPointType: "phone", Uri: "tel:+1555"},
			nil,
			{EntryPointType: "video", Uri: "https://meet.google.com/bbb"},
		}}}
	ev, _ = mapGoogleEventToDomain(conf)
	if ev.MeetLink != "https://meet.google.com/bbb" {
		t.Errorf("conference MeetLink = %q", ev.MeetLink)
	}
	if ev, _ := mapGoogleEventToDomain(&calendar.Event{Id: "3", Start: &calendar.EventDateTime{Date: "2026-04-07"}}); ev.MeetLink != "" || ev.URL != "" {
		t.Errorf("an event without links got %q / %q", ev.MeetLink, ev.URL)
	}
}
