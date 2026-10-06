package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"tocli/internal/domain"
)

// modalPanel wraps content in the modal box style, at most maxW wide, and clipped so it always
// fits between the header and the status bar.
func (m Model) modalPanel(content string, maxW int) string {
	boxW := min(max(8, m.width-4), maxW)
	// Panel chrome: 2 border cells + Padding(1, 2) = 6 columns and 4 rows. Wrap first so the
	// height clip counts the wrapped lines.
	content = lipgloss.NewStyle().Width(max(1, boxW-6)).Render(content)
	content = clipToLines(content, max(1, bodyOuterLines(m.height)-4))
	return m.styles.Panel.Width(boxW).Render(content)
}

// overlay draws box centered over frame. Zone markers are stripped from the dashboard first (the
// compositor would otherwise treat them as stray escape sequences); mouse input is ignored while
// a modal is open, so nothing needs them.
func (m Model) overlay(frame, box string) string {
	frame = zone.Scan(frame)
	x := max(0, (m.width-lipgloss.Width(box))/2)
	y := max(0, (m.height-lipgloss.Height(box))/2)
	return lipgloss.NewCompositor(
		lipgloss.NewLayer(frame),
		lipgloss.NewLayer(box).X(x).Y(y).Z(1),
	).Render()
}

func (m Model) renderHelp() string {
	var b strings.Builder

	title := m.styles.Title.Render("  Keyboard Shortcuts")
	b.WriteString(title + "\n\n")

	bindings := []struct{ key, desc string }{
		{"j / ↓", "Move down"},
		{"k / ↑", "Move up"},
		{"← / h", "Previous week (graph)"},
		{"→ / l", "Next week (graph)"},
		{"[ / ]", "Previous / next year (graph pane)"},
		{"Tab", "Next pane"},
		{"Shift+Tab", "Previous pane"},
		{"Space / Enter", "Complete or reopen task (tasks pane)"},
		{"Enter", "Event details: when, where, description, links (agenda pane)"},
		{"n", "New task (tasks pane; optional due)"},
		{"e", "Edit the selected task's title and due date (tasks pane)"},
		{"d then y/n", "Delete task (confirm)"},
		{"g", "Toggle contribution/daily rating mode (graph pane)"},
		{"1-5", "Rate selected day (graph pane, daily mode)"},
		{"t", "Write/edit a note for the selected day (graph pane, daily mode)"},
		{"e", "Export month's ratings + notes to CSV (graph pane, daily mode)"},
		{"Click / wheel", "Focus pane / pick task or day; wheel scrolls"},
		{"/", "Search events (type, ↑↓ select, enter jumps to the day)"},
		{":", "Command palette: run any action by name (or type > in the search)"},
		{"r", "Refresh data"},
		{"?", "Toggle help"},
		{"q / Ctrl+C", "Quit"},
	}

	for _, kb := range bindings {
		line := fmt.Sprintf("  %s  %s",
			m.styles.HelpKey.Render(fmt.Sprintf("%-16s", kb.key)),
			m.styles.HelpDesc.Render(kb.desc))
		b.WriteString(line + "\n")
	}

	b.WriteString("\n" + m.styles.Dim.Render("  Press ? to return"))

	return b.String()
}

// handleHelpKey: the overlay swallows input; only close and quit are live.
func (m *Model) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case quitFromKey(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help), msg.String() == "esc":
		m.mode = modeNormal
	}
	return m, nil
}

func (m *Model) handleDeleteConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if quitFromKey(msg, m.keys.Quit) {
		return m, tea.Quit
	}
	switch msg.String() {
	case "y", "Y":
		m.mode = modeNormal
		return m, m.deleteSelectedTask()
	case "n", "N", "esc":
		m.mode = modeNormal
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) handleCreateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	// Do not bind "q" here — it must go to the text field (task titles can contain "q").
	case msg.String() == "ctrl+c":
		return m, tea.Quit

	case key.Matches(msg, m.keys.Enter):
		return m, m.submitNewTask()

	case key.Matches(msg, m.keys.Tab):
		if m.createDueFocused {
			m.createDueFocused = false
			m.dueInput.Blur()
			m.createInput.Focus()
		} else {
			m.createDueFocused = true
			m.createInput.Blur()
			m.dueInput.Focus()
		}
		return m, nil

	case msg.String() == "esc":
		m.cancelCreateTask()
		return m, nil

	case key.Matches(msg, m.keys.PrevList):
		if m.editTask == nil && len(m.taskLists) > 0 {
			m.createListIndex = (m.createListIndex - 1 + len(m.taskLists)) % len(m.taskLists)
		}
		return m, nil

	case key.Matches(msg, m.keys.NextList):
		if m.editTask == nil && len(m.taskLists) > 0 {
			m.createListIndex = (m.createListIndex + 1) % len(m.taskLists)
		}
		return m, nil

	default:
		var cmd tea.Cmd
		if m.createDueFocused {
			m.dueInput, cmd = m.dueInput.Update(msg)
		} else {
			m.createInput, cmd = m.createInput.Update(msg)
		}
		return m, cmd
	}
}

