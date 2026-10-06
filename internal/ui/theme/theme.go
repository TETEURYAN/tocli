package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Base       color.Color
	Surface    color.Color
	Overlay    color.Color
	Text       color.Color
	Subtle     color.Color
	Muted      color.Color
	Primary    color.Color
	Secondary  color.Color
	Accent     color.Color
	Success    color.Color
	Warning    color.Color
	Error      color.Color
	GraphLvl0  color.Color
	GraphLvl1  color.Color
	GraphLvl2  color.Color
	GraphLvl3  color.Color
	GraphLvl4  color.Color
	RatingLow  color.Color
	RatingHigh color.Color
}

// Dark is Tokyo Night; Light is Tokyo Night Day. The dashboard starts on Dark and switches once
// the terminal reports its background color (see ForBackground).
var Dark = Theme{
	Base:      lipgloss.Color("#1a1b26"),
	Surface:   lipgloss.Color("#24283b"),
	Overlay:   lipgloss.Color("#414868"),
	Text:      lipgloss.Color("#c0caf5"),
	Subtle:    lipgloss.Color("#a9b1d6"),
	Muted:     lipgloss.Color("#565f89"),
	Primary:   lipgloss.Color("#7aa2f7"),
	Secondary: lipgloss.Color("#bb9af7"),
	Accent:    lipgloss.Color("#7dcfff"),
	Success:   lipgloss.Color("#9ece6a"),
	Warning:   lipgloss.Color("#e0af68"),
	Error:     lipgloss.Color("#f7768e"),
	GraphLvl0: lipgloss.Color("#3b4261"),
	GraphLvl1: lipgloss.Color("#1e4620"),
	GraphLvl2: lipgloss.Color("#2ea043"),
	GraphLvl3: lipgloss.Color("#3fb950"),
	GraphLvl4: lipgloss.Color("#56d364"),
	// Deliberately more saturated than Error/Success (which stay muted for
	// body text) — the rating scale needs to read as a strong red→green
	// gradient at a glance, not a subtle tint.
	RatingLow:  lipgloss.Color("#ef4444"),
	RatingHigh: lipgloss.Color("#22c55e"),
}

var Light = Theme{
	Base:      lipgloss.Color("#e1e2e7"),
	Surface:   lipgloss.Color("#d0d5e3"),
	Overlay:   lipgloss.Color("#a1a6c5"),
	Text:      lipgloss.Color("#3760bf"),
	Subtle:    lipgloss.Color("#6172b0"),
	Muted:     lipgloss.Color("#848cb5"),
	Primary:   lipgloss.Color("#2e7de9"),
	Secondary: lipgloss.Color("#9854f1"),
	Accent:    lipgloss.Color("#007197"),
	Success:   lipgloss.Color("#587539"),
	Warning:   lipgloss.Color("#8c6c3e"),
	Error:     lipgloss.Color("#f52a65"),
	GraphLvl0: lipgloss.Color("#b4b9d0"),
	GraphLvl1: lipgloss.Color("#9be9a8"),
	GraphLvl2: lipgloss.Color("#40c463"),
	GraphLvl3: lipgloss.Color("#30a14e"),
	GraphLvl4: lipgloss.Color("#216e39"),

	RatingLow:  lipgloss.Color("#dc2626"),
	RatingHigh: lipgloss.Color("#16a34a"),
}

// ForBackground picks the palette that reads well on the terminal's background.
func ForBackground(isDark bool) Theme {
	if isDark {
		return Dark
	}
	return Light
}


type Styles struct {
	T            Theme
	App          lipgloss.Style
	PanelActive  lipgloss.Style
	Panel        lipgloss.Style
	Card         lipgloss.Style
	CardActive   lipgloss.Style
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	TaskOpen     lipgloss.Style
	TaskDone     lipgloss.Style
	TaskOverdue  lipgloss.Style
	TaskSelected lipgloss.Style
	EventTime    lipgloss.Style
	EventTitle   lipgloss.Style
	EventNow     lipgloss.Style
	EventPast    lipgloss.Style
	Location     lipgloss.Style
	StatusBar    lipgloss.Style
	HelpKey      lipgloss.Style
	HelpDesc     lipgloss.Style
	ProgressFill lipgloss.Style
	ProgressBg   lipgloss.Style
	Percentage   lipgloss.Style
	Dim          lipgloss.Style
}

func NewStyles(t Theme) Styles {
	return Styles{
		T: t,

		App: lipgloss.NewStyle().
			Background(t.Base),

		PanelActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(1, 2),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Overlay).
			Padding(1, 2),

		// Dashboard pane cards: tighter than the modal Panel so several fit in one column.
		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Overlay).
			Padding(0, 1),

		CardActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Primary).
			Padding(0, 1),

		Title: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		Subtitle: lipgloss.NewStyle().
			Foreground(t.Muted).
			Italic(true),

		TaskOpen: lipgloss.NewStyle().
			Foreground(t.Text),

		TaskDone: lipgloss.NewStyle().
			Foreground(t.Muted).
			Strikethrough(true),

		TaskOverdue: lipgloss.NewStyle().
			Foreground(t.Error),

		TaskSelected: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		EventTime: lipgloss.NewStyle().
			Foreground(t.Accent).
			Width(15),

		EventTitle: lipgloss.NewStyle().
			Foreground(t.Text),

		EventNow: lipgloss.NewStyle().
			Foreground(t.Success).
			Bold(true),

		EventPast: lipgloss.NewStyle().
			Foreground(t.Muted),

		Location: lipgloss.NewStyle().
			Foreground(t.Muted).
			Italic(true),

		StatusBar: lipgloss.NewStyle().
			Foreground(t.Subtle).
			Background(t.Surface).
			Padding(0, 1),

		HelpKey: lipgloss.NewStyle().
			Foreground(t.Primary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(t.Muted),

		ProgressFill: lipgloss.NewStyle().
			Foreground(t.Success),

		ProgressBg: lipgloss.NewStyle().
			Foreground(t.Overlay),

		Percentage: lipgloss.NewStyle().
			Foreground(t.Accent).
			Bold(true),

		Dim: lipgloss.NewStyle().
			Foreground(t.Muted),
	}
}
