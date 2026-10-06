package ui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"tocli/internal/ui/components"
	"tocli/internal/ui/theme"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		if m.mode == modeCreateTask {
			w := min(64, max(24, m.width-14))
			if w > 0 {
				m.createInput.SetWidth(w)
				m.dueInput.SetWidth(w)
			}
		}
		if m.mode == modeWriteNote {
			w := min(70, max(24, m.width-14))
			if w > 0 {
				m.noteInput.SetWidth(w)
			}
		}
		m.updateLayout()
		return m, nil

	case tea.KeyPressMsg:
		switch m.mode {
		case modeConfirmDelete:
			return m.handleDeleteConfirmKey(msg)
		case modeWriteNote:
			return m.handleNoteKey(msg)
		case modeCreateTask:
			return m.handleCreateKey(msg)
		case modeHelp:
			return m.handleHelpKey(msg)
		case modeSearch:
			return m.handleSearchKey(msg)
		case modeEventDetail:
			return m.handleDetailKey(msg)
		}
		return m.handleKey(msg)

	case tea.BackgroundColorMsg:
		if m.themeMode == "auto" {
			m.applyTheme(theme.ForBackground(msg.IsDark()))
		}
		return m, nil

	case tea.MouseClickMsg:
		if m.mode == modeSearch {
			return m.handleSearchClick(msg)
		}
		if m.modalOpen() {
			return m, nil
		}
		return m.handleMouseClick(msg)

	case tea.MouseWheelMsg:
		if m.mode == modeSearch {
			return m.handleSearchWheel(msg)
		}
		if m.modalOpen() {
			return m, nil
		}
		return m.handleMouseWheel(msg)

	case searchTickMsg:
		if m.mode != modeSearch || msg.gen != m.search.gen || m.search.anim >= 1 {
			return m, nil
		}
		m.search.anim = min(1, m.search.anim+searchAnimStep)
		if m.search.anim < 1 {
			return m, searchTick(m.search.gen)
		}
		return m, nil

	case searchLoadedMsg:
		m.search.loading = false
		if msg.err != nil {
			m.search.err = msg.err
		} else {
			m.search.err = nil
			m.search.index = msg.index
		}
		m.refreshSearchResults()
		return m, nil

	case tasksLoadedMsg:
		m.loading = false
		m.tasks.Tasks = filterActiveTasks(msg.tasks)
		m.taskLists = msg.lists
		m.clampCreateListIndex()
		m.clampTaskCursor()
		return m, nil

	case createTaskDoneMsg:
		m.tasks.Tasks = filterActiveTasks(msg.tasks)
		m.taskLists = msg.lists
		m.clampCreateListIndex()
		if m.mode == modeCreateTask {
			m.mode = modeNormal
		}
		m.editTask = nil
		m.createDueFocused = false
		m.createInput.Blur()
		m.createInput.SetValue("")
		m.dueInput.Blur()
		m.dueInput.SetValue("")
		m.clampTaskCursor()
		if msg.selectID != "" {
			// The list is re-sorted after an edit; keep the cursor on the task that was edited.
			for i, t := range m.tasks.Tasks {
				if t.ID == msg.selectID {
					m.tasks.Cursor = i
					break
				}
			}
		}
		return m, nil

	case eventsLoadedMsg:
		if m.agenda.OverrideDate == nil {
			m.agenda.Events = msg.events
		}
		return m, nil

	case contributionLoadedMsg:
		if msg.data.Year != m.graphYear {
			return m, nil // a slow answer for a year the user already left
		}
		m.contribution.Data = msg.data
		m.contribution.Loading = false
		m.contribution.ClampCursor()
		m.progress.Progress = m.progressUC.Calculate()
		return m, nil

	case ratingLoadedMsg:
		if msg.data.Year != m.graphYear {
			return m, nil
		}
		m.contribution.RatingData = msg.data
		return m, nil

	case noticeMsg:
		m.notice, m.err = msg.text, nil
		return m, nil

	case yearTickMsg:
		if msg.seq != m.yearSeq {
			return m, nil // the user kept switching; only the last pause loads
		}
		return m, tea.Batch(m.loadGraphYear(), m.loadDayDetail(m.contribution.CursorDate))

	case noteSavedMsg:
		m.contribution.RatingData = msg.data
		if m.mode == modeWriteNote {
			m.mode = modeNormal
		}
		m.noteInput.Blur()
		m.noteInput.Reset()
		return m, nil

	case dayDetailLoadedMsg:
		if m.activePane == GraphPane && msg.date.Equal(m.contribution.CursorDate) {
			m.agenda.OverrideDate = &msg.date
			m.agenda.Events = msg.events
			m.agenda.DayTasks = msg.tasks
		}
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		m.notice = ""
		return m, nil

	case exportDoneMsg:
		m.err = nil
		m.notice = fmt.Sprintf("exported %s", msg.path)
		return m, nil

	case tickMsg:
		m.progress.Progress = m.progressUC.Calculate()
		return m, tickCmd()
	}

	return m, nil
}

