package domain

import (
	"errors"
	"fmt"
	"time"
)

const (
	MinRatingScore = 1
	MaxRatingScore = 5

	// MaxRatingNoteLength bounds the "how was your day" journal entry so a
	// runaway paste can't blow up ratings.json or the export CSV.
	MaxRatingNoteLength = 1000
)

var (
	ErrInvalidRatingScore = errors.New("rating score must be between 1 and 5")
	ErrRatingNoteTooLong  = fmt.Errorf("rating note must be at most %d characters", MaxRatingNoteLength)
)

// DailyRating is a user-assigned mood/productivity score, plus an optional
// free-text note, for a single day. Google Tasks/Calendar have no equivalent
// concept — this is purely local data.
type DailyRating struct {
	Date  time.Time
	Score int
	Note  string
}

// ValidateRatingScore reports whether score is in the accepted 1-5 range.
func ValidateRatingScore(score int) error {
	if score < MinRatingScore || score > MaxRatingScore {
		return ErrInvalidRatingScore
	}
	return nil
}

// ValidateRatingNote reports whether note is within the accepted length.
func ValidateRatingNote(note string) error {
	if len(note) > MaxRatingNoteLength {
		return ErrRatingNoteTooLong
	}
	return nil
}

// NormalizeRatingDate strips time-of-day so a date can be used as a stable
// lookup/storage key regardless of what time component the caller passed in.
func NormalizeRatingDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

type RatingRepository interface {
	// GetRatings returns every rated day in the given year, keyed by
	// NormalizeRatingDate(date).
	GetRatings(year int) (map[time.Time]int, error)
	SetRating(date time.Time, score int) error

	// GetNotes returns every day with a non-empty note in the given year,
	// keyed by NormalizeRatingDate(date). A note may exist independently of
	// a score (or vice versa).
	GetNotes(year int) (map[time.Time]string, error)
	SetNote(date time.Time, note string) error
}
