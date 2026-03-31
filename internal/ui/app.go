package ui

import (
	"tocli/internal/domain"
	"tocli/internal/ui/components"
	"tocli/internal/ui/theme"
	"tocli/internal/usecase"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type Pane int

const (
	TaskPane Pane = iota
	AgendaPane
	GraphPane
	paneCount

	// Below this width the layout stacks panels vertically.
	minSplitWidth = 68
)

// bodyOuterLines is vertical space for the main panel row (between header and status).
func bodyOuterLines(termHeight int) int {
	if termHeight <= 0 {
		return 0
	}
	// One line header + one line status → remaining rows for the body (may be 0 on very short terminals).
	return max(0, termHeight-2)
}

// clipToLines keeps the first maxLines lines. Bubble Tea drops excess lines from the top of the
// view when the string is taller than the terminal, which hides the header — clipping the body
// avoids overflowing the agreed layout (1 + bodyOuter + 1 rows).
func clipToLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

// clampToWidth truncates each line to maxCells display width (ANSI-aware), matching Bubble Tea's
// per-line truncation so content is not clipped unevenly on the right.
func clampToWidth(s string, maxCells int) string {
	if maxCells <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], maxCells, "")
	}
	return strings.Join(lines, "\n")
}

// finalizeFrame enforces terminal w×h: clamp line widths, drop excess rows from the bottom, then
// pad with full-width blank lines so the alt screen fills to the bottom without leftover paint.
func (m Model) finalizeFrame(s string) string {
	w, h := m.width, m.height
	if w <= 0 {
		return s
	}
	s = clampToWidth(s, w)
	if h <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	pad := strings.Repeat(" ", w)
	for len(lines) < h {
		lines = append(lines, pad)
	}
	return strings.Join(lines, "\n")
}

func panelInnerHeight(outer int) int {
	if outer <= 3 {
		return 1
	}
	// Border + padding: slightly tighter than before so content uses more of the panel.
	return max(1, outer-3)
}

func splitStackOuterHeights(total int) (tasks, agenda, contrib, progress int) {
	if total <= 0 {
		return 0, 0, 0, 0
	}
	// Fewer than four rows: give one row to the first N panes so the sum equals total (no forced 4-row minimum).
	if total < 4 {
		switch total {
		case 1:
			return 1, 0, 0, 0
		case 2:
			return 1, 1, 0, 0
		case 3:
			return 1, 1, 1, 0
		default:
			return 0, 0, 0, 0
		}
	}
	if total <= 10 {
		q := total / 4
		r := total % 4
		tasks, agenda, contrib, progress = q, q, q, q
		for i := 0; i < r; i++ {
			switch i % 4 {
			case 0:
				tasks++
			case 1:
				agenda++
			case 2:
				contrib++
			default:
				progress++
			}
		}
		return tasks, agenda, contrib, progress
	}
	progress = min(8, max(3, total*13/100))
	rest := total - progress
	tasks = max(4, rest*40/100)
	agenda = max(3, rest*30/100)
	contrib = rest - tasks - agenda
	if contrib < 3 {
		need := 3 - contrib
		contrib = 3
		tasks = max(4, tasks-need)
		if tasks+agenda+contrib+progress > total {
			tasks = total - progress - agenda - contrib
		}
		if tasks < 4 {
			tasks = 4
			agenda = max(3, total-progress-contrib-tasks)
		}
	}
	return tasks, agenda, contrib, progress
}

func splitColumnWidths(termW int) (left, right int) {
	// Narrower task column on laptop-sized terminals → more space for agenda + graph.
	ratioPct := 38
	if termW < 120 {
		ratioPct = 30
	}
	if termW < 90 {
		ratioPct = 28
	}
	left = termW * ratioPct / 100
	minLeft, minRight := 18, 22
	if termW < 90 {
		minLeft = 16
		minRight = 20
	}
	if left < minLeft {
		left = minLeft
	}
	right = termW - left - 1
	if right < minRight {
		right = minRight
		left = termW - right - 1
		if left < 16 {
			left = 16
			right = termW - left - 1
		}
	}
	return left, right
}

