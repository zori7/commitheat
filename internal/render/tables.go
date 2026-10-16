package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"commitheat/internal/stats"
)

const (
	maxBarWidth    = 40
	minAuthorWidth = 12 // the author column never shrinks below this
)

// Partial blocks in eighths, for smooth bar ends.
var eighths = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

// bar draws value/max scaled to width cells.
func bar(value, max, width int) string {
	if value <= 0 || max <= 0 || width <= 0 {
		return ""
	}
	units := value * width * 8 / max
	if units == 0 {
		units = 1 // any activity gets a sliver
	}
	return strings.Repeat("█", units/8) + eighths[units%8]
}

// Lines draws one row per bucket: period, lines added, lines deleted, and a
// bar proportional to total churn (added + deleted). Bars shrink to fit in
// width columns; width <= 0 means unlimited.
func Lines(p Painter, buckets []stats.Bucket, width int) string {
	var keyW, addW, delW, maxChurn int
	for _, bk := range buckets {
		keyW = max(keyW, len(bk.Period))
		addW = max(addW, len(Thousands(bk.Added)))
		delW = max(delW, len(Thousands(bk.Deleted)))
		maxChurn = max(maxChurn, bk.Added+bk.Deleted)
	}
	barWidth := maxBarWidth
	if width > 0 {
		// "<period>  +<added>  -<deleted>  "
		barWidth = min(barWidth, width-(keyW+2+1+addW+2+1+delW+2))
	}

	var b strings.Builder
	for _, bk := range buckets {
		line := fmt.Sprintf("%s  %s  %s",
			padRight(bk.Period, keyW),
			p.fg(addColor, "+"+padLeft(Thousands(bk.Added), addW)),
			p.fg(delColor, "-"+padLeft(Thousands(bk.Deleted), delW)),
		)
		if b := bar(bk.Added+bk.Deleted, maxChurn, barWidth); b != "" {
			line += "  " + p.fg(addColor, b)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// Authors draws a leaderboard: rank, author, commits, share of commits, and
// line counts. If the table is wider than width, the author column is
// truncated with "…"; width <= 0 means unlimited.
func Authors(p Painter, authors []stats.Author, width int) string {
	headers := []string{"#", "Author", "Commits", "Share", "Added", "Deleted"}
	total := 0
	for _, a := range authors {
		total += a.Commits
	}

	rows := make([][]string, len(authors))
	for i, a := range authors {
		who := a.Name
		if a.Email != "" {
			who += " <" + a.Email + ">"
		}
		rows[i] = []string{
			fmt.Sprint(i + 1),
			who,
			Thousands(a.Commits),
			fmt.Sprintf("%.1f%%", 100*float64(a.Commits)/float64(total)),
			"+" + Thousands(a.Added),
			"-" + Thousands(a.Deleted),
		}
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, r := range rows {
		for i, c := range r {
			widths[i] = max(widths[i], lipgloss.Width(c))
		}
	}
	if width > 0 {
		total := 2 * (len(widths) - 1) // column gaps
		for _, w := range widths {
			total += w
		}
		if over := total - width; over > 0 {
			widths[1] = max(widths[1]-over, minAuthorWidth)
		}
	}
	// Column 1 (author) is left-aligned; numbers are right-aligned.
	format := func(cells []string, paint func(col int, s string) string) string {
		out := make([]string, len(cells))
		for i, c := range cells {
			if i == 1 {
				c = padRight(ansi.Truncate(c, widths[i], "…"), widths[i])
			} else {
				c = padLeft(c, widths[i])
			}
			out[i] = paint(i, c)
		}
		return strings.TrimRight(strings.Join(out, "  "), " ")
	}

	var b strings.Builder
	b.WriteString(format(headers, func(_ int, s string) string { return p.fg(mutedColor, s) }))
	b.WriteByte('\n')
	for _, r := range rows {
		b.WriteString(format(r, func(col int, s string) string {
			switch col {
			case 4:
				return p.fg(addColor, s)
			case 5:
				return p.fg(delColor, s)
			}
			return s
		}))
		b.WriteByte('\n')
	}
	return b.String()
}
