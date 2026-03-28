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
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Pane int

const (
	TaskPane Pane = iota
	AgendaPane
	GraphPane
	paneCount
)

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
}

type tasksLoadedMsg struct {
	tasks []domain.Task
}

type eventsLoadedMsg struct {
	events []domain.Event
}

type contributionLoadedMsg struct {
	data usecase.ContributionData
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
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tasksLoadedMsg:
		m.loading = false
		m.tasks.Tasks = filterActiveTasks(msg.tasks)
		return m, nil

	case eventsLoadedMsg:
		m.agenda.Events = msg.events
		return m, nil

	case contributionLoadedMsg:
		m.contribution.Data = msg.data
		m.progress.Progress = m.progressUC.Calculate()
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
		return m.renderHelp()
	}

	header := m.renderHeader()
	body := m.renderBody()
	status := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, status)
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.Tab):
		m.activePane = (m.activePane + 1) % paneCount
		m.updateFocus()
		return m, nil

	case key.Matches(msg, m.keys.ShiftTab):
		m.activePane = (m.activePane - 1 + paneCount) % paneCount
		m.updateFocus()
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		return m, tea.Batch(m.loadTasks(), m.loadEvents(), m.loadContribution())

	case key.Matches(msg, m.keys.Up):
		switch m.activePane {
		case TaskPane:
			m.tasks.MoveUp()
		case AgendaPane:
			m.agenda.MoveUp()
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		switch m.activePane {
		case TaskPane:
			m.tasks.MoveDown()
		case AgendaPane:
			m.agenda.MoveDown()
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
}

func (m *Model) updateLayout() {
	leftWidth := m.width * 38 / 100
	rightWidth := m.width - leftWidth - 4

	topHeight := (m.height - 4) * 40 / 100
	bottomHeight := (m.height - 4) - topHeight

	m.tasks.Width = leftWidth
	m.tasks.Height = m.height - 4

	m.agenda.Width = rightWidth
	m.agenda.Height = topHeight

	m.contribution.Width = rightWidth
	m.contribution.Height = bottomHeight / 2

	m.progress.Width = rightWidth

	m.updateFocus()
}

func (m Model) renderHeader() string {
	title := lipgloss.NewStyle().
		Foreground(theme.T.Primary).
		Bold(true).
		Padding(0, 1).
		Render("⚡ tocli")

	desc := lipgloss.NewStyle().
		Foreground(theme.T.Muted).
		Render(" productivity dashboard")

	right := lipgloss.NewStyle().
		Foreground(theme.T.Muted).
		Render(time.Now().Format("Mon Jan 2, 15:04"))

	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(title+desc)-lipgloss.Width(right)-2))

	return lipgloss.NewStyle().
		Background(theme.T.Surface).
		Width(m.width).
		Padding(0, 1).
		Render(title + desc + gap + right)
}

func (m Model) renderBody() string {
	leftWidth := m.width * 38 / 100
	rightWidth := m.width - leftWidth - 6

	leftPanel := m.renderPanel(m.tasks.View(), leftWidth, m.height-4, m.activePane == TaskPane)

	agendaView := m.agenda.View()
	contribView := m.contribution.View()
	progressView := m.progress.View()

	rightContent := lipgloss.JoinVertical(lipgloss.Left,
		agendaView,
		"",
		contribView,
		"",
		progressView,
	)

	rightPanel := m.renderPanel(rightContent, rightWidth, m.height-4, m.activePane != TaskPane)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
}

func (m Model) renderPanel(content string, width, height int, active bool) string {
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

	helpKeys := []struct{ key, desc string }{
		{"↑↓/jk", "navigate"},
		{"tab", "switch pane"},
		{"space", "toggle"},
		{"r", "refresh"},
		{"?", "help"},
		{"q", "quit"},
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
		Render(left + gap + help)
}

func (m Model) renderHelp() string {
	var b strings.Builder

	title := m.styles.Title.Render("  Keyboard Shortcuts")
	b.WriteString(title + "\n\n")

	bindings := []struct{ key, desc string }{
		{"j / ↓", "Move down"},
		{"k / ↑", "Move up"},
		{"Tab", "Next pane"},
		{"Shift+Tab", "Previous pane"},
		{"Space / Enter", "Toggle task complete"},
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
		tasks, err := m.taskUC.ListAllTasks()
		if err != nil {
			return errMsg{err: err}
		}
		return tasksLoadedMsg{tasks: tasks}
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
			tasks, err := m.taskUC.ListAllTasks()
			if err != nil {
				return errMsg{err: err}
			}
			return tasksLoadedMsg{tasks: tasks}
		}
	}

	return nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
