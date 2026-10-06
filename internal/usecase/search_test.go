package usecase

import (
	"reflect"
	"testing"
	"time"

	"tocli/internal/domain"
)

var now = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func ev(id, title, loc, descr string, daysFromNow int) domain.Event {
	start := now.AddDate(0, 0, daysFromNow)
	return domain.Event{ID: id, Title: title, Location: loc, Description: descr, StartTime: start, EndTime: start.Add(time.Hour)}
}

func ids(ms []EventMatch) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.Event.ID)
	}
	return out
}

func TestSearchIgnoresCaseAndAccents(t *testing.T) {
	ix := NewEventIndex([]domain.Event{
		ev("a", "Reunião de planejamento", "", "", 1),
		ev("b", "Lunch", "Café Central", "", 2),
	})
	for _, q := range []string{"reuniao", "REUNIÃO", "reunião", "Reuniao"} {
		if got := ids(ix.Search(q, now, 0)); !reflect.DeepEqual(got, []string{"a"}) {
			t.Errorf("query %q matched %v, want [a]", q, got)
		}
	}
	if got := ids(ix.Search("cafe", now, 0)); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("location accent-insensitive match = %v, want [b]", got)
	}
}

func TestSearchRequiresEveryTerm(t *testing.T) {
	ix := NewEventIndex([]domain.Event{
		ev("a", "Sprint Planning", "Room 3A", "", 1),
		ev("b", "Sprint Review", "Room 9", "", 2),
	})
	if got := ids(ix.Search("sprint room", now, 0)); len(got) != 2 {
		t.Errorf("terms spread across title and location: got %v, want both", got)
	}
	if got := ids(ix.Search("sprint planning", now, 0)); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("got %v, want only a", got)
	}
	if got := ix.Search("sprint nonexistent", now, 0); len(got) != 0 {
		t.Errorf("a missing term must exclude the event, got %v", ids(got))
	}
}

func TestSearchBlankAndNilIndex(t *testing.T) {
	ix := NewEventIndex([]domain.Event{ev("a", "x", "", "", 0)})
	for _, q := range []string{"", "   ", "\t"} {
		if got := ix.Search(q, now, 0); got != nil {
			t.Errorf("blank query %q returned %v", q, ids(got))
		}
	}
	var nilIx *EventIndex
	if nilIx.Search("x", now, 0) != nil || nilIx.Len() != 0 {
		t.Error("a nil index must behave as empty")
	}
}

