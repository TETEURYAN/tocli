package usecase

import (
	"bytes"
	"encoding/csv"
	"strconv"
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

// ExportMonthCSV renders one row per calendar day of the given month
// ("date,score"), leaving score blank for unrated days rather than writing 0,
// so a spreadsheet doesn't mistake "no rating" for the lowest score.
func (uc *RatingUseCase) ExportMonthCSV(year int, month time.Month) ([]byte, error) {
	ratings, err := uc.repo.GetRatings(year)
	if err != nil {
		return nil, err
	}
	scores := make(map[string]int, len(ratings))
	for date, score := range ratings {
		scores[date.Format("2006-01-02")] = score
	}

	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"date", "score"}); err != nil {
		return nil, err
	}
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		score := ""
		if s := scores[d.Format("2006-01-02")]; s > 0 {
			score = strconv.Itoa(s)
		}
		if err := w.Write([]string{d.Format("2006-01-02"), score}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