func allocateRightColumnInner(rightInner int) (agendaH, contribH, progressH int) {
	if rightInner < 6 {
		agendaH = max(2, rightInner*35/100)
		progressH = max(2, rightInner*20/100)
		contribH = max(1, rightInner-agendaH-progressH)
		return agendaH, contribH, progressH
	}
	// Right column joins agenda, contrib, progress with no blank lines between blocks.
	avail := max(4, rightInner)
	progressH = min(5, max(2, avail*12/100))
	agendaH = max(2, avail*34/100)
	contribH = avail - agendaH - progressH
	if contribH < 2 {
		short := 2 - contribH
		contribH = 2
		if agendaH-short >= 2 {
			agendaH -= short
		} else {
			progressH = max(2, progressH-short)
		}
		contribH = avail - agendaH - progressH
	}
	return agendaH, contribH, progressH
}

type Model struct {
	tasks        components.TaskListModel
	agenda       components.AgendaModel
	contribution components.ContributionModel
	progress     components.ProgressBarModel

	taskUC    *usecase.TaskUseCase
	eventUC   *usecase.EventUseCase
	contribUC *usecase.ContributionUseCase
	progressUC *usecase.ProgressUseCase

	keys       KeyMap
	styles     theme.Styles
	activePane Pane
	width      int
	height     int
	ready      bool
	loading    bool
	err        error
	showHelp   bool

	creatingTask    bool
	createInput     textinput.Model
	taskLists       []domain.TaskList
	createListIndex int
}

type tasksLoadedMsg struct {
	tasks []domain.Task
	lists []domain.TaskList
}

type createTaskDoneMsg struct {
	tasks []domain.Task
	lists []domain.TaskList
}

type eventsLoadedMsg struct {
	events []domain.Event
}

type contributionLoadedMsg struct {
	data usecase.ContributionData
}

type dayDetailLoadedMsg struct {
	date   time.Time
	events []domain.Event
	tasks  []domain.Task
}

type errMsg struct {
	err error
}

type tickMsg time.Time

func NewModel(
	taskUC *usecase.TaskUseCase,
	eventUC *usecase.EventUseCase,
	contribUC *usecase.ContributionUseCase,
	progressUC *usecase.ProgressUseCase,
) Model {
	s := theme.NewStyles()
	ti := textinput.New()
	ti.Placeholder = "What needs doing?"
	ti.CharLimit = 280
	ti.Width = 40
	return Model{
		tasks:        components.NewTaskListModel(s),
		agenda:       components.NewAgendaModel(s),
		contribution: components.NewContributionModel(s),
		progress:     components.NewProgressBarModel(s),
		taskUC:       taskUC,
		eventUC:      eventUC,
		contribUC:    contribUC,
		progressUC:   progressUC,
		keys:         DefaultKeyMap(),
		styles:       s,
		activePane:   TaskPane,
		loading:      true,
		createInput:  ti,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTasks(),
		m.loadEvents(),
		m.loadContribution(),
		tickCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		if m.creatingTask {
			w := min(64, max(24, m.width-14))
			if w > 0 {
				m.createInput.Width = w
			}
		}
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		if m.creatingTask {
			return m.handleCreateKey(msg)
		}
		return m.handleKey(msg)

	case tasksLoadedMsg:
		m.loading = false
		m.tasks.Tasks = filterActiveTasks(msg.tasks)
		m.taskLists = msg.lists
		m.clampCreateListIndex()
		return m, nil

	case createTaskDoneMsg:
		m.tasks.Tasks = filterActiveTasks(msg.tasks)
		m.taskLists = msg.lists
		m.clampCreateListIndex()
		m.creatingTask = false
		m.createInput.Blur()
		m.createInput.SetValue("")
		return m, nil

	case eventsLoadedMsg:
		if m.agenda.OverrideDate == nil {
			m.agenda.Events = msg.events
		}
		return m, nil

	case contributionLoadedMsg:
		m.contribution.Data = msg.data
		m.contribution.ClampCursor()
		m.progress.Progress = m.progressUC.Calculate()
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
		return m, nil

	case tickMsg:
		m.progress.Progress = m.progressUC.Calculate()
		return m, tickCmd()
	}

	return m, nil
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Loading..."
	}

	if m.showHelp {
		return m.finalizeFrame(clipToLines(m.renderHelp(), m.height))
	}

	if m.creatingTask {
		return m.renderCreateTaskScreen()
	}

	header := m.renderHeader()
	body := clipToLines(m.renderBody(), bodyOuterLines(m.height))
	status := m.renderStatusBar()

	return m.finalizeFrame(lipgloss.JoinVertical(lipgloss.Left, header, body, status))
}