func TestSearchRanksTitleOverLocationOverDescription(t *testing.T) {
	ix := NewEventIndex([]domain.Event{
		ev("descr", "Weekly", "", "talk about budget", 1),
		ev("loc", "Weekly", "Budget room", "", 1),
		ev("title", "Budget review", "", "", 1),
	})
	want := []string{"title", "loc", "descr"}
	if got := ids(ix.Search("budget", now, 0)); !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestSearchPrefersWordStartAndPrefix(t *testing.T) {
	ix := NewEventIndex([]domain.Event{
		ev("inside", "Backlog", "", "", 1),        // "log" inside a word
		ev("start", "Log review", "", "", 1),      // "log" starts the title
		ev("word", "Review log files", "", "", 1), // "log" starts a later word
	})
	want := []string{"start", "word", "inside"}
	if got := ids(ix.Search("log", now, 0)); !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestSearchTiesBreakByClosenessToNow(t *testing.T) {
	ix := NewEventIndex([]domain.Event{
		ev("lastyear", "Standup", "", "", -365),
		ev("tomorrow", "Standup", "", "", 1),
		ev("yesterday", "Standup", "", "", -1),
		ev("nextmonth", "Standup", "", "", 30),
	})
	got := ids(ix.Search("standup", now, 0))
	// tomorrow and yesterday are equally close; the earlier one wins that tie.
	want := []string{"yesterday", "tomorrow", "nextmonth", "lastyear"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestSearchLimit(t *testing.T) {
	var events []domain.Event
	for i := 0; i < 20; i++ {
		events = append(events, ev(string(rune('a'+i)), "Standup", "", "", i))
	}
	ix := NewEventIndex(events)
	if got := len(ix.Search("standup", now, 5)); got != 5 {
		t.Errorf("limit 5 returned %d", got)
	}
	if got := len(ix.Search("standup", now, 0)); got != 20 {
		t.Errorf("no limit returned %d", got)
	}
}

func TestTitleHitsPointAtOriginalRunes(t *testing.T) {
	ix := NewEventIndex([]domain.Event{ev("a", "Reunião de planejamento", "", "", 0)})
	m := ix.Search("reuniao", now, 0)[0]
	if want := []int{0, 1, 2, 3, 4, 5, 6}; !reflect.DeepEqual(m.TitleHits, want) {
		t.Errorf("hits = %v, want %v", m.TitleHits, want)
	}

	// Two terms, the second inside a later word; hits are the union, sorted and unique.
	m = ix.Search("plan reu", now, 0)[0]
	title := []rune("Reunião de planejamento")
	var got string
	for _, i := range m.TitleHits {
		got += string(title[i])
	}
	if got != "Reuplan" {
		t.Errorf("highlighted %q, want %q", got, "Reuplan")
	}
}

func TestTitleHitsEmptyWhenOnlyOtherFieldsMatch(t *testing.T) {
	ix := NewEventIndex([]domain.Event{ev("a", "Weekly", "Rooftop", "", 0)})
	m := ix.Search("rooftop", now, 0)
	if len(m) != 1 || len(m[0].TitleHits) != 0 {
		t.Errorf("got %+v, want one match with no title highlight", m)
	}
}

func TestFoldKeepsRuneMapping(t *testing.T) {
	f := fold("Ação")
	if string(f.runes) != "acao" {
		t.Fatalf("folded = %q", string(f.runes))
	}
	if want := []int{0, 1, 2, 3}; !reflect.DeepEqual(f.orig, want) {
		t.Errorf("orig = %v, want %v", f.orig, want)
	}
	// A decomposed accent (e + U+0301) disappears but must not shift the mapping.
	f = fold("éx")
	if string(f.runes) != "ex" || !reflect.DeepEqual(f.orig, []int{0, 2}) {
		t.Errorf("decomposed: runes=%q orig=%v", string(f.runes), f.orig)
	}
}

func TestMatchText(t *testing.T) {
	score, hits, ok := MatchText("tema", "Mudar Tema escuro")
	if !ok || score == 0 {
		t.Fatalf("score=%d ok=%v", score, ok)
	}
	if want := []int{6, 7, 8, 9}; !reflect.DeepEqual(hits, want) {
		t.Errorf("hits = %v, want %v", hits, want)
	}

	if _, _, ok := MatchText("xyz", "Mudar Tema"); ok {
		t.Error("a term that is not there must not match")
	}
	if _, _, ok := MatchText("mudar xyz", "Mudar Tema"); ok {
		t.Error("every term has to match")
	}
	if _, hits, ok := MatchText("tema mudar", "Mudar Tema"); !ok || len(hits) != 9 {
		t.Errorf("terms in any order: ok=%v hits=%v", ok, hits)
	}
	if _, _, ok := MatchText("ação", "Acao rapida"); !ok {
		t.Error("accents must not matter, in either direction")
	}
	if s, h, ok := MatchText("   ", "anything"); !ok || s != 0 || h != nil {
		t.Errorf("blank query = %d %v %v, want a match with no score", s, h, ok)
	}
}

func TestMatchTextPrefersPrefixAndWordStart(t *testing.T) {
	prefix, _, _ := MatchText("ref", "Refresh data")
	word, _, _ := MatchText("ref", "Force refresh")
	inside, _, _ := MatchText("ref", "Prefer")
	if !(prefix > word && word > inside) {
		t.Errorf("scores: prefix=%d word=%d inside=%d, want prefix > word > inside", prefix, word, inside)
	}
}
