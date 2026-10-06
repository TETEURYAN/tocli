package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"tocli/internal/domain"
	"tocli/internal/usecase"
)

func searchEvent(id, title, loc string, daysFromNow int) domain.Event {
	start := time.Now().AddDate(0, 0, daysFromNow).Truncate(time.Hour)
	return domain.Event{ID: id, Title: title, Location: loc, StartTime: start, EndTime: start.Add(time.Hour)}
}

// searchModel is a model with the palette open, settled in the middle, over a known index.
func searchModel(t *testing.T, events ...domain.Event) Model {
	t.Helper()
	m := newTestModel(t)
	m.openSearch("")
	m.search.anim = 1
	m.search.loading = false
	m.search.index = usecase.NewEventIndex(events)
	return m
}

func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m, _ = press(t, m, string(r))
	}
	return m
}

// ---- geometry -------------------------------------------------------------------------------

func TestSearchGeometryStartsOnThePillAndEndsCentered(t *testing.T) {
	pill := rect{x: 50, y: 0, w: 18, h: 1}

	start := searchGeometry(120, 36, 0, pill, 3)
	if start.bar.x != pill.x || start.bar.y != 0 || start.bar.w != pill.w {
		t.Errorf("progress 0: bar = %+v, want it on the pill %+v", start.bar, pill)
	}
	if start.results.h != 0 {
		t.Error("results must stay hidden while the bar is travelling")
	}

	end := searchGeometry(120, 36, 1, pill, 3)
	if want := (120 - end.bar.w) / 2; end.bar.x != want {
		t.Errorf("settled bar x = %d, want horizontally centered (%d)", end.bar.x, want)
	}
	// Bar plus the full results area is centered vertically (within a row of rounding).
	total := searchBarH + searchMaxRows + 3
	if top, bottom := end.bar.y, 36-(end.bar.y+total); top-bottom > 1 || bottom-top > 1 {
		t.Errorf("settled block is not vertically centered: %d rows above, %d below", top, bottom)
	}
	if end.results.y != end.bar.y+searchBarH {
		t.Errorf("results at y=%d, want directly under the bar (%d)", end.results.y, end.bar.y+searchBarH)
	}
}

func TestSearchGeometryTravelsMonotonically(t *testing.T) {
	pill := rect{x: 50, y: 0, w: 18, h: 1}
	prev := searchGeometry(120, 36, 0, pill, 0).bar
	for p := 0.1; p <= 1.0001; p += 0.1 {
		cur := searchGeometry(120, 36, p, pill, 0).bar
		if cur.y < prev.y {
			t.Fatalf("progress %.1f: bar moved back up (%d -> %d)", p, prev.y, cur.y)
		}
		if cur.w < prev.w {
			t.Fatalf("progress %.1f: bar shrank (%d -> %d)", p, prev.w, cur.w)
		}
		prev = cur
	}
}

func TestSearchGeometryStaysOnScreenAtEverySize(t *testing.T) {
	for w := 10; w <= 200; w += 7 {
		// The header only has a search box on terminals at least 50 columns wide.
		pill := rect{}
		if w >= 50 {
			pill = rect{x: (w - 18) / 2, y: 0, w: 18, h: 1}
		}
		for h := 3; h <= 70; h += 3 {
			for _, p := range []float64{0, 0.3, 1} {
				for _, n := range []int{0, 1, 5, 50} {
					g := searchGeometry(w, h, p, pill, n)
					for name, r := range map[string]rect{"bar": g.bar, "results": g.results} {
						if r.h == 0 {
							continue
						}
						if r.x < 0 || r.y < 0 || r.w <= 0 || r.x+r.w > w || r.y+r.h > h {
							t.Fatalf("%dx%d p=%.1f n=%d: %s %+v is off screen", w, h, p, n, name, r)
						}
					}
					if p == 1 && (g.bar.x < 0 || g.bar.x+g.bar.w > w) {
						t.Fatalf("%dx%d settled bar %+v exceeds the terminal width", w, h, g.bar)
					}
					if g.results.h > 0 && g.results.h != g.rows+3 {
						t.Fatalf("%dx%d: results height %d for %d rows", w, h, g.results.h, g.rows)
					}
				}
			}
		}
	}
}

