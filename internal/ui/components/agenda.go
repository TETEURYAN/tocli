package components

import (
	"tocli/internal/domain"
	"tocli/internal/ui/theme"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type AgendaModel struct {
	Events       []domain.Event
	Cursor       int
	Width        int
	Height       int
	Focused      bool
	styles       theme.Styles
	scrollOffset int
}

func NewAgendaModel(s theme.Styles) AgendaModel {
	return AgendaModel{styles: s}
}

func (m *AgendaModel) View() string {
	s := m.styles
	now := time.Now()
	dateStr := now.Format("Monday, Jan 2")
	title := s.Title.Render("  Today's Agenda")
	subtitle := s.Subtitle.Render("  " + dateStr)

	if len(m.Events) == 0 {
		content := s.Dim.Render("  No events today")
		return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", content)
	}

	headerLines := 3
	scrollHint := 0
	contentHeight := m.Height - headerLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	type row struct {
		eventIdx int
		isLoc    bool
	}
	var rows []row
	for i, evt := range m.Events {
		rows = append(rows, row{eventIdx: i, isLoc: false})
		if evt.Location != "" {
			rows = append(rows, row{eventIdx: i, isLoc: true})
		}
	}

	selStart := -1
	selEnd := -1
	for i, r := range rows {
		if r.eventIdx != m.Cursor {
			continue
		}
		if selStart < 0 {
			selStart = i
		}
		selEnd = i
	}
	if selStart < 0 {
		selStart, selEnd = 0, 0
	}

	if len(rows) > contentHeight {
		scrollHint = 1
		contentHeight = m.Height - headerLines - scrollHint
		if contentHeight < 1 {
			contentHeight = 1
		}
	}

	off := m.scrollOffset
	if selStart < off {
		off = selStart
	}
	if selEnd >= off+contentHeight {
		off = selEnd - contentHeight + 1
	}
	if off < 0 {
		off = 0
	}
	if off > 0 && off > len(rows)-contentHeight {
		off = max(0, len(rows)-contentHeight)
	}
	m.scrollOffset = off

	var lines []string
	lines = append(lines, title, subtitle, "")

	end := off + contentHeight
	if end > len(rows) {
		end = len(rows)
	}
	for i := off; i < end; i++ {
		r := rows[i]
		evt := m.Events[r.eventIdx]
		if r.isLoc {
			loc := s.Location.Render(fmt.Sprintf("                  %s", evt.Location))
			lines = append(lines, loc)
			continue
		}
		lines = append(lines, m.renderEvent(evt, r.eventIdx == m.Cursor))
	}

	if scrollHint > 0 {
		info := s.Dim.Render(fmt.Sprintf("  %d-%d/%d", off+1, end, len(rows)))
		lines = append(lines, "", info)
	}

	return strings.Join(lines, "\n")
}

func (m AgendaModel) renderEvent(evt domain.Event, selected bool) string {
	s := m.styles

	timeRange := evt.FormatTimeRange()
	timeStyle := s.EventTime
	titleStyle := s.EventTitle

	if evt.IsHappening() {
		timeStyle = s.EventNow
		titleStyle = s.EventNow
		timeRange = "● " + timeRange
	} else if evt.IsPast() {
		timeStyle = s.EventPast
		titleStyle = s.EventPast
	} else {
		timeRange = "  " + timeRange
	}

	cursor := "  "
	if selected && m.Focused {
		cursor = "▸ "
		titleStyle = s.TaskSelected
	}

	return cursor + timeStyle.Render(timeRange) + " " + titleStyle.Render(evt.Title)
}

func (m *AgendaModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *AgendaModel) MoveDown() {
	if m.Cursor < len(m.Events)-1 {
		m.Cursor++
	}
}
