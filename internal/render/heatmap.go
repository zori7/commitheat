package render

import (
	"strings"
	"time"

	"commitheat/internal/stats"
)

var weekdayLabels = [7]string{"", "Mon", "", "Wed", "", "Fri", ""}

const (
	labelWidth = 4 // "Mon "
	minWeeks   = 4 // never shrink the grid below this, even if it wraps
)

// FitFrom returns the start date to draw from so the heatmap for from..to
// fits in width columns, keeping the most recent weeks. It returns from
// unchanged when the grid already fits or width <= 0 (unknown or unlimited).
func FitFrom(from, to time.Time, width int) time.Time {
	if width <= 0 {
		return from
	}
	// Each week takes 2 columns ("■ "), minus the trailing space on the last.
	maxWeeks := max((width-labelWidth+1)/2, minWeeks)
	gridStart := from.AddDate(0, 0, -int(from.Weekday()))
	if weeks := int(to.Sub(gridStart).Hours()/24)/7 + 1; weeks <= maxWeeks {
		return from
	}
	lastWeek := to.AddDate(0, 0, -int(to.Weekday()))
	return lastWeek.AddDate(0, 0, -7*(maxWeeks-1))
}

// Heatmap draws one column per week (Sunday to Saturday, top to bottom) for
// the days from..to inclusive, followed by a Less/More legend.
// from and to are calendar days at midnight UTC.
func Heatmap(p Painter, daily map[time.Time]stats.Stats, from, to time.Time) string {
	gridStart := from.AddDate(0, 0, -int(from.Weekday()))
	weeks := int(to.Sub(gridStart).Hours()/24)/7 + 1

	var counts []int
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		counts = append(counts, daily[d].Commits)
	}
	th := stats.Quantiles(counts)

	var b strings.Builder
	b.WriteString(monthHeader(gridStart, from, to, weeks))
	b.WriteByte('\n')
	for row := 0; row < 7; row++ {
		line := padRight(weekdayLabels[row], labelWidth)
		for w := 0; w < weeks; w++ {
			d := gridStart.AddDate(0, 0, w*7+row)
			if d.Before(from) || d.After(to) {
				line += "  "
				continue
			}
			line += p.level(th.Level(daily[d].Commits)) + " "
		}
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteByte('\n')
	}

	legend := "Less "
	for l := 0; l < 5; l++ {
		legend += p.level(l) + " "
	}
	legend += "More"
	b.WriteString(padLeft(legend, labelWidth+weeks*2-1))
	b.WriteByte('\n')
	return b.String()
}

// monthHeader places a month name above the first week column containing the
// 1st of that month. Labels that would collide with the previous one are dropped.
func monthHeader(gridStart, from, to time.Time, weeks int) string {
	header := []rune(strings.Repeat(" ", labelWidth+weeks*2))
	nextFree := 0
	put := func(col int, label string) {
		if col < nextFree {
			return
		}
		copy(header[col:], []rune(label))
		nextFree = col + len(label) + 1
	}

	firstLabelWeek := -1
	for w := 0; w < weeks; w++ {
		for i := 0; i < 7; i++ {
			d := gridStart.AddDate(0, 0, w*7+i)
			if d.Day() == 1 && !d.Before(from) && !d.After(to) {
				if firstLabelWeek < 0 {
					firstLabelWeek = w
				}
				break
			}
		}
	}
	// Label the partial first month too, if there is room before the next label.
	if firstLabelWeek < 0 || firstLabelWeek >= 2 {
		put(labelWidth, from.Format("Jan"))
	}
	for w := 0; w < weeks; w++ {
		for i := 0; i < 7; i++ {
			d := gridStart.AddDate(0, 0, w*7+i)
			if d.Day() == 1 && !d.Before(from) && !d.After(to) {
				put(labelWidth+w*2, d.Format("Jan"))
				break
			}
		}
	}
	return strings.TrimRight(string(header), " ")
}
