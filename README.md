# commitheat

A small Go CLI that shows git statistics for a local repository in the terminal:

- a GitHub-style **contribution heatmap** (weeks × weekdays, 5 shades of green)
- **lines added / removed** grouped by day, week, month, or year
- an **author leaderboard**

It reads your local git history only. It never creates, rewrites, or pushes commits.

```
          Oct      Nov      Dec      Jan      Feb      Mar      Apr ...
     Mon  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ...
     Wed  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ...
     Fri  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ■ ■  ■ ■ ...
                                               Less ■ ■ ■ ■ ■ More
     820 commits in the last year
```

## Status

MVP. `heatmap`, `lines`, and `authors` work, with `--json` and `--no-color` output.

## Usage

```sh
commitheat heatmap [--since 1y] [--author me@example.com] [--all]
commitheat lines   --by day|week|month|year [--since 6m] [--author ...]
commitheat authors [--since 1y] [-n 20] [--all]

# --since takes 30d, 2w, 6m, 1y, or a date like 2025-01-31
# heatmap defaults to 1y; lines and authors default to all history
# lines defaults to --by month
# --all includes every branch and ref, not just HEAD

# global flags
  -C <path>     repository path (default: current directory)
  --json        machine-readable output
  --no-color    disable ANSI colors (also respects NO_COLOR)
```

Example output of `commitheat lines --by month`:

```
2026-08  +12,430  -3,210  ████████████▌
2026-09  + 8,102  -1,994  ████████
2026-10  + 2,371  -  540  ██▍
```

## How it works

commitheat runs the `git` binary rather than using a Go git library such as go-git, which is slow on large repositories:

```sh
git log --no-merges --date=short --pretty=format:'@%ad|%aE|%aN' --numstat
```

That output goes through three steps:

1. **parse**: build `[]Commit{Date, Email, Name, Added, Deleted}` (`%aE`/`%aN` apply `.mailmap`)
2. **aggregate**: group commits into `map[date]Stats`, then roll them up by week, month, or year
3. **render**: draw the heatmap with 5 quantile buckets in GitHub colors
   (`#161b22 #0e4429 #006d32 #26a641 #39d353`), or bar tables for line stats.
   Without color (piped output, `--no-color`, `NO_COLOR`) the levels use `· ░ ▒ ▓ █`.

## Layout

```
commitheat/
├── main.go
├── cmd/            # cobra commands: heatmap, lines, authors
├── internal/
│   ├── gitlog/     # run `git log` and parse it
│   ├── stats/      # aggregation by day/week/month/year/author
│   └── render/     # heatmap + tables (lipgloss)
└── go.mod
```

## Install

macOS and Linux (x86_64 or arm64):

```sh
curl -fsSL https://raw.githubusercontent.com/zori7/commitheat/main/install.sh | sh
```

This downloads the latest release binary, checks its SHA-256 checksum, and installs it to
`/usr/local/bin` if that is writable, otherwise to `~/.local/bin`. Options:

```sh
curl -fsSL .../install.sh | COMMITHEAT_VERSION=v0.1.0 sh        # a specific release
curl -fsSL .../install.sh | COMMITHEAT_INSTALL_DIR=~/bin sh     # a different directory
```

Requirements: `git` on `PATH`.

## Build from source

Requires Go 1.22+.

```sh
go build -o commitheat .          # binary in the current folder
# or
go install .                      # binary in $(go env GOPATH)/bin

go test ./...
```

## Releasing

Binaries are built locally and uploaded to GitHub Releases (they are not committed to the repo):

```sh
scripts/build-release.sh v0.1.0                   # dist/commitheat_<os>_<arch>.tar.gz + checksums.txt
gh release create v0.1.0 dist/* --generate-notes  # tags and uploads; install.sh picks up the latest
```

## License

MIT
