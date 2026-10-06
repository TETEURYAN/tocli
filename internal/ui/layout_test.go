package ui

import "testing"

func TestComputeLayoutFillsBody(t *testing.T) {
	for w := 10; w <= 220; w++ {
		for h := 1; h <= 90; h++ {
			l := computeLayout(w, h)
			boxes := map[string]box{"tasks": l.tasks, "search": l.search, "agenda": l.agenda, "graph": l.graph, "progress": l.progress}
			for name, b := range boxes {
				if b.W < 0 || b.H < 0 {
					t.Fatalf("%dx%d: %s has negative size %+v", w, h, name, b)
				}
			}

			if l.stacked != (w < minSplitWidth) {
				t.Fatalf("%dx%d: stacked=%v", w, h, l.stacked)
			}

			if l.stacked {
				if got := l.tasks.H + l.search.H + l.agenda.H + l.graph.H + l.progress.H; got != h {
					t.Fatalf("%dx%d: stacked heights sum to %d, want %d (%+v)", w, h, got, h, l)
				}
				for name, b := range boxes {
					if b.H > 0 && b.W != w {
						t.Fatalf("%dx%d: %s width %d, want %d", w, h, name, b.W, w)
					}
				}
				continue
			}

			if l.tasks.H != h {
				t.Fatalf("%dx%d: tasks height %d, want %d", w, h, l.tasks.H, h)
			}
			if got := l.search.H + l.agenda.H + l.graph.H + l.progress.H; got != h {
				t.Fatalf("%dx%d: right column heights sum to %d, want %d (%+v)", w, h, got, h, l)
			}
			if got := l.tasks.W + 1 + l.agenda.W; got != w {
				t.Fatalf("%dx%d: columns plus gap use %d cells, want %d", w, h, got, w)
			}
			if l.agenda.W != l.graph.W || l.graph.W != l.progress.W || (l.search.H > 0 && l.search.W != l.agenda.W) {
				t.Fatalf("%dx%d: right column widths differ: %+v", w, h, l)
			}
		}
	}
}

func TestComputeLayoutDegenerateSizes(t *testing.T) {
	for _, c := range [][2]int{{0, 0}, {-1, 10}, {80, 0}, {80, -3}} {
		if l := computeLayout(c[0], c[1]); l != (layout{}) {
			t.Errorf("computeLayout(%d, %d) = %+v, want zero layout", c[0], c[1], l)
		}
	}
}

func TestComputeLayoutTypicalTerminal(t *testing.T) {
	l := computeLayout(120, bodyOuterLines(36))
	if l.graph.innerH() < 13 {
		t.Errorf("graph inner height %d too small for the full grid + legend", l.graph.innerH())
	}
	if l.agenda.innerH() < 8 {
		t.Errorf("agenda inner height %d too small", l.agenda.innerH())
	}
	if l.progress.H == 0 {
		t.Error("progress card should fit on a 120x36 terminal")
	}
}

func TestBodyOuterLines(t *testing.T) {
	for in, want := range map[int]int{-5: 0, 0: 0, 1: 0, 2: 0, 3: 1, 36: 34} {
		if got := bodyOuterLines(in); got != want {
			t.Errorf("bodyOuterLines(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestSearchRowSitsAboveTheAgenda(t *testing.T) {
	// Wide: top of the right column, one row, same width as the agenda card.
	l := computeLayout(120, bodyOuterLines(36))
	if l.search.H != 1 || l.search.W != l.agenda.W {
		t.Fatalf("wide search box = %+v, want one row as wide as the agenda (%d)", l.search, l.agenda.W)
	}
	r, ok := l.searchRect()
	if !ok {
		t.Fatal("wide layout should have a search rect")
	}
	if r.x != l.tasks.W+1 || r.y != headerRows || r.h != 1 || r.w != l.agenda.W {
		t.Errorf("wide rect = %+v, want it at the top of the right column", r)
	}

	// Stacked: between the tasks card and the agenda card.
	l = computeLayout(60, bodyOuterLines(30))
	r, ok = l.searchRect()
	if !ok || r.x != 0 || r.y != headerRows+l.tasks.H || r.w != 60 {
		t.Errorf("stacked rect = %+v ok=%v, want directly under the tasks card", r, ok)
	}
}

func TestSearchRowIsDroppedOnShortTerminals(t *testing.T) {
	for _, c := range []struct{ w, bodyH int }{{120, searchRowMinBodyWide - 1}, {60, searchRowMinBodyStacked - 1}} {
		l := computeLayout(c.w, c.bodyH)
		if l.search.H != 0 {
			t.Errorf("%dx%d body: search row should be dropped, got %+v", c.w, c.bodyH, l.search)
		}
		if _, ok := l.searchRect(); ok {
			t.Errorf("%dx%d body: searchRect should report no room", c.w, c.bodyH)
		}
	}
	for _, c := range []struct{ w, bodyH int }{{120, searchRowMinBodyWide}, {60, searchRowMinBodyStacked}} {
		if l := computeLayout(c.w, c.bodyH); l.search.H != 1 {
			t.Errorf("%dx%d body: search row should fit, got %+v", c.w, c.bodyH, l.search)
		}
	}
}
