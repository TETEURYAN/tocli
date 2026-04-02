package usecase

import (
	"strings"
	"time"
	"tocli/internal/domain"
)

type TaskUseCase struct {
	repo domain.TaskRepository
}

func NewTaskUseCase(repo domain.TaskRepository) *TaskUseCase {
	return &TaskUseCase{repo: repo}
}

func (uc *TaskUseCase) ListTaskLists() ([]domain.TaskList, error) {
	return uc.repo.ListTaskLists()
}

func (uc *TaskUseCase) ListTasks(listID string) ([]domain.Task, error) {
	return uc.repo.ListTasks(listID)
}

func (uc *TaskUseCase) CompleteTask(taskID, listID string) error {
	return uc.repo.CompleteTask(taskID, listID)
}

func (uc *TaskUseCase) ReopenTask(taskID, listID string) error {
	return uc.repo.ReopenTask(taskID, listID)
}

func (uc *TaskUseCase) CreateTask(listID, title string, due *time.Time) (domain.Task, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return domain.Task{}, domain.ErrEmptyTaskTitle
	}
	return uc.repo.CreateTask(listID, t, due)
}

func (uc *TaskUseCase) DeleteTask(taskID, listID string) error {
	return uc.repo.DeleteTask(taskID, listID)
}

// ParseOptionalTaskDue parses optional due text (Brazilian order: dia-mês-ano). Empty returns (nil, nil).
func ParseOptionalTaskDue(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	// DD-MM-YYYY, then optional hora (24h).
	layouts := []string{
		"02-01-2006 15:04",
		"02-01-2006 15:04:05",
		"02-01-2006",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, domain.ErrInvalidDue
}

func (uc *TaskUseCase) ListAllTasks() ([]domain.Task, error) {
	lists, err := uc.repo.ListTaskLists()
	if err != nil {
		return nil, err
	}

	var all []domain.Task
	for _, list := range lists {
		tasks, err := uc.repo.ListTasks(list.ID)
		if err != nil {
			continue
		}
		for i := range tasks {
			tasks[i].ListName = list.Name
			tasks[i].ListID = list.ID
		}
		all = append(all, tasks...)
	}
	return all, nil
}

func (uc *TaskUseCase) TasksCompletedOn(date time.Time) ([]domain.Task, error) {
	lists, err := uc.repo.ListTaskLists()
	if err != nil {
		return nil, err
	}

	dateStr := date.Format("2006-01-02")
	var result []domain.Task
	for _, list := range lists {
		tasks, err := uc.repo.ListTasks(list.ID)
		if err != nil {
			continue
		}
		for _, task := range tasks {
			if task.Status == domain.TaskCompleted && task.CompletedAt != nil {
				if task.CompletedAt.Format("2006-01-02") == dateStr {
					task.ListName = list.Name
					task.ListID = list.ID
					result = append(result, task)
				}
			}
		}
	}
	return result, nil
}
