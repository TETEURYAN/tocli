package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Event struct {
	ID          string
	Title       string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Location    string
	AllDay      bool
	// URL is the event's page in the calendar web UI; MeetLink is its video-call link.
	URL      string
	MeetLink string
}

func (e *Event) Duration() time.Duration {
	return e.EndTime.Sub(e.StartTime)
}

func (e *Event) IsHappening() bool {
	now := time.Now()
	return now.After(e.StartTime) && now.Before(e.EndTime)
}

func (e *Event) IsPast() bool {
	return time.Now().After(e.EndTime)
}

func (e *Event) FormatTimeRange() string {
	if e.AllDay {
		return "All day"
	}
	return fmt.Sprintf("%s – %s", e.StartTime.Format("15:04"), e.EndTime.Format("15:04"))
}

// Link is a web address found on an event.
type Link struct {
	Label string
	URL   string
}

var urlPattern = regexp.MustCompile(`https?://[^\s<>"'\x60]+`)

// ExtractURLs returns the http(s) URLs in text, in order and without duplicates. Punctuation that
// usually follows a link in prose (a full stop, a closing parenthesis) is not part of it.
func ExtractURLs(text string) []string {
	var out []string
	seen := map[string]bool{}
	for _, raw := range urlPattern.FindAllString(text, -1) {
		u := strings.TrimRight(raw, ".,;:!?")
		// A trailing ")" or "]" belongs to the link only if it has a matching opener inside it.
		for _, pair := range [][2]string{{"(", ")"}, {"[", "]"}} {
			for strings.HasSuffix(u, pair[1]) && strings.Count(u, pair[1]) > strings.Count(u, pair[0]) {
				u = strings.TrimSuffix(u, pair[1])
			}
		}
		if u != "" && !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	return out
}

// MaxEventLinks bounds the links offered for one event (they are opened with the keys 1-9).
const MaxEventLinks = 9

// Links lists everything worth opening for the event: the video call first, then links written
// in the location and description, then the event's calendar page.
func (e Event) Links() []Link {
	var out []Link
	seen := map[string]bool{}
	add := func(label, u string) {
		if u == "" || seen[u] || len(out) >= MaxEventLinks {
			return
		}
		seen[u] = true
		out = append(out, Link{Label: label, URL: u})
	}
	add("Video call", e.MeetLink)
	for _, field := range []string{e.Location, e.Description} {
		for _, u := range ExtractURLs(field) {
			add(hostOf(u), u)
		}
	}
	add("Calendar", e.URL)
	return out
}

func hostOf(u string) string {
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	return u
}