func quitFromKey(msg tea.KeyMsg, quit key.Binding) bool {
	if key.Matches(msg, quit) {
		return true
	}
	// Bubble Tea v1 may emit rune keys where bubbles' matcher misses "q".
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		return r == 'q' || r == 'Q'
	}
	return false
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case quitFromKey(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		prev := m.activePane
		m.activePane = (m.activePane + 1) % paneCount
		m.updateFocus()
		if m.activePane == GraphPane {
			return m, m.loadDayDetail(m.contribution.CursorDate)
		}
		if prev == GraphPane {
			return m, m.loadEvents()
		}
		return m, nil

	case key.Matches(msg, m.keys.ShiftTab):
		prev := m.activePane
		m.activePane = (m.activePane - 1 + paneCount) % paneCount
		m.updateFocus()
		if m.activePane == GraphPane {
			return m, m.loadDayDetail(m.contribution.CursorDate)
		}
		if prev == GraphPane {
			return m, m.loadEvents()
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, tea.Batch(m.loadTasks(), m.loadEvents(), m.loadContribution())

	case key.Matches(msg, m.keys.NewTask):
		if m.activePane == TaskPane {
			m.startCreateTask()
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
		if m.activePane == TaskPane {
			return m, m.toggleTask()
		}
		return m, nil
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

func (m *Model) updateLayout() {
	bodyOuter := bodyOuterLines(m.height)
	fullW := max(12, m.width)

	if m.width < minSplitWidth {
		tOut, aOut, cOut, pOut := splitStackOuterHeights(bodyOuter)
		innerW := max(8, fullW-4)
		m.tasks.Width = innerW
		m.tasks.Height = panelInnerHeight(tOut)
		m.agenda.Width = innerW
		m.agenda.Height = panelInnerHeight(aOut)
		m.contribution.Width = max(8, fullW-6)
		m.contribution.Height = panelInnerHeight(cOut)
		m.progress.Width = innerW
		m.progress.Height = panelInnerHeight(pOut)
		m.updateFocus()
		return
	}

	leftOuter, rightOuter := splitColumnWidths(m.width)
	taskInner := panelInnerHeight(leftOuter)
	rightInner := panelInnerHeight(rightOuter)
	agH, coH, prH := allocateRightColumnInner(rightInner)

	m.tasks.Width = max(8, taskInner)
	m.tasks.Height = taskInner

	m.agenda.Width = max(8, rightInner)
	m.agenda.Height = agH

	m.contribution.Width = max(8, rightOuter-6)
	m.contribution.Height = coH

	m.progress.Width = max(8, rightInner)
	m.progress.Height = prH

	m.updateFocus()
}

func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(theme.T.Primary).
		Bold(true).
		Padding(0, 1).
		Render("⚡ tocli")

	compact := m.width < 96 || m.height < 26
	var mid string
	if compact {
		mid = lipgloss.NewStyle().
			Foreground(theme.T.Muted).
			Render(" · dashboard")
	} else {
		mid = lipgloss.NewStyle().
			Foreground(theme.T.Muted).
			Render(" productivity dashboard")
	}

	rightFmt := "Mon Jan 2, 15:04"
	if compact && m.width < 72 {
		rightFmt = "15:04"
	}
	right := lipgloss.NewStyle().
		Foreground(theme.T.Muted).
		Render(time.Now().Format(rightFmt))

	leftPart := title + mid
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(leftPart)-lipgloss.Width(right)-2))

	return lipgloss.NewStyle().
		Background(theme.T.Surface).
		Width(m.width).
		Padding(0, 1).
		Inline(true).
		Render(leftPart + gap + right)
}

