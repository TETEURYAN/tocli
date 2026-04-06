package components

import (
	"tocli/internal/domain"
	"tocli/internal/ui/theme"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type TaskListModel struct {
	Tasks    []domain.Task
	Cursor   int
	Width    int
	Height   int
	Focused  bool
	styles   theme.Styles
	Offset   int
}

func NewTaskListModel(s theme.Styles) TaskListModel {
	return TaskListModel{styles: s}
}

func (m *TaskListModel) View() string {
	s := m.styles
	title := s.Title.Render("  Tasks")

	if len(m.Tasks) == 0 {
		content := s.Dim.Render("  No tasks loaded")
		return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
	}

	// Title + counter (+ optional legend) + blank + optional scroll row.
	chromeLines := 5
	if m.Width >= 60 {
		chromeLines = 6
	}
	contentHeight := m.Height - chromeLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	if m.Cursor >= m.Offset+contentHeight {
		m.Offset = m.Cursor - contentHeight + 1
	}
	if m.Cursor < m.Offset {
		m.Offset = m.Cursor
	}

	var lines []string
	openCount := 0
	doneCount := 0

	for _, t := range m.Tasks {
		if t.Status == domain.TaskCompleted {
			doneCount++
		} else {
			openCount++
		}
	}

	counter := s.Subtitle.Render(fmt.Sprintf("  %d open · %d done", openCount, doneCount))
	lines = append(lines, title, counter)
	if m.Width >= 60 {
		legend := s.Dim.Render("  ! urgent · * important · W work · P personal · L learn · J job · · other")
		lines = append(lines, legend)
	}
	lines = append(lines, "")

	end := m.Offset + contentHeight
	if end > len(m.Tasks) {
		end = len(m.Tasks)
	}

	for i := m.Offset; i < end; i++ {
		task := m.Tasks[i]
		line := m.renderTask(task, i == m.Cursor)
		lines = append(lines, line)
	}

	if len(m.Tasks) > contentHeight {
		scrollInfo := s.Dim.Render(fmt.Sprintf("  %d/%d", m.Cursor+1, len(m.Tasks)))
		lines = append(lines, "", scrollInfo)
	}

	return strings.Join(lines, "\n")
}

func (m TaskListModel) renderTask(task domain.Task, selected bool) string {
	s := m.styles
	pri := domain.TaskEffectivePriority(task)
	accent := theme.ListCategoryAccent(task.ListName)
	mark := theme.ListCategoryMarker(task.ListName)
	switch pri {
	case domain.TaskPriorityUrgent:
		accent = theme.T.Error
		mark = "!"
	case domain.TaskPriorityImportant:
		accent = theme.T.Warning
		mark = "*"
	}
	// Cursor + colored marker + space + title (+ optional due).
	const markCells = 2
	dueSuffix := ""
	if task.DueDate != nil {
		d := task.DueDate.In(time.Local)
		if d.Hour() == 0 && d.Minute() == 0 && d.Second() == 0 && d.Nanosecond() == 0 {
			dueSuffix = "  · " + d.Format("Jan 2")
		} else {
			dueSuffix = "  · " + d.Format("Jan 2 15:04")
		}
	}
	reserve := 10 + markCells + len([]rune(dueSuffix))
	maxWidth := m.Width - reserve
	if maxWidth < 8 {
		maxWidth = 8
	}

	title := task.Title
	if len(title) > maxWidth {
		title = title[:maxWidth-1] + "…"
	}

	markStyle := lipgloss.NewStyle().Foreground(accent)
	var textStyle lipgloss.Style
	switch {
	case task.Status == domain.TaskCompleted:
		textStyle = lipgloss.NewStyle().
			Foreground(accent).
			Strikethrough(true).
			Faint(true)
	case task.IsOverdue():
		textStyle = s.TaskOverdue
	default:
		textStyle = lipgloss.NewStyle().Foreground(accent)
	}

	if selected && m.Focused {
		markStyle = markStyle.Bold(true)
		switch {
		case task.Status == domain.TaskCompleted:
			textStyle = lipgloss.NewStyle().
				Foreground(accent).
				Strikethrough(true).
				Faint(true).
				Bold(true)
		case task.IsOverdue():
			textStyle = s.TaskOverdue.Bold(true)
		default:
			textStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
		}
	}

	markStyled := markStyle.Render(mark + " ")

	cursor := "  "
	if selected && m.Focused {
		cursor = "▸ "
	}

	line := cursor + markStyled + textStyle.Render(title)
	if dueSuffix != "" {
		line += s.Dim.Render(dueSuffix)
	}
	return line
}

func (m *TaskListModel) MoveUp() {
	if m.Cursor > 0 {
		m.Cursor--
	}
}

func (m *TaskListModel) MoveDown() {
	if m.Cursor < len(m.Tasks)-1 {
		m.Cursor++
	}
}

func (m *TaskListModel) SelectedTask() *domain.Task {
	if m.Cursor >= 0 && m.Cursor < len(m.Tasks) {
		return &m.Tasks[m.Cursor]
	}
	return nil
}
