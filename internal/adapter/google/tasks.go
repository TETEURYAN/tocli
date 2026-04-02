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
		newTask.Due = due.UTC().Format(time.RFC3339)
	}
	created, err := r.svc.Tasks.Insert(listID, newTask).Context(r.ctx).Do()
	if err != nil {
		return domain.Task{}, wrapAPIError("create task", err)
	}
	return mapGoogleTaskToDomain(created, listID), nil
}

func (r *TaskRepo) DeleteTask(taskID, listID string) error {
	err := r.svc.Tasks.Delete(listID, taskID).Context(r.ctx).Do()
	return wrapAPIError("delete task", err)
}
