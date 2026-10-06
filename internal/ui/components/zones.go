package components

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// bubblezone's Mark panics until a manager exists, so the package that marks zones owns its
// initialization; every render path (the app, tests) can then call Mark safely.
func init() { zone.NewGlobal() }

// Zone ids for mouse hit-testing. Components mark their clickable regions with
// zone.Mark; the root model calls zone.Scan on the final frame.
const (
	ZonePaneTasks  = "pane:tasks"
	ZonePaneAgenda = "pane:agenda"
	ZonePaneGraph  = "pane:graph"
)

func zoneTask(i int) string { return fmt.Sprintf("task:%d", i) }

func zoneEvent(i int) string { return fmt.Sprintf("event:%d", i) }

func zoneDay(d time.Time) string { return "day:" + d.Format("2006-01-02") }

// TaskAt returns the index of the visible task row under the mouse, if any.
func (m TaskListModel) TaskAt(msg tea.MouseMsg) (int, bool) {
	end := min(len(m.Tasks), m.Offset+max(1, m.Height))
	for i := m.Offset; i < end; i++ {
		if zone.Get(zoneTask(i)).InBounds(msg) {
			return i, true
		}
	}
	return 0, false
}

// DateAt returns the graph day cell under the mouse, if any.
func (m ContributionModel) DateAt(msg tea.MouseMsg) (time.Time, bool) {
	year := m.cursorYear()
	for d := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local); d.Year() == year; d = d.AddDate(0, 0, 1) {
		if zone.Get(zoneDay(d)).InBounds(msg) {
			return d, true
		}
	}
	return time.Time{}, false
}

// EventAt returns the index of the visible agenda event under the mouse, if any. Only today's
// agenda has clickable events; the day view shown while browsing the graph does not.
func (m AgendaModel) EventAt(msg tea.MouseMsg) (int, bool) {
	if m.OverrideDate != nil {
		return 0, false
	}
	first, last := m.visibleEvents()
	for i := first; i < last; i++ {
		if zone.Get(zoneEvent(i)).InBounds(msg) {
			return i, true
		}
	}
	return 0, false
}

// markRows pads rows to a common width (at least minW) and marks them as one zone. bubblezone
// hit-tests the rectangle between the zone's first and last cell, so rows of different widths
// would shrink the clickable area to the last row's width.
func markRows(id string, minW int, rows []string) []string {
	w := minW
	for _, r := range rows {
		w = max(w, lipgloss.Width(r))
	}
	padded := make([]string, len(rows))
	for i, r := range rows {
		padded[i] = r + strings.Repeat(" ", w-lipgloss.Width(r))
	}
	return strings.Split(zone.Mark(id, strings.Join(padded, "\n")), "\n")
}
