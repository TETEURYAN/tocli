package domain

import (
	"reflect"
	"testing"
)

func TestExtractURLs(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string
	}{
		{"none", "no links here", nil},
		{"plain", "join https://meet.google.com/abc-defg-hij now", []string{"https://meet.google.com/abc-defg-hij"}},
		{"trailing full stop", "see https://example.com/page.", []string{"https://example.com/page"}},
		{"trailing comma and others", "a https://a.io/x, b https://b.io/y; c https://c.io/z!", []string{"https://a.io/x", "https://b.io/y", "https://c.io/z"}},
		{"in parentheses", "(see https://example.com/a)", []string{"https://example.com/a"}},
		{"balanced parentheses kept", "https://en.wikipedia.org/wiki/Go_(language)", []string{"https://en.wikipedia.org/wiki/Go_(language)"}},
		{"in angle brackets", "<https://example.com/a>", []string{"https://example.com/a"}},
		{"query and fragment", "https://a.io/p?x=1&y=2#frag", []string{"https://a.io/p?x=1&y=2#frag"}},
		{"duplicates removed, order kept", "https://b.io https://a.io https://b.io", []string{"https://b.io", "https://a.io"}},
		{"http too", "http://old.example.com", []string{"http://old.example.com"}},
		{"other schemes ignored", "ftp://x.y file:///etc/passwd javascript:alert(1) mailto:a@b.c", nil},
		{"multiline", "line 1\nhttps://a.io\nline 3 https://b.io", []string{"https://a.io", "https://b.io"}},
	}
	for _, c := range cases {
		if got := ExtractURLs(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: ExtractURLs(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestEventLinksOrderAndDedup(t *testing.T) {
	e := Event{
		MeetLink:    "https://meet.google.com/abc",
		Location:    "Room 3A or https://zoom.us/j/123",
		Description: "Agenda: https://docs.example.com/doc.\nJoin: https://meet.google.com/abc (same call)",
		URL:         "https://calendar.google.com/event?eid=1",
	}
	got := e.Links()
	want := []Link{
		{"Video call", "https://meet.google.com/abc"},
		{"zoom.us", "https://zoom.us/j/123"},
		{"docs.example.com", "https://docs.example.com/doc"},
		{"Calendar", "https://calendar.google.com/event?eid=1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Links() =\n%v\nwant\n%v", got, want)
	}
}

func TestEventLinksEmptyAndCapped(t *testing.T) {
	if got := (Event{Title: "x"}).Links(); len(got) != 0 {
		t.Errorf("an event without links gave %v", got)
	}
	var d string
	for i := 0; i < 20; i++ {
		d += "https://x.io/" + string(rune('a'+i)) + " "
	}
	if got := (Event{Description: d}).Links(); len(got) != MaxEventLinks {
		t.Errorf("got %d links, want the cap of %d", len(got), MaxEventLinks)
	}
}
