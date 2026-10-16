package stats

import (
	"testing"
	"time"

	"commitheat/internal/gitlog"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func commit(date, email string, added, deleted int) gitlog.Commit {
	return gitlog.Commit{Date: day(date), Email: email, Name: email, Added: added, Deleted: deleted}
}

func TestDaily(t *testing.T) {
	got := Daily([]gitlog.Commit{
		commit("2026-10-16", "a", 1, 2),
		commit("2026-10-16", "b", 3, 4),
		commit("2026-10-15", "a", 5, 0),
	})
	if got[day("2026-10-16")] != (Stats{Commits: 2, Added: 4, Deleted: 6}) {
		t.Errorf("2026-10-16: %+v", got[day("2026-10-16")])
	}
	if got[day("2026-10-15")] != (Stats{Commits: 1, Added: 5}) {
		t.Errorf("2026-10-15: %+v", got[day("2026-10-15")])
	}
}

func TestPeriodStartAndLabel(t *testing.T) {
	d := day("2026-10-16") // a Friday
	cases := []struct {
		p           Period
		start, want string
	}{
		{Day, "2026-10-16", "2026-10-16"},
		{Week, "2026-10-12", "2026-W42"},
		{Month, "2026-10-01", "2026-10"},
		{Year, "2026-01-01", "2026"},
	}
	for _, c := range cases {
		start := c.p.Start(d)
		if !start.Equal(day(c.start)) {
			t.Errorf("%v start: got %s, want %s", c.p, start.Format(time.DateOnly), c.start)
		}
		if got := c.p.Label(start); got != c.want {
			t.Errorf("%v label: got %s, want %s", c.p, got, c.want)
		}
	}
	// Sunday belongs to the week that started the previous Monday.
	if got := Week.Start(day("2026-10-18")); !got.Equal(day("2026-10-12")) {
		t.Errorf("sunday week start: %s", got.Format(time.DateOnly))
	}
}

func TestRollupFillsGaps(t *testing.T) {
	got := Rollup([]gitlog.Commit{
		commit("2026-10-02", "a", 10, 1),
		commit("2026-08-20", "a", 5, 5),
		commit("2026-08-01", "b", 1, 0),
	}, Month)
	want := []Bucket{
		{Period: "2026-08", Start: day("2026-08-01"), Stats: Stats{Commits: 2, Added: 6, Deleted: 5}},
		{Period: "2026-09", Start: day("2026-09-01")},
		{Period: "2026-10", Start: day("2026-10-01"), Stats: Stats{Commits: 1, Added: 10, Deleted: 1}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d buckets: %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("bucket %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if Rollup(nil, Day) != nil {
		t.Error("empty rollup should be nil")
	}
}

func TestByAuthor(t *testing.T) {
	got := ByAuthor([]gitlog.Commit{
		{Date: day("2026-10-16"), Email: "Ann@Example.com", Name: "Ann New", Added: 1},
		{Date: day("2026-10-15"), Email: "bob@example.com", Name: "Bob", Added: 100},
		{Date: day("2026-10-14"), Email: "ann@example.com", Name: "Ann Old", Added: 2},
	})
	if len(got) != 2 {
		t.Fatalf("got %d authors: %+v", len(got), got)
	}
	if got[0].Name != "Ann New" || got[0].Commits != 2 || got[0].Added != 3 {
		t.Errorf("first author: %+v", got[0])
	}
	if got[1].Name != "Bob" || got[1].Commits != 1 {
		t.Errorf("second author: %+v", got[1])
	}
}

func TestQuantilesAndLevel(t *testing.T) {
	th := Quantiles([]int{0, 0, 1, 2, 3, 4, 5, 6, 7, 8})
	if th != (Thresholds{2, 4, 6}) {
		t.Fatalf("thresholds: %v", th)
	}
	for n, want := range map[int]int{0: 0, 1: 1, 2: 1, 3: 2, 4: 2, 5: 3, 6: 3, 7: 4, 100: 4} {
		if got := th.Level(n); got != want {
			t.Errorf("Level(%d) = %d, want %d", n, got, want)
		}
	}
	// A single active day lands in level 1, not the darkest level.
	if got := Quantiles([]int{0, 3, 0}).Level(3); got != 1 {
		t.Errorf("single active day level = %d", got)
	}
	if Quantiles(nil).Level(0) != 0 {
		t.Error("no activity should be level 0")
	}
}
