// Package stats aggregates commits by day, week, month, year, and author.
package stats

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"commitheat/internal/gitlog"
)

// Stats are the totals for a group of commits.
type Stats struct {
	Commits int `json:"commits"`
	Added   int `json:"added"`
	Deleted int `json:"deleted"`
}

func (s *Stats) add(c gitlog.Commit) {
	s.Commits++
	s.Added += c.Added
	s.Deleted += c.Deleted
}

// Daily groups commits by calendar day.
func Daily(commits []gitlog.Commit) map[time.Time]Stats {
	days := make(map[time.Time]Stats)
	for _, c := range commits {
		s := days[c.Date]
		s.add(c)
		days[c.Date] = s
	}
	return days
}

// Period is the size of a Rollup bucket.
type Period int

const (
	Day Period = iota
	Week
	Month
	Year
)

// ParsePeriod accepts day, week, month, or year.
func ParsePeriod(s string) (Period, error) {
	switch strings.ToLower(s) {
	case "day":
		return Day, nil
	case "week":
		return Week, nil
	case "month":
		return Month, nil
	case "year":
		return Year, nil
	}
	return 0, fmt.Errorf("invalid period %q: want day, week, month, or year", s)
}

// Start returns the first day of the period containing t. Weeks start on Monday.
func (p Period) Start(t time.Time) time.Time {
	y, m, d := t.Date()
	switch p {
	case Week:
		offset := (int(t.Weekday()) + 6) % 7 // days since Monday
		return time.Date(y, m, d-offset, 0, 0, 0, 0, time.UTC)
	case Month:
		return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	case Year:
		return time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (p Period) next(start time.Time) time.Time {
	switch p {
	case Week:
		return start.AddDate(0, 0, 7)
	case Month:
		return start.AddDate(0, 1, 0)
	case Year:
		return start.AddDate(1, 0, 0)
	}
	return start.AddDate(0, 0, 1)
}

// Label formats the period starting at start, e.g. 2026-10-16, 2026-W42, 2026-10, 2026.
func (p Period) Label(start time.Time) string {
	switch p {
	case Week:
		y, w := start.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	case Month:
		return start.Format("2006-01")
	case Year:
		return start.Format("2006")
	}
	return start.Format(time.DateOnly)
}

// Bucket is one period of a Rollup.
type Bucket struct {
	Period string    `json:"period"`
	Start  time.Time `json:"start"`
	Stats
}

// Rollup groups commits into periods, oldest first. Periods with no commits
// between the first and last commit are included with zero stats.
func Rollup(commits []gitlog.Commit, p Period) []Bucket {
	if len(commits) == 0 {
		return nil
	}
	groups := make(map[time.Time]Stats)
	first, last := p.Start(commits[0].Date), p.Start(commits[0].Date)
	for _, c := range commits {
		start := p.Start(c.Date)
		s := groups[start]
		s.add(c)
		groups[start] = s
		if start.Before(first) {
			first = start
		}
		if start.After(last) {
			last = start
		}
	}
	var buckets []Bucket
	for t := first; !t.After(last); t = p.next(t) {
		buckets = append(buckets, Bucket{Period: p.Label(t), Start: t, Stats: groups[t]})
	}
	return buckets
}

// Author is one contributor's totals.
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Stats
}

// ByAuthor groups commits by email (case-insensitive), most commits first.
// The name shown is the one from the author's most recent commit.
func ByAuthor(commits []gitlog.Commit) []Author {
	index := make(map[string]int)
	var authors []Author
	var latest []time.Time
	for _, c := range commits {
		key := strings.ToLower(c.Email)
		i, ok := index[key]
		if !ok {
			i = len(authors)
			index[key] = i
			authors = append(authors, Author{Name: c.Name, Email: c.Email})
			latest = append(latest, c.Date)
		}
		if c.Date.After(latest[i]) {
			authors[i].Name = c.Name
			latest[i] = c.Date
		}
		authors[i].add(c)
	}
	sort.SliceStable(authors, func(i, j int) bool {
		a, b := authors[i], authors[j]
		if a.Commits != b.Commits {
			return a.Commits > b.Commits
		}
		if a.Added+a.Deleted != b.Added+b.Deleted {
			return a.Added+a.Deleted > b.Added+b.Deleted
		}
		return strings.ToLower(a.Email) < strings.ToLower(b.Email)
	})
	return authors
}

// Thresholds split non-zero daily commit counts into heatmap levels 1-4.
type Thresholds [3]int

// Quantiles computes thresholds at the 25th, 50th, and 75th percentiles of
// the non-zero counts, so each of levels 1-4 covers roughly a quarter of active days.
func Quantiles(counts []int) Thresholds {
	var nz []int
	for _, n := range counts {
		if n > 0 {
			nz = append(nz, n)
		}
	}
	if len(nz) == 0 {
		return Thresholds{}
	}
	sort.Ints(nz)
	at := func(q float64) int { return nz[int(q*float64(len(nz)-1))] }
	return Thresholds{at(0.25), at(0.50), at(0.75)}
}

// Level maps a daily count to 0 (none) through 4 (busiest).
func (t Thresholds) Level(n int) int {
	switch {
	case n <= 0:
		return 0
	case n <= t[0]:
		return 1
	case n <= t[1]:
		return 2
	case n <= t[2]:
		return 3
	}
	return 4
}
