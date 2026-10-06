package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"tocli/internal/ui/theme"
	"tocli/internal/usecase"
)

func TestHeatmapEmptyDaysAreDotsActiveDaysAreBlocks(t *testing.T) {
	m := NewContributionModel(theme.NewStyles(theme.Dark))
	m.Data = usecase.ContributionData{MaxCount: 4}

	if got := m.renderCell(0); !strings.Contains(got, emptyCell) || strings.Contains(got, filledCell) {
		t.Errorf("empty day rendered as %q, want the faint dot", got)
	}
	for _, n := range []int{1, 2, 3, 4} {
		if got := m.renderCell(n); !strings.Contains(got, filledCell) {
			t.Errorf("day with %d tasks rendered as %q, want a block", n, got)
		}
	}
	if got := m.renderRatingCell(0); !strings.Contains(got, emptyCell) {
		t.Errorf("unrated day rendered as %q, want the faint dot", got)
	}
	if got := m.renderRatingCell(3); !strings.Contains(got, filledCell) {
		t.Errorf("rated day rendered as %q, want a block", got)
	}
}

func TestGraphThresholdsMatchRenderedRows(t *testing.T) {
	m := NewContributionModel(theme.NewStyles(theme.Dark))
	m.Width = 120
	m.Focused = true

	// Full view: title, subtitle, blank, months, 7 weekdays, blank, legend.
	m.Height = 13
	if n := strings.Count(m.View(), "\n") + 1; n != 13 {
		t.Errorf("full graph is %d rows, want 13", n)
	}
	if !strings.Contains(m.View(), "Less") {
		t.Error("legend should be visible at Height 13")
	}

	m.Height = 12
	if strings.Contains(m.View(), "Less") {
		t.Error("legend should be dropped below Height 13")
	}
}

func TestProgressBarFitsItsWidthAndHeight(t *testing.T) {
	m := NewProgressBarModel(theme.NewStyles(theme.Dark))
	m.Progress = usecase.YearProgress{Year: 2026, Percentage: 76.4, DaysPassed: 279, TotalDays: 365, DaysRemaining: 86}

	for _, w := range []int{14, 30, 80, 150} {
		m.Width, m.Height = w, 3
		view := m.View()
		// The card clips the title and caption; the bar line must fit on its own.
		if bar := strings.Split(view, "\n")[1]; lipgloss.Width(bar) > w {
			t.Errorf("width %d: bar line is %d cells wide: %q", w, lipgloss.Width(bar), bar)
		}
		if n := strings.Count(view, "\n") + 1; n != 3 {
			t.Errorf("width %d: %d rows at Height 3, want title + bar + caption", w, n)
		}
	}

	m.Width, m.Height = 40, 2
	if n := strings.Count(m.View(), "\n") + 1; n != 2 {
		t.Errorf("Height 2: %d rows, want title + bar", n)
	}
	if !strings.Contains(m.View(), "76.4%") {
		t.Error("percentage label missing")
	}
}
