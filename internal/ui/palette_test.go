package ui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tocli/internal/ui/components"
	"tocli/internal/ui/theme"
)

// runPalette opens the palette in command mode, types query, runs the first match and returns
// the model plus the command's result.
func runPalette(t *testing.T, m Model, query string) (Model, tea.Cmd) {
	t.Helper()
	m, _ = press(t, m, ":")
	m = typeText(t, m, query)
	if len(m.search.commands) == 0 {
		t.Fatalf("no command matches %q (available: %v)", query, commandTitles(m))
	}
	return press(t, m, "enter")
}

func commandTitles(m Model) []string {
	var out []string
	for _, c := range m.search.commands {
		out = append(out, c.cmd.title)
	}
	return out
}

func TestColonOpensThePaletteInCommandMode(t *testing.T) {
	m := newMockModel(t)
	m, cmd := press(t, m, ":")
	if m.mode != modeSearch || !m.commandMode() {
		t.Fatalf("mode=%v commandMode=%v", m.mode, m.commandMode())
	}
	if m.search.input.Value() != ">" {
		t.Errorf("input = %q, want the > prefix", m.search.input.Value())
	}
	if len(m.search.commands) < 10 {
		t.Errorf("an empty command query should list every available command, got %d", len(m.search.commands))
	}
	if cmd == nil {
		t.Error("opening should start the animation")
	}
}

func TestTypingAGreaterThanSwitchesAnOpenSearchToCommands(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, "/")
	if m.commandMode() {
		t.Fatal("/ should start in event mode")
	}
	m = typeText(t, m, ">")
	if !m.commandMode() || len(m.search.commands) == 0 || m.search.results != nil {
		t.Errorf("after >: commandMode=%v commands=%d results=%v", m.commandMode(), len(m.search.commands), m.search.results)
	}
	// Deleting the > goes back to events.
	next, _ := m.Update(termKey(t, "backspace"))
	m = asModel(t, next)
	if m.commandMode() || m.search.commands != nil {
		t.Errorf("after backspace: commandMode=%v commands=%v", m.commandMode(), m.search.commands)
	}
}

func TestCommandQueryParsing(t *testing.T) {
	for in, want := range map[string]bool{">": true, "> theme": true, "  >x": true, "theme": false, "": false, "a>b": false} {
		if got := isCommandQuery(in); got != want {
			t.Errorf("isCommandQuery(%q) = %v, want %v", in, got, want)
		}
	}
	if got := commandQuery("  > new task "); got != " new task " {
		t.Errorf("commandQuery = %q", got)
	}
}

func TestCommandsFilterByNameIgnoringCaseAndAccents(t *testing.T) {
	m := newMockModel(t)
	for _, q := range []string{"theme", "THEME", "tHeMe"} {
		got := m.matchCommands(q)
		if len(got) != 3 {
			t.Errorf("%q matched %d commands, want the 3 themes", q, len(got))
		}
	}
	if got := m.matchCommands("light"); len(got) != 1 || got[0].cmd.title != "Theme: light" {
		t.Errorf("light -> %v", got)
	}
	if got := m.matchCommands("zzz"); len(got) != 0 {
		t.Errorf("zzz matched %d", len(got))
	}
	// Terms in any order.
	if got := m.matchCommands("task new"); len(got) == 0 || got[0].cmd.title != "New task" {
		t.Errorf("'task new' should still find New task first, got %v", got)
	}
}

func TestCommandsRankPrefixAndWordStartFirst(t *testing.T) {
	m := newMockModel(t)
	got := m.matchCommands("ref")
	if len(got) == 0 || got[0].cmd.title != "Refresh data" {
		t.Errorf("'ref' should put Refresh data first, got %v", got)
	}
}

func TestUnavailableCommandsAreHidden(t *testing.T) {
	m := newMockModel(t)
	titles := func(m Model) map[string]bool {
		out := map[string]bool{}
		for _, c := range m.matchCommands("") {
			out[c.cmd.title] = true
		}
		return out
	}

	before := titles(m)
	if before["Write note for the selected day"] || before["Export this month's ratings to CSV"] {
		t.Error("daily-mode commands should be hidden in contribution mode")
	}
	if !before["Edit selected task"] || !before["Delete selected task"] {
		t.Error("task commands should be available when a task is selected")
	}

	m.contribution.Mode = components.ModeDaily
	if after := titles(m); !after["Write note for the selected day"] || !after["Export this month's ratings to CSV"] {
		t.Error("daily-mode commands should appear in daily mode")
	}

	m.tasks.Tasks = nil
	if none := titles(m); none["Edit selected task"] || none["Delete selected task"] || none["Complete or reopen selected task"] {
		t.Error("task commands must be hidden with nothing selected")
	}
}