func TestSearchGeometryWithoutPillStartsInTheHeader(t *testing.T) {
	g := searchGeometry(120, 36, 0, rect{}, 0)
	if g.bar.y != 0 || g.bar.w <= 0 {
		t.Errorf("bar without a pill = %+v, want a header-height start", g.bar)
	}
}

// ---- flow -----------------------------------------------------------------------------------

func TestSlashOpensSearchFromEveryPane(t *testing.T) {
	for _, pane := range []Pane{TaskPane, AgendaPane, GraphPane} {
		m := newTestModel(t)
		m.activePane = pane
		m, cmd := press(t, m, "/")
		if m.mode != modeSearch {
			t.Fatalf("pane %v: / left mode = %v", pane, m.mode)
		}
		if cmd == nil {
			t.Fatalf("pane %v: opening search must start loading and animating", pane)
		}
		if m.search.anim != 0 || !m.search.loading {
			t.Errorf("pane %v: anim=%v loading=%v on open", pane, m.search.anim, m.search.loading)
		}
	}
}

func TestSearchTypingFiltersAndQIsJustALetter(t *testing.T) {
	m := searchModel(t,
		searchEvent("a", "Quarterly review", "", 2),
		searchEvent("b", "Lunch", "", 3),
	)
	m = typeText(t, m, "q")
	if m.mode != modeSearch {
		t.Fatal("typing q must not quit or leave the palette")
	}
	if got := len(m.search.results); got != 1 || m.search.results[0].Event.ID != "a" {
		t.Errorf("results for %q = %+v", "q", m.search.results)
	}
	m = typeText(t, m, "uarter")
	if len(m.search.results) != 1 {
		t.Errorf("results after %q = %d, want 1", m.search.input.Value(), len(m.search.results))
	}
	m = typeText(t, m, "xyz")
	if len(m.search.results) != 0 {
		t.Errorf("a non-matching query should empty the results, got %d", len(m.search.results))
	}
}

func TestSearchNavigationClamps(t *testing.T) {
	m := searchModel(t,
		searchEvent("a", "Standup", "", 1),
		searchEvent("b", "Standup", "", 2),
		searchEvent("c", "Standup", "", 3),
	)
	m = typeText(t, m, "standup")
	m, _ = press(t, m, "up")
	if m.search.sel != 0 {
		t.Errorf("up at the top moved the selection to %d", m.search.sel)
	}
	for i := 0; i < 10; i++ {
		next, _ := m.Update(termKey(t, "down"))
		m = asModel(t, next)
	}
	if m.search.sel != 2 {
		t.Errorf("down past the end left selection at %d, want 2", m.search.sel)
	}
	next, _ := m.Update(termKey(t, "ctrl+p"))
	if got := asModel(t, next).search.sel; got != 1 {
		t.Errorf("ctrl+p selection = %d, want 1", got)
	}
}

func TestSearchEscapeClosesAndResets(t *testing.T) {
	m := searchModel(t, searchEvent("a", "Standup", "", 1))
	m = typeText(t, m, "stand")
	m, _ = press(t, m, "esc")
	if m.mode != modeNormal {
		t.Fatalf("esc left mode %v", m.mode)
	}
	if m.search.input.Value() != "" || len(m.search.results) != 0 {
		t.Error("closing should clear the query and results")
	}
}

func TestSearchEnterJumpsToTheDay(t *testing.T) {
	target := searchEvent("a", "Standup", "", 0)
	m := searchModel(t, target)
	m.contribution.Data.Year = target.StartTime.Year()
	m = typeText(t, m, "standup")

	m, cmd := press(t, m, "enter")
	if m.mode != modeNormal || m.activePane != GraphPane {
		t.Fatalf("after enter: mode=%v pane=%v, want normal + graph", m.mode, m.activePane)
	}
	y, mo, d := target.StartTime.In(time.Local).Date()
	if cy, cm, cd := m.contribution.CursorDate.Date(); cy != y || cm != mo || cd != d {
		t.Errorf("cursor on %v, want %d-%d-%d", m.contribution.CursorDate, y, mo, d)
	}
	if cmd == nil {
		t.Error("enter should load the day's detail")
	}
}