func (m Model) renderBody() string {
	bodyOuter := bodyOuterLines(m.height)
	if m.width < minSplitWidth {
		return m.renderBodyStacked(bodyOuter)
	}
	return m.renderBodyWide(bodyOuter)
}

func (m Model) renderBodyStacked(bodyOuter int) string {
	if bodyOuter <= 0 {
		return ""
	}
	tOut, aOut, cOut, pOut := splitStackOuterHeights(bodyOuter)
	w := m.width
	parts := make([]string, 0, 4)
	if tOut > 0 {
		parts = append(parts, m.renderPanel(m.tasks.View(), w, tOut, m.activePane == TaskPane))
	}
	if aOut > 0 {
		parts = append(parts, m.renderPanel(m.agenda.View(), w, aOut, m.activePane == AgendaPane))
	}
	if cOut > 0 {
		parts = append(parts, m.renderPanel(m.contribution.View(), w, cOut, m.activePane == GraphPane))
	}
	if pOut > 0 {
		parts = append(parts, m.renderPanel(m.progress.View(), w, pOut, m.activePane == GraphPane))
	}
	if len(parts) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderBodyWide(bodyOuter int) string {
	if bodyOuter <= 0 {
		return ""
	}
	leftOuter, rightOuter := splitColumnWidths(m.width)

	leftPanel := m.renderPanel(m.tasks.View(), leftOuter, bodyOuter, m.activePane == TaskPane)

	agendaView := m.agenda.View()
	contribView := m.contribution.View()
	progressView := m.progress.View()

	rightContent := lipgloss.JoinVertical(lipgloss.Left,
		agendaView,
		contribView,
		progressView,
	)

	rightPanel := m.renderPanel(rightContent, rightOuter, bodyOuter, m.activePane != TaskPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
}

func (m Model) renderPanel(content string, width, height int, active bool) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	style := m.styles.Panel
	if active {
		style = m.styles.PanelActive
	}
	return style.Width(width).Height(height).Render(content)
}

func (m Model) renderStatusBar() string {
	var parts []string

	paneName := [...]string{"tasks", "agenda", "graph"}[m.activePane]
	paneInfo := m.styles.HelpKey.Render(fmt.Sprintf(" %s ", paneName))
	parts = append(parts, paneInfo)

	if m.loading {
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.T.Warning).Render(" loading..."))
	} else if m.err != nil {
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.T.Error).Render(fmt.Sprintf(" %v", m.err)))
	}

	var helpKeys []struct{ key, desc string }
	switch {
	case m.activePane == GraphPane && m.width < 92:
		helpKeys = []struct{ key, desc string }{
			{"←→↑↓", "navigate"},
			{"tab", "pane"},
			{"?", "help"},
			{"q", "quit"},
		}
	case m.activePane == GraphPane:
		helpKeys = []struct{ key, desc string }{
			{"←→", "week"},
			{"↑↓", "day"},
			{"tab", "switch pane"},
			{"r", "refresh"},
			{"?", "help"},
			{"q", "quit"},
		}
	case m.width < 92:
		helpKeys = []struct{ key, desc string }{
			{"tab", "pane"},
			{"n", "new"},
			{"r", "refresh"},
			{"?", "help"},
			{"q", "quit"},
		}
	default:
		helpKeys = []struct{ key, desc string }{
			{"↑↓/jk", "navigate"},
			{"tab", "switch pane"},
			{"space", "done/reopen"},
			{"n", "new task"},
			{"r", "refresh"},
			{"?", "help"},
			{"q", "quit"},
		}
	}

	var helpParts []string
	for _, h := range helpKeys {
		helpParts = append(helpParts,
			m.styles.HelpKey.Render(h.key)+" "+m.styles.HelpDesc.Render(h.desc))
	}

	help := strings.Join(helpParts, "  ")
	left := strings.Join(parts, "")
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(help)-2))

	return lipgloss.NewStyle().
		Background(theme.T.Surface).
		Width(m.width).
		Padding(0, 1).
		Inline(true).
		Render(left + gap + help)
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
		{"Tab", "Next pane"},
		{"Shift+Tab", "Previous pane"},
		{"Space / Enter", "Complete or reopen task"},
		{"n", "New task (tasks pane)"},
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

