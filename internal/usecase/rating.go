package usecase

import (
	"time"
	"tocli/internal/domain"
)

// RatingData mirrors ContributionData's DayCounts shape (map keyed by
// "2006-01-02") so the UI can render both grids through the same lookup
// pattern. A missing key (or, equivalently, a zero value) means "not rated" —
// valid scores start at 1, so zero is an unambiguous sentinel.
type RatingData struct {
	DayScores map[string]int
	Year      int
}

type RatingUseCase struct {
	repo domain.RatingRepository
}

func NewRatingUseCase(repo domain.RatingRepository) *RatingUseCase {
	return &RatingUseCase{repo: repo}
}

func (uc *RatingUseCase) Generate(year int) RatingData {
	data := RatingData{Year: year, DayScores: make(map[string]int)}

	ratings, err := uc.repo.GetRatings(year)
	if err != nil {
		return data
	}

	for date, score := range ratings {
		if date.Year() != year {
			continue
		}
		data.DayScores[date.Format("2006-01-02")] = score
	}
	return data
}

func (uc *RatingUseCase) SetRating(date time.Time, score int) error {
	if err := domain.ValidateRatingScore(score); err != nil {
		return err
	}
	return uc.repo.SetRating(date, score)
}
