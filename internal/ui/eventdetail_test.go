package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"tocli/internal/domain"
)

// fakeOpener records the URLs "opened" instead of launching a browser.
func fakeOpener(t *testing.T, fail error) *[]string {
	t.Helper()
	var opened []string
	old := openURL
	openURL = func(u string) error {
		if fail != nil {
			return fail
		}
		opened = append(opened, u)
		return nil
	}
	t.Cleanup(func() { openURL = old })
	return &opened
}

func linkedEvent() domain.Event {
	start := time.Now().Add(2 * time.Hour).Truncate(time.Minute)
	return domain.Event{
		ID: "e1", Title: "Sprint Planning", Location: "Room 3A",
		Description: "Plan the sprint.\nNotes: https://docs.example.com/plan",
		MeetLink:    "https://meet.google.com/abc",
		URL:         "https://calendar.google.com/e/1",
		StartTime:   start, EndTime: start.Add(time.Hour),
	}
}

func detailModel(t *testing.T, e domain.Event) Model {
	t.Helper()
	m := newTestModel(t)
	m.openEventDetail(e)
	return m
}

// ---- pure helpers ---------------------------------------------------------------------------

func TestWhenText(t *testing.T) {
	d := func(day, h, min int) time.Time { return time.Date(2026, 10, day, h, min, 0, 0, time.Local) }
	cases := []struct {
		name string
		e    domain.Event
		want string
	}{
		{"timed", domain.Event{StartTime: d(6, 10, 0), EndTime: d(6, 11, 0)}, "Tue Oct 6, 2026 · 10:00–11:00 (1h)"},
		{"90 minutes", domain.Event{StartTime: d(6, 9, 0), EndTime: d(6, 10, 30)}, "Tue Oct 6, 2026 · 09:00–10:30 (1h 30m)"},
		{"45 minutes", domain.Event{StartTime: d(6, 16, 0), EndTime: d(6, 16, 45)}, "Tue Oct 6, 2026 · 16:00–16:45 (45m)"},
		{"all day", domain.Event{AllDay: true, StartTime: d(6, 0, 0), EndTime: d(7, 0, 0).Add(-time.Nanosecond)}, "Tue Oct 6, 2026 · all day"},
		{"multi-day all day", domain.Event{AllDay: true, StartTime: d(6, 0, 0), EndTime: d(8, 0, 0).Add(-time.Nanosecond)}, "Tue Oct 6 – Wed Oct 7, 2026 · all day · 2 days"},
		{"across midnight", domain.Event{StartTime: d(6, 22, 0), EndTime: d(7, 1, 0)}, "Tue Oct 6 22:00 – Wed Oct 7 01:00"},
	}
	for _, c := range cases {
		if got := whenText(c.e); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDurationText(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0: "", -time.Hour: "", 30 * time.Minute: "30m", time.Hour: "1h", 90 * time.Minute: "1h 30m", 25 * time.Hour: "25h",
	} {
		if got := durationText(d); got != want {
			t.Errorf("durationText(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestRelativeDay(t *testing.T) {
	now := time.Date(2026, 10, 6, 9, 0, 0, 0, time.Local)
	at := func(days int) time.Time { return now.AddDate(0, 0, days).Add(11 * time.Hour) }
	for days, want := range map[int]string{
		0: "today", 1: "tomorrow", -1: "yesterday", 3: "in 3 days", -5: "5 days ago",
		14: "in 2 weeks", -21: "3 weeks ago", 120: "", -400: "",
	} {
		if got := relativeDay(at(days), now); got != want {
			t.Errorf("%+d days: %q, want %q", days, got, want)
		}
	}
	// By calendar date: 23:30 today is still today even though it is almost tomorrow.
	if got := relativeDay(time.Date(2026, 10, 6, 23, 30, 0, 0, time.Local), now); got != "today" {
		t.Errorf("late tonight = %q", got)
	}
}

// ---- opening and closing --------------------------------------------------------------------

func TestEnterOnTheAgendaOpensTheSelectedEventsDetails(t *testing.T) {
	m := newMockModel(t)
	m = drain(t, m, m.loadEvents())
	m, _ = press(t, m, "tab") // agenda
	m, _ = press(t, m, "down")
	want := m.agenda.Events[m.agenda.Cursor]

	m, _ = press(t, m, "enter")
	if m.mode != modeEventDetail || m.detail.event.ID != want.ID {
		t.Fatalf("mode=%v event=%q, want the details of %q", m.mode, m.detail.event.ID, want.ID)
	}
	m, _ = press(t, m, "esc")
	if m.mode != modeNormal {
		t.Errorf("esc left mode %v", m.mode)
	}
	m, _ = press(t, m, "enter")
	m, _ = press(t, m, "enter")
	if m.mode != modeNormal {
		t.Errorf("enter inside the panel should close it, mode %v", m.mode)
	}
}

func TestSpaceInTheAgendaDoesNotOpenDetails(t *testing.T) {
	m := newMockModel(t)
	m = drain(t, m, m.loadEvents())
	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "space")
	if m.mode != modeNormal {
		t.Errorf("space in the agenda opened mode %v", m.mode)
	}
}

func TestEnterInTheTasksPaneStillCompletesTheTask(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")
	m, cmd := press(t, m, "enter")
	if m.mode != modeNormal || cmd == nil {
		t.Fatalf("mode=%v cmd=%v, want the toggle command", m.mode, cmd != nil)
	}
	m = feed(t, m, cmd)
	for _, x := range m.tasks.Tasks {
		if x.ID == task.ID && x.Status != domain.TaskCompleted {
			t.Error("enter did not complete the task")
		}
	}
}

func TestEnterOnAnEmptyAgendaDoesNothing(t *testing.T) {
	m := newMockModel(t)
	m.agenda.Events = nil
	m.activePane = AgendaPane
	m, _ = press(t, m, "enter")
	if m.mode != modeNormal {
		t.Errorf("mode = %v", m.mode)
	}
}

func TestQuitStillWorksInsideTheDetailPanel(t *testing.T) {
	m := detailModel(t, linkedEvent())
	_, cmd := press(t, m, "q")
	if cmd == nil {
		t.Fatal("q returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q produced %T, want a quit", cmd())
	}
}

// ---- links ----------------------------------------------------------------------------------

func TestOKeyOpensTheVideoCallFirst(t *testing.T) {
	opened := fakeOpener(t, nil)
	m := detailModel(t, linkedEvent())
	m, cmd := press(t, m, "o")
	m = feed(t, m, cmd)

	if len(*opened) != 1 || (*opened)[0] != "https://meet.google.com/abc" {
		t.Errorf("opened %v, want the Meet link first", *opened)
	}
	if !strings.Contains(m.notice, "Video call") {
		t.Errorf("notice = %q", m.notice)
	}
	if m.mode != modeEventDetail {
		t.Error("opening a link should leave the panel open")
	}
}

func TestNumberKeysOpenTheNumberedLink(t *testing.T) {
	opened := fakeOpener(t, nil)
	m := detailModel(t, linkedEvent())
	links := m.detail.event.Links()
	if len(links) != 3 {
		t.Fatalf("expected 3 links (call, doc, calendar), got %v", links)
	}
	for i, l := range links {
		var cmd tea.Cmd
		m, cmd = press(t, m, string(rune('1'+i)))
		m = feed(t, m, cmd)
		if got := (*opened)[len(*opened)-1]; got != l.URL {
			t.Errorf("key %d opened %q, want %q", i+1, got, l.URL)
		}
	}
}

func TestANumberBeyondTheLinksSaysSoAndOpensNothing(t *testing.T) {
	opened := fakeOpener(t, nil)
	m := detailModel(t, linkedEvent())
	m, cmd := press(t, m, "9")
	if cmd != nil || len(*opened) != 0 || !strings.Contains(m.notice, "no link 9") {
		t.Errorf("cmd=%v opened=%v notice=%q", cmd != nil, *opened, m.notice)
	}
}

func TestAnEventWithoutLinksHasNothingToOpen(t *testing.T) {
	opened := fakeOpener(t, nil)
	m := detailModel(t, domain.Event{Title: "Lunch", StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)})
	m, cmd := press(t, m, "o")
	m = feed(t, m, cmd)
	if len(*opened) != 0 || !strings.Contains(m.notice, "no links") {
		t.Errorf("opened=%v notice=%q", *opened, m.notice)
	}
	if strings.Contains(stripANSI(m.renderEventDetailStatusBar()), "open link") {
		t.Error("the status bar offers to open a link that does not exist")
	}
}

func TestAnOpenerFailureIsShownNotSwallowed(t *testing.T) {
	fakeOpener(t, errors.New("no browser"))
	m := detailModel(t, linkedEvent())
	m, cmd := press(t, m, "o")
	m = feed(t, m, cmd)
	if m.err == nil || !strings.Contains(m.err.Error(), "no browser") {
		t.Errorf("err = %v", m.err)
	}
}

func TestNoticeMessageClearsAPreviousError(t *testing.T) {
	m := newTestModel(t)
	m.err = errors.New("old")
	next, _ := m.Update(noticeMsg{text: "done"})
	m = asModel(t, next)
	if m.err != nil || m.notice != "done" {
		t.Errorf("err=%v notice=%q", m.err, m.notice)
	}
}

// ---- content and scrolling ------------------------------------------------------------------

func TestDetailContentShowsEverythingTheAgendaRowHides(t *testing.T) {
	m := detailModel(t, linkedEvent())
	out := stripANSI(m.eventDetailContent())
	for _, want := range []string{
		"Sprint Planning", "When", "(1h)", "Where", "Room 3A", "Plan the sprint.",
		"1  Video call", "meet.google.com/abc", "2  docs.example.com", "3  Calendar",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("detail is missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "today") {
		t.Errorf("an event later today should say so:\n%s", out)
	}
}

func TestDetailOmitsSectionsThatAreEmpty(t *testing.T) {
	m := detailModel(t, domain.Event{Title: "Quiet", StartTime: time.Now().Add(time.Hour), EndTime: time.Now().Add(2 * time.Hour)})
	out := stripANSI(m.eventDetailContent())
	for _, no := range []string{"Where", "Video call", "Calendar", "lines "} {
		if strings.Contains(out, no) {
			t.Errorf("an event without that data still shows %q:\n%s", no, out)
		}
	}
}

func longEvent() domain.Event {
	var b strings.Builder
	for i := 1; i <= 40; i++ {
		b.WriteString("Description line number ")
		b.WriteString(strings.Repeat("x", i%7))
		b.WriteString(" ")
		b.WriteString(string(rune('A' + i%26)))
		b.WriteString("\n")
	}
	e := linkedEvent()
	e.Description = b.String()
	return e
}

func TestLongDescriptionsScrollAndClamp(t *testing.T) {
	m := detailModel(t, longEvent())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = asModel(t, next)
	m.detail = detailState{event: longEvent()}

	l := m.detailLayout()
	if l.descTotal <= l.descRows {
		t.Fatalf("test setup: %d lines should not fit %d rows", l.descTotal, l.descRows)
	}
	if !strings.Contains(stripANSI(m.eventDetailContent()), "↑↓ scroll") {
		t.Error("a scrollable description should say so")
	}

	m, _ = press(t, m, "up")
	if m.detail.scroll != 0 {
		t.Errorf("scrolling above the top moved to %d", m.detail.scroll)
	}
	m, _ = press(t, m, "down")
	m, _ = press(t, m, "j")
	if m.detail.scroll != 2 {
		t.Errorf("scroll = %d after two steps down", m.detail.scroll)
	}
	for i := 0; i < 100; i++ {
		m, _ = press(t, m, "down")
	}
	if want := l.descTotal - l.descRows; m.detail.scroll != want {
		t.Errorf("scroll = %d at the bottom, want %d", m.detail.scroll, want)
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	m = asModel(t, next)
	if m.detail.scroll != l.descTotal-2*l.descRows {
		t.Errorf("pgup scroll = %d", m.detail.scroll)
	}
	// The visible window really changes with the scroll offset.
	m.detail.scroll = 0
	top := stripANSI(m.eventDetailContent())
	m.detail.scroll = 10
	if stripANSI(m.eventDetailContent()) == top {
		t.Error("scrolling did not change what is shown")
	}
}

func TestDetailNeverExceedsTheRowsTheModalCanUse(t *testing.T) {
	for _, e := range []domain.Event{longEvent(), linkedEvent(), {Title: strings.Repeat("A very long title ", 12), StartTime: time.Now(), EndTime: time.Now().Add(time.Hour)}} {
		for w := 12; w <= 140; w += 16 {
			for h := 8; h <= 50; h += 7 {
				m := newTestModel(t)
				next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
				m = asModel(t, next)
				m.openEventDetail(e)

				frame := m.render()
				lines := strings.Split(frame, "\n")
				if len(lines) != h {
					t.Fatalf("%dx%d: frame has %d lines", w, h, len(lines))
				}
				for i, l := range lines {
					if lw := lipgloss.Width(l); lw > w {
						t.Fatalf("%dx%d: line %d is %d cells", w, h, i, lw)
					}
				}
			}
		}
	}
}

func TestDetailStatusBarHints(t *testing.T) {
	m := detailModel(t, linkedEvent())
	bar := stripANSI(m.renderEventDetailStatusBar())
	for _, want := range []string{"esc close", "open link", "open #"} {
		if !strings.Contains(bar, want) {
			t.Errorf("status bar %q lacks %q", bar, want)
		}
	}
	m.detail = detailState{event: domain.Event{Title: "x", MeetLink: "https://meet.google.com/a"}}
	if bar := stripANSI(m.renderEventDetailStatusBar()); strings.Contains(bar, "open #") {
		t.Errorf("a single link needs no numbered hint: %q", bar)
	}
}

// ---- other ways in --------------------------------------------------------------------------

func TestTabInTheSearchOpensDetailsInsteadOfJumping(t *testing.T) {
	e := searchEvent("a", "Offsite", "", 3)
	e.MeetLink = "https://meet.google.com/zzz"
	m := searchModel(t, e)
	m = typeText(t, m, "offsite")

	m, _ = press(t, m, "tab")
	if m.mode != modeEventDetail || m.detail.event.ID != "a" {
		t.Fatalf("mode=%v event=%q", m.mode, m.detail.event.ID)
	}
	if m.search.input.Value() != "" {
		t.Error("the palette kept its query after closing")
	}
}

func TestTabInCommandModeAndWithoutResultsDoesNothing(t *testing.T) {
	m := searchModel(t, searchEvent("a", "Offsite", "", 3))
	m = typeText(t, m, "zzz")
	m, _ = press(t, m, "tab")
	if m.mode != modeSearch {
		t.Errorf("tab with no result left mode %v", m.mode)
	}
	m = searchModel(t, searchEvent("a", "Offsite", "", 3))
	m = typeText(t, m, ">")
	m, _ = press(t, m, "tab")
	if m.mode != modeSearch {
		t.Errorf("tab in command mode left mode %v", m.mode)
	}
}

func TestPaletteCommandShowsTheSelectedEventsDetails(t *testing.T) {
	m := newMockModel(t)
	m = drain(t, m, m.loadEvents())
	m, _ = runPalette(t, m, "details")
	if m.mode != modeEventDetail || m.detail.event.ID != m.agenda.Events[m.agenda.Cursor].ID {
		t.Errorf("mode=%v event=%q", m.mode, m.detail.event.ID)
	}

	empty := newMockModel(t)
	empty.agenda.Events = nil
	for _, c := range empty.matchCommands("details") {
		t.Errorf("the details command is offered with no events: %q", c.cmd.title)
	}
}

func TestClickingAnAgendaEventOpensItsDetails(t *testing.T) {
	m := newMockModel(t)
	m = drain(t, m, m.loadEvents())
	want := m.agenda.Events[1]

	// Zones are registered when a frame is scanned; find the row on screen.
	frame := stripANSI(zone.Scan(m.render()))
	row, col := -1, -1
	for y, line := range strings.Split(frame, "\n") {
		if i := strings.Index(line, want.Title); i >= 0 && strings.Contains(line, ":") {
			row, col = y, len([]rune(line[:i]))
			break
		}
	}
	if row < 0 {
		t.Fatalf("event %q not on screen:\n%s", want.Title, frame)
	}

	var got Model
	for i := 0; i < 60; i++ { // bubblezone resolves zones on its own goroutine
		next, _ := m.Update(tea.MouseClickMsg{X: col + 1, Y: row, Button: tea.MouseLeft})
		got = asModel(t, next)
		if got.mode == modeEventDetail {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got.mode != modeEventDetail || got.detail.event.ID != want.ID {
		t.Fatalf("clicking %q: mode=%v event=%q", want.Title, got.mode, got.detail.event.ID)
	}
	if got.activePane != AgendaPane || got.agenda.Cursor != 1 {
		t.Errorf("pane=%v cursor=%d, want the agenda focused on that event", got.activePane, got.agenda.Cursor)
	}
}

func TestMockEventsCarryLinksForTheDemo(t *testing.T) {
	m := newMockModel(t)
	m = drain(t, m, m.loadEvents())
	var withLinks int
	for _, e := range m.agenda.Events {
		if len(e.Links()) > 0 {
			withLinks++
		}
	}
	if withLinks == 0 {
		t.Error("no mock event has a link, the detail panel cannot be demoed offline")
	}
}
