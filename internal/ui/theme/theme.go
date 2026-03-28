package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Base      lipgloss.Color
	Surface   lipgloss.Color
	Overlay   lipgloss.Color
	Text      lipgloss.Color
	Subtle    lipgloss.Color
	Muted     lipgloss.Color
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	GraphLvl0 lipgloss.Color
	GraphLvl1 lipgloss.Color
	GraphLvl2 lipgloss.Color
	GraphLvl3 lipgloss.Color
	GraphLvl4 lipgloss.Color
}

var TokyoNight = Theme{
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
	GraphLvl0: lipgloss.Color("#292e42"),
	GraphLvl1: lipgloss.Color("#1e4620"),
	GraphLvl2: lipgloss.Color("#2ea043"),
	GraphLvl3: lipgloss.Color("#3fb950"),
	GraphLvl4: lipgloss.Color("#56d364"),
}

var T = TokyoNight

type Styles struct {
	App          lipgloss.Style
	PanelActive  lipgloss.Style
	Panel        lipgloss.Style
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

func NewStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Background(T.Base),

		PanelActive: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(T.Primary).
			Padding(1, 2),

		Panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(T.Overlay).
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Foreground(T.Primary).
			Bold(true).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(T.Muted).
			Italic(true),

		TaskOpen: lipgloss.NewStyle().
			Foreground(T.Text),

		TaskDone: lipgloss.NewStyle().
			Foreground(T.Muted).
			Strikethrough(true),

		TaskOverdue: lipgloss.NewStyle().
			Foreground(T.Error),

		TaskSelected: lipgloss.NewStyle().
			Foreground(T.Primary).
			Bold(true),

		EventTime: lipgloss.NewStyle().
			Foreground(T.Accent).
			Width(15),

		EventTitle: lipgloss.NewStyle().
			Foreground(T.Text),

		EventNow: lipgloss.NewStyle().
			Foreground(T.Success).
			Bold(true),

		EventPast: lipgloss.NewStyle().
			Foreground(T.Muted),

		Location: lipgloss.NewStyle().
			Foreground(T.Muted).
			Italic(true),

		StatusBar: lipgloss.NewStyle().
			Foreground(T.Subtle).
			Background(T.Surface).
			Padding(0, 1),

		HelpKey: lipgloss.NewStyle().
			Foreground(T.Primary).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(T.Muted),

		ProgressFill: lipgloss.NewStyle().
			Foreground(T.Success),

		ProgressBg: lipgloss.NewStyle().
			Foreground(T.Overlay),

		Percentage: lipgloss.NewStyle().
			Foreground(T.Accent).
			Bold(true),

		Dim: lipgloss.NewStyle().
			Foreground(T.Muted),
	}
}
