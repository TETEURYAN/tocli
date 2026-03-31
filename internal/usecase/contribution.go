package usecase

import (
	"tocli/internal/domain"
	"time"
)

const WeeksInYear = 53

type ContributionData struct {
	Weeks     [WeeksInYear][7]int
	DayCounts map[string]int
	MaxCount  int
	Year      int
	Total     int
}

type ContributionUseCase struct {
	repo domain.TaskRepository
}

func NewContributionUseCase(repo domain.TaskRepository) *ContributionUseCase {
	return &ContributionUseCase{repo: repo}
}

func (uc *ContributionUseCase) Generate(year int) ContributionData {
	data := ContributionData{
		Year:      year,
		DayCounts: make(map[string]int),
	}

	lists, err := uc.repo.ListTaskLists()
	if err != nil {
		return data
	}

	for _, list := range lists {
		tasks, err := uc.repo.ListTasks(list.ID)
		if err != nil {
			continue
		}
		for _, task := range tasks {
			if task.Status == domain.TaskCompleted && task.CompletedAt != nil {
				if task.CompletedAt.Year() == year {
					key := task.CompletedAt.Format("2006-01-02")
					data.DayCounts[key]++
				}
			}
		}
	}

	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	startOffset := int(jan1.Weekday())

	for day := 0; day < 366; day++ {
		date := jan1.AddDate(0, 0, day)
		if date.Year() != year {
			break
		}

		idx := date.YearDay() - 1 + startOffset
		week := idx / 7
		weekday := idx % 7

		if week >= WeeksInYear {
			break
		}

		key := date.Format("2006-01-02")
		count := data.DayCounts[key]
		data.Weeks[week][weekday] = count
		data.Total += count

		if count > data.MaxCount {
			data.MaxCount = count
		}
	}

	return data
}
