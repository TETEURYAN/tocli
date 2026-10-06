package components

import (
	"strings"
	"testing"

	"tocli/internal/domain"
	"tocli/internal/ui/theme"
)

func events(locations ...string) []domain.Event {
	var out []domain.Event
	for _, loc := range locations {
		out = append(out, domain.Event{Title: "e", Location: loc})
	}
	return out
}

func TestVisibleEventsShowsEverythingWithoutHeight(t *testing.T) {
	m := AgendaModel{Events: events("", "x", "")}
	if first, last := m.visibleEvents(); first != 0 || last != 3 {
		t.Errorf("window = [%d,%d), want [0,3)", first, last)
	}
}

func TestVisibleEventsRespectsBudget(t *testing.T) {
	// 2-row events (with location); header takes 3 rows, so Height 9 leaves 6 rows = 3 events.
	m := AgendaModel{Events: events("a", "b", "c", "d", "e"), Height: 9}
	first, last := m.visibleEvents()
	if first != 0 || last != 3 {
		t.Errorf("window = [%d,%d), want [0,3)", first, last)
	}
}

func TestVisibleEventsFollowsCursorOnlyWhenFocused(t *testing.T) {
	m := AgendaModel{Events: events("a", "b", "c", "d", "e"), Height: 9, Cursor: 4}

	if first, _ := m.visibleEvents(); first != 0 {
		t.Errorf("unfocused window starts at %d, want 0", first)
	}

	m.Focused = true
	first, last := m.visibleEvents()
	if m.Cursor < first || m.Cursor >= last {
		t.Errorf("cursor %d outside focused window [%d,%d)", m.Cursor, first, last)
	}
}

func TestVisibleEventsAlwaysShowsAtLeastOne(t *testing.T) {
	m := AgendaModel{Events: events("a", "b"), Height: 1, Focused: true, Cursor: 1}
	first, last := m.visibleEvents()
	if last-first < 1 {
		t.Errorf("empty window [%d,%d)", first, last)
	}
}

func TestAgendaViewReportsHiddenEvents(t *testing.T) {
	m := NewAgendaModel(theme.NewStyles(theme.Dark))
	m.Events = events("a", "b", "c", "d", "e")
	m.Height = 9
	if v := m.View(); !strings.Contains(v, "of 5") {
		t.Errorf("subtitle should say how many events are shown, got:\n%s", v)
	}
	m.Height = 40
	if v := m.View(); strings.Contains(v, "of 5") {
		t.Errorf("no hidden-events note expected when everything fits, got:\n%s", v)
	}
}
