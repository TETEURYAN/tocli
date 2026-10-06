package ui

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"tocli/internal/domain"
)

// ---- the key map itself -----------------------------------------------------------------------

// producibleKeys is every key string a real terminal can make Bubble Tea deliver: the decoded
// form of each printable ASCII character, each named key and each ctrl+letter.
func producibleKeys(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for r := rune(0x21); r <= 0x7e; r++ {
		out[termKey(t, string(r)).String()] = true
	}
	for name := range namedKeys {
		out[termKey(t, name).String()] = true
	}
	for c := 'a'; c <= 'z'; c++ {
		out[termKey(t, "ctrl+"+string(c)).String()] = true
	}
	return out
}

func TestEveryKeyBindingIsAKeyATerminalCanProduce(t *testing.T) {
	// A binding written in another library's notation (" " instead of "space") never matches
	// anything and the key silently does nothing. This catches that for every binding at once.
	can := producibleKeys(t)
	v := reflect.ValueOf(DefaultKeyMap())
	for i := 0; i < v.NumField(); i++ {
		b := v.Field(i).Interface().(key.Binding)
		name := v.Type().Field(i).Name
		if len(b.Keys()) == 0 {
			t.Errorf("binding %s has no keys", name)
		}
		for _, k := range b.Keys() {
			if !can[k] {
				t.Errorf("binding %s uses %q, which no terminal key produces (known keys include %q)", name, k, sampleKeys(can))
			}
		}
	}
	if can[" "] {
		t.Error("the space bar must be named \"space\", not \" \"")
	}
}

func sampleKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		if len(k) > 1 {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out[:min(len(out), 8)]
}

func TestBindingsMatchTheKeysTheyAreMeantFor(t *testing.T) {
	k := DefaultKeyMap()
	cases := []struct {
		binding key.Binding
		name    string
		keys    []string
	}{
		{k.Up, "Up", []string{"up", "k"}},
		{k.Down, "Down", []string{"down", "j"}},
		{k.Left, "Left", []string{"left", "h"}},
		{k.Right, "Right", []string{"right", "l"}},
		{k.Tab, "Tab", []string{"tab"}},
		{k.ShiftTab, "ShiftTab", []string{"shift+tab"}},
		{k.Enter, "Enter", []string{"enter"}},
		{k.Space, "Space", []string{"space"}},
		{k.Refresh, "Refresh", []string{"r"}},
		{k.Help, "Help", []string{"?"}},
		{k.Quit, "Quit", []string{"q", "ctrl+c"}},
		{k.NewTask, "NewTask", []string{"n"}},
		{k.EditTask, "EditTask", []string{"e"}},
		{k.DeleteTask, "DeleteTask", []string{"d"}},
		{k.ToggleGraphMode, "ToggleGraphMode", []string{"g"}},
		{k.ExportCSV, "ExportCSV", []string{"e"}},
		{k.WriteNote, "WriteNote", []string{"t"}},
		{k.PrevList, "PrevList", []string{"["}},
		{k.NextList, "NextList", []string{"]"}},
		{k.PrevYear, "PrevYear", []string{"["}},
		{k.NextYear, "NextYear", []string{"]"}},
		{k.Search, "Search", []string{"/"}},
		{k.Command, "Command", []string{":"}},
	}
	for _, c := range cases {
		for _, name := range c.keys {
			if !key.Matches(termKey(t, name), c.binding) {
				t.Errorf("%s does not match the %q key (it sends %q)", c.name, name, termKey(t, name).String())
			}
		}
	}
	// And the bindings are not so loose that other keys trigger them.
	for _, c := range []struct {
		b    key.Binding
		name string
		not  string
	}{{k.Space, "Space", "enter"}, {k.Enter, "Enter", "space"}, {k.Tab, "Tab", "shift+tab"}, {k.Quit, "Quit", "w"}} {
		if key.Matches(termKey(t, c.not), c.b) {
			t.Errorf("%s wrongly matches %q", c.name, c.not)
		}
	}
}

func TestTheDecoderProducesWhatTheFixtureNamesPromise(t *testing.T) {
	// If these ever change, the key names used across the tests would silently mean something else.
	for name, want := range map[string]string{
		"space": "space", "enter": "enter", "esc": "esc", "tab": "tab", "shift+tab": "shift+tab",
		"backspace": "backspace", "up": "up", "pgup": "pgup", "pgdown": "pgdown", "ctrl+c": "ctrl+c",
		"ctrl+s": "ctrl+s", "ctrl+p": "ctrl+p", "q": "q", "?": "?", "/": "/", ":": ":", "[": "[",
	} {
		if got := termKey(t, name).String(); got != want {
			t.Errorf("%q decodes to %q, want %q", name, got, want)
		}
	}
}

