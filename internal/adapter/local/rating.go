package local

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
	"tocli/internal/domain"
)

// RatingRepo implements domain.RatingRepository backed by a JSON file at
// RatingsPath(), keyed by "2006-01-02" date strings. Whole-file read/write is
// fine here: a year of daily ratings is a few hundred entries at most.
type RatingRepo struct {
	mu   sync.Mutex
	path string
}

func NewRatingRepo() (*RatingRepo, error) {
	path, err := RatingsPath()
	if err != nil {
		return nil, err
	}
	return &RatingRepo{path: path}, nil
}

func (r *RatingRepo) GetRatings(year int) (map[time.Time]int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, err := r.load()
	if err != nil {
		return nil, err
	}

	result := make(map[time.Time]int)
	for dateStr, score := range stored {
		date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			continue
		}
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

	r.mu.Lock()
	defer r.mu.Unlock()

	stored, err := r.load()
	if err != nil {
		return err
	}
	key := domain.NormalizeRatingDate(date).Format("2006-01-02")
	stored[key] = score
	return r.save(stored)
}

func (r *RatingRepo) load() (map[string]int, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]int), nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return make(map[string]int), nil
	}
	var stored map[string]int
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	if stored == nil {
		stored = make(map[string]int)
	}
	return stored, nil
}

func (r *RatingRepo) save(stored map[string]int) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0600)
}