func TestEveryCommandHasATitleAndARunFunction(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range newMockModel(t).paletteCommands() {
		if strings.TrimSpace(c.title) == "" || c.run == nil {
			t.Errorf("incomplete command %+v", c)
		}
		if seen[c.title] {
			t.Errorf("duplicate command title %q", c.title)
		}
		seen[c.title] = true
	}
}

// ---- running commands -------------------------------------------------------------------------

func TestCommandNewTask(t *testing.T) {
	m, _ := runPalette(t, newMockModel(t), "new task")
	if m.mode != modeCreateTask || m.editTask != nil {
		t.Errorf("mode=%v editTask=%v, want a blank new-task form", m.mode, m.editTask)
	}
}

func TestCommandEditSelectedTask(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")
	m, _ = runPalette(t, m, "edit")
	if m.mode != modeCreateTask || m.editTask == nil || m.editTask.ID != task.ID {
		t.Fatalf("mode=%v editTask=%+v, want the form editing %q", m.mode, m.editTask, task.ID)
	}
	if m.createInput.Value() != task.Title {
		t.Errorf("title field = %q", m.createInput.Value())
	}
}

func TestCommandCompleteSelectedTask(t *testing.T) {
	m := newMockModel(t)
	task := selectTask(t, &m, "Buy groceries")
	m, cmd := runPalette(t, m, "complete")
	if m.mode != modeNormal || cmd == nil {
		t.Fatalf("mode=%v cmd=%v", m.mode, cmd != nil)
	}
	m = feed(t, m, cmd)
	for _, x := range m.tasks.Tasks {
		if x.ID == task.ID && x.Status == 0 {
			t.Error("the task is still open")
		}
	}
}

func TestCommandDeleteAsksForConfirmationFirst(t *testing.T) {
	m := newMockModel(t)
	before := len(m.tasks.Tasks)
	m, _ = runPalette(t, m, "delete")
	if m.mode != modeConfirmDelete {
		t.Fatalf("mode = %v, want the y/n confirmation (a command must not delete on its own)", m.mode)
	}
	if len(m.tasks.Tasks) != before {
		t.Error("the task was deleted without confirmation")
	}
	m, _ = press(t, m, "n")
	if m.mode != modeNormal || len(m.tasks.Tasks) != before {
		t.Error("answering n should cancel")
	}
}

func TestCommandGoToToday(t *testing.T) {
	m := graphModel(t)
	m, _ = press(t, m, "[")
	m, _ = press(t, m, "[")
	m.activePane = TaskPane
	m, cmd := runPalette(t, m, "today")
	today := time.Now()
	if m.graphYear != today.Year() || m.activePane != GraphPane {
		t.Errorf("year=%d pane=%v, want the current year with the graph focused", m.graphYear, m.activePane)
	}
	if c := m.contribution.CursorDate; c.Year() != today.Year() || c.YearDay() != today.YearDay() {
		t.Errorf("cursor = %v, want today", c)
	}
	if cmd == nil {
		t.Error("going to another year must load it")
	}
}

func TestCommandsPreviousAndNextYear(t *testing.T) {
	m := newMockModel(t)
	year := m.graphYear
	m, _ = runPalette(t, m, "previous year")
	if m.graphYear != year-1 || m.activePane != GraphPane {
		t.Errorf("previous: year=%d pane=%v", m.graphYear, m.activePane)
	}
	m, _ = runPalette(t, m, "next year")
	m, _ = runPalette(t, m, "next year")
	if m.graphYear != year+1 {
		t.Errorf("next twice: year=%d, want %d", m.graphYear, year+1)
	}
}

func TestCommandToggleGraphMode(t *testing.T) {
	m := newMockModel(t)
	m, _ = runPalette(t, m, "daily rating")
	if m.contribution.Mode != components.ModeDaily || m.activePane != GraphPane {
		t.Errorf("mode=%v pane=%v", m.contribution.Mode, m.activePane)
	}
	m, _ = runPalette(t, m, "contribution and daily")
	if m.contribution.Mode != components.ModeContribution {
		t.Error("running it again should switch back")
	}
}

