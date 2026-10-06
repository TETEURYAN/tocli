package ui

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// drain runs a command and everything it fans out into (batches), feeding each message back to
// the model. Search animation ticks are not followed, so it always terminates.
func drain(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = drain(t, m, c)
		}
		return m
	}
	if _, isTick := msg.(searchTickMsg); isTick {
		return m
	}
	next, follow := m.Update(msg)
	m = asModel(t, next)
	if _, isYearTick := msg.(yearTickMsg); isYearTick {
		return drain(t, m, follow)
	}
	return m
}

func graphModel(t *testing.T) Model {
	t.Helper()
	m := newMockModel(t)
	m = drain(t, m, m.loadGraphYear())
	m.activePane = GraphPane
	m.updateFocus()
	return m
}

func TestYearKeysChangeTheGraphYearKeepingMonthAndDay(t *testing.T) {
	m := graphModel(t)
	thisYear := time.Now().Year()
	m.contribution.CursorDate = time.Date(thisYear, time.March, 14, 0, 0, 0, 0, time.Local)

	m, cmd := press(t, m, "[")
	if m.graphYear != thisYear-1 || m.contribution.Data.Year != thisYear-1 {
		t.Fatalf("[ : graphYear=%d data=%d, want %d", m.graphYear, m.contribution.Data.Year, thisYear-1)
	}
	if c := m.contribution.CursorDate; c.Year() != thisYear-1 || c.Month() != time.March || c.Day() != 14 {
		t.Errorf("cursor = %v, want 14 March of the previous year", c)
	}
	if !m.contribution.Loading {
		t.Error("the new year should be marked loading right away")
	}
	if cmd == nil {
		t.Error("switching year must schedule the (debounced) load")
	}

	m, _ = press(t, m, "]")
	m, _ = press(t, m, "]")
	if m.graphYear != thisYear+1 {
		t.Errorf("graphYear = %d, want %d", m.graphYear, thisYear+1)
	}
}

func TestYearKeysOnlyActInTheGraphPane(t *testing.T) {
	m := newMockModel(t)
	year := m.graphYear
	for _, pane := range []Pane{TaskPane, AgendaPane} {
		m.activePane = pane
		m, _ = press(t, m, "[")
		m, _ = press(t, m, "]")
		if m.graphYear != year {
			t.Fatalf("pane %v: year changed to %d", pane, m.graphYear)
		}
	}
}

func TestYearSwitchingIsDebouncedToTheLastPause(t *testing.T) {
	m := graphModel(t)
	var ticks []tea.Cmd
	for i := 0; i < 3; i++ {
		var cmd tea.Cmd
		m, cmd = press(t, m, "[")
		ticks = append(ticks, cmd)
	}
	if m.graphYear != time.Now().Year()-3 {
		t.Fatalf("graphYear = %d after three presses", m.graphYear)
	}

	// The first two pauses were interrupted: their ticks must do nothing.
	for i := 0; i < 2; i++ {
		next, cmd := m.Update(ticks[i]())
		m = asModel(t, next)
		if cmd != nil {
			t.Errorf("stale tick %d started a load", i)
		}
	}
	next, cmd := m.Update(ticks[2]())
	m = asModel(t, next)
	if cmd == nil {
		t.Fatal("the last tick should load the year")
	}
	m = drain(t, m, cmd)
	if m.contribution.Loading {
		t.Error("the graph is still loading after the data arrived")
	}
	if m.contribution.Data.Year != m.graphYear {
		t.Errorf("data is for %d, want %d", m.contribution.Data.Year, m.graphYear)
	}
}

func TestLateAnswersForAnotherYearAreDiscarded(t *testing.T) {
	m := graphModel(t)
	m, _ = press(t, m, "[") // now showing last year, loading

	stale := m.contribUC.Generate(time.Now().Year()) // the answer for the year we just left
	next, _ := m.Update(contributionLoadedMsg{data: stale})
	m = asModel(t, next)
	if m.contribution.Data.Year != time.Now().Year()-1 || !m.contribution.Loading {
		t.Errorf("a late answer for %d overwrote the graph: data=%d loading=%v",
			stale.Year, m.contribution.Data.Year, m.contribution.Loading)
	}

	staleRatings := m.ratingUC.Generate(time.Now().Year())
	next, _ = m.Update(ratingLoadedMsg{data: staleRatings})
	if got := asModel(t, next).contribution.RatingData.Year; got != time.Now().Year()-1 {
		t.Errorf("late ratings for another year were applied (year %d)", got)
	}
}

