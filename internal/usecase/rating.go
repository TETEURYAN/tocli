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
// valid scores start at 1, so zero is an unambiguous sentinel. DayNotes only
// holds entries for days with a non-empty note.
type RatingData struct {
	DayScores map[string]int
	DayNotes  map[string]string
	Year      int
}

type RatingUseCase struct {
	repo domain.RatingRepository
}

func NewRatingUseCase(repo domain.RatingRepository) *RatingUseCase {
	return &RatingUseCase{repo: repo}
}

func (uc *RatingUseCase) Generate(year int) RatingData {
	data := RatingData{Year: year, DayScores: make(map[string]int), DayNotes: make(map[string]string)}

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

	notes, err := uc.repo.GetNotes(year)
	if err != nil {
		return data
	}
	for date, note := range notes {
		if date.Year() != year || note == "" {
			continue
		}
		data.DayNotes[date.Format("2006-01-02")] = note
	}

	return data
}

func (uc *RatingUseCase) SetRating(date time.Time, score int) error {
	if err := domain.ValidateRatingScore(score); err != nil {
		return err
	}
	return uc.repo.SetRating(date, score)
}

func (uc *RatingUseCase) SetNote(date time.Time, note string) error {
	if err := domain.ValidateRatingNote(note); err != nil {
		return err
	}
	return uc.repo.SetNote(date, note)
}

// ExportMonthCSV renders one row per calendar day of the given month
// ("date,score,note"), leaving score blank for unrated days rather than
// writing 0, so a spreadsheet doesn't mistake "no rating" for the lowest
// score. Notes are written as-is; encoding/csv quotes them automatically if
// they contain commas, quotes, or newlines.
func (uc *RatingUseCase) ExportMonthCSV(year int, month time.Month) ([]byte, error) {
	ratings, err := uc.repo.GetRatings(year)
	if err != nil {
		return nil, err
	}
	scores := make(map[string]int, len(ratings))
	for date, score := range ratings {
		scores[date.Format("2006-01-02")] = score
	}

	notes, err := uc.repo.GetNotes(year)
	if err != nil {
		return nil, err
	}
	notesByDay := make(map[string]string, len(notes))
	for date, note := range notes {
		notesByDay[date.Format("2006-01-02")] = note
	}

	first := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"date", "score", "note"}); err != nil {
		return nil, err
	}
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		score := ""
		if s := scores[key]; s > 0 {
			score = strconv.Itoa(s)
		}
		if err := w.Write([]string{key, score, notesByDay[key]}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
