package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ListCategory int

const (
	ListCatDefault ListCategory = iota
	ListCatWork
	ListCatJob
	ListCatPersonal
	ListCatLearning
)

func classifyListName(name string) ListCategory {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case "work", "trabalho", "office":
		return ListCatWork
	case "job", "jobs", "emprego":
		return ListCatJob
	case "personal", "pessoal", "home", "casa":
		return ListCatPersonal
	case "learning", "study", "estudos", "education":
		return ListCatLearning
	}
	if strings.Contains(n, "learn") || strings.Contains(n, "study") || strings.Contains(n, "course") || strings.Contains(n, "curso") {
		return ListCatLearning
	}
	if strings.Contains(n, "personal") || strings.Contains(n, "pessoal") {
		return ListCatPersonal
	}
	if strings.Contains(n, "job") {
		return ListCatJob
	}
	if strings.Contains(n, "work") || strings.Contains(n, "trabalho") {
		return ListCatWork
	}
	return ListCatDefault
}

// ListCategoryAccent is the list-type color used for the task marker and title.
func ListCategoryAccent(name string) lipgloss.Color {
	switch classifyListName(name) {
	case ListCatWork:
		return T.Primary
	case ListCatJob:
		return T.Warning
	case ListCatPersonal:
		return T.Success
	case ListCatLearning:
		return T.Secondary
	default:
		return T.Subtle
	}
}

// ListCategoryMarker is a single-column hint shown before the task title (W/J/P/L/·).
func ListCategoryMarker(name string) string {
	switch classifyListName(name) {
	case ListCatWork:
		return "W"
	case ListCatJob:
		return "J"
	case ListCatPersonal:
		return "P"
	case ListCatLearning:
		return "L"
	default:
		return "·"
	}
}
