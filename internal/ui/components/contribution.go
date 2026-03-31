package components

import (
	"fmt"
	"strings"
	"time"
	"tocli/internal/ui/theme"
	"tocli/internal/usecase"

	"github.com/charmbracelet/lipgloss"
)

type monthInfo struct {
	firstWeekday int
	numDays      int
	numWeekCols  int
}

type gridLayout struct {
	year      int
	months    [12]monthInfo
	totalCols int
}

func computeGrid(year int) gridLayout {
	g := gridLayout{year: year}
	for mo := 0; mo < 12; mo++ {
		first := time.Date(year, time.Month(mo+1), 1, 0, 0, 0, 0, time.Local)
		last := first.AddDate(0, 1, -1)
		fw := int(first.Weekday())
		nd := last.Day()
		nw := (nd-1+fw)/7 + 1
		g.months[mo] = monthInfo{firstWeekday: fw, numDays: nd, numWeekCols: nw}
		g.totalCols += nw
	}
	return g
}

func (g gridLayout) dateAt(month, weekCol, weekday int) (time.Time, bool) {
	if month < 0 || month >= 12 || weekday < 0 || weekday > 6 {
		return time.Time{}, false
	}
	mi := g.months[month]
	if weekCol < 0 || weekCol >= mi.numWeekCols {
		return time.Time{}, false
	}
	day := weekCol*7 + weekday - mi.firstWeekday + 1
	if day < 1 || day > mi.numDays {
		return time.Time{}, false
	}
	return time.Date(g.year, time.Month(month+1), day, 0, 0, 0, 0, time.Local), true
}

func (g gridLayout) posFor(d time.Time) (month, weekCol, weekday int) {
	month = int(d.Month()) - 1
	weekday = int(d.Weekday())
	weekCol = (d.Day() - 1 + g.months[month].firstWeekday) / 7
	return
}

func colsInRange(g gridLayout, start, end int) int {
	w := 0
	for mo := start; mo < end; mo++ {
		w += g.months[mo].numWeekCols
	}
	return w
}

func visibleMonths(g gridLayout, curMonth, avail int) (int, int) {
	start, end := curMonth, curMonth+1
	for {
		expanded := false
		if start > 0 && colsInRange(g, start-1, end) <= avail {
			start--
			expanded = true
		}
		if end < 12 && colsInRange(g, start, end+1) <= avail {
			end++
			expanded = true
		}
		if !expanded {
			break
		}
	}
	return start, end
}

type ContributionModel struct {
	Data       usecase.ContributionData
	Width      int
	Height     int
	Focused    bool
	CursorDate time.Time
	styles     theme.Styles
}

func NewContributionModel(s theme.Styles) ContributionModel {
	now := time.Now()
	return ContributionModel{
		styles:     s,
		CursorDate: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
	}
}

func (m ContributionModel) SelectedDate() time.Time {
	return m.CursorDate
}

