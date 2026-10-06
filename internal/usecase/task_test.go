package usecase

import (
	"errors"
	"testing"
	"time"

	"tocli/internal/adapter/mock"
	"tocli/internal/domain"
)

func firstTask(t *testing.T, uc *TaskUseCase) domain.Task {
	t.Helper()
	tasks, err := uc.ListAllTasks()
	if err != nil || len(tasks) == 0 {
		t.Fatalf("no tasks to work with: %v", err)
	}
	for _, task := range tasks {
		if task.Status == domain.TaskOpen {
			return task
		}
	}
	t.Fatal("no open task")
	return domain.Task{}
}

func TestUpdateTaskChangesTitleAndDue(t *testing.T) {
	uc := NewTaskUseCase(mock.NewTaskRepo())
	task := firstTask(t, uc)

	due := time.Date(2030, 5, 17, 0, 0, 0, 0, time.Local)
	got, err := uc.UpdateTask(task.ID, task.ListID, "  Renamed task  ", &due)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Renamed task" {
		t.Errorf("title = %q, want it trimmed", got.Title)
	}
	if got.DueDate == nil || !got.DueDate.Equal(due) {
		t.Errorf("due = %v, want %v", got.DueDate, due)
	}

	all, _ := uc.ListAllTasks()
	var found bool
	for _, x := range all {
		if x.ID == task.ID {
			found = true
			if x.Title != "Renamed task" {
				t.Errorf("the change was not persisted, title = %q", x.Title)
			}
		}
	}
	if !found {
		t.Error("task disappeared after the update")
	}
}

func TestUpdateTaskNilDueClearsIt(t *testing.T) {
	uc := NewTaskUseCase(mock.NewTaskRepo())
	task := firstTask(t, uc)
	due := time.Now().Add(48 * time.Hour)
	if _, err := uc.UpdateTask(task.ID, task.ListID, "x", &due); err != nil {
		t.Fatal(err)
	}
	got, err := uc.UpdateTask(task.ID, task.ListID, "x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.DueDate != nil {
		t.Errorf("due = %v, want it cleared", got.DueDate)
	}
}

func TestUpdateTaskRejectsBlankTitleAndUnknownTask(t *testing.T) {
	uc := NewTaskUseCase(mock.NewTaskRepo())
	task := firstTask(t, uc)

	for _, title := range []string{"", "   ", "\t\n"} {
		if _, err := uc.UpdateTask(task.ID, task.ListID, title, nil); !errors.Is(err, domain.ErrEmptyTaskTitle) {
			t.Errorf("title %q: err = %v, want ErrEmptyTaskTitle", title, err)
		}
	}
	if _, err := uc.UpdateTask("nope", task.ListID, "x", nil); err == nil {
		t.Error("updating an unknown task should fail")
	}
	if _, err := uc.UpdateTask(task.ID, "nope", "x", nil); err == nil {
		t.Error("updating in an unknown list should fail")
	}
}

func TestParseOptionalTaskDue(t *testing.T) {
	if d, err := ParseOptionalTaskDue("  "); d != nil || err != nil {
		t.Errorf("blank = %v, %v, want nil, nil", d, err)
	}
	d, err := ParseOptionalTaskDue("07-04-2026")
	if err != nil || d == nil || d.Year() != 2026 || d.Month() != 4 || d.Day() != 7 || d.Hour() != 0 {
		t.Errorf("date only = %v, %v", d, err)
	}
	d, err = ParseOptionalTaskDue("07-04-2026 18:30")
	if err != nil || d == nil || d.Hour() != 18 || d.Minute() != 30 {
		t.Errorf("date + time = %v, %v", d, err)
	}
	if _, err := ParseOptionalTaskDue("2026-04-07"); !errors.Is(err, domain.ErrInvalidDue) {
		t.Errorf("ISO order should be rejected, got %v", err)
	}
}
