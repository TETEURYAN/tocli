package usecase

import "time"

type YearProgress struct {
	Year          int
	DaysPassed    int
	DaysRemaining int
	TotalDays     int
	Percentage    float64
}

type ProgressUseCase struct{}

func NewProgressUseCase() *ProgressUseCase {
	return &ProgressUseCase{}
}

func (uc *ProgressUseCase) Calculate() YearProgress {
	now := time.Now()
	year := now.Year()

	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
	nextJan1 := time.Date(year+1, 1, 1, 0, 0, 0, 0, now.Location())

	totalDays := int(nextJan1.Sub(jan1).Hours() / 24)
	daysPassed := now.YearDay()
	daysRemaining := totalDays - daysPassed
	percentage := float64(daysPassed) / float64(totalDays) * 100

	return YearProgress{
		Year:          year,
		DaysPassed:    daysPassed,
		DaysRemaining: daysRemaining,
		TotalDays:     totalDays,
		Percentage:    percentage,
	}
}