func (m *ContributionModel) ClampCursor() {
	year := m.Data.Year
	if year == 0 {
		year = time.Now().Year()
	}
	first := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	last := time.Date(year, 12, 31, 0, 0, 0, 0, time.Local)
	if m.CursorDate.Before(first) || m.CursorDate.After(last) {
		now := time.Now()
		if now.Year() == year {
			m.CursorDate = time.Date(year, now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		} else {
			m.CursorDate = first
		}
	}
}

func (m *ContributionModel) cursorYear() int {
	if m.Data.Year != 0 {
		return m.Data.Year
	}
	return time.Now().Year()
}

func (m *ContributionModel) MoveRight() {
	next := m.CursorDate.AddDate(0, 0, 7)
	if next.Year() == m.cursorYear() {
		m.CursorDate = next
	}
}

func (m *ContributionModel) MoveLeft() {
	prev := m.CursorDate.AddDate(0, 0, -7)
	if prev.Year() == m.cursorYear() {
		m.CursorDate = prev
	}
}

func (m *ContributionModel) MoveDown() {
	next := m.CursorDate.AddDate(0, 0, 1)
	if next.Year() == m.cursorYear() {
		m.CursorDate = next
	}
}

func (m *ContributionModel) MoveUp() {
	prev := m.CursorDate.AddDate(0, 0, -1)
	if prev.Year() == m.cursorYear() {
		m.CursorDate = prev
	}
}

func (m ContributionModel) dayCount(d time.Time) int {
	if m.Data.DayCounts == nil {
		return 0
	}
	return m.Data.DayCounts[d.Format("2006-01-02")]
}

func (m ContributionModel) View() string {
	s := m.styles
	year := m.cursorYear()
	grid := computeGrid(year)

	const fullOverhead = 6
	const compactOverhead = 2

	avail := m.Width - fullOverhead
	useGaps := false
	compact := false
	startMonth, endMonth := 0, 12

	switch {
	case avail >= grid.totalCols+11:
		useGaps = true
	case avail >= grid.totalCols:
		// fits without gaps
	default:
		compact = true
		avail = m.Width - compactOverhead
		if avail < grid.totalCols {
			curMonth := int(m.CursorDate.Month()) - 1
			startMonth, endMonth = visibleMonths(grid, curMonth, avail)
		}
	}

	showSubtitle := m.Height >= 7
	showMonth := m.Height >= 8
	showLegend := m.Height >= 11

	var lines []string

	title := s.Title.Render(fmt.Sprintf("  %d Contribution Graph", year))
	lines = append(lines, title)

	if showSubtitle {
		if m.Focused {
			dateStr := m.CursorDate.Format("Mon, Jan 2")
			count := m.dayCount(m.CursorDate)
			sub := s.Subtitle.Render(fmt.Sprintf("  %d tasks this year", m.Data.Total)) +
				"  " + lipgloss.NewStyle().Foreground(theme.T.Primary).Bold(true).Render("▸ "+dateStr) +
				" " + s.Dim.Render(fmt.Sprintf("· %d tasks", count))
			lines = append(lines, sub)
		} else {
			lines = append(lines, s.Subtitle.Render(fmt.Sprintf("  %d tasks completed this year", m.Data.Total)))
		}
	}

	if showMonth {
		lines = append(lines, "")
		lines = append(lines, m.renderMonthHeaders(grid, startMonth, endMonth, useGaps, compact))
	}

	dayLabels := [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	curMo, curWC, curWD := grid.posFor(m.CursorDate)

	for wd := 0; wd < 7; wd++ {
		var row strings.Builder

		if compact {
			if wd%2 == 1 {
				row.WriteString(s.Dim.Render(string(dayLabels[wd][0])) + " ")
			} else {
				row.WriteString("  ")
			}
		} else {
			if wd%2 == 1 {
				row.WriteString(s.Dim.Render(fmt.Sprintf("%-3s ", dayLabels[wd])))
			} else {
				row.WriteString("    ")
			}
		}

		for mo := startMonth; mo < endMonth; mo++ {
			mi := grid.months[mo]
			for wc := 0; wc < mi.numWeekCols; wc++ {
				dt, valid := grid.dateAt(mo, wc, wd)
				if !valid {
					row.WriteString(" ")
					continue
				}
				count := m.dayCount(dt)
				isCursor := m.Focused && mo == curMo && wc == curWC && wd == curWD
				if isCursor {
					row.WriteString(lipgloss.NewStyle().
						Foreground(theme.T.Primary).Bold(true).Render("◆"))
				} else {
					row.WriteString(m.renderCell(count))
				}
			}
			if useGaps && mo < endMonth-1 {
				row.WriteString(" ")
			}
		}

		if compact {
			lines = append(lines, row.String())
		} else {
			lines = append(lines, "  "+row.String())
		}
	}

	if showLegend {
		lines = append(lines, "")
		lines = append(lines, m.renderLegend(compact))
	}

	return strings.Join(lines, "\n")
}

func (m ContributionModel) renderCell(count int) string {
	level := m.getLevel(count)
	var color lipgloss.Color
	switch level {
	case 0:
		color = theme.T.GraphLvl0
	case 1:
		color = theme.T.GraphLvl1
	case 2:
		color = theme.T.GraphLvl2
	case 3:
		color = theme.T.GraphLvl3
	default:
		color = theme.T.GraphLvl4
	}
	return lipgloss.NewStyle().Foreground(color).Render("█")
}

func (m ContributionModel) getLevel(count int) int {
	if count == 0 {
		return 0
	}
	mx := m.Data.MaxCount
	if mx == 0 {
		return 1
	}
	ratio := float64(count) / float64(mx)
	switch {
	case ratio <= 0.25:
		return 1
	case ratio <= 0.50:
		return 2
	case ratio <= 0.75:
		return 3
	default:
		return 4
	}
}

func (m ContributionModel) renderLegend(compact bool) string {
	s := m.styles
	less := s.Dim.Render("Less ")
	more := s.Dim.Render(" More")
	cells := []string{m.renderCell(0), m.renderCell(1)}
	if m.Data.MaxCount > 1 {
		cells = append(cells, m.renderCell(m.Data.MaxCount/2))
	}
	if m.Data.MaxCount > 2 {
		cells = append(cells, m.renderCell(m.Data.MaxCount*3/4))
	}
	cells = append(cells, m.renderCell(m.Data.MaxCount))
	indent := "    "
	if compact {
		indent = "  "
	}
	return indent + less + strings.Join(cells, "") + more
}

func (m ContributionModel) renderMonthHeaders(grid gridLayout, startMonth, endMonth int, useGaps, compact bool) string {
	names := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	var hdr strings.Builder

	if compact {
		hdr.WriteString("  ")
	} else {
		hdr.WriteString("      ")
	}

	for mo := startMonth; mo < endMonth; mo++ {
		w := grid.months[mo].numWeekCols
		name := names[mo]
		if w >= len(name) {
			hdr.WriteString(name)
			hdr.WriteString(strings.Repeat(" ", w-len(name)))
		} else {
			hdr.WriteString(name[:w])
		}
		if useGaps && mo < endMonth-1 {
			hdr.WriteString(" ")
		}
	}

	return m.styles.Dim.Render(hdr.String())
}
