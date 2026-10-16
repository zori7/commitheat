package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"commitheat/internal/gitlog"
	"commitheat/internal/render"
	"commitheat/internal/stats"
)

func newLinesCmd(a *app) *cobra.Command {
	var by, since, author string
	var all bool
	c := &cobra.Command{
		Use:   "lines",
		Short: "Show lines added and removed per day, week, month, or year",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			period, err := stats.ParsePeriod(by)
			if err != nil {
				return err
			}
			from, err := parseSince(since, a.today())
			if err != nil {
				return err
			}
			commits, err := gitlog.Read(cmd.Context(), gitlog.Options{Dir: a.dir, Since: from, Author: author, All: all})
			if err != nil {
				return err
			}
			buckets := stats.Rollup(commits, period)

			if a.json {
				if buckets == nil {
					buckets = []stats.Bucket{}
				}
				return a.writeJSON(buckets)
			}
			if len(buckets) == 0 {
				fmt.Fprintln(a.out, "no commits found")
				return nil
			}
			fmt.Fprint(a.out, render.Lines(a.painter(), buckets, a.width()))
			return nil
		},
	}
	c.Flags().StringVar(&by, "by", "month", "group by day, week, month, or year")
	c.Flags().StringVar(&since, "since", "", "start of the range: 30d, 2w, 6m, 1y, or YYYY-MM-DD (default: all history)")
	c.Flags().StringVar(&author, "author", "", "only commits whose author name or email matches this pattern")
	c.Flags().BoolVar(&all, "all", false, "include commits from all branches and refs, not just HEAD")
	return c
}
