package components

import (
	"fmt"

	"charm.land/bubbles/v2/progress"

	"tocli/internal/ui/theme"
	"tocli/internal/usecase"
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

// bar builds a bubbles progress bar for the current palette and width. It is rendered
// statically (ViewAs), so there is no animation state to keep between frames.
func (m ProgressBarModel) bar() progress.Model {
	// "  " indent + bar + " " + "100.0%" (6) = 9 cells of chrome.
	w := max(4, m.Width-10)
	b := progress.New(
		progress.WithWidth(w),
		progress.WithoutPercentage(),
		progress.WithFillCharacters(progress.DefaultFullCharFullBlock, '░'),
		progress.WithColors(m.styles.T.Primary, m.styles.T.Success),
	)
	b.EmptyColor = m.styles.T.Overlay
	return b
}

func (m ProgressBarModel) View() string {
	s := m.styles
	p := m.Progress

	title := s.Title.Render(fmt.Sprintf("  %d Progress", p.Year))
	pct := s.Percentage.Render(fmt.Sprintf("%.1f%%", p.Percentage))
	barLine := "  " + m.bar().ViewAs(p.Percentage/100) + " " + pct

	// Two rows leave room for the bar only; the caption needs a third.
	if m.Height > 0 && m.Height <= 2 {
		return title + "\n" + barLine
	}

	details := s.Dim.Render(fmt.Sprintf(
		"  Day %d of %d · %d days remaining",
		p.DaysPassed, p.TotalDays, p.DaysRemaining,
	))
	return title + "\n" + barLine + "\n" + details
}
