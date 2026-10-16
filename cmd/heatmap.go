package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"commitheat/internal/gitlog"
	"commitheat/internal/render"
	"commitheat/internal/stats"
)

type heatmapDay struct {
	Date    string `json:"date"`
	Commits int    `json:"commits"`
	Level   int    `json:"level"`
}

type heatmapJSON struct {
	From    string       `json:"from"`
	To      string       `json:"to"`
	Commits int          `json:"commits"`
	Days    []heatmapDay `json:"days"`
}

func newHeatmapCmd(a *app) *cobra.Command {
	var since, author string
	var all bool
	c := &cobra.Command{
		Use:   "heatmap",
		Short: "Show a GitHub-style contribution heatmap",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			today := a.today()
			from, err := parseSince(since, today)
			if err != nil {
				return err
			}
			if from.IsZero() {
				return fmt.Errorf("--since must not be empty")
			}
			commits, err := gitlog.Read(cmd.Context(), gitlog.Options{Dir: a.dir, Since: from, Author: author, All: all})
			if err != nil {
				return err
			}
			daily := stats.Daily(commits)

			if a.json {
				return a.writeJSON(heatmapData(daily, from, today, len(commits)))
			}

			// In a narrow terminal, show only the most recent weeks that fit.
			shown := render.FitFrom(from, today, a.width())
			caption := sinceCaption(since, from)
			total := len(commits)
			trimmed := shown.After(from)
			if trimmed {
				total = 0
				for _, c := range commits {
					if !c.Date.Before(shown) {
						total++
					}
				}
				caption = "since " + shown.Format(time.DateOnly)
			}
			fmt.Fprint(a.out, render.Heatmap(a.painter(), daily, shown, today))
			fmt.Fprintf(a.out, "%s %s %s\n", render.Thousands(total), plural(total, "commit"), caption)
			if trimmed {
				fmt.Fprintln(a.out, "widen the terminal to see more")
			}
			return nil
		},
	}
	c.Flags().StringVar(&since, "since", "1y", "start of the range: 30d, 2w, 6m, 1y, or YYYY-MM-DD")
	c.Flags().StringVar(&author, "author", "", "only commits whose author name or email matches this pattern")
	c.Flags().BoolVar(&all, "all", false, "include commits from all branches and refs, not just HEAD")
	return c
}

func heatmapData(daily map[time.Time]stats.Stats, from, to time.Time, total int) heatmapJSON {
	var counts []int
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		counts = append(counts, daily[d].Commits)
	}
	th := stats.Quantiles(counts)
	out := heatmapJSON{From: from.Format(time.DateOnly), To: to.Format(time.DateOnly), Commits: total}
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		n := daily[d].Commits
		out.Days = append(out.Days, heatmapDay{Date: d.Format(time.DateOnly), Commits: n, Level: th.Level(n)})
	}
	return out
}

func sinceCaption(since string, from time.Time) string {
	if m := relativeSince.FindStringSubmatch(since); m != nil && m[1] == "1" {
		unit := map[string]string{"d": "day", "w": "week", "m": "month", "y": "year"}[m[2]]
		return "in the last " + unit
	}
	if m := relativeSince.FindStringSubmatch(since); m != nil {
		unit := map[string]string{"d": "days", "w": "weeks", "m": "months", "y": "years"}[m[2]]
		return fmt.Sprintf("in the last %s %s", m[1], unit)
	}
	return "since " + from.Format(time.DateOnly)
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
