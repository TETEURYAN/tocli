package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"

	"tocli/internal/usecase"
)

// The search palette: a box above the agenda that, when activated, glides to the middle of the
// screen as a search bar with the results listed underneath it.

const (
	zoneSearch = "search:open"

	searchBarH    = 3 // border + one input row + border
	searchMaxRows = 8 // result rows shown at once
	searchBoxMaxW = 72

	// searchAnimStep is the animation progress per frame; at ~60 fps this is about 100 ms.
	searchAnimStep   = 0.16
	searchFrameDelay = 16 * time.Millisecond

	// How far around today the palette looks for appointments.
	searchWindow = 365 * 24 * time.Hour
	// Upper bound on rendered/navigable hits; far above what anyone scrolls through.
	searchMaxResults = 200
)

type searchState struct {
	input   textinput.Model
	index   *usecase.EventIndex
	loading bool
	err     error
	results []usecase.EventMatch
	// commands are the hits when the query starts with ">" (command mode); then results is empty.
	commands []commandMatch
	sel      int
	anim     float64 // 0 = bar still on the search box, 1 = settled in the middle
	gen      int     // invalidates the tick chain of a previous opening
}

type searchLoadedMsg struct {
	index *usecase.EventIndex
	err   error
}

type searchTickMsg struct{ gen int }

func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.Placeholder = "Search events…  (> for commands)"
	ti.CharLimit = 120
	return ti
}

// ---- geometry -------------------------------------------------------------------------------

type rect struct{ x, y, w, h int }

func (r rect) contains(px, py int) bool {
	return px >= r.x && px < r.x+r.w && py >= r.y && py < r.y+r.h
}

// searchGeom is where the bar and the results box are drawn.
type searchGeom struct {
	bar     rect
	results rect // h == 0 while hidden
	rows    int  // result rows that fit inside results
}

func lerp(a, b int, t float64) int { return a + int(math.Round(float64(b-a)*t)) }

func easeOutCubic(p float64) float64 {
	p = min(max(p, 0), 1)
	q := 1 - p
	return 1 - q*q*q
}

// searchGeometry is pure. The bar starts on pill (the search box above the agenda) and travels to the
// final position, which centers the bar plus the full-height results area vertically; the box
// grows from the pill's width to its final width on the way. Results only appear once the bar has
// settled, and the bar's final position does not depend on how many hits there are, so it never
// jumps while typing.
func searchGeometry(termW, termH int, progress float64, pill rect, nResults int) searchGeom {
	boxW := min(searchBoxMaxW, termW-4)
	if termW < 24 {
		boxW = termW
	}
	boxW = max(boxW, 1)
	finalX := (termW - boxW) / 2

	maxRows := max(0, min(searchMaxRows, termH-searchBarH-3-2))
	reserved := searchBarH
	if maxRows > 0 {
		reserved += maxRows + 3 // results: rows + footer + border
	}
	finalY := min(max(0, (termH-reserved)/2), max(0, termH-searchBarH))

	e := easeOutCubic(progress)
	start := pill
	if start.w <= 0 {
		start = rect{x: finalX, y: 0, w: boxW}
	}
	g := searchGeom{bar: rect{
		x: lerp(start.x, finalX, e),
		y: lerp(start.y, finalY, e),
		w: lerp(start.w, boxW, e),
		h: searchBarH,
	}}

	if maxRows > 0 && progress >= 1 {
		g.rows = min(max(nResults, 1), maxRows)
		g.results = rect{x: g.bar.x, y: g.bar.y + searchBarH, w: g.bar.w, h: g.rows + 3}
	}
	return g
}

// ---- search box above the agenda -------------------------------------------------------------

// searchRect is where the search box is on screen, if the terminal has room for it.
func (m Model) searchRect() (rect, bool) {
	return computeLayout(m.width, bodyOuterLines(m.height)).searchRect()
}

// renderSearchBox draws the one-row search box that sits above the agenda card: a key chip and a
// prompt on the bar background, clickable as a whole. While the palette is open the box has left
// (the bar is drawn travelling from this spot to the middle of the screen), so its slot is blank.
func (m Model) renderSearchBox(b box) string {
	if b.W <= 0 || b.H <= 0 {
		return ""
	}
	t := m.styles.T
	blank := lipgloss.NewStyle().Width(b.W).Render("")
	if m.mode == modeSearch {
		return blank
	}

	chip := lipgloss.NewStyle().Foreground(t.Base).Background(t.Primary).Bold(true).Render(" / ")
	label := " search events…  (> commands)"
	switch {
	case b.W < 20:
		label = " search"
	case b.W < 36:
		label = " search events…"
	}
	on := lipgloss.NewStyle().Foreground(t.Muted).Background(t.Surface)
	row := chip + on.Render(label)
	if pad := b.W - lipgloss.Width(row); pad > 0 {
		row += on.Render(strings.Repeat(" ", pad))
	}
	return zone.Mark(zoneSearch, ansi.Truncate(row, b.W, ""))
}