func (m Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		lists, err := m.taskUC.ListTaskLists()
		if err != nil {
			return errMsg{err: err}
		}
		tasks, err := m.taskUC.ListAllTasks()
		if err != nil {
			return errMsg{err: err}
		}
		return tasksLoadedMsg{tasks: tasks, lists: lists}
	}
}

func (m Model) loadEvents() tea.Cmd {
	return func() tea.Msg {
		events, err := m.eventUC.GetTodayEvents()
		if err != nil {
			return errMsg{err: err}
		}
		return eventsLoadedMsg{events: events}
	}
}

func (m Model) loadContribution() tea.Cmd {
	return func() tea.Msg {
		data := m.contribUC.Generate(time.Now().Year())
		return contributionLoadedMsg{data: data}
	}
}

func (m Model) loadDayDetail(date time.Time) tea.Cmd {
	return func() tea.Msg {
		events, _ := m.eventUC.GetEventsForDate(date)
		tasks, _ := m.taskUC.TasksCompletedOn(date)
		return dayDetailLoadedMsg{date: date, events: events, tasks: tasks}
	}
}

func (m Model) toggleTask() tea.Cmd {
	task := m.tasks.SelectedTask()
	if task == nil {
		return nil
	}

	taskID := task.ID
	listID := task.ListID

	if task.Status == domain.TaskOpen {
		return func() tea.Msg {
			if err := m.taskUC.CompleteTask(taskID, listID); err != nil {
				return errMsg{err: err}
			}
			lists, err := m.taskUC.ListTaskLists()
			if err != nil {
				return errMsg{err: err}
			}
			tasks, err := m.taskUC.ListAllTasks()
			if err != nil {
				return errMsg{err: err}
			}
			return tasksLoadedMsg{tasks: tasks, lists: lists}
		}
	}

	if task.Status == domain.TaskCompleted {
		return func() tea.Msg {
			if err := m.taskUC.ReopenTask(taskID, listID); err != nil {
				return errMsg{err: err}
			}
			lists, err := m.taskUC.ListTaskLists()
			if err != nil {
				return errMsg{err: err}
			}
			tasks, err := m.taskUC.ListAllTasks()
			if err != nil {
				return errMsg{err: err}
			}
			return tasksLoadedMsg{tasks: tasks, lists: lists}
		}
	}

	return nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) handleCreateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	// Do not bind "q" here — it must go to the text field (task titles can contain "q").
	case msg.String() == "ctrl+c":
		return m, tea.Quit

	case key.Matches(msg, m.keys.Enter):
		return m, m.submitNewTask()

	case msg.String() == "esc":
		m.cancelCreateTask()
		return m, nil

	case key.Matches(msg, m.keys.PrevList):
		if len(m.taskLists) > 0 {
			m.createListIndex = (m.createListIndex - 1 + len(m.taskLists)) % len(m.taskLists)
		}
		return m, nil

	case key.Matches(msg, m.keys.NextList):
		if len(m.taskLists) > 0 {
			m.createListIndex = (m.createListIndex + 1) % len(m.taskLists)
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.createInput, cmd = m.createInput.Update(msg)
		return m, cmd
	}
}

