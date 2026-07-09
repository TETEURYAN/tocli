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

// ratingEntry is the on-disk shape for a single day: a score and/or a note,
// either of which may be absent.
type ratingEntry struct {
	Score int    `json:"score"`
	Note  string `json:"note,omitempty"`
}

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
	for dateStr, entry := range stored {
		date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			continue
		}
		if date.Year() == year {
			result[date] = entry.Score
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
	entry := stored[key]
	entry.Score = score
	stored[key] = entry
	return r.save(stored)
}

func (r *RatingRepo) GetNotes(year int) (map[time.Time]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, err := r.load()
	if err != nil {
		return nil, err
	}

	result := make(map[time.Time]string)
	for dateStr, entry := range stored {
		if entry.Note == "" {
			continue
		}
		date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
		if err != nil {
			continue
		}
		if date.Year() == year {
			result[date] = entry.Note
		}
	}
	return result, nil
}

func (r *RatingRepo) SetNote(date time.Time, note string) error {
	if err := domain.ValidateRatingNote(note); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	stored, err := r.load()
	if err != nil {
		return err
	}
	key := domain.NormalizeRatingDate(date).Format("2006-01-02")
	entry := stored[key]
	entry.Note = note
	stored[key] = entry
	return r.save(stored)
}

func (r *RatingRepo) load() (map[string]ratingEntry, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]ratingEntry), nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return make(map[string]ratingEntry), nil
	}

	var stored map[string]ratingEntry
	if err := json.Unmarshal(data, &stored); err == nil {
		if stored == nil {
			stored = make(map[string]ratingEntry)
		}
		return stored, nil
	}

	// Fall back to the pre-note format ("date": score) written by earlier
	// versions of tocli, so existing ratings.json files keep working.
	var legacy map[string]int
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, err
	}
	stored = make(map[string]ratingEntry, len(legacy))
	for k, score := range legacy {
		stored[k] = ratingEntry{Score: score}
	}
	return stored, nil
}

func (r *RatingRepo) save(stored map[string]ratingEntry) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0600)
}
