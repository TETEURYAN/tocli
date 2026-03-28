package domain

import "time"

type TaskRepository interface {
	ListTaskLists() ([]TaskList, error)
	ListTasks(listID string) ([]Task, error)
	CompleteTask(taskID, listID string) error
	CreateTask(listID, title string) (Task, error)
}

type EventRepository interface {
	GetEvents(start, end time.Time) ([]Event, error)
}
