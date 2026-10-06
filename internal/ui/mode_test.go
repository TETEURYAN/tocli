package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"tocli/internal/domain"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	m := NewModel(nil, nil, nil, nil, nil)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = asModel(t, next)
	m.tasks.Tasks = []domain.Task{{ID: "1", Title: "write tests", Status: domain.TaskOpen}}
	m.taskLists = []domain.TaskList{{ID: "l", Name: "Work"}}
	return m
}

// Update returns either Model or *Model depending on which handler ran.
func asModel(t *testing.T, v tea.Model) Model {
	t.Helper()
	switch m := v.(type) {
	case Model:
		return m
	case *Model:
		return *m
	}
	t.Fatalf("unexpected model type %T", v)
	return Model{}
}

func press(t *testing.T, m Model, s string) (Model, tea.Cmd) {
	t.Helper()
	k := tea.KeyPressMsg{Text: s}
	if len(s) == 1 {
		k.Code = rune(s[0])
	}
	switch s {
	case "esc":
		k = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		k = tea.KeyPressMsg{Code: tea.KeyTab}
	}
	next, cmd := m.Update(k)
	return asModel(t, next), cmd
}

func TestHelpModeSwallowsKeysAndCloses(t *testing.T) {
	m := newTestModel(t)
	m, _ = press(t, m, "?")
	if m.mode != modeHelp {
		t.Fatalf("mode = %v, want help", m.mode)
	}
	m, _ = press(t, m, "n") // must not open the create form underneath
	if m.mode != modeHelp {
		t.Fatalf("n while help open changed mode to %v", m.mode)
	}
	m, _ = press(t, m, "esc")
	if m.mode != modeNormal {
		t.Fatalf("esc did not close help, mode = %v", m.mode)
	}
	m, _ = press(t, m, "?")
	m, _ = press(t, m, "?")
	if m.mode != modeNormal {
		t.Fatalf("? did not toggle help closed, mode = %v", m.mode)
	}
}

func TestCreateTaskModeTypingQDoesNotQuit(t *testing.T) {
	m := newTestModel(t)
	m, _ = press(t, m, "n")
	if m.mode != modeCreateTask {
		t.Fatalf("mode = %v, want create", m.mode)
	}
	m, cmd := press(t, m, "q")
	if m.mode != modeCreateTask {
		t.Fatalf("typing q left create mode: %v", m.mode)
	}
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("typing q in the task title quit the app")
		}
	}
	if got := m.createInput.Value(); got != "q" {
		t.Fatalf("title = %q, want %q", got, "q")
	}
	m, _ = press(t, m, "esc")
	if m.mode != modeNormal || m.createInput.Value() != "" {
		t.Fatalf("esc should cancel and clear: mode=%v title=%q", m.mode, m.createInput.Value())
	}
}

func TestDeleteConfirmFlow(t *testing.T) {
	m := newTestModel(t)
	m, _ = press(t, m, "d")
	if m.mode != modeConfirmDelete {
		t.Fatalf("mode = %v, want confirm delete", m.mode)
	}
	m, _ = press(t, m, "n")
	if m.mode != modeNormal {
		t.Fatalf("n should cancel, mode = %v", m.mode)
	}

	m.tasks.Tasks = nil
	m, _ = press(t, m, "d")
	if m.mode != modeNormal {
		t.Fatalf("d with no task selected must not prompt, mode = %v", m.mode)
	}
}

func TestAsyncCompletionOnlyLeavesItsOwnMode(t *testing.T) {
	m := newTestModel(t)
	m, _ = press(t, m, "?")

	next, _ := m.Update(createTaskDoneMsg{})
	m = asModel(t, next)
	if m.mode != modeHelp {
		t.Fatalf("createTaskDoneMsg closed the help overlay, mode = %v", m.mode)
	}
	next, _ = m.Update(noteSavedMsg{})
	m = asModel(t, next)
	if m.mode != modeHelp {
		t.Fatalf("noteSavedMsg closed the help overlay, mode = %v", m.mode)
	}

	m, _ = press(t, m, "esc")
	m, _ = press(t, m, "n")
	next, _ = m.Update(createTaskDoneMsg{})
	if got := asModel(t, next).mode; got != modeNormal {
		t.Fatalf("createTaskDoneMsg should leave create mode, mode = %v", got)
	}
}

func TestNoteModeFromGraphPane(t *testing.T) {
	m := newTestModel(t)
	m, _ = press(t, m, "tab")
	m, _ = press(t, m, "tab")
	if m.activePane != GraphPane {
		t.Fatalf("pane = %v, want graph", m.activePane)
	}
	m, _ = press(t, m, "g")
	m.contribution.CursorDate = time.Now()
	m, _ = press(t, m, "t")
	if m.mode != modeWriteNote {
		t.Fatalf("mode = %v, want write note", m.mode)
	}
	m, _ = press(t, m, "esc")
	if m.mode != modeNormal {
		t.Fatalf("esc should leave note mode, mode = %v", m.mode)
	}
}

func TestModalOpenTracksMode(t *testing.T) {
	m := newTestModel(t)
	m, _ = press(t, m, "?")
	if !m.modalOpen() {
		t.Fatal("modalOpen() should be true in help mode")
	}
	m, _ = press(t, m, "esc")
	if m.modalOpen() {
		t.Fatal("modalOpen() should be false in normal mode")
	}
}
