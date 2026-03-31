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
	DayTasks     []domain.Task
	OverrideDate *time.Time
	Cursor       int
	Width        int
	Height       int
	Focused      bool
	styles       theme.Styles
}

func NewAgendaModel(s theme.Styles) AgendaModel {
	return AgendaModel{styles: s}
}

func (m AgendaModel) View() string {
	if m.OverrideDate != nil {
		return m.renderDayDetail()
	}
	return m.renderTodayAgenda()
}

func (m AgendaModel) renderTodayAgenda() string {
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

func (m AgendaModel) renderDayDetail() string {
	s := m.styles
	dateStr := m.OverrideDate.Format("Monday, Jan 2, 2006")
	title := s.Title.Render("  Day Detail")
	subtitle := s.Subtitle.Render("  " + dateStr)

	var lines []string
	lines = append(lines, title, subtitle, "")

	hasContent := false

	if len(m.Events) > 0 {
		hasContent = true
		lines = append(lines, s.Dim.Render("  Events:"))
		for _, evt := range m.Events {
			lines = append(lines, m.renderEvent(evt, false))
			if evt.Location != "" {
				loc := s.Location.Render(fmt.Sprintf("                  %s", evt.Location))
				lines = append(lines, loc)
			}
		}
	}

	if len(m.DayTasks) > 0 {
		hasContent = true
		if len(m.Events) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, s.Dim.Render("  Completed tasks:"))
		for _, task := range m.DayTasks {
			lines = append(lines, "  "+s.TaskDone.Render("✓ "+task.Title))
		}
	}

	if !hasContent {
		lines = append(lines, s.Dim.Render("  No events or tasks"))
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
