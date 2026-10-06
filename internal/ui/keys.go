package ui

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Tab       key.Binding
	ShiftTab  key.Binding
	Enter     key.Binding
	Space     key.Binding
	Refresh   key.Binding
	Help      key.Binding
	Quit      key.Binding
	NewTask   key.Binding
	PrevList   key.Binding
	NextList   key.Binding
	DeleteTask key.Binding
	ToggleGraphMode key.Binding
	ExportCSV  key.Binding
	WriteNote  key.Binding
	Search     key.Binding
	Command    key.Binding
	EditTask   key.Binding
	PrevYear   key.Binding
	NextYear   key.Binding

	// Display-only groupings for the status bar hints (the real bindings above do the work).
	Navigate key.Binding
	Details  key.Binding
	YearNav  key.Binding
	WeekNav  key.Binding
	DayNav   key.Binding
	Rate     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev pane"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "done/reopen"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		NewTask: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new task"),
		),
		PrevList: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "prev list"),
		),
		NextList: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next list"),
		),
		DeleteTask: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		ToggleGraphMode: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "toggle mode"),
		),
		ExportCSV: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "export csv"),
		),
		WriteNote: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "day note"),
		),
		PrevYear: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous year"),
		),
		NextYear: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next year"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "details"),
		),
		YearNav: key.NewBinding(
			key.WithKeys("[", "]"),
			key.WithHelp("[ ]", "year"),
		),
		EditTask: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Command: key.NewBinding(
			key.WithKeys(":"),
			key.WithHelp(":", "commands"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Navigate: key.NewBinding(
			key.WithKeys("up", "down", "k", "j"),
			key.WithHelp("↑↓/jk", "navigate"),
		),
		WeekNav: key.NewBinding(
			key.WithKeys("left", "right", "h", "l"),
			key.WithHelp("←→", "week"),
		),
		DayNav: key.NewBinding(
			key.WithKeys("up", "down", "k", "j"),
			key.WithHelp("↑↓", "day"),
		),
		Rate: key.NewBinding(
			key.WithKeys("1", "2", "3", "4", "5"),
			key.WithHelp("1-5", "rate day"),
		),
	}
}
