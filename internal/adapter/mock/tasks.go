package mock

import (
	"tocli/internal/domain"
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"
)

type TaskRepo struct {
	lists       []domain.TaskList
	tasks       map[string][]domain.Task
	taskIDSeq   uint64
}

func NewTaskRepo() *TaskRepo {
	r := &TaskRepo{
		lists: []domain.TaskList{
			{ID: "list-1", Name: "Work"},
			{ID: "list-2", Name: "Personal"},
			{ID: "list-3", Name: "Learning"},
		},
		tasks: make(map[string][]domain.Task),
	}
	r.seed()
	return r
}

func (r *TaskRepo) ListTaskLists() ([]domain.TaskList, error) {
	return r.lists, nil
}

func (r *TaskRepo) ListTasks(listID string) ([]domain.Task, error) {
	tasks, ok := r.tasks[listID]
	if !ok {
		return nil, fmt.Errorf("list %s not found", listID)
	}
	return tasks, nil
}

func (r *TaskRepo) CompleteTask(taskID, listID string) error {
	tasks, ok := r.tasks[listID]
	if !ok {
		return fmt.Errorf("list %s not found", listID)
	}
	for i := range tasks {
		if tasks[i].ID == taskID {
			tasks[i].Complete()
			r.tasks[listID] = tasks
			return nil
		}
	}
	return fmt.Errorf("task %s not found", taskID)
}

func (r *TaskRepo) ReopenTask(taskID, listID string) error {
	tasks, ok := r.tasks[listID]
	if !ok {
		return fmt.Errorf("list %s not found", listID)
	}
	for i := range tasks {
		if tasks[i].ID == taskID {
			tasks[i].Reopen()
			r.tasks[listID] = tasks
			return nil
		}
	}
	return fmt.Errorf("task %s not found", taskID)
}

func (r *TaskRepo) CreateTask(listID, title string, due *time.Time) (domain.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Task{}, domain.ErrEmptyTaskTitle
	}
	if _, ok := r.tasks[listID]; !ok {
		return domain.Task{}, fmt.Errorf("list %s not found", listID)
	}
	id := fmt.Sprintf("new-%d", atomic.AddUint64(&r.taskIDSeq, 1))
	task := domain.Task{
		ID:       id,
		Title:    title,
		Status:   domain.TaskOpen,
		ListID:   listID,
		DueDate:  due,
	}
	for _, list := range r.lists {
		if list.ID == listID {
			task.ListName = list.Name
			break
		}
	}
	r.tasks[listID] = append(r.tasks[listID], task)
	return task, nil
}

func (r *TaskRepo) DeleteTask(taskID, listID string) error {
	tasks, ok := r.tasks[listID]
	if !ok {
		return fmt.Errorf("list %s not found", listID)
	}
	for i := range tasks {
		if tasks[i].ID == taskID {
			r.tasks[listID] = append(tasks[:i], tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("task %s not found", taskID)
}

func (r *TaskRepo) seed() {
	now := time.Now()
	rng := rand.New(rand.NewSource(42))

	workTasks := []struct {
		title string
		done  bool
	}{
		{"Review PR #342 – auth refactor", false},
		{"Update API documentation", false},
		{"Fix pagination bug on /users", true},
		{"Write integration tests for payments", false},
		{"Deploy staging environment", false},
		{"Prepare sprint retrospective", true},
		{"Optimize database queries", false},
		{"Code review: new caching layer", false},
	}

	personalTasks := []struct {
		title string
		done  bool
	}{
		{"Buy groceries", false},
		{"Schedule dentist appointment", false},
		{"Pay electricity bill", true},
		{"Call mom", false},
		{"Organize desk", true},
		{"Plan weekend trip", false},
	}

	learningTasks := []struct {
		title string
		done  bool
	}{
		{"Read 'Designing Data-Intensive Apps' ch. 5", false},
		{"Complete Go concurrency exercises", true},
		{"Watch distributed systems lecture", false},
		{"Practice LeetCode – dynamic programming", false},
		{"Write blog post about clean architecture", false},
	}

	buildTasks := func(listID string, items []struct {
		title string
		done  bool
	}) []domain.Task {
		var tasks []domain.Task
		for i, item := range items {
			t := domain.Task{
				ID:     fmt.Sprintf("%s-task-%d", listID, i+1),
				Title:  item.title,
				ListID: listID,
			}
			due := now.AddDate(0, 0, rng.Intn(14)-3)
			t.DueDate = &due

			if item.done {
				completed := now.AddDate(0, 0, -rng.Intn(5))
				t.Status = domain.TaskCompleted
				t.CompletedAt = &completed
			}
			tasks = append(tasks, t)
		}
		return tasks
	}

	r.tasks["list-1"] = buildTasks("list-1", workTasks)
	r.tasks["list-2"] = buildTasks("list-2", personalTasks)
	r.tasks["list-3"] = buildTasks("list-3", learningTasks)

	r.seedHistorical(rng)
}

func (r *TaskRepo) seedHistorical(rng *rand.Rand) {
	now := time.Now()
	year := now.Year()
	jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)

	for d := 0; d < now.YearDay()-1; d++ {
		date := jan1.AddDate(0, 0, d)
		weekday := date.Weekday()

		var maxTasks int
		switch weekday {
		case time.Saturday, time.Sunday:
			maxTasks = 2
		default:
			maxTasks = 5
		}

		count := rng.Intn(maxTasks + 1)
		for i := 0; i < count; i++ {
			completed := date.Add(time.Duration(9+rng.Intn(10)) * time.Hour)
			t := domain.Task{
				ID:          fmt.Sprintf("hist-%d-%d", d, i),
				Title:       fmt.Sprintf("Historical task %d-%d", d, i),
				Status:      domain.TaskCompleted,
				CompletedAt: &completed,
				ListID:      r.lists[rng.Intn(len(r.lists))].ID,
			}
			r.tasks[t.ListID] = append(r.tasks[t.ListID], t)
		}
	}
}
