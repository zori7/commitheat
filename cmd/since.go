package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var relativeSince = regexp.MustCompile(`^(\d+)([dwmy])$`)

// parseSince turns "30d", "2w", "6m", "1y", or "2025-01-31" into a start date
// relative to today. An empty string means no lower bound (the zero time).
func parseSince(s string, today time.Time) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if m := relativeSince.FindStringSubmatch(s); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --since %q: %w", s, err)
		}
		switch m[2] {
		case "d":
			return today.AddDate(0, 0, -n), nil
		case "w":
			return today.AddDate(0, 0, -7*n), nil
		case "m":
			return today.AddDate(0, -n, 0), nil
		case "y":
			return today.AddDate(-n, 0, 0), nil
		}
	}
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since %q: want a duration like 30d, 2w, 6m, 1y or a date like 2025-01-31", s)
	}
	if t.After(today) {
		return time.Time{}, fmt.Errorf("invalid --since %q: date is in the future", s)
	}
	return t, nil
}
