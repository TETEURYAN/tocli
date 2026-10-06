package google

import (
	"context"
	"time"
	"tocli/internal/domain"

	tasks "google.golang.org/api/tasks/v1"
)

// TaskRepo implements domain.TaskRepository using the Google Tasks API.
type TaskRepo struct {
	svc *tasks.Service
	ctx context.Context
}

// NewTaskRepo builds a repository backed by Google Tasks.
func NewTaskRepo(ctx context.Context, client *Client) *TaskRepo {
	return &TaskRepo{svc: client.Tasks, ctx: ctx}
}

func (r *TaskRepo) ListTaskLists() ([]domain.TaskList, error) {
	resp, err := r.svc.Tasklists.List().Context(r.ctx).MaxResults(100).Do()
	if err != nil {
		return nil, wrapAPIError("list task lists", err)
	}
	lists := make([]domain.TaskList, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item == nil {
			continue
		}
		lists = append(lists, domain.TaskList{ID: item.Id, Name: item.Title})
	}
	return lists, nil
}

func (r *TaskRepo) ListTasks(listID string) ([]domain.Task, error) {
	var all []domain.Task
	pageToken := ""
	for {
		call := r.svc.Tasks.List(listID).
			Context(r.ctx).
			MaxResults(100).
			ShowCompleted(true).
			ShowHidden(true)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, wrapAPIError("list tasks", err)
		}
		for _, item := range resp.Items {
			if item == nil {
				continue
			}
			all = append(all, mapGoogleTaskToDomain(item, listID))
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return all, nil
}

func (r *TaskRepo) CompleteTask(taskID, listID string) error {
	patch := &tasks.Task{Status: "completed"}
	_, err := r.svc.Tasks.Patch(listID, taskID, patch).Context(r.ctx).Do()
	return wrapAPIError("complete task", err)
}

func (r *TaskRepo) ReopenTask(taskID, listID string) error {
	patch := &tasks.Task{Status: "needsAction"}
	patch.NullFields = []string{"completed"}
	_, err := r.svc.Tasks.Patch(listID, taskID, patch).Context(r.ctx).Do()
	return wrapAPIError("reopen task", err)
}

func (r *TaskRepo) CreateTask(listID, title string, due *time.Time) (domain.Task, error) {
	newTask := &tasks.Task{Title: title}
	if due != nil {
		newTask.Due = dueForGoogle(*due)
	}
	created, err := r.svc.Tasks.Insert(listID, newTask).Context(r.ctx).Do()
	if err != nil {
		return domain.Task{}, wrapAPIError("create task", err)
	}
	return mapGoogleTaskToDomain(created, listID), nil
}

func (r *TaskRepo) UpdateTask(taskID, listID, title string, due *time.Time) (domain.Task, error) {
	patch := &tasks.Task{Title: title}
	if due != nil {
		patch.Due = dueForGoogle(*due)
	} else {
		patch.NullFields = []string{"Due"}
	}
	updated, err := r.svc.Tasks.Patch(listID, taskID, patch).Context(r.ctx).Do()
	if err != nil {
		return domain.Task{}, wrapAPIError("update task", err)
	}
	return mapGoogleTaskToDomain(updated, listID), nil
}

// dueForGoogle converts a local due time to what Google Tasks stores: the calendar date at
// midnight UTC. The API keeps only the date (the time of day is discarded), so the date has to be
// taken in the user's zone first; converting the instant to UTC could move it to another day.
func dueForGoogle(t time.Time) string {
	d := t.In(time.Local)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func (r *TaskRepo) DeleteTask(taskID, listID string) error {
	err := r.svc.Tasks.Delete(listID, taskID).Context(r.ctx).Do()
	return wrapAPIError("delete task", err)
}
