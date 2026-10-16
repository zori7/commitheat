// Package cmd wires the commitheat subcommands together.
package cmd

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"

	"commitheat/internal/render"
)

// Version is set at build time with -ldflags "-X commitheat/cmd.Version=...".
var Version = "dev"

// app holds global flags and injectable dependencies shared by subcommands.
type app struct {
	dir     string
	json    bool
	noColor bool

	out   io.Writer
	now   func() time.Time
	color func() termenv.Profile
	width func() int // output width in columns; 0 means unlimited
}

// today returns the local calendar date as midnight UTC, matching gitlog.Commit.Date.
func (a *app) today() time.Time {
	y, m, d := a.now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (a *app) painter() render.Painter {
	if a.noColor || os.Getenv("NO_COLOR") != "" {
		return render.NewPainter(termenv.Ascii)
	}
	return render.NewPainter(a.color())
}

// terminalWidth returns the width of stdout if it is a terminal, else 0
// so piped or redirected output is never squeezed.
func terminalWidth() int {
	fd := os.Stdout.Fd()
	if !term.IsTerminal(fd) {
		return 0
	}
	w, _, err := term.GetSize(fd)
	if err != nil {
		return 0
	}
	return w
}

func (a *app) writeJSON(v any) error {
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// NewRootCmd builds the commitheat command tree.
func NewRootCmd() *cobra.Command {
	a := &app{
		out:   os.Stdout,
		now:   time.Now,
		color: lipgloss.ColorProfile,
		width: terminalWidth,
	}
	return newRootCmd(a)
}

func newRootCmd(a *app) *cobra.Command {
	root := &cobra.Command{
		Use:           "commitheat",
		Short:         "Git statistics in your terminal: contribution heatmap, line counts, authors",
		Long:          "commitheat reads the local git history of a repository and shows a contribution\nheatmap, lines added and removed over time, and an author leaderboard.\nIt never modifies the repository.",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(a.out)

	f := root.PersistentFlags()
	f.StringVarP(&a.dir, "dir", "C", ".", "repository path")
	f.BoolVar(&a.json, "json", false, "machine-readable JSON output")
	f.BoolVar(&a.noColor, "no-color", false, "disable ANSI colors (also respects NO_COLOR)")

	root.AddCommand(newHeatmapCmd(a), newLinesCmd(a), newAuthorsCmd(a))
	return root
}