func TestCommandsWriteNoteAndExportOnlyInDailyMode(t *testing.T) {
	m := newMockModel(t)
	m.contribution.Mode = components.ModeDaily
	m, _ = runPalette(t, m, "note")
	if m.mode != modeWriteNote {
		t.Errorf("mode = %v, want the note editor", m.mode)
	}
	m, _ = press(t, m, "esc")

	m, cmd := runPalette(t, m, "export")
	if cmd == nil {
		t.Fatal("export should run a command")
	}
	_ = m
}

func TestCommandFocusPanes(t *testing.T) {
	for query, want := range map[string]Pane{"focus tasks": TaskPane, "focus agenda": AgendaPane, "focus graph": GraphPane} {
		m := newMockModel(t)
		m.activePane = Pane((int(want) + 1) % int(paneCount))
		m, _ = runPalette(t, m, query)
		if m.activePane != want || m.mode != modeNormal {
			t.Errorf("%q: pane=%v mode=%v, want %v", query, m.activePane, m.mode, want)
		}
	}
}

func TestCommandThemesSwitchAtRuntime(t *testing.T) {
	m := newMockModel(t)
	if m.themeMode != "auto" {
		t.Fatalf("initial themeMode = %q", m.themeMode)
	}
	darkBar, darkProgress := m.renderStatusBar(), m.progress.View()
	m, _ = runPalette(t, m, "theme: light")
	if m.themeMode != "light" || !reflect.DeepEqual(m.styles.T, theme.Light) {
		t.Errorf("light: themeMode=%q, palette applied=%v", m.themeMode, reflect.DeepEqual(m.styles.T, theme.Light))
	}
	// The components got the new palette too, not just the model.
	if m.renderStatusBar() == darkBar {
		t.Error("the status bar looks the same after switching to the light theme")
	}
	if m.progress.View() == darkProgress {
		t.Error("the progress card (a component) kept the dark palette")
	}

	m, _ = runPalette(t, m, "theme: dark")
	if m.themeMode != "dark" || !reflect.DeepEqual(m.styles.T, theme.Dark) {
		t.Errorf("dark: themeMode=%q", m.themeMode)
	}

	// A pinned theme ignores the terminal's background report...
	next, _ := m.Update(tea.BackgroundColorMsg{})
	if got := asModel(t, next).styles.T; !reflect.DeepEqual(got, theme.Dark) {
		t.Error("a pinned theme was overridden by the terminal background")
	}
	// ...following the terminal again re-queries it and then listens.
	m, cmd := runPalette(t, m, "follow the terminal")
	if m.themeMode != "auto" {
		t.Errorf("themeMode = %q, want auto", m.themeMode)
	}
	if cmd == nil {
		t.Error("going back to auto must ask the terminal for its background color")
	}
}

func TestCommandRefreshAndHelpAndQuit(t *testing.T) {
	m, cmd := runPalette(t, newMockModel(t), "refresh")
	if !m.loading || cmd == nil {
		t.Errorf("refresh: loading=%v cmd=%v", m.loading, cmd != nil)
	}
	m = drain(t, m, cmd)
	if m.loading {
		t.Error("loading should clear once the tasks arrive")
	}

	m, _ = runPalette(t, newMockModel(t), "keyboard shortcuts")
	if m.mode != modeHelp {
		t.Errorf("mode = %v, want help", m.mode)
	}

	_, cmd = runPalette(t, newMockModel(t), "quit")
	if cmd == nil {
		t.Fatal("quit returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("quit command produced %T", cmd())
	}
}

func TestRunningACommandClosesThePaletteFirst(t *testing.T) {
	// A command that opens a modal must land in that modal, not back in the palette.
	m, _ := runPalette(t, newMockModel(t), "new task")
	if m.mode != modeCreateTask {
		t.Errorf("mode = %v", m.mode)
	}
	if m.search.input.Value() != "" || m.search.commands != nil {
		t.Error("the palette kept its query after closing")
	}
}

func TestCommandEnterWithNoMatchDoesNothing(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, ":")
	m = typeText(t, m, "zzzz")
	m, cmd := press(t, m, "enter")
	if m.mode != modeSearch || cmd != nil {
		t.Errorf("mode=%v cmd=%v, want the palette to stay open", m.mode, cmd != nil)
	}
}

