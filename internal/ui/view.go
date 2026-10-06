package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"

	"tocli/internal/ui/components"
)

// clipToLines keeps the first maxLines lines. Bubble Tea drops excess lines from the top of the
// view when the string is taller than the terminal, which hides the header — clipping the body
// avoids overflowing the agreed layout (1 + bodyOuter + 1 rows).
func clipToLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

// clampToWidth truncates each line to maxCells display width (ANSI-aware), matching Bubble Tea's
// per-line truncation so content is not clipped unevenly on the right.
func clampToWidth(s string, maxCells int) string {
	if maxCells <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], maxCells, "")
	}
	return strings.Join(lines, "\n")
}

// finalizeFrame enforces terminal w×h: clamp line widths, drop excess rows from the bottom, then
// pad with full-width blank lines so the alt screen fills to the bottom without leftover paint.
func (m Model) finalizeFrame(s string) string {
	w, h := m.width, m.height
	if w <= 0 {
		return s
	}
	s = clampToWidth(s, w)
	if h <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	pad := strings.Repeat(" ", w)
	for len(lines) < h {
		lines = append(lines, pad)
	}
	return strings.Join(lines, "\n")
}

// View declares the frame plus terminal state (alt screen, mouse) as Bubble Tea v2 expects.
func (m Model) View() tea.View {
	v := tea.NewView(zone.Scan(m.render()))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) render() string {
	if !m.ready {
		return "\n  Loading..."
	}

	// The dashboard is always drawn; modals are floated on top of it as a centered box.
	status := m.renderStatusBar()
	var box string
	switch {
	case m.mode == modeHelp:
		box = m.modalPanel(m.renderHelp(), 100)
	case m.mode == modeCreateTask:
		box = m.modalPanel(m.createTaskContent(), 78)
		status = m.renderCreateStatusBar()
	case m.mode == modeWriteNote:
		box = m.modalPanel(m.writeNoteContent(), 78)
		status = m.renderWriteNoteStatusBar()
	case m.mode == modeSearch:
		status = m.renderSearchStatusBar()
	case m.mode == modeEventDetail:
		box = m.modalPanel(m.eventDetailContent(), detailBoxMaxW)
		status = m.renderEventDetailStatusBar()
	}

	header := m.renderHeader()
	body := clipToLines(m.renderBody(), bodyOuterLines(m.height))
	frame := m.finalizeFrame(lipgloss.JoinVertical(lipgloss.Left, header, body, status))
	if m.mode == modeSearch {
		return m.overlaySearch(frame)
	}
	if box == "" {
		return frame
	}
	return m.overlay(frame, box)
}

func (m *Model) updateLayout() {
	l := computeLayout(m.width, bodyOuterLines(m.height))
	m.tasks.Width, m.tasks.Height = max(8, l.tasks.innerW()), l.tasks.innerH()
	m.agenda.Width, m.agenda.Height = max(8, l.agenda.innerW()), l.agenda.innerH()
	m.contribution.Width, m.contribution.Height = max(8, l.graph.innerW()), l.graph.innerH()
	m.progress.Width, m.progress.Height = max(8, l.progress.innerW()), l.progress.innerH()
	m.updateFocus()
}

func (m Model) renderBody() string {
	l := computeLayout(m.width, bodyOuterLines(m.height))

	tasks := m.renderCard(components.ZonePaneTasks, l.tasks, m.tasks.View(), m.activePane == TaskPane)
	agenda := m.renderCard(components.ZonePaneAgenda, l.agenda, m.agenda.View(), m.activePane == AgendaPane)
	graph := m.renderCard(components.ZonePaneGraph, l.graph, m.contribution.View(), m.activePane == GraphPane)
	progress := m.renderCard("", l.progress, m.progress.View(), false)
	search := m.renderSearchBox(l.search)

	if l.stacked {
		return joinNonEmpty(tasks, search, agenda, graph, progress)
	}
	right := joinNonEmpty(search, agenda, graph, progress)
	return lipgloss.JoinHorizontal(lipgloss.Top, tasks, " ", right)
}

// joinNonEmpty stacks the non-empty blocks vertically (panes that did not fit render as "").
func joinNonEmpty(blocks ...string) string {
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if b != "" {
			parts = append(parts, b)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderCard draws content inside a bordered card of exactly b's outer size. Content is clipped
// to the card's inner area first (lipgloss Height is only a minimum and Width wraps), so a card
// can never push its neighbours around. A non-empty zoneID makes the whole card clickable.
func (m Model) renderCard(zoneID string, b box, content string, active bool) string {
	if b.W <= 0 || b.H <= 0 {
		return ""
	}
	style := m.styles.Card
	if active {
		style = m.styles.CardActive
	}
	content = clampToWidth(clipToLines(content, b.innerH()), b.innerW())
	out := style.Width(b.W).Height(b.H).Render(content)
	if zoneID != "" {
		out = zone.Mark(zoneID, out)
	}
	return out
}
