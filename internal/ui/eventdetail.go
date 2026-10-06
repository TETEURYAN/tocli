package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tocli/internal/domain"
	"tocli/internal/platform"
)

// The event detail panel: everything the agenda row cannot show (full date and duration, the
// location, the description, and the links in it), with a way to open those links.

// openURL is how links are opened; tests replace it so no browser is launched.
var openURL = platform.OpenURL

type detailState struct {
	event  domain.Event
	scroll int // first visible description line
}

// noticeMsg reports the outcome of a background action in the status bar.
type noticeMsg struct{ text string }

func (m *Model) openEventDetail(e domain.Event) {
	m.mode = modeEventDetail
	m.err = nil
	m.detail = detailState{event: e}
}

// agendaEvent is the event under the agenda cursor, if the agenda lists any.
func (m Model) agendaEvent() (domain.Event, bool) {
	if c := m.agenda.Cursor; c >= 0 && c < len(m.agenda.Events) {
		return m.agenda.Events[c], true
	}
	return domain.Event{}, false
}

func (m *Model) handleDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	links := m.detail.event.Links()
	switch s := msg.String(); {
	case quitFromKey(msg, m.keys.Quit):
		return m, tea.Quit
	case s == "esc" || s == "enter":
		m.mode = modeNormal
		return m, nil
	case s == "o":
		return m, m.openLink(0)
	case len(s) == 1 && s[0] >= '1' && s[0] <= '9':
		n := int(s[0] - '1')
		if n >= len(links) {
			m.notice = fmt.Sprintf("this event has no link %d", n+1)
			return m, nil
		}
		return m, m.openLink(n)
	case s == "up" || s == "k":
		m.scrollDetail(-1)
	case s == "down" || s == "j":
		m.scrollDetail(1)
	case s == "pgup":
		m.scrollDetail(-m.detailLayout().descRows)
	case s == "pgdown":
		m.scrollDetail(m.detailLayout().descRows)
	}
	return m, nil
}

// openLink opens the n-th link of the open event in the browser.
func (m Model) openLink(n int) tea.Cmd {
	links := m.detail.event.Links()
	if n < 0 || n >= len(links) {
		return func() tea.Msg { return noticeMsg{text: "this event has no links"} }
	}
	link := links[n]
	return func() tea.Msg {
		if err := openURL(link.URL); err != nil {
			return errMsg{err: err}
		}
		return noticeMsg{text: "opened " + link.Label + " in the browser"}
	}
}

func (m *Model) scrollDetail(delta int) {
	l := m.detailLayout()
	m.detail.scroll = min(max(m.detail.scroll+delta, 0), max(0, l.descTotal-l.descRows))
}

// ---- layout ---------------------------------------------------------------------------------

const detailBoxMaxW = 78

// detailLayout is the arithmetic of the panel: how wide the text is, and how many description
// lines fit once the fixed parts (title, when, where, links) have taken their rows.
type detailLayout struct {
	inner     int      // text width inside the box
	title     []string // wrapped
	desc      []string // wrapped description
	descTotal int
	descRows  int // description rows that fit (0 when there is no description)
	links     []domain.Link
}

func (m Model) detailLayout() detailLayout {
	e := m.detail.event
	boxW := min(max(8, m.width-4), detailBoxMaxW)
	l := detailLayout{inner: max(1, boxW-6), links: e.Links()}

	l.title = strings.Split(ansi.Wrap(strings.TrimSpace(e.Title), l.inner, ""), "\n")
	if d := strings.TrimSpace(e.Description); d != "" {
		l.desc = strings.Split(ansi.Wrap(strings.ReplaceAll(d, "\r\n", "\n"), l.inner, ""), "\n")
	}
	l.descTotal = len(l.desc)

	// Rows the modal can use: everything between header and status bar, minus the panel's chrome.
	budget := max(1, bodyOuterLines(m.height)-4)
	fixed := len(l.title) + 1 // title + when
	if strings.TrimSpace(e.Location) != "" {
		fixed++
	}
	if len(l.links) > 0 {
		fixed += 1 + len(l.links) + 1 // blank, one row per link, and the links' blank above
	}
	if l.descTotal > 0 {
		fixed += 2 // blank above the description and the scroll note below it
		l.descRows = min(l.descTotal, max(2, budget-fixed))
	}
	return l
}

// ---- rendering ------------------------------------------------------------------------------

