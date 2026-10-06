package ui

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"tocli/internal/ui/components"
)

// barStyles are the text styles used on the header and status bars. Every segment carries the
// bar's Surface background explicitly: an ANSI reset after a styled segment would otherwise
// punch holes in the bar's own background.
type barStyles struct {
	Title, Muted, Key, Desc, Warn, Err, OK lipgloss.Style
}

func (m Model) barStyles() barStyles {
	t := m.styles.T
	on := func(fg color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fg).Background(t.Surface)
	}
	return barStyles{
		Title: on(t.Primary).Bold(true),
		Muted: on(t.Muted),
		Key:   on(t.Primary).Bold(true),
		Desc:  on(t.Muted),
		Warn:  on(t.Warning),
		Err:   on(t.Error),
		OK:    on(t.Success),
	}
}

var sgrReset = regexp.MustCompile(`\x1b\[0?m`)

// bar lays out one full-width line: left segment, filler, right segment.
func (m Model) bar(left, right string) string {
	bg := lipgloss.NewStyle().Background(m.styles.T.Surface)
	gap := bg.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right)-2)))
	line := left + gap + right

	// Styled segments end with an SGR reset, which also clears the bar background for whatever
	// follows (bubbles/help puts an unstyled space between each key and its label). Re-assert
	// the background after every reset so the bar stays one solid strip.
	if on, _, ok := strings.Cut(bg.Render("x"), "x"); ok && on != "" {
		line = sgrReset.ReplaceAllStringFunc(line, func(reset string) string { return reset + on })
	}
	// Inline(true) pads but never truncates, so cut to the inner width ourselves.
	line = ansi.Truncate(line, max(0, m.width-2), "")
	return bg.Width(m.width).Padding(0, 1).Inline(true).Render(line)
}

// hint is a one-off display binding (key + label) for the modal status bars.
func hint(k, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(k), key.WithHelp(k, desc))
}

func (m Model) helpModel(width int) help.Model {
	b := m.barStyles()
	h := help.New()
	h.ShortSeparator = "  "
	h.Styles = help.Styles{
		ShortKey:       b.Key,
		ShortDesc:      b.Desc,
		ShortSeparator: b.Desc,
		Ellipsis:       b.Desc,
	}
	h.SetWidth(width)
	return h
}

// fitHints renders the first tier of bindings that fits width; if even the last one does not,
// bubbles/help truncates it with an ellipsis.
func (m Model) fitHints(width int, tiers ...[]key.Binding) string {
	unlimited := m.helpModel(0)
	for _, tier := range tiers {
		if lipgloss.Width(unlimited.ShortHelpView(tier)) <= width {
			return unlimited.ShortHelpView(tier)
		}
	}
	return m.helpModel(width).ShortHelpView(tiers[len(tiers)-1])
}

// paneHints returns the key hints for the active pane, from most to least detailed. Every list
// ends with help + quit, which are the last things to be dropped.
func (m Model) paneHints() [][]key.Binding {
	k := m.keys
	switch m.activePane {
	case GraphPane:
		var daily []key.Binding
		if m.contribution.Mode == components.ModeDaily {
			daily = []key.Binding{k.WriteNote, k.ExportCSV, k.Rate}
		}
		with := func(b ...key.Binding) []key.Binding { return append(append([]key.Binding{}, daily...), b...) }
		return [][]key.Binding{
			with(k.WeekNav, k.DayNav, k.YearNav, k.ToggleGraphMode, k.Tab, k.Refresh, k.Search, k.Help, k.Quit),
			with(k.WeekNav, k.DayNav, k.YearNav, k.ToggleGraphMode, k.Tab, k.Help, k.Quit),
			with(k.Navigate, k.ToggleGraphMode, k.Tab, k.Help, k.Quit),
			with(k.ToggleGraphMode, k.Tab, k.Help, k.Quit), // keeps note/export/rate in daily mode
			{k.ToggleGraphMode, k.Tab, k.Help, k.Quit},
			{k.Tab, k.Help, k.Quit},
			{k.Help, k.Quit},
		}
	case AgendaPane:
		return [][]key.Binding{
			{k.Navigate, k.Details, k.Tab, k.Refresh, k.Search, k.Help, k.Quit},
			{k.Navigate, k.Details, k.Tab, k.Help, k.Quit},
			{k.Tab, k.Search, k.Help, k.Quit},
			{k.Tab, k.Help, k.Quit},
			{k.Help, k.Quit},
		}
	default:
		return [][]key.Binding{
			{k.Navigate, k.Tab, k.Space, k.NewTask, k.EditTask, k.DeleteTask, k.Refresh, k.Search, k.Help, k.Quit},
			{k.Tab, k.NewTask, k.EditTask, k.DeleteTask, k.Search, k.Help, k.Quit},
			{k.Tab, k.NewTask, k.DeleteTask, k.Search, k.Help, k.Quit},
			{k.Tab, k.NewTask, k.Help, k.Quit},
			{k.Tab, k.Help, k.Quit},
			{k.Help, k.Quit},
		}
	}
}

func (m Model) renderHeader() string {
	b := m.barStyles()
	title := b.Title.Padding(0, 1).Render("⚡ tocli")

	compact := m.width < 96 || m.height < 26
	mid := " productivity dashboard"
	if compact {
		mid = " · dashboard"
	}

	rightFmt := "Mon Jan 2, 15:04"
	if compact && m.width < 72 {
		rightFmt = "15:04"
	}
	left := title + b.Muted.Render(mid)
	right := b.Muted.Render(time.Now().Format(rightFmt))

	return m.bar(left, right)
}

func (m Model) renderStatusBar() string {
	b := m.barStyles()

	if m.mode == modeConfirmDelete {
		left := b.Warn.Render(" Delete this task? ")
		return m.bar(left, m.fitHints(m.width-lipgloss.Width(left)-3,
			[]key.Binding{hint("y", "confirm"), hint("n", "cancel")}))
	}

	paneName := [...]string{"tasks", "agenda", "graph"}[m.activePane]
	left := b.Key.Render(fmt.Sprintf(" %s ", paneName))
	switch {
	case m.loading:
		left += b.Warn.Render(" loading...")
	case m.err != nil:
		left += b.Err.Render(fmt.Sprintf(" %v", m.err))
	case m.notice != "":
		left += b.OK.Render(" " + m.notice)
	}
	return m.bar(left, m.fitHints(m.width-lipgloss.Width(left)-3, m.paneHints()...))
}

// modalStatusBar is the status line while a modal owns the keyboard.
func (m Model) modalStatusBar(label string, hints ...key.Binding) string {
	b := m.barStyles()
	left := b.Key.Render(" " + label + " ")
	if m.err != nil {
		left += b.Err.Render(fmt.Sprintf("%v ", m.err))
	}
	return m.bar(left, m.fitHints(m.width-lipgloss.Width(left)-3, hints))
}

func (m Model) renderCreateStatusBar() string {
	if m.editTask != nil {
		return m.modalStatusBar("edit task",
			hint("esc", "cancel"), hint("enter", "save"), hint("tab", "field"))
	}
	return m.modalStatusBar("new task",
		hint("esc", "cancel"), hint("enter", "save"), hint("tab", "field"), hint("[ ]", "list"))
}

func (m Model) renderWriteNoteStatusBar() string {
	return m.modalStatusBar("day note", hint("esc", "cancel"), hint("ctrl+s", "save"))
}