// ---- helpers --------------------------------------------------------------------------------

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func taskByID(m Model, id string) *domain.Task {
	for i := range m.tasks.Tasks {
		if m.tasks.Tasks[i].ID == id {
			return &m.tasks.Tasks[i]
		}
	}
	return nil
}

func cursorOn(m *Model, id string) {
	for i, task := range m.tasks.Tasks {
		if task.ID == id {
			m.tasks.Cursor = i
		}
	}
}

// ---- the space bar (the reported bug) -------------------------------------------------------

func TestSpaceCompletesAndReopensTheSelectedTask(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")
	if task.Status != domain.TaskOpen {
		t.Fatal("test setup: the task should start open")
	}

	m, cmd := press(t, m, "space")
	if cmd == nil {
		t.Fatal("space on the selected task did nothing: no command was returned")
	}
	m = feed(t, m, cmd)
	if got := taskByID(m, task.ID); got == nil || got.Status != domain.TaskCompleted {
		t.Fatalf("after space the task is %+v, want completed", got)
	}

	// Space again on that same task reopens it.
	cursorOn(&m, task.ID)
	m, cmd = press(t, m, "space")
	m = feed(t, m, cmd)
	if got := taskByID(m, task.ID); got == nil || got.Status != domain.TaskOpen {
		t.Fatalf("a second space should reopen the task, got %+v", got)
	}
}

func TestSpaceAndEnterDoTheSameOnTheTasksPane(t *testing.T) {
	for _, k := range []string{"space", "enter"} {
		m := newMockModel(t)
		task := selectTask(t, &m, "Buy groceries")
		m, cmd := press(t, m, k)
		m = feed(t, m, cmd)
		if got := taskByID(m, task.ID); got == nil || got.Status != domain.TaskCompleted {
			t.Errorf("%s did not complete the task", k)
		}
	}
}

func TestSpaceWithNoTaskSelectedIsHarmless(t *testing.T) {
	m := newMockModel(t)
	m.tasks.Tasks = nil
	m, cmd := press(t, m, "space")
	if cmd != nil || m.mode != modeNormal {
		t.Errorf("cmd=%v mode=%v", cmd != nil, m.mode)
	}
}

func TestSpaceDoesNothingOutsideTheTasksPane(t *testing.T) {
	m := newMockModel(t)
	m = drain(t, m, m.loadEvents())
	before := len(m.tasks.Tasks)
	for _, pane := range []Pane{AgendaPane, GraphPane} {
		m.activePane = pane
		m.updateFocus()
		var cmd tea.Cmd
		m, cmd = press(t, m, "space")
		if cmd != nil || m.mode != modeNormal || len(m.tasks.Tasks) != before {
			t.Errorf("space in pane %v: cmd=%v mode=%v", pane, cmd != nil, m.mode)
		}
	}
}

func TestSpaceIsTextInsideForms(t *testing.T) {
	// The create form: a space is a character, not "toggle".
	m := newMockModel(t)
	m, _ = press(t, m, "n")
	m = typeText(t, m, "buy milk")
	if got := m.createInput.Value(); got != "buy milk" {
		t.Errorf("title = %q, want the space kept", got)
	}

	// The note editor.
	g := graphModel(t)
	g.contribution.Mode = 1 // daily
	g, _ = press(t, g, "t")
	g.noteInput.SetValue("")
	g = typeText(t, g, "a good day")
	if got := g.noteInput.Value(); got != "a good day" {
		t.Errorf("note = %q", got)
	}

	// The search box: two terms separated by a space.
	s := searchModel(t, searchEvent("a", "Sprint Planning", "Room 3A", 1), searchEvent("b", "Sprint Review", "Room 9", 2))
	s = typeText(t, s, "sprint room")
	if len(s.search.results) != 2 {
		t.Errorf("'sprint room' matched %d events, want both", len(s.search.results))
	}
	s = typeText(t, s, " 3a")
	if len(s.search.results) != 1 {
		t.Errorf("a third term should narrow to one event, got %d", len(s.search.results))
	}
}

// ---- keyboard matrix: every key, in every context, as a terminal sends it -------------------

type keyCase struct {
	name  string
	setup func(t *testing.T) Model
	keys  []string
	check func(t *testing.T, m Model, cmd tea.Cmd)
}

