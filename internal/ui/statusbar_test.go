package ui

import (
	"image/color"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"tocli/internal/ui/components"
	"tocli/internal/ui/theme"
)

func TestBarsAreExactlyTerminalWidth(t *testing.T) {
	m := newTestModel(t)
	for w := 20; w <= 200; w += 3 {
		m.width = w
		for _, pane := range []Pane{TaskPane, AgendaPane, GraphPane} {
			m.activePane = pane
			for _, mode := range []components.GraphMode{components.ModeContribution, components.ModeDaily} {
				m.contribution.Mode = mode
				for name, bar := range map[string]string{
					"status": m.renderStatusBar(),
					"header": m.renderHeader(),
					"create": m.renderCreateStatusBar(),
					"note":   m.renderWriteNoteStatusBar(),
				} {
					if got := lipgloss.Width(bar); got != w {
						t.Fatalf("%s bar at width %d pane %v is %d cells wide", name, w, pane, got)
					}
					if strings.Contains(bar, "\n") {
						t.Fatalf("%s bar at width %d wrapped onto several lines", name, w)
					}
				}
			}
		}
	}
}

func TestStatusHintsDegradeButKeepHelpAndQuit(t *testing.T) {
	m := newTestModel(t)
	for _, pane := range []Pane{TaskPane, AgendaPane, GraphPane} {
		m.activePane = pane

		m.width = 200
		wide := m.renderStatusBar()
		if !strings.Contains(wide, "quit") || !strings.Contains(wide, "help") {
			t.Errorf("pane %v: wide bar lost help/quit: %q", pane, wide)
		}

		m.width = 60
		narrow := m.renderStatusBar()
		if !strings.Contains(narrow, "quit") || !strings.Contains(narrow, "help") {
			t.Errorf("pane %v: 60-column bar dropped help/quit instead of lower-priority hints: %q", pane, narrow)
		}
		if lipgloss.Width(narrow) != 60 {
			t.Errorf("pane %v: 60-column bar is %d cells", pane, lipgloss.Width(narrow))
		}
	}
}

func TestStatusBarHasNoHolesInItsBackground(t *testing.T) {
	m := newTestModel(t)
	m.width = 120
	bar := m.renderStatusBar()
	// Every reset must be followed by the background being set again, never by visible text.
	for _, reset := range sgrReset.FindAllStringIndex(bar, -1) {
		rest := bar[reset[1]:]
		if rest != "" && !strings.HasPrefix(rest, "\x1b[") {
			t.Fatalf("text follows a reset without a background: %q", rest[:min(12, len(rest))])
		}
	}
}

func TestThemeSwitchReachesComponents(t *testing.T) {
	m := newTestModel(t)
	dark := m.renderStatusBar()
	m = m.WithTheme("light")
	if m.themeMode != "light" {
		t.Fatalf("themeMode = %q", m.themeMode)
	}
	if light := m.renderStatusBar(); light == dark {
		t.Error("status bar did not change when switching to the light theme")
	}

	// A pinned theme ignores the terminal's background report.
	next, _ := m.Update(tea.BackgroundColorMsg{Color: color.Black})
	if got := asModel(t, next).styles.T; !reflect.DeepEqual(got, theme.Light) {
		t.Error("pinned light theme was overridden by BackgroundColorMsg")
	}

	// Auto follows it.
	auto := newTestModel(t)
	next, _ = auto.Update(tea.BackgroundColorMsg{Color: color.White})
	if got := asModel(t, next).styles.T; !reflect.DeepEqual(got, theme.Light) {
		t.Error("auto theme did not switch to Light on a light background")
	}
}

func TestDailyModeKeepsItsHintsOnMidWidths(t *testing.T) {
	m := newTestModel(t)
	m.activePane = GraphPane
	m.loading = false
	m.contribution.Mode = components.ModeDaily
	for _, w := range []int{100, 110, 140} {
		m.width = w
		bar := m.renderStatusBar()
		for _, want := range []string{"day note", "rate day", "quit"} {
			if !strings.Contains(bar, want) {
				t.Errorf("width %d: daily-mode bar lost %q", w, want)
			}
		}
	}
}
