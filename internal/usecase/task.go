package usecase

import (
	"strings"
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

func (uc *TaskUseCase) CreateTask(listID, title string) (domain.Task, error) {
	t := strings.TrimSpace(title)
	if t == "" {
		return domain.Task{}, domain.ErrEmptyTaskTitle
	}
	return uc.repo.CreateTask(listID, t)
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