// ---- opening, closing, loading ----------------------------------------------------------------

// openSearch opens the palette, optionally pre-filled (">" opens it straight into command mode).
func (m *Model) openSearch(prefill string) tea.Cmd {
	m.mode = modeSearch
	m.err = nil
	m.search.gen++
	m.search.anim = 0
	m.search.sel = 0
	m.search.results = nil
	m.search.commands = nil
	m.search.err = nil
	m.search.loading = true
	m.search.input.Reset()
	if prefill != "" {
		m.search.input.SetValue(prefill)
		m.search.input.CursorEnd()
		m.refreshSearchResults()
	}
	return tea.Batch(m.search.input.Focus(), m.loadSearchIndex(), searchTick(m.search.gen))
}

func (m *Model) closeSearch() {
	m.mode = modeNormal
	m.search.input.Blur()
	m.search.input.Reset()
	m.search.results = nil
	m.search.commands = nil
	m.search.sel = 0
}

func searchTick(gen int) tea.Cmd {
	return tea.Tick(searchFrameDelay, func(time.Time) tea.Msg { return searchTickMsg{gen: gen} })
}

// loadSearchIndex fetches a wide window of events once per opening; typing then filters in memory.
func (m Model) loadSearchIndex() tea.Cmd {
	return func() tea.Msg {
		if m.eventUC == nil {
			return searchLoadedMsg{index: usecase.NewEventIndex(nil)}
		}
		now := time.Now()
		events, err := m.eventUC.GetEventsInRange(now.Add(-searchWindow), now.Add(searchWindow))
		if err != nil {
			return searchLoadedMsg{err: err}
		}
		return searchLoadedMsg{index: usecase.NewEventIndex(events)}
	}
}

func (m *Model) refreshSearchResults() {
	q := m.search.input.Value()
	if isCommandQuery(q) {
		m.search.commands = m.matchCommands(commandQuery(q))
		m.search.results = nil
	} else {
		m.search.commands = nil
		m.search.results = m.search.index.Search(q, time.Now(), searchMaxResults)
	}
	m.search.sel = 0
}

// commandMode is true while the palette lists commands instead of events.
func (m Model) commandMode() bool { return isCommandQuery(m.search.input.Value()) }

// searchCount is how many rows the palette currently lists, in either mode.
func (m Model) searchCount() int {
	if m.commandMode() {
		return len(m.search.commands)
	}
	return len(m.search.results)
}

// ---- input ----------------------------------------------------------------------------------

func (m *Model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	g := m.currentSearchGeometry()
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.closeSearch()
		return m, nil
	case "enter":
		return m, m.openSearchResult()
	case "tab":
		// Details of the selected event, instead of jumping to its day.
		if !m.commandMode() && m.search.sel >= 0 && m.search.sel < len(m.search.results) {
			e := m.search.results[m.search.sel].Event
			m.closeSearch()
			m.openEventDetail(e)
		}
		return m, nil
	case "up", "ctrl+p":
		m.moveSearchSel(-1)
		return m, nil
	case "down", "ctrl+n":
		m.moveSearchSel(1)
		return m, nil
	case "pgup":
		m.moveSearchSel(-max(1, g.rows))
		return m, nil
	case "pgdown":
		m.moveSearchSel(max(1, g.rows))
		return m, nil
	}

	before := m.search.input.Value()
	var cmd tea.Cmd
	m.search.input, cmd = m.search.input.Update(msg)
	if m.search.input.Value() != before {
		m.refreshSearchResults()
	}
	return m, cmd
}

func (m *Model) moveSearchSel(delta int) {
	n := m.searchCount()
	if n == 0 {
		return
	}
	m.search.sel = min(max(m.search.sel+delta, 0), n-1)
}

// openSearchResult jumps to the selected appointment's day: it closes the palette, focuses the
// graph on that date (switching the graph's year when the event is in another year) and lets the
// agenda show the day.
func (m *Model) openSearchResult() tea.Cmd {
	if m.search.sel < 0 || m.search.sel >= m.searchCount() {
		return nil
	}
	if m.commandMode() {
		// Close first, so a command that opens a modal or changes the mode starts from the
		// dashboard rather than from the palette.
		run := m.search.commands[m.search.sel].cmd.run
		m.closeSearch()
		return run(m)
	}
	start := m.search.results[m.search.sel].Event.StartTime.In(time.Local)
	m.closeSearch()
	return m.goToDate(time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local))
}

