# rapt_cli

Rapt is a speed-reading CLI. It flashes one word at a time in the middle of
your terminal ([RSVP](https://en.wikipedia.org/wiki/Rapid_serial_visual_presentation)-style)
at an adjustable words-per-minute rate, and remembers where you left off in
each file.

## Features

* ingest a text file
* show one word at a time, proceeding at N ms per word
* user hotkeys
  * speed +/-
  * advance/retreat through the text
  * quit
  * load text file
* remember where the user left off in a file

## Install

Download a binary for your platform from the
[releases page](https://github.com/fingerskier/rapt_cli/releases), or build
from source:

```sh
go install github.com/fingerskier/rapt_cli@latest
```

## Usage

```sh
rapt book.txt          # open a file (resumes where you left off)
rapt                   # reopen the last file you were reading
rapt -wpm 450 book.txt # override the saved reading speed
```

The reader starts paused — press `space` to begin.

### Hotkeys

| Key             | Action                          |
| --------------- | ------------------------------- |
| `space`         | play / pause                    |
| `←` / `→` (`h` / `l`) | retreat / advance one word |
| `↑` / `↓` or `+` / `-` | speed up / slow down by 25 wpm |
| `r`             | restart from the beginning      |
| `o`             | open a different text file      |
| `q` / `Ctrl-C`  | quit (position is saved)        |

Reading speed and per-file positions are stored in
`~/.config/rapt/state.json` (or the platform equivalent).

## Development

```sh
go test ./...
go build -o rapt .
```

## Releasing

Push a version tag and CI builds binaries for Linux, macOS, and Windows
(amd64 + arm64) and attaches them to a GitHub release:

```sh
git tag v0.1.0
git push origin v0.1.0
```
