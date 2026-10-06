package usecase

import (
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"tocli/internal/domain"
)

// EventMatch is one search hit. TitleHits are rune indexes into Event.Title that matched the
// query, so the UI can highlight them without redoing the (accent-insensitive) matching.
type EventMatch struct {
	Event     domain.Event
	Score     int
	TitleHits []int
}

// folded is a string lowercased and stripped of accents, keeping for every folded rune the index
// of the original rune it came from (so "Reunião" folded is "reuniao" and index 5 maps back to 'ã').
type folded struct {
	runes []rune
	orig  []int
}

func fold(s string) folded {
	var f folded
	for i, r := range []rune(s) {
		for _, d := range norm.NFD.String(string(r)) {
			if unicode.Is(unicode.Mn, d) {
				continue
			}
			f.runes = append(f.runes, unicode.ToLower(d))
			f.orig = append(f.orig, i)
		}
	}
	return f
}

// indexOf returns the first position of needle in f.runes, or -1.
func (f folded) indexOf(needle []rune) int {
	if len(needle) == 0 || len(needle) > len(f.runes) {
		return -1
	}
	for i := 0; i+len(needle) <= len(f.runes); i++ {
		match := true
		for j, r := range needle {
			if f.runes[i+j] != r {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// wordStart reports whether position i begins a word (start of text or after a non-letter/digit).
func (f folded) wordStart(i int) bool {
	return i == 0 || !(unicode.IsLetter(f.runes[i-1]) || unicode.IsDigit(f.runes[i-1]))
}

type eventDoc struct {
	event                  domain.Event
	title, location, descr folded
}

// EventIndex is a search index over a fixed set of events. Folding every field once up front
// keeps each keystroke's search cheap even with a year or two of calendar data.
type EventIndex struct {
	docs []eventDoc
}

func NewEventIndex(events []domain.Event) *EventIndex {
	ix := &EventIndex{docs: make([]eventDoc, len(events))}
	for i, e := range events {
		ix.docs[i] = eventDoc{
			event:    e,
			title:    fold(e.Title),
			location: fold(e.Location),
			descr:    fold(e.Description),
		}
	}
	return ix
}

func (ix *EventIndex) Len() int {
	if ix == nil {
		return 0
	}
	return len(ix.docs)
}

// Field weights: a hit in the title matters most.
const (
	weightTitle    = 30
	weightLocation = 20
	weightDescr    = 10
	bonusWordStart = 5
	bonusPrefix    = 5
)

// Search returns events matching every whitespace-separated term of query (case- and
// accent-insensitive), best first. Score ties are broken by how close the event is to now, so
// "this week's standup" outranks last year's. A blank query matches nothing. limit <= 0 means no
// limit.
func (ix *EventIndex) Search(query string, now time.Time, limit int) []EventMatch {
	if ix == nil {
		return nil
	}
	terms := queryTerms(query)
	if len(terms) == 0 {
		return nil
	}

	var out []EventMatch
docs:
	for _, d := range ix.docs {
		score := 0
		var hits []int
		for _, term := range terms {
			best := 0
			if i := d.title.indexOf(term); i >= 0 {
				s := weightTitle
				if d.title.wordStart(i) {
					s += bonusWordStart
				}
				if i == 0 {
					s += bonusPrefix
				}
				best = s
				for k := i; k < i+len(term); k++ {
					hits = append(hits, d.title.orig[k])
				}
			}
			if i := d.location.indexOf(term); i >= 0 {
				best = max(best, weightLocation+boolInt(d.location.wordStart(i))*bonusWordStart)
			}
			if i := d.descr.indexOf(term); i >= 0 {
				best = max(best, weightDescr+boolInt(d.descr.wordStart(i))*bonusWordStart)
			}
			if best == 0 {
				continue docs // every term has to match somewhere
			}
			score += best
		}
		out = append(out, EventMatch{Event: d.event, Score: score, TitleHits: uniqueSorted(hits)})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		di, dj := distance(out[i].Event.StartTime, now), distance(out[j].Event.StartTime, now)
		if di != dj {
			return di < dj
		}
		return out[i].Event.StartTime.Before(out[j].Event.StartTime)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func distance(a, b time.Time) time.Duration {
	d := a.Sub(b)
	if d < 0 {
		return -d
	}
	return d
}

func uniqueSorted(xs []int) []int {
	if len(xs) == 0 {
		return nil
	}
	sort.Ints(xs)
	out := xs[:1]
	for _, x := range xs[1:] {
		if x != out[len(out)-1] {
			out = append(out, x)
		}
	}
	return out
}

// MatchText reports whether every whitespace-separated term of query occurs in text, ignoring
// case and accents, with the same scoring as event search (word starts and prefixes score
// higher). hits are rune indexes into text for highlighting. A blank query matches everything
// with score 0.
func MatchText(query, text string) (score int, hits []int, ok bool) {
	terms := queryTerms(query)
	if len(terms) == 0 {
		return 0, nil, true
	}
	f := fold(text)
	for _, term := range terms {
		i := f.indexOf(term)
		if i < 0 {
			return 0, nil, false
		}
		score += weightTitle
		if f.wordStart(i) {
			score += bonusWordStart
		}
		if i == 0 {
			score += bonusPrefix
		}
		for k := i; k < i+len(term); k++ {
			hits = append(hits, f.orig[k])
		}
	}
	return score, uniqueSorted(hits), true
}

// queryTerms splits a query into folded terms.
func queryTerms(query string) [][]rune {
	var terms [][]rune
	for _, t := range strings.Fields(query) {
		if f := fold(t); len(f.runes) > 0 {
			terms = append(terms, f.runes)
		}
	}
	return terms
}