func (m Model) currentSearchGeometry() searchGeom {
	pill, _ := m.searchRect()
	return searchGeometry(m.width, m.height, m.search.anim, pill, m.searchCount())
}

func (m *Model) handleSearchClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	mouse := msg.Mouse()
	g := m.currentSearchGeometry()
	switch {
	case g.results.h > 0 && g.results.contains(mouse.X, mouse.Y):
		// Rows start below the top border; the footer and bottom border are not clickable.
		row := mouse.Y - g.results.y - 1
		first := m.searchFirstRow(g.rows)
		if row >= 0 && row < g.rows && first+row < m.searchCount() {
			m.search.sel = first + row
			return m, m.openSearchResult()
		}
	case g.bar.contains(mouse.X, mouse.Y):
	default:
		m.closeSearch()
	}
	return m, nil
}

func (m *Model) handleSearchWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseWheelUp:
		m.moveSearchSel(-1)
	case tea.MouseWheelDown:
		m.moveSearchSel(1)
	}
	return m, nil
}

// searchFirstRow is the index of the first visible result: the window follows the selection.
func (m Model) searchFirstRow(rows int) int {
	if rows <= 0 || m.search.sel < rows {
		return 0
	}
	return m.search.sel - rows + 1
}

// ---- rendering ------------------------------------------------------------------------------

// dimFrame fades the dashboard behind the palette: faint on, re-asserted after every reset.
func dimFrame(s string) string {
	const faint = "\x1b[2m"
	return faint + sgrReset.ReplaceAllStringFunc(s, func(r string) string { return r + faint })
}

func (m Model) overlaySearch(frame string) string {
	frame = dimFrame(zone.Scan(frame))
	g := m.currentSearchGeometry()

	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(frame),
		lipgloss.NewLayer(m.renderSearchBar(g)).X(g.bar.x).Y(g.bar.y).Z(1),
	}
	switch {
	case g.results.h > 0 && strings.TrimSpace(m.search.input.Value()) != "":
		layers = append(layers, lipgloss.NewLayer(m.renderSearchResults(g)).X(g.results.x).Y(g.results.y).Z(1))
	case m.search.anim >= 1 && g.bar.y+searchBarH < m.height:
		// Nothing typed yet: a one-line hint under the bar.
		hint := m.styles.Dim.Render("type to search events · > for commands · esc close")
		hint = ansi.Truncate(hint, g.bar.w, "")
		layers = append(layers, lipgloss.NewLayer(hint).X(g.bar.x+1).Y(g.bar.y+searchBarH).Z(1))
	}
	return lipgloss.NewCompositor(layers...).Render()
}

func (m Model) renderSearchBar(g searchGeom) string {
	t := m.styles.T
	in := m.search.input
	inner := max(1, g.bar.w-4)
	in.SetWidth(max(1, inner-lipgloss.Width(in.Prompt)))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(0, 1).
		Width(g.bar.w).
		Height(searchBarH).
		Render(ansi.Truncate(in.View(), inner, ""))
}

func (m Model) renderSearchResults(g searchGeom) string {
	t := m.styles.T
	inner := max(1, g.results.w-4)
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Overlay).
		Padding(0, 1).
		Width(g.results.w)

	var lines []string
	switch {
	case m.commandMode():
		lines = m.commandLines(g, inner)
	case m.search.err != nil && m.search.index.Len() == 0:
		lines = []string{m.styles.TaskOverdue.Render("couldn't load events: " + m.search.err.Error())}
	case m.search.loading && len(m.search.results) == 0:
		lines = []string{m.styles.Dim.Render("loading events…")}
	case len(m.search.results) == 0:
		lines = []string{m.styles.Dim.Render(fmt.Sprintf("no events match “%s”", strings.TrimSpace(m.search.input.Value())))}
	default:
		first := m.searchFirstRow(g.rows)
		for i := first; i < min(first+g.rows, len(m.search.results)); i++ {
			lines = append(lines, m.renderSearchRow(m.search.results[i], i == m.search.sel, inner))
		}
	}
	for len(lines) < g.rows {
		lines = append(lines, "")
	}

	verb := "open"
	if m.commandMode() {
		verb = "run"
	}
	extra := ""
	if !m.commandMode() {
		extra = " · tab details"
	}
	footer := m.styles.Dim.Render("esc close · enter " + verb + extra)
	if n := m.searchCount(); n > 0 {
		footer = m.styles.Dim.Render(fmt.Sprintf("%d of %d · ↑↓ select · enter %s%s · esc close", m.search.sel+1, n, verb, extra))
	}
	lines = append(lines, ansi.Truncate(footer, inner, "…"))
	return style.Render(strings.Join(lines, "\n"))
}

