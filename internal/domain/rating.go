package domain

import (
	"errors"
	"time"
)

const (
	MinRatingScore = 1
	MaxRatingScore = 5
)

var ErrInvalidRatingScore = errors.New("rating score must be between 1 and 5")

// DailyRating is a user-assigned mood/productivity score for a single day.
// Google Tasks/Calendar have no equivalent concept — this is purely local data.
type DailyRating struct {
	Date  time.Time
	Score int
}

// ValidateRatingScore reports whether score is in the accepted 1-5 range.
func ValidateRatingScore(score int) error {
	if score < MinRatingScore || score > MaxRatingScore {
		return ErrInvalidRatingScore
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
}
