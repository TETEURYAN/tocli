package components

import (
	"tocli/internal/ui/theme"
	"tocli/internal/usecase"
	"fmt"
	"strings"
)

type ProgressBarModel struct {
	Progress usecase.YearProgress
	Width    int
	Height   int
	styles   theme.Styles
}

func NewProgressBarModel(s theme.Styles) ProgressBarModel {
	return ProgressBarModel{styles: s}
}

func (m ProgressBarModel) View() string {
	s := m.styles
	p := m.Progress

	title := s.Title.Render(fmt.Sprintf("  %d Progress", p.Year))

	barWidth := m.Width - 10
	if barWidth < 4 {
		barWidth = 4
	}

	filled := int(float64(barWidth) * p.Percentage / 100)
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	bar := s.ProgressFill.Render(strings.Repeat("█", filled)) +
		s.ProgressBg.Render(strings.Repeat("░", empty))

	pctStr := s.Percentage.Render(fmt.Sprintf("%.1f%%", p.Percentage))

	details := s.Dim.Render(fmt.Sprintf(
		"  Day %d of %d · %d days remaining",
		p.DaysPassed, p.TotalDays, p.DaysRemaining,
	))

	if m.Height > 0 && m.Height <= 2 {
		return strings.Join([]string{title, "  " + pctStr}, "\n")
	}
	if m.Height > 0 && m.Height <= 3 {
		return strings.Join([]string{
			title,
			"  " + bar + " " + pctStr,
		}, "\n")
	}

	return strings.Join([]string{
		title,
		"",
		"  " + bar + " " + pctStr,
		details,
	}, "\n")
}