// searchDate is the date column: "Tue Oct  6", or "Oct  6 2025" for another year.
func searchDate(t time.Time) string {
	t = t.In(time.Local)
	if t.Year() == time.Now().Year() {
		return t.Format("Mon Jan _2")
	}
	return t.Format("Jan _2 2006")
}

func (m Model) renderSearchRow(match usecase.EventMatch, selected bool, width int) string {
	t := m.styles.T
	e := match.Event

	bg := lipgloss.NewStyle()
	if selected {
		bg = bg.Background(t.Surface)
	}
	text := bg.Foreground(t.Text)
	muted := bg.Foreground(t.Muted)
	if e.IsPast() {
		text = bg.Foreground(t.Subtle)
	}
	hit := bg.Foreground(t.Accent).Bold(true)

	marker := bg.Render("  ")
	if selected {
		marker = bg.Foreground(t.Primary).Bold(true).Render("▸ ")
	}
	clock := "all day"
	if !e.AllDay {
		clock = e.StartTime.In(time.Local).Format("15:04")
	}
	prefix := marker + muted.Render(fmt.Sprintf("%-11s %-7s", searchDate(e.StartTime), clock)) + bg.Render("  ")

	room := max(1, width-lipgloss.Width(prefix))
	title := highlightRunes(e.Title, match.TitleHits, text, hit)
	line := prefix + ansi.Truncate(title, room, bg.Render("…"))
	if loc := strings.TrimSpace(e.Location); loc != "" {
		if left := room - lipgloss.Width(e.Title); left >= len(loc)+5 {
			line += muted.Render("  · " + loc)
		}
	}
	// Pad so the selection background spans the whole row.
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += bg.Render(strings.Repeat(" ", pad))
	}
	return ansi.Truncate(line, width, "")
}

// highlightRunes renders s with the runes at the given indexes in the hit style.
func highlightRunes(s string, hits []int, base, hit lipgloss.Style) string {
	if len(hits) == 0 {
		return base.Render(s)
	}
	set := make(map[int]bool, len(hits))
	for _, i := range hits {
		set[i] = true
	}
	var b strings.Builder
	var run []rune
	runHit := false
	flush := func() {
		if len(run) == 0 {
			return
		}
		if runHit {
			b.WriteString(hit.Render(string(run)))
		} else {
			b.WriteString(base.Render(string(run)))
		}
		run = run[:0]
	}
	for i, r := range []rune(s) {
		if set[i] != runHit {
			flush()
			runHit = set[i]
		}
		run = append(run, r)
	}
	flush()
	return b.String()
}

// searchHint documents the palette in the status bar.
func (m Model) renderSearchStatusBar() string {
	if m.commandMode() {
		return m.modalStatusBar("commands", hint("esc", "close"), hint("enter", "run"), hint("↑↓", "select"))
	}
	return m.modalStatusBar("search", hint("esc", "close"), hint("enter", "open"), hint("↑↓", "select"), hint("tab", "details"), hint(">", "commands"))
}

// commandLines renders the visible command rows (or the empty state).
func (m Model) commandLines(g searchGeom, inner int) []string {
	if len(m.search.commands) == 0 {
		return []string{m.styles.Dim.Render(fmt.Sprintf("no command matches “%s”", strings.TrimSpace(commandQuery(m.search.input.Value()))))}
	}
	var lines []string
	first := m.searchFirstRow(g.rows)
	for i := first; i < min(first+g.rows, len(m.search.commands)); i++ {
		lines = append(lines, m.renderCommandRow(m.search.commands[i], i == m.search.sel, inner))
	}
	return lines
}

// renderCommandRow is "▸ Title                     key": the title with the matched letters
// highlighted, the shortcut (if any) right-aligned.
func (m Model) renderCommandRow(cm commandMatch, selected bool, width int) string {
	t := m.styles.T
	bg := lipgloss.NewStyle()
	if selected {
		bg = bg.Background(t.Surface)
	}
	marker := bg.Render("  ")
	if selected {
		marker = bg.Foreground(t.Primary).Bold(true).Render("▸ ")
	}
	keyCell := ""
	if cm.cmd.key != "" {
		keyCell = bg.Foreground(t.Primary).Bold(true).Render(cm.cmd.key)
	}
	room := max(1, width-lipgloss.Width(marker)-lipgloss.Width(keyCell)-1)
	title := ansi.Truncate(highlightRunes(cm.cmd.title, cm.hits, bg.Foreground(t.Text), bg.Foreground(t.Accent).Bold(true)), room, bg.Render("…"))
	pad := max(1, width-lipgloss.Width(marker)-lipgloss.Width(title)-lipgloss.Width(keyCell))
	return ansi.Truncate(marker+title+bg.Render(strings.Repeat(" ", pad))+keyCell, width, "")
}
