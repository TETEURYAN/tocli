package theme

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// RatingColor interpolates linearly (in RGB space) between RatingLow (score
// 1, red) and RatingHigh (score 5, green) — a more saturated pair than
// Error/Success so the gradient reads clearly at a glance. Scores outside
// 1-5 — in particular 0, the "not rated" sentinel — return the same neutral
// color the contribution graph uses for empty days.
func (t Theme) RatingColor(score int) color.Color {
	if score < 1 || score > 5 {
		return t.GraphLvl0
	}
	frac := float64(score-1) / float64(MaxRatingScore-1)
	return lerpColor(t.RatingLow, t.RatingHigh, frac)
}

// MaxRatingScore mirrors domain.MaxRatingScore; kept local to avoid a
// theme -> domain import for a single constant.
const MaxRatingScore = 5

func lerpColor(a, b color.Color, t float64) color.Color {
	ar, ag, ab := toRGB8(a)
	br, bg, bb := toRGB8(b)
	r := lerpChannel(ar, br, t)
	g := lerpChannel(ag, bg, t)
	bl := lerpChannel(ab, bb, t)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, bl))
}

func lerpChannel(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

func toRGB8(c color.Color) (r, g, b uint8) {
	r16, g16, b16, _ := c.RGBA()
	return uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8)
}
