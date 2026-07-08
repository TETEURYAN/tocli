package mock

import (
	"math/rand"
	"time"
	"tocli/internal/domain"
)

// RatingRepo is an in-memory domain.RatingRepository for -offline mode.
// Ratings never touch disk here, so a fresh set of example scores is
// generated on every run.
type RatingRepo struct {
	scores map[time.Time]int
}

func NewRatingRepo() *RatingRepo {
	r := &RatingRepo{scores: make(map[time.Time]int)}
	r.seed()
	return r
}

func (r *RatingRepo) GetRatings(year int) (map[time.Time]int, error) {
	result := make(map[time.Time]int)
	for date, score := range r.scores {
		if date.Year() == year {
			result[date] = score
		}
	}
	return result, nil
}

func (r *RatingRepo) SetRating(date time.Time, score int) error {
	if err := domain.ValidateRatingScore(score); err != nil {
		return err
	}
	r.scores[domain.NormalizeRatingDate(date)] = score
	return nil
}

// seed fills the last few weeks with example scores so the daily rating
// graph has something to show immediately in -offline mode.
func (r *RatingRepo) seed() {
	now := time.Now()
	rng := rand.New(rand.NewSource(7))

	for d := 0; d < 21; d++ {
		date := domain.NormalizeRatingDate(now.AddDate(0, 0, -d))
		if rng.Intn(4) == 0 {
			continue // leave some days unrated
		}
		r.scores[date] = 1 + rng.Intn(5)
	}
}
