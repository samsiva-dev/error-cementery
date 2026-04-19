# ⚰ Error Cemetery

A CLI tool to **bury** errors when you fix them and **dig them up** when history repeats itself.

Stop solving the same error twice. Record what broke and how you fixed it, then search your personal graveyard the next time it haunts you.

---

## Features

- **Three-pass search**: exact hash → BM25 full-text → Claude semantic re-ranking
- **Interactive TUI**: Bubbletea-powered forms for burying and browsing
- **Tag support**: Categorise errors for easier retrieval
- **Clipboard integration**: Pre-fill error text straight from your clipboard
- **Smart mode**: Optional Claude AI re-ranking for fuzzy / conceptual matches
- **Persistent storage**: SQLite database in your home directory

---

## Installation

### Build from source

```bash
git clone https://github.com/samsiva-dev/error-cemetery
cd error-cemetery
make install   # builds and copies binary to ./cemetery
```

Or just build the binary:

```bash
make build     # output: bin/cemetery
```

Add the binary to a directory on your `$PATH`.

---

## Commands

### `bury` — Record an error and its fix

```bash
cemetery bury            # open interactive form
cemetery bury --clip     # pre-fill error text from clipboard
```

You will be prompted for:
- The error message / description
- The fix you applied
- Optional tags

### `dig` — Search for a buried error

```bash
cemetery dig "connection refused"
cemetery dig --clip              # use clipboard content as query
cemetery dig --smart "..."       # enable Claude semantic re-ranking
```

Search uses a three-pass ranking strategy:

| Pass | Method | Trigger |
|------|--------|---------|
| 1 | Exact hash match | Always |
| 2 | BM25 full-text search (FTS5) | Always |
| 3 | Claude semantic re-ranking | `--smart` flag or `smart_mode = true` in config |

### `visit` — Browse the full graveyard

```bash
cemetery visit
```

Opens a scrollable TUI listing all buried errors with their fixes and metadata.

### `stats` — Show cemetery statistics

```bash
cemetery stats
```

Prints the total number of buried errors and the top 10 most-used tags.

### `config` — Open the config file

```bash
cemetery config
```

Creates the config file if it does not exist, then opens it in `$EDITOR` (falls back to `nano`).

---

## Configuration

Config is stored at `~/.config/cemetery/config.toml` (follows the OS config dir convention).

```toml
[cemetery]
db_path    = ""      # default: ~/.local/share/cemetery/cemetery.db
smart_mode = false   # set true to always use Claude re-ranking

[claude]
api_key = ""         # or set ANTHROPIC_API_KEY env var
model   = "claude-haiku-4-5-20251001"
```

The `ANTHROPIC_API_KEY` environment variable takes precedence over the value in the config file.

---

## Smart mode (Claude AI)

Smart mode uses the Anthropic API to semantically re-rank FTS candidates, making it useful when your query is conceptually related but doesn't share exact keywords with the stored error.

Enable it per-command with `--smart`, or permanently in the config:

```toml
[cemetery]
smart_mode = true
```

Smart mode requires a valid `api_key` under `[claude]` or the `ANTHROPIC_API_KEY` environment variable.

---

## Development

```bash
make build   # compile
make test    # run all tests
make clean   # remove build artifacts
```

**Tech stack**: Go · [Cobra](https://github.com/spf13/cobra) · [Bubbletea](https://github.com/charmbracelet/bubbletea) · [SQLite (modernc)](https://pkg.go.dev/modernc.org/sqlite) · [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go)
