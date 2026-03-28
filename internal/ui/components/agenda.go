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
	Events  []domain.Event
	Cursor  int
	Width   int
	Height  int
	Focused bool
	styles  theme.Styles
}

func NewAgendaModel(s theme.Styles) AgendaModel {
	return AgendaModel{styles: s}
}

func (m AgendaModel) View() string {
	s := m.styles
	now := time.Now()
	dateStr := now.Format("Monday, Jan 2")
	title := s.Title.Render("  Today's Agenda")
	subtitle := s.Subtitle.Render("  " + dateStr)

	if len(m.Events) == 0 {
		content := s.Dim.Render("  No events today")
		return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", content)
	}

	var lines []string
	lines = append(lines, title, subtitle, "")

	for i, evt := range m.Events {
		line := m.renderEvent(evt, i == m.Cursor)
		lines = append(lines, line)

		if evt.Location != "" {
			loc := s.Location.Render(fmt.Sprintf("                  %s", evt.Location))
			lines = append(lines, loc)
		}
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
