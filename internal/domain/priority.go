package domain

import (
	"cmp"
	"slices"
	"strings"
)

// TaskPriority groups tasks for display order: urgent first, then important, then normal.
// Google Tasks has no priority field in the API; we infer it from list names and title prefixes.
type TaskPriority int

const (
	TaskPriorityUrgent TaskPriority = iota
	TaskPriorityImportant
	TaskPriorityNormal
)

// PriorityFromListName maps common Google Tasks / Calendar list titles to a priority band.
func PriorityFromListName(listName string) TaskPriority {
	n := strings.ToLower(strings.TrimSpace(listName))
	if matchesUrgentList(n) {
		return TaskPriorityUrgent
	}
	if matchesImportantList(n) {
		return TaskPriorityImportant
	}
	return TaskPriorityNormal
}

func matchesUrgentList(n string) bool {
	keywords := []string{
		"urgent", "urgente", "asap", "critical", "crítico", "critico",
		"alta prioridade", "high priority", "firefight",
	}
	for _, k := range keywords {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

func matchesImportantList(n string) bool {
	keywords := []string{
		"important", "importante", "star", "starred", "favorit", "priority", "prioridade",
		"focus", "foco",
	}
	for _, k := range keywords {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}

// TaskEffectivePriority returns title-based priority if set, otherwise from list name.
func TaskEffectivePriority(t Task) TaskPriority {
	title := strings.TrimSpace(t.Title)
	switch {
	case strings.HasPrefix(title, "[!]"), strings.HasPrefix(title, "🔴"):
		return TaskPriorityUrgent
	case strings.HasPrefix(title, "[*]"), strings.HasPrefix(title, "⭐"), strings.HasPrefix(title, "★"):
		return TaskPriorityImportant
	}
	return PriorityFromListName(t.ListName)
}

// SortOpenTasksByPriorityAndDue sorts in place: priority ascending, then due date (nil last), then title.
func SortOpenTasksByPriorityAndDue(tasks []Task) {
	slices.SortStableFunc(tasks, compareOpenByPriorityDue)
}

// SortDoneTasksForDisplay sorts in place: priority ascending, then completed time (newest first).
func SortDoneTasksForDisplay(tasks []Task) {
	slices.SortStableFunc(tasks, compareDoneByPriorityCompleted)
}

func compareOpenByPriorityDue(a, b Task) int {
	if c := cmp.Compare(int(TaskEffectivePriority(a)), int(TaskEffectivePriority(b))); c != 0 {
		return c
	}
	switch {
	case a.DueDate == nil && b.DueDate == nil:
		return strings.Compare(a.Title, b.Title)
	case a.DueDate == nil:
		return 1
	case b.DueDate == nil:
		return -1
	default:
		if c := a.DueDate.Compare(*b.DueDate); c != 0 {
			return c
		}
		return strings.Compare(a.Title, b.Title)
	}
}

func compareDoneByPriorityCompleted(a, b Task) int {
	if c := cmp.Compare(int(TaskEffectivePriority(a)), int(TaskEffectivePriority(b))); c != 0 {
		return c
	}
	switch {
	case a.CompletedAt == nil && b.CompletedAt == nil:
		return strings.Compare(a.Title, b.Title)
	case a.CompletedAt == nil:
		return 1
	case b.CompletedAt == nil:
		return -1
	default:
		if c := b.CompletedAt.Compare(*a.CompletedAt); c != 0 {
			return c
		}
		return strings.Compare(a.Title, b.Title)
	}
}
