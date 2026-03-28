package google

import (
	"tocli/internal/domain"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type TasksCLI struct {
	binaryPath string
}

func NewTasksCLI(binaryPath string) *TasksCLI {
	if binaryPath == "" {
		binaryPath = "google"
	}
	return &TasksCLI{binaryPath: binaryPath}
}

type cliTaskList struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type cliTask struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Notes     string `json:"notes"`
	Status    string `json:"status"`
	Due       string `json:"due"`
	Completed string `json:"completed"`
}

func (c *TasksCLI) ListTaskLists() ([]domain.TaskList, error) {
	out, err := c.run("tasks", "tasklists", "list", "--format=json")
	if err != nil {
		return nil, fmt.Errorf("list task lists: %w", err)
	}

	var raw []cliTaskList
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse task lists: %w", err)
	}

	lists := make([]domain.TaskList, len(raw))
	for i, r := range raw {
		lists[i] = domain.TaskList{ID: r.ID, Name: r.Title}
	}
	return lists, nil
}

func (c *TasksCLI) ListTasks(listID string) ([]domain.Task, error) {
	out, err := c.run("tasks", "tasks", "list", "--tasklist="+listID, "--format=json")
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	var raw []cliTask
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse tasks: %w", err)
	}

	tasks := make([]domain.Task, len(raw))
	for i, r := range raw {
		tasks[i] = domain.Task{
			ID:     r.ID,
			Title:  r.Title,
			Notes:  r.Notes,
			ListID: listID,
		}

		if strings.EqualFold(r.Status, "completed") {
			tasks[i].Status = domain.TaskCompleted
		}

		if r.Due != "" {
			if t, err := time.Parse(time.RFC3339, r.Due); err == nil {
				tasks[i].DueDate = &t
			}
		}

		if r.Completed != "" {
			if t, err := time.Parse(time.RFC3339, r.Completed); err == nil {
				tasks[i].CompletedAt = &t
			}
		}
	}
	return tasks, nil
}

func (c *TasksCLI) CompleteTask(taskID, listID string) error {
	_, err := c.run("tasks", "tasks", "update", taskID, "--tasklist="+listID, "--status=completed")
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}
	return nil
}

func (c *TasksCLI) ReopenTask(taskID, listID string) error {
	_, err := c.run("tasks", "tasks", "update", taskID, "--tasklist="+listID, "--status=needsAction")
	if err != nil {
		return fmt.Errorf("reopen task: %w", err)
	}
	return nil
}

func (c *TasksCLI) CreateTask(listID, title string) (domain.Task, error) {
	_, _ = listID, title
	return domain.Task{}, fmt.Errorf("CreateTask is not implemented for the Google Tasks CLI adapter")
}

func (c *TasksCLI) run(args ...string) ([]byte, error) {
	cmd := exec.Command(c.binaryPath, args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s: %s", err, string(exitErr.Stderr))
		}
		return nil, err
	}
	return out, nil
}