func TestSearchEnterOnAnotherYearSwitchesTheGraphToThatYear(t *testing.T) {
	old := searchEvent("a", "Offsite", "", -400)
	m := searchModel(t, old)
	m = typeText(t, m, "offsite")

	m, cmd := press(t, m, "enter")
	wantYear := old.StartTime.In(time.Local).Year()
	if m.mode != modeNormal || m.activePane != GraphPane {
		t.Fatalf("after enter: mode=%v pane=%v", m.mode, m.activePane)
	}
	if m.graphYear != wantYear || m.contribution.Data.Year != wantYear {
		t.Errorf("graph year = %d (data %d), want %d", m.graphYear, m.contribution.Data.Year, wantYear)
	}
	if y, mo, d := old.StartTime.In(time.Local).Date(); m.contribution.CursorDate.Year() != y ||
		m.contribution.CursorDate.Month() != mo || m.contribution.CursorDate.Day() != d {
		t.Errorf("cursor = %v, want the event's day", m.contribution.CursorDate)
	}
	if !m.contribution.Loading {
		t.Error("the new year should show as loading until its data arrives")
	}
	if cmd == nil {
		t.Error("jumping to another year must load that year")
	}
}

func TestSearchEnterWithNoResultsDoesNothing(t *testing.T) {
	m := searchModel(t, searchEvent("a", "Standup", "", 1))
	m = typeText(t, m, "zzz")
	m, cmd := press(t, m, "enter")
	if m.mode != modeSearch || cmd != nil {
		t.Errorf("enter with no hit: mode=%v cmd=%v, want to stay open", m.mode, cmd != nil)
	}
}

func TestSearchAnimationRunsToCompletionAndIgnoresStaleTicks(t *testing.T) {
	m := newTestModel(t)
	m.openSearch("")
	gen := m.search.gen

	for i := 0; i < 20 && m.search.anim < 1; i++ {
		next, _ := m.Update(searchTickMsg{gen: gen})
		m = asModel(t, next)
	}
	if m.search.anim != 1 {
		t.Fatalf("animation stuck at %v", m.search.anim)
	}

	// Close, reopen: ticks of the first opening must not advance the second.
	m.closeSearch()
	m.openSearch("")
	next, _ := m.Update(searchTickMsg{gen: gen})
	if got := asModel(t, next).search.anim; got != 0 {
		t.Errorf("a stale tick advanced the new animation to %v", got)
	}

	// And a tick after closing is a no-op.
	m.closeSearch()
	next, cmd := m.Update(searchTickMsg{gen: m.search.gen})
	if asModel(t, next).mode != modeNormal || cmd != nil {
		t.Error("ticks after closing must be ignored")
	}
}

func TestSearchLoadedUpdatesResultsForWhatWasAlreadyTyped(t *testing.T) {
	m := newTestModel(t)
	m.openSearch("")
	m = typeText(t, m, "stand")
	if len(m.search.results) != 0 {
		t.Fatal("no index yet, nothing to match")
	}
	idx := usecase.NewEventIndex([]domain.Event{searchEvent("a", "Standup", "", 1)})
	next, _ := m.Update(searchLoadedMsg{index: idx})
	m = asModel(t, next)
	if m.search.loading || len(m.search.results) != 1 {
		t.Errorf("after load: loading=%v results=%d", m.search.loading, len(m.search.results))
	}
}

func TestSearchLoadErrorKeepsThePreviousIndex(t *testing.T) {
	m := searchModel(t, searchEvent("a", "Standup", "", 1))
	next, _ := m.Update(searchLoadedMsg{err: errTest})
	m = asModel(t, next)
	if m.search.err == nil {
		t.Error("the error should be recorded")
	}
	if m.search.index.Len() != 1 {
		t.Error("a failed reload must not throw away the events already loaded")
	}
}

var errTest = &testError{}

type testError struct{}

func (*testError) Error() string { return "calendar unreachable" }

// ---- rendering ------------------------------------------------------------------------------

