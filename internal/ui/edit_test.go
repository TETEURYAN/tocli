package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"tocli/internal/adapter/mock"
	"tocli/internal/domain"
	"tocli/internal/usecase"
)

// newMockModel is a model wired to the in-memory repositories, with its tasks already loaded.
func newMockModel(t *testing.T) Model {
	t.Helper()
	tasks := mock.NewTaskRepo()
	m := NewModel(
		usecase.NewTaskUseCase(tasks),
		usecase.NewEventUseCase(mock.NewEventRepo()),
		usecase.NewContributionUseCase(tasks),
		usecase.NewRatingUseCase(mock.NewRatingRepo()),
		usecase.NewProgressUseCase(),
	)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = asModel(t, next)
	return feed(t, m, m.loadTasks())
}

// feed runs a command to completion and delivers its message to the model.
func feed(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	next, _ := m.Update(cmd())
	return asModel(t, next)
}

func selectTask(t *testing.T, m *Model, title string) domain.Task {
	t.Helper()
	for i, task := range m.tasks.Tasks {
		if strings.Contains(task.Title, title) {
			m.tasks.Cursor = i
			return task
		}
	}
	t.Fatalf("no task containing %q in %d tasks", title, len(m.tasks.Tasks))
	return domain.Task{}
}

func TestEditOpensThePrefilledForm(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")

	m, _ = press(t, m, "e")
	if m.mode != modeCreateTask || m.editTask == nil {
		t.Fatalf("e: mode=%v editTask=%v, want the form in edit mode", m.mode, m.editTask)
	}
	if m.editTask.ID != task.ID {
		t.Errorf("editing %q, want %q", m.editTask.ID, task.ID)
	}
	if got := m.createInput.Value(); got != task.Title {
		t.Errorf("title field = %q, want %q", got, task.Title)
	}
	if got, want := m.dueInput.Value(), formatDueField(task.DueDate); got != want {
		t.Errorf("due field = %q, want %q", got, want)
	}
	if view := stripANSI(m.createTaskContent()); !strings.Contains(view, "Edit task") || !strings.Contains(view, "can't be changed") {
		t.Errorf("form should say it is editing and that the list is fixed:\n%s", view)
	}
	if bar := stripANSI(m.renderCreateStatusBar()); !strings.Contains(bar, "edit task") || strings.Contains(bar, "list") {
		t.Errorf("status bar = %q, want an edit label without the [ ] list hint", bar)
	}
}

func TestEditSavesTitleAndDueAndKeepsTheCursorOnTheTask(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")
	m, _ = press(t, m, "e")

	m.createInput.SetValue("Buy groceries and flowers")
	m.dueInput.SetValue("31-12-2031")
	m, cmd := press(t, m, "enter")
	if cmd == nil {
		t.Fatal("enter should save")
	}
	m = feed(t, m, cmd)

	if m.mode != modeNormal || m.editTask != nil {
		t.Fatalf("after saving: mode=%v editTask=%v", m.mode, m.editTask)
	}
	sel := m.tasks.SelectedTask()
	if sel == nil || sel.ID != task.ID {
		t.Fatalf("cursor is on %+v, want the edited task %q (the list was re-sorted)", sel, task.ID)
	}
	if sel.Title != "Buy groceries and flowers" {
		t.Errorf("title = %q", sel.Title)
	}
	if sel.DueDate == nil || sel.DueDate.Year() != 2031 || sel.DueDate.Month() != 12 || sel.DueDate.Day() != 31 {
		t.Errorf("due = %v, want 2031-12-31", sel.DueDate)
	}
}

func TestEditWithAnEmptiedDueFieldClearsTheDueDate(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")
	if task.DueDate == nil {
		t.Skip("seed task has no due date")
	}
	m, _ = press(t, m, "e")
	m.dueInput.SetValue("")
	m, cmd := press(t, m, "enter")
	m = feed(t, m, cmd)

	if sel := m.tasks.SelectedTask(); sel == nil || sel.ID != task.ID || sel.DueDate != nil {
		t.Errorf("selected = %+v, want the same task without a due date", sel)
	}
}