func TestCommandSelectionClampsAndSurvivesRefiltering(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, ":")
	for i := 0; i < 100; i++ {
		next, _ := m.Update(termKey(t, "down"))
		m = asModel(t, next)
	}
	if m.search.sel != len(m.search.commands)-1 {
		t.Errorf("sel = %d, want the last of %d", m.search.sel, len(m.search.commands))
	}
	m = typeText(t, m, "q") // narrows the list; the selection must restart inside it
	if m.search.sel != 0 {
		t.Errorf("after typing, sel = %d, want 0", m.search.sel)
	}
}

// ---- rendering --------------------------------------------------------------------------------

func TestCommandPaletteShowsTitlesShortcutsAndFooter(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, ":")
	m.search.anim = 1
	out := stripANSI(m.render())
	for _, want := range []string{"New task", "Edit selected task", "Complete or reopen selected task", "1 of ", "enter run"} {
		if !strings.Contains(out, want) {
			t.Errorf("command list is missing %q", want)
		}
	}
	// The shortcut is shown on the same row as its command.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "New task") && strings.Contains(line, "│") {
			if !strings.Contains(line, " n ") && !strings.Contains(line, " n│") && !strings.Contains(line, "n │") {
				t.Errorf("no shortcut next to New task: %q", line)
			}
			return
		}
	}
	t.Error("did not find the New task row in the palette")
}

func TestCommandPaletteEmptyState(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, ":")
	m.search.anim = 1
	m = typeText(t, m, "zzz")
	if out := stripANSI(m.render()); !strings.Contains(out, "no command matches") {
		t.Error("expected the empty-state message")
	}
}

func TestCommandPaletteStatusBarAndHint(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, "/")
	m.search.anim = 1
	if bar := stripANSI(m.renderSearchStatusBar()); !strings.Contains(bar, "commands") {
		t.Errorf("the event search bar should mention >: %q", bar)
	}
	if out := stripANSI(m.render()); !strings.Contains(out, "> for commands") {
		t.Error("the empty search should hint at the command mode")
	}
	m = typeText(t, m, ">")
	if bar := stripANSI(m.renderSearchStatusBar()); !strings.Contains(bar, "run") {
		t.Errorf("command mode status bar should say enter runs: %q", bar)
	}
}

func TestCommandPaletteRendersAtEverySizeWithinTheTerminal(t *testing.T) {
	for _, query := range []string{">", ">e", ">zzz"} {
		for w := 12; w <= 180; w += 24 {
			for h := 3; h <= 60; h += 11 {
				m := newMockModel(t)
				next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
				m = asModel(t, next)
				m.openSearch("")
				m.search.anim = 1
				m = typeText(t, m, query)

				lines := strings.Split(m.render(), "\n")
				if len(lines) != h {
					t.Fatalf("%q %dx%d: %d lines", query, w, h, len(lines))
				}
				for i, l := range lines {
					if lw := lipgloss.Width(l); lw > w {
						t.Fatalf("%q %dx%d: line %d is %d cells", query, w, h, i, lw)
					}
				}
			}
		}
	}
}

func TestClickingACommandRunsIt(t *testing.T) {
	m := newMockModel(t)
	m, _ = press(t, m, ":")
	m.search.anim = 1
	g := m.currentSearchGeometry()

	// The second row is "Edit selected task" (a task is selected in the mock data).
	if got := m.search.commands[1].cmd.title; got != "Edit selected task" {
		t.Fatalf("row 2 is %q", got)
	}
	next, _ := m.Update(tea.MouseClickMsg{X: g.results.x + 6, Y: g.results.y + 1 + 1, Button: tea.MouseLeft})
	m = asModel(t, next)
	if m.mode != modeCreateTask || m.editTask == nil {
		t.Errorf("clicking the row left mode=%v editTask=%v, want the edit form", m.mode, m.editTask)
	}
}

func TestSearchBoxLabelMentionsCommandsWhenThereIsRoom(t *testing.T) {
	m := newMockModel(t)
	l := computeLayout(m.width, bodyOuterLines(m.height))
	if got := stripANSI(m.renderSearchBox(l.search)); !strings.Contains(got, "> commands") {
		t.Errorf("box = %q, want it to advertise the command mode", got)
	}
	if got := stripANSI(m.renderSearchBox(box{W: 25, H: 1})); strings.Contains(got, "> commands") || !strings.Contains(got, "search") {
		t.Errorf("narrow box = %q", got)
	}
}