func (m *Model) startCreateTask() {
	m.editTask = nil
	m.mode = modeCreateTask
	m.err = nil
	m.createDueFocused = false
	m.createInput.SetValue("")
	m.dueInput.SetValue("")
	w := min(64, max(24, m.width-14))
	if w > 0 {
		m.createInput.SetWidth(w)
		m.dueInput.SetWidth(w)
	}
	m.createListIndex = 0
	if sel := m.tasks.SelectedTask(); sel != nil {
		for i, l := range m.taskLists {
			if l.ID == sel.ListID {
				m.createListIndex = i
				break
			}
		}
	}
	m.clampCreateListIndex()
	m.dueInput.Blur()
	m.createInput.Focus()
}

// startEditTask opens the task form pre-filled with t. The list cannot be changed, only the title
// and the due date; an emptied due field removes the due date.
func (m *Model) startEditTask(t domain.Task) {
	m.startCreateTask()
	m.editTask = &t
	m.createInput.SetValue(t.Title)
	m.createInput.CursorEnd()
	m.dueInput.SetValue(formatDueField(t.DueDate))
}

// formatDueField renders a due date the way the form parses it: date only at local midnight,
// date and time otherwise.
func formatDueField(due *time.Time) string {
	if due == nil {
		return ""
	}
	d := due.In(time.Local)
	if d.Hour() == 0 && d.Minute() == 0 && d.Second() == 0 {
		return d.Format("02-01-2006")
	}
	return d.Format("02-01-2006 15:04")
}

func (m *Model) startWriteNote() {
	m.mode = modeWriteNote
	m.err = nil
	m.noteDate = m.contribution.CursorDate
	m.noteInput.SetValue(m.contribution.DayNote(m.noteDate))
	w := min(70, max(24, m.width-14))
	if w > 0 {
		m.noteInput.SetWidth(w)
	}
	m.noteInput.Focus()
}

func (m *Model) cancelWriteNote() {
	m.mode = modeNormal
	m.noteInput.Blur()
	m.noteInput.Reset()
	m.err = nil
}

func (m *Model) handleNoteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	// Do not bind "q" here — it must go to the textarea (notes can contain the letter q).
	case msg.String() == "ctrl+c":
		return m, tea.Quit

	case msg.String() == "esc":
		m.cancelWriteNote()
		return m, nil

	case msg.String() == "ctrl+s":
		return m, m.saveNote(m.noteDate, strings.TrimSpace(m.noteInput.Value()))

	default:
		var cmd tea.Cmd
		m.noteInput, cmd = m.noteInput.Update(msg)
		return m, cmd
	}
}

func (m *Model) cancelCreateTask() {
	m.mode = modeNormal
	m.editTask = nil
	m.createDueFocused = false
	m.createInput.Blur()
	m.createInput.SetValue("")
	m.dueInput.Blur()
	m.dueInput.SetValue("")
	m.err = nil
}

func (m Model) createTaskContent() string {
	listLine := m.styles.Subtitle.Render("  List: ") +
		m.styles.Dim.Render("no lists — try refresh (r)") +
		m.styles.Subtitle.Render("  ·  [ / ] change list")
	if len(m.taskLists) > 0 {
		name := m.taskLists[m.createListIndex].Name
		listLine = m.styles.Subtitle.Render("  List: ") +
			m.styles.HelpKey.Render(name) +
			m.styles.Subtitle.Render("  ·  [ / ] change list")
	}

	heading := "  New task"
	if m.editTask != nil {
		heading = "  Edit task"
		name := m.editTask.ListName
		if name == "" {
			name = "—"
		}
		listLine = m.styles.Subtitle.Render("  List: ") + m.styles.HelpKey.Render(name) +
			m.styles.Subtitle.Render("  ·  can't be changed")
	}
	title := m.styles.Title.Render(heading)
	inputLine := "  " + m.createInput.View()
	dueLine := "  " + m.dueInput.View()
	hint := m.styles.Dim.Render("  enter save  ·  tab title/due  ·  esc cancel  ·  ctrl+c quit")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		listLine,
		"",
		inputLine,
		"",
		dueLine,
		"",
		hint,
	)

	return content
}

func (m Model) writeNoteContent() string {
	title := m.styles.Title.Render("  Day note — " + m.noteDate.Format("Mon, Jan 2 2006"))
	inputLine := lipgloss.NewStyle().PaddingLeft(2).Render(m.noteInput.View())
	hint := m.styles.Dim.Render(fmt.Sprintf("  ctrl+s save  ·  esc cancel  ·  ctrl+c quit  ·  %d/%d",
		len(m.noteInput.Value()), domain.MaxRatingNoteLength))

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		inputLine,
		"",
		hint,
	)

	return content
}