func mockModel(t *testing.T) Model { return newMockModel(t) }

func TestKeyboardMatrix(t *testing.T) {
	today := func() time.Time {
		n := time.Now()
		return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
	}
	dailyGraph := func(t *testing.T) Model {
		m := graphModel(t)
		m.contribution.Mode = 1
		m.contribution.CursorDate = today()
		return m
	}
	onPane := func(p Pane) func(t *testing.T) Model {
		return func(t *testing.T) Model {
			m := newMockModel(t)
			m = drain(t, m, m.loadEvents())
			m = drain(t, m, m.loadGraphYear())
			m.activePane = p
			m.updateFocus()
			if p == GraphPane {
				m.contribution.CursorDate = today()
			}
			return m
		}
	}
	mode := func(want mode) func(t *testing.T, m Model, cmd tea.Cmd) {
		return func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.mode != want {
				t.Errorf("mode = %v, want %v", m.mode, want)
			}
		}
	}

	cases := []keyCase{
		// -- global, from the dashboard
		{"? opens help", mockModel, []string{"?"}, mode(modeHelp)},
		{"/ opens event search", mockModel, []string{"/"}, mode(modeSearch)},
		{": opens the command palette", mockModel, []string{":"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.mode != modeSearch || !m.commandMode() {
				t.Errorf("mode=%v commandMode=%v", m.mode, m.commandMode())
			}
		}},
		{"r refreshes", mockModel, []string{"r"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if !m.loading || cmd == nil {
				t.Errorf("loading=%v cmd=%v", m.loading, cmd != nil)
			}
		}},
		{"q quits", mockModel, []string{"q"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if !isQuit(cmd) {
				t.Error("q did not quit")
			}
		}},
		{"ctrl+c quits", mockModel, []string{"ctrl+c"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if !isQuit(cmd) {
				t.Error("ctrl+c did not quit")
			}
		}},
		{"tab cycles forward", mockModel, []string{"tab", "tab"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.activePane != GraphPane {
				t.Errorf("pane = %v, want graph after two tabs", m.activePane)
			}
		}},
		{"tab wraps to the first pane", mockModel, []string{"tab", "tab", "tab"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.activePane != TaskPane {
				t.Errorf("pane = %v, want tasks", m.activePane)
			}
		}},
		{"shift+tab goes back", mockModel, []string{"shift+tab"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.activePane != GraphPane {
				t.Errorf("pane = %v, want graph (wrapping backwards)", m.activePane)
			}
		}},

		// -- tasks pane
		{"j moves down", onPane(TaskPane), []string{"j"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.tasks.Cursor != 1 {
				t.Errorf("cursor = %d", m.tasks.Cursor)
			}
		}},
		{"down arrow moves down", onPane(TaskPane), []string{"down", "down"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.tasks.Cursor != 2 {
				t.Errorf("cursor = %d", m.tasks.Cursor)
			}
		}},
		{"k and up move up, clamped at the top", onPane(TaskPane), []string{"j", "j", "k", "up", "up", "k"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.tasks.Cursor != 0 {
				t.Errorf("cursor = %d", m.tasks.Cursor)
			}
		}},
		{"n opens the new task form", onPane(TaskPane), []string{"n"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.mode != modeCreateTask || m.editTask != nil {
				t.Errorf("mode=%v editTask=%v", m.mode, m.editTask)
			}
		}},
		{"e opens the edit form", onPane(TaskPane), []string{"e"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.mode != modeCreateTask || m.editTask == nil {
				t.Errorf("mode=%v editTask=%v", m.mode, m.editTask)
			}
		}},
		{"d asks to confirm the deletion", onPane(TaskPane), []string{"d"}, mode(modeConfirmDelete)},
		{"enter completes the task", onPane(TaskPane), []string{"enter"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Error("enter returned no command")
			}
		}},
		{"space completes the task", onPane(TaskPane), []string{"space"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Error("space returned no command")
			}
		}},

		// -- agenda pane
		{"j/k move through events", onPane(AgendaPane), []string{"j", "j", "k"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.agenda.Cursor != 1 {
				t.Errorf("cursor = %d", m.agenda.Cursor)
			}
		}},
		{"enter opens the event details", onPane(AgendaPane), []string{"enter"}, mode(modeEventDetail)},
		{"space does not open details", onPane(AgendaPane), []string{"space"}, mode(modeNormal)},
		{"n does nothing in the agenda", onPane(AgendaPane), []string{"n"}, mode(modeNormal)},

		// -- graph pane
		{"l and right move a week forward", onPane(GraphPane), []string{"l", "right"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if want := today().AddDate(0, 0, 14); !sameDay(m.contribution.CursorDate, want) && want.Year() == m.graphYear {
				t.Errorf("cursor = %v, want %v", m.contribution.CursorDate, want)
			}
		}},
		{"h and left move a week back", onPane(GraphPane), []string{"h", "left"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if want := today().AddDate(0, 0, -14); !sameDay(m.contribution.CursorDate, want) {
				t.Errorf("cursor = %v, want %v", m.contribution.CursorDate, want)
			}
		}},
		{"j and down move a day forward", onPane(GraphPane), []string{"j", "down"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if want := today().AddDate(0, 0, 2); !sameDay(m.contribution.CursorDate, want) {
				t.Errorf("cursor = %v, want %v", m.contribution.CursorDate, want)
			}
		}},
		{"k and up move a day back", onPane(GraphPane), []string{"k", "up"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if want := today().AddDate(0, 0, -2); !sameDay(m.contribution.CursorDate, want) {
				t.Errorf("cursor = %v, want %v", m.contribution.CursorDate, want)
			}
		}},
		{"g switches the graph mode", onPane(GraphPane), []string{"g"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.contribution.Mode != 1 {
				t.Errorf("mode = %v, want daily", m.contribution.Mode)
			}
		}},
		{"[ and ] change the year", onPane(GraphPane), []string{"[", "[", "]"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.graphYear != time.Now().Year()-1 {
				t.Errorf("year = %d", m.graphYear)
			}
		}},
		{"t opens the note editor in daily mode", dailyGraph, []string{"t"}, mode(modeWriteNote)},
		{"t does nothing in contribution mode", onPane(GraphPane), []string{"t"}, mode(modeNormal)},
		{"1-5 rate the day in daily mode", dailyGraph, []string{"4"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Error("rating a day returned no command")
			}
		}},
		{"e exports in daily mode", dailyGraph, []string{"e"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if cmd == nil || m.mode != modeNormal {
				t.Errorf("cmd=%v mode=%v", cmd != nil, m.mode)
			}
		}},
		{"space does nothing on the graph", onPane(GraphPane), []string{"space"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if cmd != nil || m.mode != modeNormal {
				t.Errorf("cmd=%v mode=%v", cmd != nil, m.mode)
			}
		}},

		// -- help
		{"esc closes help", mockModel, []string{"?", "esc"}, mode(modeNormal)},
		{"? closes help", mockModel, []string{"?", "?"}, mode(modeNormal)},
		{"space and enter leave help open", mockModel, []string{"?", "space", "enter"}, mode(modeHelp)},
		{"q quits from help", mockModel, []string{"?", "q"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if !isQuit(cmd) {
				t.Error("q should quit")
			}
		}},

		// -- create / edit form
		{"typing, tab to the due field and enter save", onPane(TaskPane), []string{"n", "a", "space", "b", "tab", "1", "0", "-", "1", "2", "-", "2", "0", "3", "0", "enter"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Error("enter should save")
			}
			if m.createInput.Value() != "a b" || m.dueInput.Value() != "10-12-2030" {
				t.Errorf("title=%q due=%q", m.createInput.Value(), m.dueInput.Value())
			}
		}},
		{"esc cancels the form", onPane(TaskPane), []string{"n", "x", "esc"}, mode(modeNormal)},
		{"q is typed in the form, it does not quit", onPane(TaskPane), []string{"n", "q"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if isQuit(cmd) || m.createInput.Value() != "q" {
				t.Errorf("quit=%v title=%q", isQuit(cmd), m.createInput.Value())
			}
		}},
		{"ctrl+c quits from the form", onPane(TaskPane), []string{"n", "ctrl+c"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if !isQuit(cmd) {
				t.Error("ctrl+c should quit")
			}
		}},
		{"backspace edits the title", onPane(TaskPane), []string{"n", "a", "b", "backspace"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.createInput.Value() != "a" {
				t.Errorf("title = %q", m.createInput.Value())
			}
		}},

		// -- note editor
		{"enter makes a new line in the note", dailyGraph, []string{"t", "a", "enter", "b"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if v := m.noteInput.Value(); v == "" || m.mode != modeWriteNote {
				t.Errorf("note=%q mode=%v", v, m.mode)
			}
		}},
		{"ctrl+s saves the note", dailyGraph, []string{"t", "a", "ctrl+s"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if cmd == nil {
				t.Error("ctrl+s should save")
			}
		}},
		{"esc cancels the note", dailyGraph, []string{"t", "esc"}, mode(modeNormal)},
		{"q is typed in the note", dailyGraph, []string{"t", "q"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if isQuit(cmd) || m.mode != modeWriteNote {
				t.Errorf("quit=%v mode=%v", isQuit(cmd), m.mode)
			}
		}},

		// -- delete confirmation
		{"y confirms the deletion", onPane(TaskPane), []string{"d", "y"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if m.mode != modeNormal || cmd == nil {
				t.Errorf("mode=%v cmd=%v", m.mode, cmd != nil)
			}
		}},
		{"n cancels the deletion", onPane(TaskPane), []string{"d", "n"}, mode(modeNormal)},
		{"esc cancels the deletion", onPane(TaskPane), []string{"d", "esc"}, mode(modeNormal)},
		{"space does not confirm the deletion", onPane(TaskPane), []string{"d", "space"}, mode(modeConfirmDelete)},

		// -- search
		{"esc closes the search", mockModel, []string{"/", "esc"}, mode(modeNormal)},
		{"ctrl+c quits from the search", mockModel, []string{"/", "ctrl+c"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if !isQuit(cmd) {
				t.Error("ctrl+c should quit")
			}
		}},
		{"q is typed in the search", mockModel, []string{"/", "q"}, func(t *testing.T, m Model, cmd tea.Cmd) {
			if isQuit(cmd) || m.search.input.Value() != "q" {
				t.Errorf("quit=%v query=%q", isQuit(cmd), m.search.input.Value())
			}
		}},
		{"backspace edits the query", mockModel, []string{"/", "a", "b", "backspace"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.search.input.Value() != "a" {
				t.Errorf("query = %q", m.search.input.Value())
			}
		}},
		{"ctrl+n and ctrl+p move the selection", func(t *testing.T) Model {
			return searchModel(t, searchEvent("a", "Standup", "", 1), searchEvent("b", "Standup", "", 2), searchEvent("c", "Standup", "", 3))
		}, []string{"s", "t", "a", "ctrl+n", "ctrl+n", "ctrl+p"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.search.sel != 1 {
				t.Errorf("sel = %d, want 1", m.search.sel)
			}
		}},
		{"pgdown and pgup page the results", func(t *testing.T) Model {
			var ev []domain.Event
			for i := 1; i <= 30; i++ {
				ev = append(ev, searchEvent(string(rune('a'+i%26))+string(rune('a'+i/26)), "Standup", "", i))
			}
			return searchModel(t, ev...)
		}, []string{"s", "t", "pgdown", "pgdown", "pgup"}, func(t *testing.T, m Model, _ tea.Cmd) {
			if m.search.sel <= 0 {
				t.Errorf("sel = %d, paging did not move", m.search.sel)
			}
		}},
		{"tab opens the details from the search", func(t *testing.T) Model {
			return searchModel(t, searchEvent("a", "Standup", "", 1))
		}, []string{"s", "t", "tab"}, mode(modeEventDetail)},

		// -- event details
		{"esc and enter close the details", onPane(AgendaPane), []string{"enter", "esc"}, mode(modeNormal)},
		{"q quits from the details", onPane(AgendaPane), []string{"enter", "q"}, func(t *testing.T, _ Model, cmd tea.Cmd) {
			if !isQuit(cmd) {
				t.Error("q should quit")
			}
		}},
		{"space is ignored in the details", onPane(AgendaPane), []string{"enter", "space"}, mode(modeEventDetail)},
		{"j and k scroll without leaving", onPane(AgendaPane), []string{"enter", "j", "k"}, mode(modeEventDetail)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := c.setup(t)
			var cmd tea.Cmd
			for _, k := range c.keys {
				m, cmd = press(t, m, k)
			}
			c.check(t, m, cmd)
		})
	}
}

func TestListKeysCycleTheTargetListWhenCreating(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, "n")
	n := len(m.taskLists)
	if n < 2 {
		t.Fatalf("test setup: need several lists, have %d", n)
	}
	start := m.createListIndex

	m, _ = press(t, m, "]")
	if want := (start + 1) % n; m.createListIndex != want {
		t.Errorf("] moved the list from %d to %d, want %d", start, m.createListIndex, want)
	}
	m, _ = press(t, m, "[")
	m, _ = press(t, m, "[")
	if want := (start - 1 + n) % n; m.createListIndex != want {
		t.Errorf("[ [ from %d gave %d, want %d (wrapping)", start, m.createListIndex, want)
	}
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}