func (m *Model) startCreateTask() {
	m.creatingTask = true
	m.err = nil
	m.createInput.SetValue("")
	w := min(64, max(24, m.width-14))
	if w > 0 {
		m.createInput.Width = w
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
	m.createInput.Focus()
}

func (m *Model) cancelCreateTask() {
	m.creatingTask = false
	m.createInput.Blur()
	m.createInput.SetValue("")
	m.err = nil
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

func (m Model) submitNewTask() tea.Cmd {
	if len(m.taskLists) == 0 {
		return func() tea.Msg {
			return errMsg{err: fmt.Errorf("no task lists available")}
		}
	}
	listID := m.taskLists[m.createListIndex].ID
	title := m.createInput.Value()
	return func() tea.Msg {
		_, err := m.taskUC.CreateTask(listID, title)
		if err != nil {
			return errMsg{err: err}
		}
		lists, err := m.taskUC.ListTaskLists()
		if err != nil {
			return errMsg{err: err}
		}
		tasks, err := m.taskUC.ListAllTasks()
		if err != nil {
			return errMsg{err: err}
		}
		return createTaskDoneMsg{tasks: tasks, lists: lists}
	}
}

func (m Model) renderCreateTaskScreen() string {
	header := m.renderHeader()
	bodyH := bodyOuterLines(m.height)

	listLine := m.styles.Subtitle.Render("  List: ") +
		m.styles.Dim.Render("no lists — try refresh (r)") +
		m.styles.Subtitle.Render("  ·  [ / ] change list")
	if len(m.taskLists) > 0 {
		name := m.taskLists[m.createListIndex].Name
		listLine = m.styles.Subtitle.Render("  List: ") +
			m.styles.HelpKey.Render(name) +
			m.styles.Subtitle.Render("  ·  [ / ] change list")
	}

	title := m.styles.Title.Render("  New task")
	inputLine := "  " + m.createInput.View()
	hint := m.styles.Dim.Render("  enter save  ·  esc cancel  ·  ctrl+c quit")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		listLine,
		"",
		inputLine,
		"",
		hint,
	)

	if bodyH <= 0 {
		status := m.renderCreateStatusBar()
		return m.finalizeFrame(lipgloss.JoinVertical(lipgloss.Left, header, "", status))
	}

	boxW := min(max(8, m.width-4), 78)
	innerH := max(1, panelInnerHeight(bodyH))
	if innerH > bodyH {
		innerH = max(1, bodyH)
	}
	panel := m.styles.Panel.Width(boxW).Height(innerH).Render(content)
	body := lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Center, panel,
		lipgloss.WithWhitespaceBackground(theme.T.Base))
	body = clipToLines(body, bodyH)
	status := m.renderCreateStatusBar()

	return m.finalizeFrame(lipgloss.JoinVertical(lipgloss.Left, header, body, status))
}

func (m Model) renderCreateStatusBar() string {
	left := m.styles.HelpKey.Render(" new task ")
	if m.err != nil {
		left += lipgloss.NewStyle().Foreground(theme.T.Error).Render(fmt.Sprintf("%v ", m.err))
	}
	helpParts := []string{
		m.styles.HelpKey.Render("esc") + " " + m.styles.HelpDesc.Render("cancel"),
		m.styles.HelpKey.Render("enter") + " " + m.styles.HelpDesc.Render("save"),
		m.styles.HelpKey.Render("[ ]") + " " + m.styles.HelpDesc.Render("list"),
	}
	help := strings.Join(helpParts, "  ")
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(help)-2))
	return lipgloss.NewStyle().
		Background(theme.T.Surface).
		Width(m.width).
		Padding(0, 1).
		Inline(true).
		Render(left + gap + help)
}

func filterActiveTasks(tasks []domain.Task) []domain.Task {
	var open, done []domain.Task
	for _, t := range tasks {
		if t.Status == domain.TaskOpen {
			open = append(open, t)
		} else if t.CompletedAt != nil {
			today := time.Now().Truncate(24 * time.Hour)
			if t.CompletedAt.After(today) {
				done = append(done, t)
			}
		}
	}
	return append(open, done...)
}