func TestYearBoundsStopWithANotice(t *testing.T) {
	m := graphModel(t)
	m.graphYear = minGraphYear
	m.contribution.SetYear(minGraphYear, time.Date(minGraphYear, 6, 1, 0, 0, 0, 0, time.Local))
	m, cmd := press(t, m, "[")
	if m.graphYear != minGraphYear || cmd != nil || !strings.Contains(m.notice, "covers") {
		t.Errorf("below the minimum: year=%d cmd=%v notice=%q", m.graphYear, cmd != nil, m.notice)
	}

	top := time.Now().Year() + maxFutureYears
	m.graphYear = top
	m.contribution.SetYear(top, time.Date(top, 6, 1, 0, 0, 0, 0, time.Local))
	m, cmd = press(t, m, "]")
	if m.graphYear != top || cmd != nil {
		t.Errorf("above the maximum: year=%d cmd=%v", m.graphYear, cmd != nil)
	}
}

func TestSameDayInYear(t *testing.T) {
	leapDay := time.Date(2024, 2, 29, 0, 0, 0, 0, time.Local)
	if got := sameDayInYear(leapDay, 2025); got.Month() != time.February || got.Day() != 28 {
		t.Errorf("29 Feb -> 2025 = %v, want 28 Feb", got)
	}
	if got := sameDayInYear(leapDay, 2028); got.Month() != time.February || got.Day() != 29 {
		t.Errorf("29 Feb -> 2028 = %v, want 29 Feb", got)
	}
	d := time.Date(2026, 12, 31, 0, 0, 0, 0, time.Local)
	if got := sameDayInYear(d, 2020); got.Month() != time.December || got.Day() != 31 || got.Year() != 2020 {
		t.Errorf("31 Dec -> 2020 = %v", got)
	}
	jan31 := time.Date(2026, 1, 31, 0, 0, 0, 0, time.Local)
	if got := sameDayInYear(jan31, 2027); got.Month() != time.January || got.Day() != 31 {
		t.Errorf("31 Jan must stay in January, got %v", got)
	}
}

func TestRefreshReloadsTheYearBeingViewed(t *testing.T) {
	m := graphModel(t)
	m.graphYear = time.Now().Year() - 2
	msg := m.loadContribution()()
	if got := msg.(contributionLoadedMsg).data.Year; got != m.graphYear {
		t.Errorf("refresh loaded %d, want the viewed year %d", got, m.graphYear)
	}
	if got := m.loadRatings()().(ratingLoadedMsg).data.Year; got != m.graphYear {
		t.Errorf("refresh loaded ratings for %d, want %d", got, m.graphYear)
	}
}

func TestCursorStaysInsideTheViewedYear(t *testing.T) {
	m := graphModel(t)
	m, _ = press(t, m, "[")
	y := m.graphYear
	m.contribution.CursorDate = time.Date(y, 12, 31, 0, 0, 0, 0, time.Local)
	for i := 0; i < 3; i++ {
		next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
		m = asModel(t, next)
	}
	if m.contribution.CursorDate.Year() != y {
		t.Errorf("moving right past 31 Dec left year %d (cursor %v)", y, m.contribution.CursorDate)
	}
}

func TestGraphShowsTheViewedYearAndLoadingState(t *testing.T) {
	m := graphModel(t)
	m.width, m.height = 120, 36
	m.updateLayout()
	m, _ = press(t, m, "[")
	m.contribution.Focused = true
	view := stripANSI(m.contribution.View())
	want := time.Now().Year() - 1
	if !strings.Contains(view, "Contribution Graph") || !strings.Contains(view, itoa(want)) {
		t.Errorf("graph does not show %d:\n%s", want, view)
	}
	if !strings.Contains(view, "loading") {
		t.Errorf("graph should say it is loading:\n%s", view)
	}
}

func TestGoToDateInTheSameYearDoesNotReloadTheGraph(t *testing.T) {
	m := graphModel(t)
	seq := m.yearSeq
	day := time.Date(m.graphYear, 5, 5, 0, 0, 0, 0, time.Local)
	m2 := m
	cmd := m2.goToDate(day)
	if m2.graphYear != m.graphYear || m2.yearSeq != seq || m2.contribution.Loading {
		t.Errorf("same-year jump touched the year state: year=%d seq=%d loading=%v", m2.graphYear, m2.yearSeq, m2.contribution.Loading)
	}
	if !m2.contribution.CursorDate.Equal(day) || cmd == nil {
		t.Errorf("cursor=%v cmd=%v", m2.contribution.CursorDate, cmd != nil)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }
