package usecase

import (
	"tocli/internal/domain"
	"time"
)

type EventUseCase struct {
	repo domain.EventRepository
}

func NewEventUseCase(repo domain.EventRepository) *EventUseCase {
	return &EventUseCase{repo: repo}
}

func (uc *EventUseCase) GetTodayEvents() ([]domain.Event, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)
	return uc.repo.GetEvents(start, end)
}

func (uc *EventUseCase) GetWeekEvents() ([]domain.Event, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(7 * 24 * time.Hour)
	return uc.repo.GetEvents(start, end)
}

func (uc *EventUseCase) GetEventsForDate(date time.Time) ([]domain.Event, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)
	return uc.repo.GetEvents(start, end)
}
