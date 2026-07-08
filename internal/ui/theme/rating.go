package theme

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RatingColor interpolates linearly (in RGB space) between Error (score 1,
// red) and Success (score 5, green). Scores outside 1-5 — in particular 0,
// the "not rated" sentinel — return the same neutral color the contribution
// graph uses for empty days.
func RatingColor(score int) lipgloss.Color {
	if score < 1 || score > 5 {
		return T.GraphLvl0
	}
	t := float64(score-1) / float64(MaxRatingScore-1)
	return lerpColor(T.Error, T.Success, t)
}

// MaxRatingScore mirrors domain.MaxRatingScore; kept local to avoid a
// theme -> domain import for a single constant.
const MaxRatingScore = 5

func lerpColor(a, b lipgloss.Color, t float64) lipgloss.Color {
	ar, ag, ab := hexToRGB(string(a))
	br, bg, bb := hexToRGB(string(b))
	r := lerpChannel(ar, br, t)
	g := lerpChannel(ag, bg, t)
	bl := lerpChannel(ab, bb, t)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, bl))
}

func lerpChannel(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

func hexToRGB(hex string) (r, g, b uint8) {
	hex = strings.TrimPrefix(hex, "#")
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}
