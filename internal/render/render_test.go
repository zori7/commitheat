package render

import (
	"strings"
	"testing"
	"time"

	"github.com/muesli/termenv"

	"commitheat/internal/stats"
)

var plain = NewPainter(termenv.Ascii)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestThousands(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 12430: "12,430", 1234567: "1,234,567", -4500: "-4,500"} {
		if got := Thousands(n); got != want {
			t.Errorf("Thousands(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestBar(t *testing.T) {
	if got := bar(10, 10, maxBarWidth); got != strings.Repeat("█", maxBarWidth) {
		t.Errorf("full bar: %q", got)
	}
	if got := bar(5, 10, maxBarWidth); got != strings.Repeat("█", maxBarWidth/2) {
		t.Errorf("half bar: %q", got)
	}
	if got := bar(1, 1_000_000, maxBarWidth); got != "▏" {
		t.Errorf("tiny value should get a sliver, got %q", got)
	}
	if bar(0, 10, maxBarWidth) != "" {
		t.Error("zero should draw nothing")
	}
	if bar(10, 10, 0) != "" {
		t.Error("zero width should draw nothing")
	}
}

func TestHeatmapPlain(t *testing.T) {
	// Three weeks: Sun 2026-09-27 through Sat 2026-10-17, starting mid-week.
	from, to := day("2026-09-30"), day("2026-10-16")
	daily := map[time.Time]stats.Stats{
		day("2026-09-30"): {Commits: 1},
		day("2026-10-05"): {Commits: 2},
		day("2026-10-16"): {Commits: 9},
	}
	got := Heatmap(plain, daily, from, to)
	want := strings.Join([]string{
		"    Oct",
		"      · ·",
		"Mon   ▒ ·",
		"      · ·",
		"Wed ░ · ·",
		"    · · ·",
		"Fri · · █",
		"    · ·",
		"Less · ░ ▒ ▓ █ More",
		"",
	}, "\n")
	if got != want {
		t.Errorf("heatmap mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestHeatmapColorUsesGitHubPalette(t *testing.T) {
	p := NewPainter(termenv.TrueColor)
	got := Heatmap(p, nil, day("2026-10-01"), day("2026-10-07"))
	// #161b22 is rgb(22, 27, 34).
	if !strings.Contains(got, "38;2;22;27;34m■") {
		t.Errorf("expected level-0 truecolor cells, got:\n%q", got)
	}
}

func TestLinesPlain(t *testing.T) {
	got := Lines(plain, []stats.Bucket{
		{Period: "2026-08", Stats: stats.Stats{Added: 12430, Deleted: 3210}},
		{Period: "2026-09", Stats: stats.Stats{Added: 8102, Deleted: 1994}},
		{Period: "2026-10"},
	}, 0)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines:\n%s", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "2026-08  +12,430  -3,210  ") {
		t.Errorf("row 0: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "2026-09  + 8,102  -1,994  ") {
		t.Errorf("row 1: %q", lines[1])
	}
	if lines[2] != "2026-10  +     0  -    0" {
		t.Errorf("row 2: %q", lines[2])
	}
	if !strings.HasSuffix(lines[0], strings.Repeat("█", maxBarWidth)) {
		t.Errorf("largest bucket should have a full bar: %q", lines[0])
	}
}

func TestAuthorsPlain(t *testing.T) {
	got := Authors(plain, []stats.Author{
		{Name: "Ann", Email: "ann@example.com", Stats: stats.Stats{Commits: 3, Added: 1200, Deleted: 30}},
		{Name: "Bob", Email: "bob@example.com", Stats: stats.Stats{Commits: 1, Added: 5, Deleted: 0}},
	}, 0)
	want := strings.Join([]string{
		"#  Author                 Commits  Share   Added  Deleted",
		"1  Ann <ann@example.com>        3  75.0%  +1,200      -30",
		"2  Bob <bob@example.com>        1  25.0%      +5       -0",
		"",
	}, "\n")
	if got != want {
		t.Errorf("authors mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFitFrom(t *testing.T) {
	from, to := day("2025-10-16"), day("2026-10-16") // 53 week columns
	if got := FitFrom(from, to, 0); !got.Equal(from) {
		t.Errorf("unlimited width changed from: %s", got.Format(time.DateOnly))
	}
	if got := FitFrom(from, to, 200); !got.Equal(from) {
		t.Errorf("wide terminal changed from: %s", got.Format(time.DateOnly))
	}
	// 40 columns: label (4) + 18 weeks * 2 - 1 = 39 fits, 19 weeks would need 41.
	got := FitFrom(from, to, 40)
	if got.Weekday() != time.Sunday || !got.Equal(day("2026-06-14")) {
		t.Errorf("FitFrom(40) = %s (%s), want 2026-06-14 (Sunday)", got.Format(time.DateOnly), got.Weekday())
	}
	for _, line := range strings.Split(Heatmap(plain, nil, got, to), "\n") {
		if w := len([]rune(line)); w > 40 {
			t.Errorf("line is %d columns wide, want <= 40: %q", w, line)
		}
	}
	// Absurdly narrow terminals still get a minimum grid rather than nothing.
	if got := FitFrom(from, to, 5); !got.Equal(day("2026-09-20")) {
		t.Errorf("FitFrom(5) = %s, want 4 weeks from 2026-09-20", got.Format(time.DateOnly))
	}
}

func TestLinesNarrow(t *testing.T) {
	buckets := []stats.Bucket{
		{Period: "2026-08", Stats: stats.Stats{Added: 12430, Deleted: 3210}},
		{Period: "2026-09", Stats: stats.Stats{Added: 100, Deleted: 0}},
	}
	// Prefix "2026-08  +12,430  -3,210  " is 26 columns, leaving 10 for the bar.
	lines := strings.Split(strings.TrimRight(Lines(plain, buckets, 36), "\n"), "\n")
	if want := "2026-08  +12,430  -3,210  " + strings.Repeat("█", 10); lines[0] != want {
		t.Errorf("row 0: got %q, want %q", lines[0], want)
	}
	// Too narrow for any bar: rows stay intact, bars are dropped.
	for _, line := range strings.Split(strings.TrimRight(Lines(plain, buckets, 20), "\n"), "\n") {
		if strings.ContainsAny(line, "█▏▎▍▌▋▊▉") {
			t.Errorf("expected no bar at width 20: %q", line)
		}
	}
}

func TestAuthorsNarrow(t *testing.T) {
	authors := []stats.Author{
		{Name: "Alexandria Ocasio-Smith", Email: "alexandria@example.com", Stats: stats.Stats{Commits: 3}},
		{Name: "Bob", Email: "bob@example.com", Stats: stats.Stats{Commits: 1}},
	}
	got := Authors(plain, authors, 50)
	for _, line := range strings.Split(strings.TrimRight(got, "\n"), "\n") {
		if w := len([]rune(line)); w > 50 {
			t.Errorf("line is %d columns wide, want <= 50: %q", w, line)
		}
	}
	if !strings.Contains(got, "1  Alexandria Oca…  ") {
		t.Errorf("expected author column truncated to fit:\n%s", got)
	}
	// Numbers are never truncated.
	if !strings.Contains(got, "75.0%") || !strings.Contains(got, "25.0%") {
		t.Errorf("numeric columns should stay intact:\n%s", got)
	}
}
