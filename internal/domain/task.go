package domain

import (
	"errors"
	"time"
)

var ErrEmptyTaskTitle = errors.New("task title cannot be empty")

type TaskStatus int

const (
	TaskOpen TaskStatus = iota
	TaskCompleted
)

type Task struct {
	ID          string
	Title       string
	Notes       string
	Status      TaskStatus
	DueDate     *time.Time
	CompletedAt *time.Time
	ListID      string
	ListName    string
}

func (t *Task) IsOverdue() bool {
	if t.DueDate == nil || t.Status == TaskCompleted {
		return false
	}
	return time.Now().After(*t.DueDate)
}

func (t *Task) Complete() {
	now := time.Now()
	t.Status = TaskCompleted
	t.CompletedAt = &now
}

func (t *Task) Reopen() {
	t.Status = TaskOpen
	t.CompletedAt = nil
}

type TaskList struct {
	ID   string
	Name string
}
