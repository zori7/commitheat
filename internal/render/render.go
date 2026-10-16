// Package render draws heatmaps and tables for the terminal.
package render

import (
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// GitHub's contribution colors, from no activity (level 0) to busiest (level 4).
var levelColors = [5]string{"#161b22", "#0e4429", "#006d32", "#26a641", "#39d353"}

// Without color, levels are told apart by glyph density instead.
var levelGlyphs = [5]string{"·", "░", "▒", "▓", "█"}

const (
	cell       = "■"
	addColor   = "#26a641"
	delColor   = "#f85149"
	mutedColor = "#8b949e"
)

// Painter applies colors for a terminal color profile.
type Painter struct {
	r *lipgloss.Renderer
}

// NewPainter returns a Painter for profile. termenv.Ascii disables color.
func NewPainter(profile termenv.Profile) Painter {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(profile)
	return Painter{r: r}
}

// Color reports whether this painter emits ANSI colors.
func (p Painter) Color() bool { return p.r.ColorProfile() != termenv.Ascii }

func (p Painter) fg(hex, s string) string {
	if !p.Color() {
		return s
	}
	return p.r.NewStyle().Foreground(lipgloss.Color(hex)).Render(s)
}

func (p Painter) level(l int) string {
	if !p.Color() {
		return levelGlyphs[l]
	}
	return p.fg(levelColors[l], cell)
}

// Thousands formats n with comma separators: 12430 -> "12,430".
func Thousands(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func padLeft(s string, width int) string {
	if n := width - lipgloss.Width(s); n > 0 {
		return strings.Repeat(" ", n) + s
	}
	return s
}

func padRight(s string, width int) string {
	if n := width - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}
