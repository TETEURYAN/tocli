package components

import (
	"tocli/internal/ui/theme"
	"tocli/internal/usecase"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type ContributionModel struct {
	Data    usecase.ContributionData
	Width   int
	Height  int
	Focused bool
	styles  theme.Styles
}

func NewContributionModel(s theme.Styles) ContributionModel {
	return ContributionModel{styles: s}
}

func (m ContributionModel) maxWeeksFromWidth() int {
	w := m.Width - 6
	if w < 2 {
		w = 2
	}
	maxWeeks := w / 2
	if maxWeeks > usecase.WeeksInYear {
		maxWeeks = usecase.WeeksInYear
	}
	if maxWeeks < 1 {
		maxWeeks = 1
	}
	return maxWeeks
}

func (m ContributionModel) weekWindow(maxWeeks int) (startWeek int) {
	startWeek = 0
	now := time.Now()
	if now.Year() == m.Data.Year {
		currentWeek := (now.YearDay() - 1 + int(time.Date(m.Data.Year, 1, 1, 0, 0, 0, 0, time.Local).Weekday())) / 7
		if currentWeek >= maxWeeks {
			startWeek = currentWeek - maxWeeks + 1
		}
	}
	return startWeek
}

func (m ContributionModel) View() string {
	s := m.styles
	maxWeeks := m.maxWeeksFromWidth()
	startWeek := m.weekWindow(maxWeeks)

	showSubtitle := m.Height >= 7
	showMonth := m.Height >= 8
	showLegend := m.Height >= 10
	dayRows := 7
	if m.Height < 8 {
		dayRows = max(1, m.Height-1)
		if dayRows > 7 {
			dayRows = 7
		}
	}

	var lines []string
	title := s.Title.Render(fmt.Sprintf("  %d Contribution Graph", m.Data.Year))
	lines = append(lines, title)
	if showSubtitle {
		subtitle := s.Subtitle.Render(fmt.Sprintf("  %d tasks completed this year", m.Data.Total))
		lines = append(lines, subtitle)
	}

	if showMonth {
		lines = append(lines, "")
		monthLabels := m.renderMonthLabels(maxWeeks, startWeek)
		lines = append(lines, "  "+monthLabels)
	} else if showSubtitle && m.Height >= 8 {
		lines = append(lines, "")
	}

	dayLabels := [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	startDay := 0
	if dayRows < 7 {
		startDay = (7 - dayRows) / 2
	}
	for d := startDay; d < startDay+dayRows && d < 7; d++ {
		day := d
		var row strings.Builder

		if day%2 == 1 {
			row.WriteString(s.Dim.Render(fmt.Sprintf("%-3s ", dayLabels[day])))
		} else {
			row.WriteString("    ")
		}

		for week := startWeek; week < startWeek+maxWeeks && week < usecase.WeeksInYear; week++ {
			count := m.Data.Weeks[week][day]
			cell := m.renderCell(count)
			row.WriteString(cell)
		}

		lines = append(lines, "  "+row.String())
	}

	if showLegend {
		lines = append(lines, "")
		legend := m.renderLegend()
		lines = append(lines, "  "+legend)
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

	return lipgloss.NewStyle().Foreground(color).Render("██")
}

func (m ContributionModel) getLevel(count int) int {
	if count == 0 {
		return 0
	}
	max := m.Data.MaxCount
	if max == 0 {
		return 1
	}
	ratio := float64(count) / float64(max)
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

func (m ContributionModel) renderLegend() string {
	s := m.styles
	less := s.Dim.Render("Less ")
	more := s.Dim.Render(" More")

	cells := []string{
		m.renderCell(0),
		m.renderCell(1),
	}
	if m.Data.MaxCount > 1 {
		cells = append(cells, m.renderCell(m.Data.MaxCount/2))
	}
	if m.Data.MaxCount > 2 {
		cells = append(cells, m.renderCell(m.Data.MaxCount*3/4))
	}
	cells = append(cells, m.renderCell(m.Data.MaxCount))

	return "    " + less + strings.Join(cells, "") + more
}

func (m ContributionModel) renderMonthLabels(maxWeeks, startWeek int) string {
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	jan1 := time.Date(m.Data.Year, 1, 1, 0, 0, 0, 0, time.Local)
	offset := int(jan1.Weekday())

	labels := make([]byte, maxWeeks*2)
	for i := range labels {
		labels[i] = ' '
	}

	for mo := 0; mo < 12; mo++ {
		firstOfMonth := time.Date(m.Data.Year, time.Month(mo+1), 1, 0, 0, 0, 0, time.Local)
		dayOfYear := firstOfMonth.YearDay() - 1
		weekIdx := (dayOfYear + offset) / 7

		pos := (weekIdx - startWeek) * 2
		if pos >= 0 && pos+3 <= len(labels) {
			name := months[mo]
			for j := 0; j < len(name) && pos+j < len(labels); j++ {
				labels[pos+j] = name[j]
			}
		}
	}

	return m.styles.Dim.Render("    " + string(labels))
}