func TestEditRejectsBadInputAndStaysOpen(t *testing.T) {
	for name, tc := range map[string]struct{ title, due, wantErr string }{
		"blank title": {"   ", "", "empty"},
		"bad date":    {"ok", "2026-04-07", "inválida"},
	} {
		m := newMockModel(t)
		selectTask(t, &m, "Buy groceries")
		m, _ = press(t, m, "e")
		m.createInput.SetValue(tc.title)
		m.dueInput.SetValue(tc.due)
		m, cmd := press(t, m, "enter")
		m = feed(t, m, cmd)

		if m.mode != modeCreateTask || m.editTask == nil {
			t.Errorf("%s: the form closed (mode=%v), the user would lose their edit", name, m.mode)
		}
		if m.err == nil || !strings.Contains(strings.ToLower(m.err.Error()), tc.wantErr) {
			t.Errorf("%s: err = %v, want it to mention %q", name, m.err, tc.wantErr)
		}
	}
}

func TestEditEscapeCancelsAndNextNewTaskIsNotAnEdit(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")
	before := len(m.tasks.Tasks)

	m, _ = press(t, m, "e")
	m.createInput.SetValue("changed my mind")
	m, _ = press(t, m, "esc")
	if m.mode != modeNormal || m.editTask != nil {
		t.Fatalf("esc: mode=%v editTask=%v", m.mode, m.editTask)
	}
	if got := m.tasks.Tasks[m.tasks.Cursor].Title; got != task.Title {
		t.Errorf("cancelled edit changed the task to %q", got)
	}

	m, _ = press(t, m, "n")
	if m.editTask != nil || m.createInput.Value() != "" {
		t.Errorf("n after a cancelled edit: editTask=%v title=%q, want a blank new-task form", m.editTask, m.createInput.Value())
	}
	m.createInput.SetValue("brand new")
	m, cmd := press(t, m, "enter")
	m = feed(t, m, cmd)
	if len(m.tasks.Tasks) != before+1 {
		t.Errorf("creating after cancelling an edit gave %d tasks, want %d", len(m.tasks.Tasks), before+1)
	}
}

func TestEditIgnoresListSwitchingKeys(t *testing.T) {
	m := newMockModel(t)
	selectTask(t, &m, "Buy groceries")
	m, _ = press(t, m, "e")
	idx := m.createListIndex
	m, _ = press(t, m, "]")
	m, _ = press(t, m, "[")
	if m.createListIndex != idx {
		t.Errorf("list index changed from %d to %d while editing", idx, m.createListIndex)
	}
}

func TestEditKeyOnlyActsInTheTasksPane(t *testing.T) {
	m := newMockModel(t)
	selectTask(t, &m, "Buy groceries")

	m, _ = press(t, m, "tab") // agenda
	m, _ = press(t, m, "e")
	if m.mode != modeNormal {
		t.Errorf("e in the agenda pane opened mode %v", m.mode)
	}

	m, _ = press(t, m, "tab") // graph: e is the CSV export there, not edit
	m, _ = press(t, m, "e")
	if m.mode != modeNormal || m.editTask != nil {
		t.Errorf("e in the graph pane: mode=%v editTask=%v", m.mode, m.editTask)
	}

	empty := newTestModel(t)
	empty.tasks.Tasks = nil // nothing to select
	empty, _ = press(t, empty, "e")
	if empty.mode != modeNormal {
		t.Errorf("e with no task selected opened mode %v", empty.mode)
	}
}

func TestFormatDueField(t *testing.T) {
	if got := formatDueField(nil); got != "" {
		t.Errorf("nil = %q", got)
	}
	day := time.Date(2026, 4, 7, 0, 0, 0, 0, time.Local)
	if got := formatDueField(&day); got != "07-04-2026" {
		t.Errorf("local midnight = %q, want the date only", got)
	}
	at := time.Date(2026, 4, 7, 18, 5, 0, 0, time.Local)
	if got := formatDueField(&at); got != "07-04-2026 18:05" {
		t.Errorf("with a time = %q", got)
	}
	// What the form shows must parse back to the same instant.
	for _, d := range []time.Time{day, at} {
		parsed, err := usecase.ParseOptionalTaskDue(formatDueField(&d))
		if err != nil || parsed == nil || !parsed.Equal(d) {
			t.Errorf("%v does not survive format+parse: %v, %v", d, parsed, err)
		}
	}
}
