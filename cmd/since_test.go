package cmd

import (
	"testing"
	"time"
)

func TestParseSince(t *testing.T) {
	today := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"30d":        "2026-09-16",
		"2w":         "2026-10-02",
		"6m":         "2026-04-16",
		"1y":         "2025-10-16",
		"2025-01-31": "2025-01-31",
	}
	for in, want := range cases {
		got, err := parseSince(in, today)
		if err != nil {
			t.Errorf("parseSince(%q): %v", in, err)
			continue
		}
		if got.Format(time.DateOnly) != want {
			t.Errorf("parseSince(%q) = %s, want %s", in, got.Format(time.DateOnly), want)
		}
	}
	if got, err := parseSince("", today); err != nil || !got.IsZero() {
		t.Errorf("empty since: %v, %v", got, err)
	}
	for _, bad := range []string{"1h", "y", "-1d", "2026-13-01", "2027-01-01", "yesterday"} {
		if _, err := parseSince(bad, today); err == nil {
			t.Errorf("parseSince(%q): expected error", bad)
		}
	}
}
