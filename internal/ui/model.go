package ui

import (
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"tocli/internal/domain"
	"tocli/internal/ui/components"
	"tocli/internal/ui/theme"
	"tocli/internal/usecase"
)

// mode is what the keyboard is currently driving. Modes are mutually exclusive.
type mode int

const (
	modeNormal        mode = iota // dashboard: panes, tasks, graph
	modeHelp                      // keyboard shortcuts overlay
	modeCreateTask                // "New task" form
	modeWriteNote                 // day note editor
	modeConfirmDelete             // y/n prompt in the status bar
	modeSearch                    // search palette over the dashboard
	modeEventDetail               // details of one calendar event
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

	taskUC     *usecase.TaskUseCase
	eventUC    *usecase.EventUseCase
	contribUC  *usecase.ContributionUseCase
	ratingUC   *usecase.RatingUseCase
	progressUC *usecase.ProgressUseCase

	keys       KeyMap
	styles     theme.Styles
	themeMode  string // "auto" follows the terminal background; "dark"/"light" pin the palette
	activePane Pane
	width      int
	height     int
	ready      bool
	loading    bool
	err        error
	notice     string

	// mode is the single interaction state: the dashboard, or exactly one modal on top of it.
	mode mode

	createInput      textinput.Model
	dueInput         textinput.Model
	createDueFocused bool
	taskLists        []domain.TaskList
	createListIndex  int
	// editTask is the task being edited while modeCreateTask shows the form in "edit" mode; nil
	// means the form is creating a new task.
	editTask *domain.Task

	noteInput textarea.Model
	noteDate  time.Time

	search searchState
	detail detailState

	// graphYear is the year the graph shows (it can differ from the current year); yearSeq
	// debounces fast year switching.
	graphYear int
	yearSeq   int
}

type tasksLoadedMsg struct {
	tasks []domain.Task
	lists []domain.TaskList
}

type createTaskDoneMsg struct {
	tasks []domain.Task
	lists []domain.TaskList
	// selectID, when set, is the task to put the cursor on after the reload (the one just edited).
	selectID string
}

type eventsLoadedMsg struct {
	events []domain.Event
}

type contributionLoadedMsg struct {
	data usecase.ContributionData
}

type ratingLoadedMsg struct {
	data usecase.RatingData
}

type dayDetailLoadedMsg struct {
	date   time.Time
	events []domain.Event
	tasks  []domain.Task
}

type errMsg struct {
	err error
}

type exportDoneMsg struct {
	path string
}

type noteSavedMsg struct {
	data usecase.RatingData
}

type tickMsg time.Time

// yearTickMsg fires once the user stopped switching years; seq discards all but the last.
type yearTickMsg struct{ seq int }

func NewModel(
	taskUC *usecase.TaskUseCase,
	eventUC *usecase.EventUseCase,
	contribUC *usecase.ContributionUseCase,
	ratingUC *usecase.RatingUseCase,
	progressUC *usecase.ProgressUseCase,
) Model {
	s := theme.NewStyles(theme.Dark)
	ti := textinput.New()
	ti.Placeholder = "What needs doing?"
	ti.CharLimit = 280
	ti.SetWidth(40)
	dueTi := textinput.New()
	dueTi.Placeholder = "Date: DD-MM-YYYY or DD-MM-YYYY HH:MM (optional)"
	dueTi.CharLimit = 32
	dueTi.SetWidth(40)
	noteTa := textarea.New()
	noteTa.Placeholder = "How was this day?"
	noteTa.CharLimit = domain.MaxRatingNoteLength
	noteTa.SetWidth(50)
	noteTa.SetHeight(6)
	noteTa.ShowLineNumbers = false
	return Model{
		tasks:        components.NewTaskListModel(s),
		agenda:       components.NewAgendaModel(s),
		contribution: components.NewContributionModel(s),
		progress:     components.NewProgressBarModel(s),
		taskUC:       taskUC,
		eventUC:      eventUC,
		contribUC:    contribUC,
		ratingUC:     ratingUC,
		progressUC:   progressUC,
		keys:         DefaultKeyMap(),
		styles:       s,
		themeMode:    "auto",
		activePane:   TaskPane,
		loading:      true,
		createInput:  ti,
		dueInput:     dueTi,
		noteInput:    noteTa,
		search:       searchState{input: newSearchInput()},
		graphYear:    time.Now().Year(),
	}
}

// WithTheme pins the palette ("dark" or "light"); any other value keeps "auto", which follows
// the terminal background reported at startup.
func (m Model) WithTheme(mode string) Model {
	if mode == "dark" || mode == "light" {
		m.setThemeMode(mode)
	}
	return m
}

// applyTheme rebuilds every style from t and hands it to the components.
func (m *Model) applyTheme(t theme.Theme) {
	m.styles = theme.NewStyles(t)
	m.tasks.SetStyles(m.styles)
	m.agenda.SetStyles(m.styles)
	m.contribution.SetStyles(m.styles)
	m.progress.SetStyles(m.styles)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		m.loadTasks(),
		m.loadEvents(),
		m.loadContribution(),
		m.loadRatings(),
		tickCmd(),
	)
}
