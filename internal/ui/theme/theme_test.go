package theme

import (
	"reflect"
	"testing"
)

func TestPalettesAreComplete(t *testing.T) {
	for name, th := range map[string]Theme{"Dark": Dark, "Light": Light} {
		v := reflect.ValueOf(th)
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).IsNil() {
				t.Errorf("%s palette is missing %s", name, v.Type().Field(i).Name)
			}
		}
	}
}

func TestForBackground(t *testing.T) {
	if !reflect.DeepEqual(ForBackground(true), Dark) {
		t.Error("dark background should pick the Dark palette")
	}
	if !reflect.DeepEqual(ForBackground(false), Light) {
		t.Error("light background should pick the Light palette")
	}
}

func TestStylesCarryTheirTheme(t *testing.T) {
	if got := NewStyles(Light).T; !reflect.DeepEqual(got, Light) {
		t.Error("NewStyles(Light) should keep the Light theme on Styles.T")
	}
}

func TestRatingColor(t *testing.T) {
	for _, th := range []Theme{Dark, Light} {
		if !reflect.DeepEqual(th.RatingColor(0), th.GraphLvl0) {
			t.Error("unrated (0) should use the empty-cell color")
		}
		if !reflect.DeepEqual(th.RatingColor(6), th.GraphLvl0) {
			t.Error("out-of-range scores should use the empty-cell color")
		}
		lo, hi := th.RatingColor(1), th.RatingColor(MaxRatingScore)
		if reflect.DeepEqual(lo, hi) {
			t.Error("score 1 and score 5 must differ")
		}
	}
}
