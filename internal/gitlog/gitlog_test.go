package gitlog

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	in := strings.Join([]string{
		"@2026-10-16|ann@example.com|Ann",
		"10\t2\tmain.go",
		"-\t-\tlogo.png",
		"3\t0\tREADME.md",
		"",
		"@2026-10-15|bob@example.com|Bob | The Builder",
		"@2026-10-14|ann@example.com|Ann",
		"1\t1\told.go => new.go",
	}, "\n")

	got, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := []Commit{
		{Date: day("2026-10-16"), Email: "ann@example.com", Name: "Ann", Added: 13, Deleted: 2},
		{Date: day("2026-10-15"), Email: "bob@example.com", Name: "Bob | The Builder"},
		{Date: day("2026-10-14"), Email: "ann@example.com", Name: "Ann", Added: 1, Deleted: 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d commits, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("commit %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{
		"1\t2\tfile.go",               // numstat before header
		"@2026-13-01|a@b|A",           // bad date
		"@2026-10-16|a@b",             // missing name
		"@2026-10-16|a@b|A\nx\t1\tf",  // bad count
		"@2026-10-16|a@b|A\n1 2 file", // not tab separated
	} {
		if _, err := Parse(strings.NewReader(in)); err == nil {
			t.Errorf("Parse(%q): expected error", in)
		}
	}
}

func TestReadFromRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	git := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
		cmd.Env = append(cmd.Env, env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commit := func(date, email, file, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		git(nil, "add", file)
		ts := date + "T12:00:00+00:00"
		git([]string{
			"GIT_AUTHOR_NAME=" + email, "GIT_AUTHOR_EMAIL=" + email, "GIT_AUTHOR_DATE=" + ts,
			"GIT_COMMITTER_NAME=" + email, "GIT_COMMITTER_EMAIL=" + email, "GIT_COMMITTER_DATE=" + ts,
		}, "commit", "-q", "-m", "c")
	}

	git(nil, "init", "-q", "-b", "main")

	empty, err := Read(context.Background(), Options{Dir: dir})
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty repo: got %v, %v", empty, err)
	}

	commit("2025-01-01", "ann@example.com", "a.txt", "1\n2\n3\n")
	commit("2025-06-01", "bob@example.com", "a.txt", "1\n")

	all, err := Read(context.Background(), Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0].Email != "bob@example.com" || all[0].Deleted != 2 || all[1].Added != 3 {
		t.Fatalf("unexpected commits: %+v", all)
	}

	since, err := Read(context.Background(), Options{Dir: dir, Since: day("2025-03-01")})
	if err != nil || len(since) != 1 || since[0].Email != "bob@example.com" {
		t.Fatalf("since filter: got %+v, %v", since, err)
	}

	byAnn, err := Read(context.Background(), Options{Dir: dir, Author: "ann@"})
	if err != nil || len(byAnn) != 1 || byAnn[0].Email != "ann@example.com" {
		t.Fatalf("author filter: got %+v, %v", byAnn, err)
	}

	if _, err := Read(context.Background(), Options{Dir: t.TempDir()}); err == nil {
		t.Fatal("expected error outside a repository")
	}
}

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}
