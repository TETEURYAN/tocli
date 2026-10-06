package ui

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"tocli/internal/domain"
	"tocli/internal/usecase"
)

func (m Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		lists, err := m.taskUC.ListTaskLists()
		if err != nil {
			return errMsg{err: err}
		}
		tasks, err := m.taskUC.ListAllTasks()
		if err != nil {
			return errMsg{err: err}
		}
		return tasksLoadedMsg{tasks: tasks, lists: lists}
	}
}

func (m Model) loadEvents() tea.Cmd {
	return func() tea.Msg {
		events, err := m.eventUC.GetTodayEvents()
		if err != nil {
			return errMsg{err: err}
		}
		return eventsLoadedMsg{events: events}
	}
}

func (m Model) loadContribution() tea.Cmd {
	year := m.graphYear
	return func() tea.Msg {
		return contributionLoadedMsg{data: m.contribUC.Generate(year)}
	}
}

func (m Model) loadRatings() tea.Cmd {
	year := m.graphYear
	return func() tea.Msg {
		return ratingLoadedMsg{data: m.ratingUC.Generate(year)}
	}
}

// loadGraphYear reloads everything the graph shows for m.graphYear.
func (m Model) loadGraphYear() tea.Cmd {
	return tea.Batch(m.loadContribution(), m.loadRatings())
}

func (m Model) setDayRating(date time.Time, score int) tea.Cmd {
	return func() tea.Msg {
		if err := m.ratingUC.SetRating(date, score); err != nil {
			return errMsg{err: err}
		}
		data := m.ratingUC.Generate(date.Year())
		return ratingLoadedMsg{data: data}
	}
}

func (m Model) saveNote(date time.Time, note string) tea.Cmd {
	return func() tea.Msg {
		if err := m.ratingUC.SetNote(date, note); err != nil {
			return errMsg{err: err}
		}
		data := m.ratingUC.Generate(date.Year())
		return noteSavedMsg{data: data}
	}
}

// exportRatingsCSV writes the currently viewed month's daily ratings (one row
// per calendar day, blank score for unrated days) to a CSV file in the
// working directory, named after that month.
func (m Model) exportRatingsCSV() tea.Cmd {
	date := m.contribution.CursorDate
	return func() tea.Msg {
		csvData, err := m.ratingUC.ExportMonthCSV(date.Year(), date.Month())
		if err != nil {
			return errMsg{err: err}
		}
		path := fmt.Sprintf("tocli-ratings-%s.csv", date.Format("2006-01"))
		if err := os.WriteFile(path, csvData, 0644); err != nil {
			return errMsg{err: fmt.Errorf("export failed: %w", err)}
		}
		return exportDoneMsg{path: path}
	}
}

func (m Model) loadDayDetail(date time.Time) tea.Cmd {
	return func() tea.Msg {
		events, _ := m.eventUC.GetEventsForDate(date)
		tasks, _ := m.taskUC.TasksCompletedOn(date)
		return dayDetailLoadedMsg{date: date, events: events, tasks: tasks}
	}
}

func (m Model) toggleTask() tea.Cmd {
	task := m.tasks.SelectedTask()
	if task == nil {
		return nil
	}

	taskID := task.ID
	listID := task.ListID

	if task.Status == domain.TaskOpen {
		return func() tea.Msg {
			if err := m.taskUC.CompleteTask(taskID, listID); err != nil {
				return errMsg{err: err}
			}
			lists, err := m.taskUC.ListTaskLists()
			if err != nil {
				return errMsg{err: err}
			}
			tasks, err := m.taskUC.ListAllTasks()
			if err != nil {
				return errMsg{err: err}
			}
			return tasksLoadedMsg{tasks: tasks, lists: lists}
		}
	}

	if task.Status == domain.TaskCompleted {
		return func() tea.Msg {
			if err := m.taskUC.ReopenTask(taskID, listID); err != nil {
				return errMsg{err: err}
			}
			lists, err := m.taskUC.ListTaskLists()
			if err != nil {
				return errMsg{err: err}
			}
			tasks, err := m.taskUC.ListAllTasks()
			if err != nil {
				return errMsg{err: err}
			}
			return tasksLoadedMsg{tasks: tasks, lists: lists}
		}
	}

	return nil
}

func (m *Model) deleteSelectedTask() tea.Cmd {
	task := m.tasks.SelectedTask()
	if task == nil {
		return nil
	}
	taskID, listID := task.ID, task.ListID
	return func() tea.Msg {
		if err := m.taskUC.DeleteTask(taskID, listID); err != nil {
			return errMsg{err: err}
		}
		lists, err := m.taskUC.ListTaskLists()
		if err != nil {
			return errMsg{err: err}
		}
		tasks, err := m.taskUC.ListAllTasks()
		if err != nil {
			return errMsg{err: err}
		}
		return tasksLoadedMsg{tasks: tasks, lists: lists}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) submitNewTask() tea.Cmd {
	if m.editTask != nil {
		return m.submitEditTask()
	}
	if len(m.taskLists) == 0 {
		return func() tea.Msg {
			return errMsg{err: fmt.Errorf("no task lists available")}
		}
	}
	due, err := usecase.ParseOptionalTaskDue(m.dueInput.Value())
	if err != nil {
		return func() tea.Msg {
			return errMsg{err: err}
		}
	}
	var duePtr *time.Time
	if due != nil {
		d := *due
		duePtr = &d
	}
	listID := m.taskLists[m.createListIndex].ID
	title := m.createInput.Value()
	return func() tea.Msg {
		_, err := m.taskUC.CreateTask(listID, title, duePtr)
		if err != nil {
			return errMsg{err: err}
		}
		lists, err := m.taskUC.ListTaskLists()
		if err != nil {
			return errMsg{err: err}
		}
		tasks, err := m.taskUC.ListAllTasks()
		if err != nil {
			return errMsg{err: err}
		}
		return createTaskDoneMsg{tasks: tasks, lists: lists}
	}
}

// submitEditTask saves the edited title and due date, then reloads every task so the list order
// (priority, due date) is recomputed. An empty due field removes the due date.
func (m Model) submitEditTask() tea.Cmd {
	due, err := usecase.ParseOptionalTaskDue(m.dueInput.Value())
	if err != nil {
		return func() tea.Msg { return errMsg{err: err} }
	}
	task := *m.editTask
	title := m.createInput.Value()
	return func() tea.Msg {
		if _, err := m.taskUC.UpdateTask(task.ID, task.ListID, title, due); err != nil {
			return errMsg{err: err}
		}
		lists, err := m.taskUC.ListTaskLists()
		if err != nil {
			return errMsg{err: err}
		}
		tasks, err := m.taskUC.ListAllTasks()
		if err != nil {
			return errMsg{err: err}
		}
		return createTaskDoneMsg{tasks: tasks, lists: lists, selectID: task.ID}
	}
}

func filterActiveTasks(tasks []domain.Task) []domain.Task {
	var open, done []domain.Task
	for _, t := range tasks {
		if t.Status == domain.TaskOpen {
			open = append(open, t)
		} else if t.CompletedAt != nil {
			today := time.Now().Truncate(24 * time.Hour)
			if t.CompletedAt.After(today) {
				done = append(done, t)
			}
		}
	}
	domain.SortOpenTasksByPriorityAndDue(open)
	domain.SortDoneTasksForDisplay(done)
	return append(open, done...)
}