func (m Model) eventDetailContent() string {
	t := m.styles.T
	e := m.detail.event
	l := m.detailLayout()
	label := func(s string) string { return lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("%-6s", s)) }

	var rows []string
	for _, line := range l.title {
		rows = append(rows, lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(line))
	}
	rows = append(rows, label("When")+m.whenLine(e, l.inner-6))
	if loc := strings.TrimSpace(e.Location); loc != "" {
		rows = append(rows, label("Where")+ansi.Truncate(loc, max(1, l.inner-6), "…"))
	}

	if l.descTotal > 0 {
		rows = append(rows, "")
		first := min(m.detail.scroll, max(0, l.descTotal-l.descRows))
		for _, line := range l.desc[first:min(first+l.descRows, l.descTotal)] {
			rows = append(rows, lipgloss.NewStyle().Foreground(t.Text).Render(line))
		}
		note := ""
		if l.descTotal > l.descRows {
			note = fmt.Sprintf("lines %d–%d of %d · ↑↓ scroll", first+1, first+l.descRows, l.descTotal)
		}
		rows = append(rows, m.styles.Dim.Render(note))
	}

	if len(l.links) > 0 {
		rows = append(rows, "")
		for i, link := range l.links {
			key := lipgloss.NewStyle().Foreground(t.Primary).Bold(true).Render(fmt.Sprint(i + 1))
			name := lipgloss.NewStyle().Foreground(t.Accent).Render(link.Label)
			rows = append(rows, ansi.Truncate(key+"  "+name+"  "+m.styles.Dim.Render(link.URL), l.inner, "…"))
		}
	}
	return strings.Join(rows, "\n")
}

// whenLine is "Tue Oct 6, 2026 · 10:00–11:00 (1h) · today", cut to width.
func (m Model) whenLine(e domain.Event, width int) string {
	t := m.styles.T
	now := time.Now()
	s := whenText(e)
	if rel := relativeDay(e.StartTime, now); rel != "" {
		s += "  ·  " + rel
	}
	line := lipgloss.NewStyle().Foreground(t.Text).Render(ansi.Truncate(s, max(1, width), "…"))
	if !e.AllDay && e.IsHappening() {
		line += "  " + lipgloss.NewStyle().Foreground(t.Success).Bold(true).Render("● now")
	}
	return line
}

func (m Model) renderEventDetailStatusBar() string {
	hints := []key.Binding{hint("esc", "close")}
	if n := len(m.detail.event.Links()); n > 0 {
		hints = append(hints, hint("o", "open link"))
		if n > 1 {
			hints = append(hints, hint("1-9", "open #"))
		}
	}
	if l := m.detailLayout(); l.descTotal > l.descRows {
		hints = append(hints, hint("↑↓", "scroll"))
	}
	return m.modalStatusBar("event", hints...)
}

// ---- pure helpers ---------------------------------------------------------------------------

// whenText describes when an event is: the date, then the time range and duration (or "all day").
func whenText(e domain.Event) string {
	start, end := e.StartTime.In(time.Local), e.EndTime.In(time.Local)
	day := func(t time.Time) string { return t.Format("Mon Jan 2, 2006") }
	sameDay := start.Year() == end.Year() && start.YearDay() == end.YearDay()

	if e.AllDay {
		if sameDay || end.Before(start) {
			return day(start) + " · all day"
		}
		days := int(time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.Local).
			Sub(time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)).Hours()/24) + 1
		return fmt.Sprintf("%s – %s · all day · %d days", start.Format("Mon Jan 2"), day(end), days)
	}
	if sameDay {
		s := fmt.Sprintf("%s · %s–%s", day(start), start.Format("15:04"), end.Format("15:04"))
		if d := durationText(end.Sub(start)); d != "" {
			s += " (" + d + ")"
		}
		return s
	}
	return fmt.Sprintf("%s %s – %s %s", start.Format("Mon Jan 2"), start.Format("15:04"), end.Format("Mon Jan 2"), end.Format("15:04"))
}

// durationText is "45m", "1h" or "1h 30m"; "" for a non-positive duration.
func durationText(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	mins := int(d.Round(time.Minute) / time.Minute)
	h, m := mins/60, mins%60
	switch {
	case h == 0:
		return fmt.Sprintf("%dm", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// relativeDay says how far a day is from today, by calendar date: "today", "tomorrow",
// "in 3 days", "2 weeks ago"... Empty beyond about a quarter, where it stops being informative.
func relativeDay(t, now time.Time) string {
	t, now = t.In(time.Local), now.In(time.Local)
	days := int(time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.Local).
		Sub(time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local)).Hours() / 24)
	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "tomorrow"
	case days == -1:
		return "yesterday"
	case days > 1 && days < 14:
		return fmt.Sprintf("in %d days", days)
	case days < -1 && days > -14:
		return fmt.Sprintf("%d days ago", -days)
	case days >= 14 && days <= 90:
		return fmt.Sprintf("in %d weeks", days/7)
	case days <= -14 && days >= -90:
		return fmt.Sprintf("%d weeks ago", -days/7)
	}
	return ""
}
