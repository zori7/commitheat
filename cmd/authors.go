package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"commitheat/internal/gitlog"
	"commitheat/internal/render"
	"commitheat/internal/stats"
)

func newAuthorsCmd(a *app) *cobra.Command {
	var since string
	var all bool
	var limit int
	c := &cobra.Command{
		Use:   "authors",
		Short: "Show an author leaderboard by commits and lines changed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			from, err := parseSince(since, a.today())
			if err != nil {
				return err
			}
			commits, err := gitlog.Read(cmd.Context(), gitlog.Options{Dir: a.dir, Since: from, All: all})
			if err != nil {
				return err
			}
			authors := stats.ByAuthor(commits)
			if limit > 0 && len(authors) > limit {
				authors = authors[:limit]
			}

			if a.json {
				if authors == nil {
					authors = []stats.Author{}
				}
				return a.writeJSON(authors)
			}
			if len(authors) == 0 {
				fmt.Fprintln(a.out, "no commits found")
				return nil
			}
			fmt.Fprint(a.out, render.Authors(a.painter(), authors, a.width()))
			return nil
		},
	}
	c.Flags().StringVar(&since, "since", "", "start of the range: 30d, 2w, 6m, 1y, or YYYY-MM-DD (default: all history)")
	c.Flags().BoolVar(&all, "all", false, "include commits from all branches and refs, not just HEAD")
	c.Flags().IntVarP(&limit, "limit", "n", 20, "show at most this many authors (0 for all)")
	return c
}