func quitFromKey(msg tea.KeyPressMsg, quit key.Binding) bool {
	if key.Matches(msg, quit) {
		return true
	}
	// Fallback in case the binding matcher misses a plain "q".
	return msg.Text == "q" || msg.Text == "Q"
}

// ratingDigit reports whether msg is a "1".."5" rune keypress, returning the
// score it represents.
func ratingDigit(msg tea.KeyPressMsg) (int, bool) {
	if len(msg.Text) != 1 {
		return 0, false
	}
	r := rune(msg.Text[0])
	if r < '1' || r > '5' {
		return 0, false
	}
	return int(r - '0'), true
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.activePane == GraphPane && m.contribution.Mode == components.ModeDaily {
		if score, ok := ratingDigit(msg); ok {
			return m, m.setDayRating(m.contribution.CursorDate, score)
		}
	}

	switch {
	case quitFromKey(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp
		return m, nil

	case key.Matches(msg, m.keys.Search):
		return m, m.openSearch("")

	case key.Matches(msg, m.keys.Command):
		return m, m.openSearch(">")

	case key.Matches(msg, m.keys.ToggleGraphMode):
		if m.activePane == GraphPane {
			m.contribution.ToggleMode()
		}
		return m, nil

	case m.activePane == TaskPane && key.Matches(msg, m.keys.EditTask):
		// "e" is also the graph's CSV export, so this has to come first and be pane-gated.
		if sel := m.tasks.SelectedTask(); sel != nil {
			m.startEditTask(*sel)
		}
		return m, nil

	case key.Matches(msg, m.keys.ExportCSV):
		if m.activePane == GraphPane && m.contribution.Mode == components.ModeDaily {
			return m, m.exportRatingsCSV()
		}
		return m, nil

	case key.Matches(msg, m.keys.WriteNote):
		if m.activePane == GraphPane && m.contribution.Mode == components.ModeDaily {
			m.startWriteNote()
		}
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		return m, m.setPane((m.activePane + 1) % paneCount)

	case key.Matches(msg, m.keys.ShiftTab):
		return m, m.setPane((m.activePane - 1 + paneCount) % paneCount)

	case key.Matches(msg, m.keys.Refresh):
		return m, m.refreshAll()

	case key.Matches(msg, m.keys.NewTask):
		if m.activePane == TaskPane {
			m.startCreateTask()
		}
		return m, nil

	case key.Matches(msg, m.keys.DeleteTask):
		if m.activePane == TaskPane && m.tasks.SelectedTask() != nil {
			m.mode = modeConfirmDelete
			m.err = nil
		}
		return m, nil

	case key.Matches(msg, m.keys.Up):
		switch m.activePane {
		case TaskPane:
			m.tasks.MoveUp()
		case AgendaPane:
			m.agenda.MoveUp()
		case GraphPane:
			m.contribution.MoveUp()
			return m, m.loadDayDetail(m.contribution.CursorDate)
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		switch m.activePane {
		case TaskPane:
			m.tasks.MoveDown()
		case AgendaPane:
			m.agenda.MoveDown()
		case GraphPane:
			m.contribution.MoveDown()
			return m, m.loadDayDetail(m.contribution.CursorDate)
		}
		return m, nil

	case m.activePane == GraphPane && key.Matches(msg, m.keys.PrevYear):
		return m, m.shiftYear(-1)

	case m.activePane == GraphPane && key.Matches(msg, m.keys.NextYear):
		return m, m.shiftYear(1)

	case key.Matches(msg, m.keys.Left):
		if m.activePane == GraphPane {
			m.contribution.MoveLeft()
			return m, m.loadDayDetail(m.contribution.CursorDate)
		}
		return m, nil

	case key.Matches(msg, m.keys.Right):
		if m.activePane == GraphPane {
			m.contribution.MoveRight()
			return m, m.loadDayDetail(m.contribution.CursorDate)
		}
		return m, nil

	case key.Matches(msg, m.keys.Space), key.Matches(msg, m.keys.Enter):
		switch {
		case m.activePane == TaskPane:
			return m, m.toggleTask()
		case m.activePane == AgendaPane && key.Matches(msg, m.keys.Enter):
			if e, ok := m.agendaEvent(); ok {
				m.openEventDetail(e)
			}
		}
		return m, nil
	}

	return m, nil
}

// setPane focuses p, reloading the agenda source when entering or leaving the graph.
func (m *Model) setPane(p Pane) tea.Cmd {
	prev := m.activePane
	m.activePane = p
	m.updateFocus()
	if p == prev {
		return nil
	}
	if p == GraphPane {
		return m.loadDayDetail(m.contribution.CursorDate)
	}
	if prev == GraphPane {
		return m.loadEvents()
	}
	return nil
}

// Bounds for year navigation: nothing useful exists before 2000, and a few years ahead is
// plenty for planning.
const (
	minGraphYear    = 2000
	maxFutureYears  = 5
	yearNavDebounce = 150 * time.Millisecond
)

// shiftYear moves the graph one year back or forward. The new year shows immediately (empty,
// "loading…"), keeping the cursor on the same month and day; the data is fetched once the user
// pauses, so tapping [ several times does not fire a load per key.
func (m *Model) shiftYear(delta int) tea.Cmd {
	year := m.graphYear + delta
	if year < minGraphYear || year > time.Now().Year()+maxFutureYears {
		m.notice = fmt.Sprintf("the graph covers %d–%d", minGraphYear, time.Now().Year()+maxFutureYears)
		return nil
	}
	m.setGraphYear(year, sameDayInYear(m.contribution.CursorDate, year))
	m.yearSeq++
	seq := m.yearSeq
	return tea.Tick(yearNavDebounce, func(time.Time) tea.Msg { return yearTickMsg{seq: seq} })
}

func (m *Model) setGraphYear(year int, cursor time.Time) {
	m.graphYear = year
	m.contribution.SetYear(year, cursor)
}

// sameDayInYear is d's month and day in another year (Feb 29 becomes Feb 28 when needed).
func sameDayInYear(d time.Time, year int) time.Time {
	day := d.Day()
	last := time.Date(year, d.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()
	return time.Date(year, d.Month(), min(day, last), 0, 0, 0, 0, time.Local)
}

// goToDate focuses the graph on day (switching its year if needed) and shows that day in the
// agenda. It loads straight away: this is a deliberate jump, not repeated key presses.
func (m *Model) goToDate(day time.Time) tea.Cmd {
	if day.Year() != m.graphYear {
		m.setGraphYear(day.Year(), day)
		m.yearSeq++ // a pending debounced load for the old target is now obsolete
		return tea.Batch(m.setPane(GraphPane), m.loadGraphYear(), m.loadDayDetail(day))
	}
	m.contribution.CursorDate = day
	return tea.Batch(m.setPane(GraphPane), m.loadDayDetail(day))
}

func (m Model) modalOpen() bool {
	return m.mode != modeNormal
}

func (m *Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	switch {
	case zone.Get(zoneSearch).InBounds(msg):
		return m, m.openSearch("")

	case zone.Get(components.ZonePaneTasks).InBounds(msg):
		cmd := m.setPane(TaskPane)
		if i, ok := m.tasks.TaskAt(msg); ok {
			m.tasks.Cursor = i
		}
		return m, cmd

	case zone.Get(components.ZonePaneGraph).InBounds(msg):
		cmd := m.setPane(GraphPane)
		if d, ok := m.contribution.DateAt(msg); ok {
			m.contribution.CursorDate = d
			return m, tea.Batch(cmd, m.loadDayDetail(d))
		}
		return m, cmd

	case zone.Get(components.ZonePaneAgenda).InBounds(msg):
		cmd := m.setPane(AgendaPane)
		if i, ok := m.agenda.EventAt(msg); ok {
			m.agenda.Cursor = i
			m.openEventDetail(m.agenda.Events[i])
		}
		return m, cmd
	}
	return m, nil
}

// handleMouseWheel scrolls the pane under the pointer, mirroring the keyboard up/down (and
// week prev/next for the graph).
func (m *Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	up := msg.Button == tea.MouseWheelUp
	if !up && msg.Button != tea.MouseWheelDown {
		return m, nil
	}
	switch {
	case zone.Get(components.ZonePaneTasks).InBounds(msg):
		if up {
			m.tasks.MoveUp()
		} else {
			m.tasks.MoveDown()
		}
	case zone.Get(components.ZonePaneGraph).InBounds(msg):
		if up {
			m.contribution.MoveLeft()
		} else {
			m.contribution.MoveRight()
		}
		return m, m.loadDayDetail(m.contribution.CursorDate)
	case zone.Get(components.ZonePaneAgenda).InBounds(msg):
		if up {
			m.agenda.MoveUp()
		} else {
			m.agenda.MoveDown()
		}
	}
	return m, nil
}

func (m *Model) updateFocus() {
	m.tasks.Focused = m.activePane == TaskPane
	m.agenda.Focused = m.activePane == AgendaPane
	m.contribution.Focused = m.activePane == GraphPane
	if m.activePane != GraphPane && m.agenda.OverrideDate != nil {
		m.agenda.OverrideDate = nil
		m.agenda.DayTasks = nil
	}
}

func (m *Model) clampCreateListIndex() {
	if len(m.taskLists) == 0 {
		m.createListIndex = 0
		return
	}
	if m.createListIndex >= len(m.taskLists) {
		m.createListIndex = 0
	}
}

func (m *Model) clampTaskCursor() {
	if len(m.tasks.Tasks) == 0 {
		m.tasks.Cursor = 0
		return
	}
	if m.tasks.Cursor >= len(m.tasks.Tasks) {
		m.tasks.Cursor = len(m.tasks.Tasks) - 1
	}
}
