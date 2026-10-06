package ui

// Below this width the layout stacks the panes vertically.
const minSplitWidth = 68

// box is the outer size (border included) of one pane card.
type box struct{ W, H int }

// Cards have a 1-cell border and Padding(0, 1): 4 columns and 2 rows of chrome.
func (b box) innerW() int { return max(1, b.W-4) }
func (b box) innerH() int { return max(1, b.H-2) }

// layout is the outer size of every pane for a given terminal size. A pane with a zero box does
// not fit and is not drawn.
type layout struct {
	stacked bool
	tasks   box
	// search is the one-row search box sitting directly above the agenda card (H == 0 when the
	// terminal is too short to spare the row; the / key still works).
	search                  box
	agenda, graph, progress box
}

// Minimum body heights (rows between header and status bar) to spend a row on the search box.
const (
	searchRowMinBodyWide    = 12
	searchRowMinBodyStacked = 14
)

// headerRows is the height of the header above the body.
const headerRows = 1

// searchRect is where the search box is on screen (absolute terminal coordinates), for the
// palette's travel animation to start from.
func (l layout) searchRect() (rect, bool) {
	if l.search.H <= 0 {
		return rect{}, false
	}
	if l.stacked {
		return rect{x: 0, y: headerRows + l.tasks.H, w: l.search.W, h: l.search.H}, true
	}
	return rect{x: l.tasks.W + 1, y: headerRows, w: l.search.W, h: l.search.H}, true
}

// bodyOuterLines is the vertical space between the one-line header and the one-line status bar.
func bodyOuterLines(termHeight int) int {
	if termHeight <= 0 {
		return 0
	}
	return max(0, termHeight-2)
}

// computeLayout is pure: it only depends on the terminal width and the body height, so every
// size can be unit-tested without rendering anything.
func computeLayout(w, bodyH int) layout {
	if w <= 0 || bodyH <= 0 {
		return layout{}
	}
	if w < minSplitWidth {
		s := 0
		if bodyH >= searchRowMinBodyStacked {
			s = 1
		}
		t, a, g, p := splitStackOuterHeights(bodyH - s)
		return layout{
			stacked:  true,
			tasks:    box{w, t},
			search:   box{w, s},
			agenda:   box{w, a},
			graph:    box{w, g},
			progress: box{w, p},
		}
	}
	left, right := splitColumnWidths(w)
	s := 0
	if bodyH >= searchRowMinBodyWide {
		s = 1
	}
	a, g, p := splitRightColumn(bodyH - s)
	return layout{
		tasks:    box{left, bodyH},
		search:   box{right, s},
		agenda:   box{right, a},
		graph:    box{right, g},
		progress: box{right, p},
	}
}

// splitRightColumn divides the right column's height between agenda, graph and the year
// progress card. The graph wants 15 outer rows (13 inner: title, subtitle, months, 7 days, legend); the
// agenda takes whatever is left, and the progress card shrinks first on short terminals.
func splitRightColumn(h int) (agenda, graph, progress int) {
	if h <= 0 {
		return 0, 0, 0
	}
	switch {
	case h >= 24:
		progress = 5 // title, bar, caption
	case h >= 9:
		progress = 4 // title, bar
	}
	rest := h - progress
	if rest < 19 {
		agenda = rest / 2
		return agenda, rest - agenda, progress
	}
	graph = max(15, rest*40/100)
	return rest - graph, graph, progress
}

func splitStackOuterHeights(total int) (tasks, agenda, contrib, progress int) {
	if total <= 0 {
		return 0, 0, 0, 0
	}
	// Fewer than four rows: give one row to the first N panes so the sum equals total (no forced 4-row minimum).
	if total < 4 {
		switch total {
		case 1:
			return 1, 0, 0, 0
		case 2:
			return 1, 1, 0, 0
		case 3:
			return 1, 1, 1, 0
		default:
			return 0, 0, 0, 0
		}
	}
	if total < 16 {
		q := total / 4
		r := total % 4
		tasks, agenda, contrib, progress = q, q, q, q
		for i := 0; i < r; i++ {
			switch i % 4 {
			case 0:
				tasks++
			case 1:
				agenda++
			case 2:
				contrib++
			default:
				progress++
			}
		}
		return tasks, agenda, contrib, progress
	}
	progress = min(8, max(5, total*13/100))
	rest := total - progress
	tasks = max(4, rest*40/100)
	agenda = max(3, rest*30/100)
	contrib = rest - tasks - agenda
	if contrib < 3 {
		need := 3 - contrib
		contrib = 3
		tasks = max(4, tasks-need)
		if tasks+agenda+contrib+progress > total {
			tasks = total - progress - agenda - contrib
		}
		if tasks < 4 {
			tasks = 4
			agenda = max(3, total-progress-contrib-tasks)
		}
	}
	return tasks, agenda, contrib, progress
}

func splitColumnWidths(termW int) (left, right int) {
	// Narrower task column on laptop-sized terminals → more space for agenda + graph.
	ratioPct := 38
	if termW < 120 {
		ratioPct = 30
	}
	if termW < 90 {
		ratioPct = 28
	}
	left = termW * ratioPct / 100
	minLeft, minRight := 18, 22
	if termW < 90 {
		minLeft = 16
		minRight = 20
	}
	if left < minLeft {
		left = minLeft
	}
	right = termW - left - 1
	if right < minRight {
		right = minRight
		left = termW - right - 1
		if left < 16 {
			left = 16
			right = termW - left - 1
		}
	}
	return left, right
}