func TestSearchRendersAtEverySizeWithinTheTerminal(t *testing.T) {
	events := []domain.Event{
		searchEvent("a", "Reunião de planejamento trimestral com a equipe de design", "Auditório principal do campus", 2),
		searchEvent("b", "Reunião", "", 3),
	}
	for _, anim := range []float64{0, 0.5, 1} {
		for w := 12; w <= 180; w += 24 {
			for h := 3; h <= 60; h += 11 {
				m := newTestModel(t)
				next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
				m = asModel(t, next)
				m.openSearch("")
				m.search.anim = anim
				m.search.loading = false
				m.search.index = usecase.NewEventIndex(events)
				m = typeText(t, m, "reuniao")

				out := m.render()
				lines := strings.Split(out, "\n")
				if len(lines) != h {
					t.Fatalf("%dx%d anim=%.1f: %d lines, want %d", w, h, anim, len(lines), h)
				}
				for i, l := range lines {
					if lw := lipgloss.Width(l); lw > w {
						t.Fatalf("%dx%d anim=%.1f: line %d is %d cells wide", w, h, anim, i, lw)
					}
				}
			}
		}
	}
}

func TestSearchPaletteShowsResultsDimsBackgroundAndTheHint(t *testing.T) {
	m := searchModel(t,
		searchEvent("a", "Reunião de alinhamento", "Sala 2", 2),
		searchEvent("b", "Lunch", "Café", 3),
	)
	m = typeText(t, m, "reuniao")
	out := m.render()
	for _, want := range []string{"Reunião de alinhamento", "Sala 2", "1 of 1", "› reuniao"} {
		if !strings.Contains(stripANSI(out), want) {
			t.Errorf("palette is missing %q", want)
		}
	}
	if strings.Contains(stripANSI(out), "Lunch") {
		t.Error("a non-matching event is listed")
	}
	if !strings.Contains(out, "\x1b[2m") {
		t.Error("the dashboard behind the palette should be dimmed (faint)")
	}

	empty := searchModel(t, searchEvent("a", "Standup", "", 1))
	if !strings.Contains(stripANSI(empty.render()), "type to search") {
		t.Error("an empty query should show the hint under the bar")
	}
}

func TestSearchStatesInTheResultsBox(t *testing.T) {
	m := searchModel(t, searchEvent("a", "Standup", "", 1))
	m = typeText(t, m, "zzz")
	if out := stripANSI(m.render()); !strings.Contains(out, "no events match") {
		t.Error("expected the empty-state message")
	}

	m.search.loading = true
	m.search.index = nil
	m.search.results = nil
	if out := stripANSI(m.render()); !strings.Contains(out, "loading events") {
		t.Error("expected the loading message")
	}

	m.search.loading = false
	m.search.err = errTest
	if out := stripANSI(m.render()); !strings.Contains(out, "calendar unreachable") {
		t.Error("expected the load error to be shown")
	}
}

func TestSearchBoxIsAboveTheAgendaNotInTheHeader(t *testing.T) {
	m := newTestModel(t)
	if h := stripANSI(m.renderHeader()); strings.Contains(h, "search") {
		t.Errorf("the header should not carry the search box any more: %q", h)
	}

	lines := strings.Split(stripANSI(zone.Scan(m.render())), "\n")
	r, ok := m.searchRect()
	if !ok {
		t.Fatal("a 120x36 terminal has room for the search box")
	}
	row := string([]rune(lines[r.y]))
	if !strings.Contains(row, "/  search events") {
		t.Errorf("row %d should hold the search box, got %q", r.y, row)
	}
	// The agenda card's top border is the very next row, in the same column.
	next := []rune(lines[r.y+1])
	if next[r.x] != '╭' {
		t.Errorf("row below the search box starts with %q at x=%d, want the agenda's top-left corner", next[r.x], r.x)
	}
	if !strings.Contains(strings.Join(lines[r.y+1:r.y+4], "\n"), "Today's Agenda") {
		t.Error("the agenda should sit right under the search box")
	}
}

