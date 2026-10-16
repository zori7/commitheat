// Package gitlog runs `git log` and parses its output into commits.
package gitlog

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Commit is one non-merge commit with its line counts summed over all files.
type Commit struct {
	// Date is the author date as a calendar day, stored as midnight UTC.
	Date    time.Time
	Email   string
	Name    string
	Added   int
	Deleted int
}

// Options selects which commits to read.
type Options struct {
	Dir    string    // repository path; empty means the current directory
	Since  time.Time // zero means no lower bound
	Author string    // passed to git's --author (regex on name and email)
	All    bool      // read every ref instead of only HEAD
}

// %aE and %aN respect .mailmap, so one person's aliases merge into one author.
const prettyFormat = "@%ad|%aE|%aN"

// Args returns the arguments passed to git for opts.
func Args(opts Options) []string {
	args := []string{
		"log", "--no-merges", "--no-color", "--no-show-signature",
		"--date=short", "--pretty=format:" + prettyFormat, "--numstat",
	}
	if opts.All {
		args = append(args, "--all")
	}
	if !opts.Since.IsZero() {
		args = append(args, "--since="+opts.Since.Format(time.DateOnly))
	}
	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}
	return args
}

// Read runs git log in opts.Dir and returns the parsed commits, newest first.
// A repository with no commits yields an empty slice.
func Read(ctx context.Context, opts Options) ([]Commit, error) {
	cmd := exec.CommandContext(ctx, "git", Args(opts)...)
	cmd.Dir = opts.Dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("git not found on PATH")
		}
		return nil, fmt.Errorf("running git: %w", err)
	}

	commits, parseErr := Parse(out)
	// Drain the pipe so git can exit even if parsing stopped early.
	_, _ = io.Copy(io.Discard, out)

	if err := cmd.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "does not have any commits yet") {
			return nil, nil
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git log: %s", msg)
	}
	if parseErr != nil {
		return nil, parseErr
	}

	// git filters --since by committer date; we bucket by author date.
	if !opts.Since.IsZero() {
		since := opts.Since.UTC().Truncate(24 * time.Hour)
		kept := commits[:0]
		for _, c := range commits {
			if !c.Date.Before(since) {
				kept = append(kept, c)
			}
		}
		commits = kept
	}
	return commits, nil
}

// Parse reads output produced with the format from Args.
func Parse(r io.Reader) ([]Commit, error) {
	var commits []Commit
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "@") {
			c, err := parseHeader(line[1:])
			if err != nil {
				return commits, fmt.Errorf("git log line %d: %w", lineNo, err)
			}
			commits = append(commits, c)
			continue
		}
		if len(commits) == 0 {
			return nil, fmt.Errorf("git log line %d: numstat line before any commit header", lineNo)
		}
		added, deleted, err := parseNumstat(line)
		if err != nil {
			return commits, fmt.Errorf("git log line %d: %w", lineNo, err)
		}
		cur := &commits[len(commits)-1]
		cur.Added += added
		cur.Deleted += deleted
	}
	return commits, sc.Err()
}

// parseHeader parses "2026-10-16|me@example.com|My Name".
func parseHeader(s string) (Commit, error) {
	parts := strings.SplitN(s, "|", 3)
	if len(parts) != 3 {
		return Commit{}, fmt.Errorf("malformed commit header %q", s)
	}
	date, err := time.Parse(time.DateOnly, parts[0])
	if err != nil {
		return Commit{}, fmt.Errorf("bad commit date %q", parts[0])
	}
	return Commit{Date: date, Email: parts[1], Name: parts[2]}, nil
}

// parseNumstat parses "12\t3\tpath". Binary files show "-\t-\tpath" and count as zero.
func parseNumstat(line string) (added, deleted int, err error) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return 0, 0, fmt.Errorf("malformed numstat line %q", line)
	}
	if parts[0] == "-" && parts[1] == "-" {
		return 0, 0, nil
	}
	if added, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, fmt.Errorf("bad added count in %q", line)
	}
	if deleted, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, fmt.Errorf("bad deleted count in %q", line)
	}
	return added, deleted, nil
}
