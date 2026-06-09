package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	defaultWPM = 300
	minWPM     = 60
	maxWPM     = 1500
	wpmStep    = 25
)

type mode int

const (
	modeRead mode = iota
	modePrompt
)

type app struct {
	state   *State
	words   []string
	path    string
	idx     int
	wpm     int
	paused  bool
	mode    mode
	input   []byte // file-path prompt buffer
	message string // transient status message
}

func main() {
	wpmFlag := flag.Int("wpm", 0, "words per minute (overrides saved setting)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "rapt — a speed-reading CLI\n\nUsage: rapt [flags] [file]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("rapt", version)
		return
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "rapt: must be run in an interactive terminal")
		os.Exit(1)
	}

	st := loadState()
	a := &app{state: st, wpm: st.WPM, paused: true}
	if a.wpm == 0 {
		a.wpm = defaultWPM
	}
	if *wpmFlag > 0 {
		a.wpm = clamp(*wpmFlag, minWPM, maxWPM)
	}

	path := flag.Arg(0)
	if path == "" {
		path = st.LastFile
	}
	if path != "" {
		if err := a.loadFile(path); err != nil {
			a.message = err.Error()
		}
	}

	if err := a.run(); err != nil {
		fmt.Fprintln(os.Stderr, "rapt:", err)
		os.Exit(1)
	}
}

func (a *app) run() error {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	// Alternate screen, hidden cursor; restored on exit.
	os.Stdout.WriteString("\x1b[?1049h\x1b[?25l")
	defer func() {
		os.Stdout.WriteString("\x1b[?25h\x1b[?1049l")
		term.Restore(fd, oldState)
	}()

	keys := make(chan keyEvent, 16)
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				close(keys)
				return
			}
			parseKeys(buf[:n], func(ev keyEvent) { keys <- ev })
		}
	}()

	timer := time.NewTimer(a.interval())
	defer timer.Stop()

	a.render()
	for {
		select {
		case ev, ok := <-keys:
			if !ok {
				a.saveAll()
				return nil
			}
			if quit := a.handleKey(ev, timer); quit {
				a.saveAll()
				return nil
			}
			a.render()
		case <-timer.C:
			if !a.paused && len(a.words) > 0 {
				if a.idx < len(a.words)-1 {
					a.idx++
				} else {
					a.paused = true
					a.message = "end of text"
					a.saveAll()
				}
				a.render()
			}
			timer.Reset(a.interval())
		}
	}
}

// handleKey processes one key event; it returns true when the app should quit.
func (a *app) handleKey(ev keyEvent, timer *time.Timer) bool {
	if ev.kind == kindCtrlC {
		return true
	}
	if a.mode == modePrompt {
		a.handlePromptKey(ev)
		return false
	}

	switch ev.kind {
	case kindUp:
		a.setWPM(a.wpm + wpmStep)
	case kindDown:
		a.setWPM(a.wpm - wpmStep)
	case kindLeft:
		a.step(-1)
	case kindRight:
		a.step(1)
	case kindChar:
		switch ev.ch {
		case 'q':
			return true
		case ' ':
			a.togglePause(timer)
		case '+', '=':
			a.setWPM(a.wpm + wpmStep)
		case '-', '_':
			a.setWPM(a.wpm - wpmStep)
		case 'h':
			a.step(-1)
		case 'l':
			a.step(1)
		case 'r':
			a.idx = 0
			a.message = ""
		case 'o':
			a.mode = modePrompt
			a.input = a.input[:0]
			a.paused = true
		}
	}
	return false
}

func (a *app) handlePromptKey(ev keyEvent) {
	switch ev.kind {
	case kindEsc:
		a.mode = modeRead
	case kindEnter:
		a.mode = modeRead
		path := strings.TrimSpace(string(a.input))
		if path == "" {
			return
		}
		if err := a.loadFile(path); err != nil {
			a.message = err.Error()
		}
	case kindBackspace:
		if len(a.input) > 0 {
			a.input = a.input[:len(a.input)-1]
		}
	case kindChar:
		a.input = append(a.input, ev.ch)
	}
}

func (a *app) togglePause(timer *time.Timer) {
	if len(a.words) == 0 {
		return
	}
	if a.paused && a.idx >= len(a.words)-1 {
		a.idx = 0 // restart after reaching the end
	}
	a.paused = !a.paused
	a.message = ""
	if !a.paused {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(a.interval())
	} else {
		a.saveAll()
	}
}

func (a *app) step(delta int) {
	if len(a.words) == 0 {
		return
	}
	a.idx = clamp(a.idx+delta, 0, len(a.words)-1)
	a.message = ""
}

func (a *app) setWPM(wpm int) {
	a.wpm = clamp(wpm, minWPM, maxWPM)
}

func (a *app) interval() time.Duration {
	return time.Minute / time.Duration(a.wpm)
}

func (a *app) loadFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	words := strings.Fields(string(data))
	if len(words) == 0 {
		return fmt.Errorf("%s: no words found", path)
	}
	a.saveAll() // remember position in the previous file
	a.words = words
	a.path = abs
	a.paused = true
	a.message = ""
	a.idx = a.state.Files[abs]
	if a.idx < 0 || a.idx >= len(words) {
		a.idx = 0
	}
	a.state.LastFile = abs
	return nil
}

func (a *app) saveAll() {
	a.state.WPM = a.wpm
	if a.path != "" {
		a.state.Files[a.path] = a.idx
		a.state.LastFile = a.path
	}
	if err := a.state.save(); err != nil {
		a.message = "could not save state: " + err.Error()
	}
}

func (a *app) render() {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		w, h = 80, 24
	}

	var sb strings.Builder
	sb.WriteString("\x1b[2J\x1b[H")

	if len(a.words) > 0 {
		word := a.words[a.idx]
		col := (w-utf8.RuneCountInString(word))/2 + 1
		if col < 1 {
			col = 1
		}
		fmt.Fprintf(&sb, "\x1b[%d;%dH%s", h/2, col, word)
	} else {
		msg := "no file loaded — press o to open one"
		fmt.Fprintf(&sb, "\x1b[%d;%dH%s", h/2, max(1, (w-len(msg))/2+1), msg)
	}

	status := a.statusLine()
	fmt.Fprintf(&sb, "\x1b[%d;1H\x1b[7m%s\x1b[0m", h-1, pad(status, w))
	hints := " space play/pause  ←/→ word  ↑/↓ speed  r restart  o open  q quit"
	fmt.Fprintf(&sb, "\x1b[%d;1H%s", h, truncate(hints, w))

	os.Stdout.WriteString(sb.String())
}

func (a *app) statusLine() string {
	if a.mode == modePrompt {
		return " open file: " + string(a.input) + "_"
	}
	var parts []string
	if a.path != "" {
		parts = append(parts, filepath.Base(a.path), fmt.Sprintf("%d/%d", a.idx+1, len(a.words)))
	}
	parts = append(parts, fmt.Sprintf("%d wpm", a.wpm))
	if a.paused {
		parts = append(parts, "paused")
	}
	if a.message != "" {
		parts = append(parts, a.message)
	}
	return " " + strings.Join(parts, "  ·  ")
}

func pad(s string, w int) string {
	s = truncate(s, w)
	return s + strings.Repeat(" ", w-utf8.RuneCountInString(s))
}

func truncate(s string, w int) string {
	if utf8.RuneCountInString(s) <= w {
		return s
	}
	runes := []rune(s)
	return string(runes[:w])
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
