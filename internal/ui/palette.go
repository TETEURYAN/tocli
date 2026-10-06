package ui

import (
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"tocli/internal/ui/components"
	"tocli/internal/ui/theme"
	"tocli/internal/usecase"
)

// The command side of the search palette: typing ">" (or opening it with ":") switches the
// results from events to actions, filtered by what you type.

// paletteCommand is one action in the command list.
type paletteCommand struct {
	title string
	// key is the shortcut shown at the right edge, if the action has one.
	key string
	// available hides commands that make no sense right now (nil = always available).
	available func(m Model) bool
	run       func(m *Model) tea.Cmd
}

type commandMatch struct {
	cmd   paletteCommand
	hits  []int // rune indexes into cmd.title that matched
	score int
}

// isCommandQuery reports whether the palette input asks for commands rather than events.
func isCommandQuery(q string) bool { return strings.HasPrefix(strings.TrimLeft(q, " "), ">") }

func commandQuery(q string) string { return strings.TrimPrefix(strings.TrimLeft(q, " "), ">") }

func hasSelectedTask(m Model) bool { return m.tasks.SelectedTask() != nil }

func inDailyMode(m Model) bool { return m.contribution.Mode == components.ModeDaily }

// paletteCommands is the full list, in the order shown for an empty query: tasks, graph, view,
// then the app itself.
func (m Model) paletteCommands() []paletteCommand {
	return []paletteCommand{
		{title: "New task", key: "n", run: func(m *Model) tea.Cmd { m.startCreateTask(); return nil }},
		{title: "Edit selected task", key: "e", available: hasSelectedTask, run: func(m *Model) tea.Cmd {
			m.startEditTask(*m.tasks.SelectedTask())
			return nil
		}},
		{title: "Complete or reopen selected task", key: "space", available: hasSelectedTask, run: func(m *Model) tea.Cmd {
			return m.toggleTask()
		}},
		{title: "Delete selected task", key: "d", available: hasSelectedTask, run: func(m *Model) tea.Cmd {
			m.err = nil
			m.mode = modeConfirmDelete
			return nil
		}},

		{title: "Show details of the selected event", key: "enter", available: func(m Model) bool {
			_, ok := m.agendaEvent()
			return ok
		}, run: func(m *Model) tea.Cmd {
			if e, ok := m.agendaEvent(); ok {
				m.openEventDetail(e)
			}
			return nil
		}},
		{title: "Go to today", run: func(m *Model) tea.Cmd {
			n := time.Now()
			return m.goToDate(time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local))
		}},
		{title: "Graph: previous year", key: "[", run: func(m *Model) tea.Cmd {
			return tea.Batch(m.setPane(GraphPane), m.shiftYear(-1))
		}},
		{title: "Graph: next year", key: "]", run: func(m *Model) tea.Cmd {
			return tea.Batch(m.setPane(GraphPane), m.shiftYear(1))
		}},
		{title: "Graph: switch between contribution and daily rating", key: "g", run: func(m *Model) tea.Cmd {
			m.contribution.ToggleMode()
			return m.setPane(GraphPane)
		}},
		{title: "Write note for the selected day", key: "t", available: inDailyMode, run: func(m *Model) tea.Cmd {
			m.startWriteNote()
			return nil
		}},
		{title: "Export this month's ratings to CSV", key: "e", available: inDailyMode, run: func(m *Model) tea.Cmd {
			return m.exportRatingsCSV()
		}},

		{title: "Focus tasks", run: func(m *Model) tea.Cmd { return m.setPane(TaskPane) }},
		{title: "Focus agenda", run: func(m *Model) tea.Cmd { return m.setPane(AgendaPane) }},
		{title: "Focus graph", run: func(m *Model) tea.Cmd { return m.setPane(GraphPane) }},
		{title: "Theme: dark", run: func(m *Model) tea.Cmd { return m.setThemeMode("dark") }},
		{title: "Theme: light", run: func(m *Model) tea.Cmd { return m.setThemeMode("light") }},
		{title: "Theme: follow the terminal", run: func(m *Model) tea.Cmd { return m.setThemeMode("auto") }},

		{title: "Refresh data", key: "r", run: func(m *Model) tea.Cmd { return m.refreshAll() }},
		{title: "Show keyboard shortcuts", key: "?", run: func(m *Model) tea.Cmd {
			m.mode = modeHelp
			return nil
		}},
		{title: "Quit", key: "q", run: func(m *Model) tea.Cmd { return tea.Quit }},
	}
}

// matchCommands returns the available commands matching query, best first (ties keep the order
// of paletteCommands).
func (m Model) matchCommands(query string) []commandMatch {
	var out []commandMatch
	for _, c := range m.paletteCommands() {
		if c.available != nil && !c.available(m) {
			continue
		}
		score, hits, ok := usecase.MatchText(query, c.title)
		if !ok {
			continue
		}
		out = append(out, commandMatch{cmd: c, hits: hits, score: score})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	return out
}

// refreshAll reloads tasks, today's events and the graph's year (the r key).
func (m *Model) refreshAll() tea.Cmd {
	m.loading = true
	return tea.Batch(m.loadTasks(), m.loadEvents(), m.loadGraphYear())
}

// setThemeMode switches the palette at runtime: "dark" and "light" pin it, anything else follows
// the terminal again (which needs a fresh background-color query).
func (m *Model) setThemeMode(mode string) tea.Cmd {
	switch mode {
	case "dark":
		m.themeMode = "dark"
		m.applyTheme(theme.Dark)
		return nil
	case "light":
		m.themeMode = "light"
		m.applyTheme(theme.Light)
		return nil
	}
	m.themeMode = "auto"
	return tea.RequestBackgroundColor
}