func TestSearchBoxLeavesItsSlotWhileThePaletteIsOpen(t *testing.T) {
	m := newTestModel(t)
	if !strings.Contains(stripANSI(m.renderSearchBox(computeLayout(m.width, bodyOuterLines(m.height)).search)), "search") {
		t.Fatal("the closed palette should show the search box")
	}
	m.openSearch("")
	got := stripANSI(m.renderSearchBox(computeLayout(m.width, bodyOuterLines(m.height)).search))
	if strings.TrimSpace(got) != "" {
		t.Errorf("slot should be blank while the palette is open (the bar left it), got %q", got)
	}
	if lipgloss.Width(got) != computeLayout(m.width, bodyOuterLines(m.height)).search.W {
		t.Error("the blank slot must keep the box's width so the layout does not shift")
	}
}

func TestSearchBoxHasTheSameSizeOpenOrClosed(t *testing.T) {
	for _, size := range [][2]int{{120, 36}, {80, 24}, {60, 30}, {100, 14}} {
		m := newTestModel(t)
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = asModel(t, next)
		closed := m.render()
		m.openSearch("")
		m.search.anim = 0
		open := m.render()
		if strings.Count(closed, "\n") != strings.Count(open, "\n") {
			t.Errorf("%v: frame height changed when the palette opened", size)
		}
	}
}

func TestPaletteStartsExactlyWhereTheBoxIs(t *testing.T) {
	m := newTestModel(t)
	box, ok := m.searchRect()
	if !ok {
		t.Fatal("no search rect")
	}
	m.openSearch("")
	g := m.currentSearchGeometry() // anim == 0
	if g.bar.x != box.x || g.bar.y != box.y || g.bar.w != box.w {
		t.Errorf("bar starts at %+v, want it on the search box %+v", g.bar, box)
	}
}

func TestClickingTheSearchBoxOpensThePalette(t *testing.T) {
	m := newTestModel(t)
	// Zones are registered when a frame is rendered and scanned.
	view := m.View()
	_ = view
	r, _ := m.searchRect()
	next, cmd := m.Update(tea.MouseClickMsg{X: r.x + 3, Y: r.y, Button: tea.MouseLeft})
	// bubblezone resolves zones asynchronously; give its worker a moment.
	for i := 0; i < 50 && asModel(t, next).mode != modeSearch; i++ {
		time.Sleep(10 * time.Millisecond)
		next, cmd = m.Update(tea.MouseClickMsg{X: r.x + 3, Y: r.y, Button: tea.MouseLeft})
	}
	if got := asModel(t, next).mode; got != modeSearch {
		t.Fatalf("click on the search box left mode %v", got)
	}
	if cmd == nil {
		t.Error("opening should start loading and animating")
	}
}

func TestSearchHighlightRunes(t *testing.T) {
	base := lipgloss.NewStyle()
	hit := lipgloss.NewStyle().Bold(true)
	if got := highlightRunes("Reunião", nil, base, hit); stripANSI(got) != "Reunião" {
		t.Errorf("no hits changed the text: %q", got)
	}
	got := highlightRunes("Reunião de design", []int{0, 1, 2, 11, 12}, base, hit)
	if stripANSI(got) != "Reunião de design" {
		t.Errorf("highlighting altered the text: %q", stripANSI(got))
	}
	if !strings.Contains(got, "\x1b[1m") {
		t.Error("hits should be rendered bold")
	}
}

func TestSearchDateColumn(t *testing.T) {
	now := time.Now()
	if got := searchDate(now); len(got) != len(now.Format("Mon Jan _2")) {
		t.Errorf("this year's date = %q", got)
	}
	if got := searchDate(now.AddDate(-1, 0, 0)); !strings.Contains(got, now.AddDate(-1, 0, 0).Format("2006")) {
		t.Errorf("another year's date = %q, want the year", got)
	}
}

func TestPaletteIgnoredByDashboardMouseAndKeys(t *testing.T) {
	m := searchModel(t, searchEvent("a", "Standup", "", 1))
	m, _ = press(t, m, "n") // would open the create form on the dashboard
	if m.mode != modeSearch {
		t.Errorf("n inside the palette changed mode to %v", m.mode)
	}
	if m.search.input.Value() != "n" {
		t.Errorf("n should have been typed, query = %q", m.search.input.Value())
	}
}

var sgrAny = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes SGR sequences so assertions can look at plain text.
func stripANSI(s string) string {
	return sgrAny.ReplaceAllString(s, "")
}
